package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"cinema-backend/config"
	"cinema-backend/internal/delivery/http/middleware"
	v1 "cinema-backend/internal/delivery/http/v1"
	"cinema-backend/internal/delivery/ws"
	"cinema-backend/internal/domain"
	"cinema-backend/internal/repository/mongo_repo"
	"cinema-backend/internal/repository/redis_repo"
	"cinema-backend/internal/service"
	"cinema-backend/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)



// จัดเก็บข้อมูลประวัติกิจกรรม
type MongoAuditLogRepository struct {
	collection *mongo.Collection
}

func NewMongoAuditLogRepository(db *mongo.Database) *MongoAuditLogRepository {
	return &MongoAuditLogRepository{collection: db.Collection("audit_logs")}
}

func (r *MongoAuditLogRepository) Store(ctx context.Context, logEntry *domain.AuditLog) error {
	_, err := r.collection.InsertOne(ctx, logEntry)
	if err != nil {
		log.Printf("[MongoDB Async Worker] Failed to store audit log: %v", err)
		return err
	}
	log.Printf("[MongoDB Async Worker] Audit Log Stored: %s - %s", logEntry.Event, logEntry.Details)
	return nil
}

func main() {
	// (.env) ห้ามทำการ Hardcode 
	cfg := config.LoadConfig()

	// ป้องกันแอปพังถ้าไม่ได้ใส่ Secret ใน .env
	if cfg.JWTAccessSecret == "" {
		cfg.JWTAccessSecret = "fallback_access_key_cinema"
	}
	if cfg.JWTRefreshSecret == "" {
		cfg.JWTRefreshSecret = "fallback_refresh_key_cinema"
	}

	// เชื่อมต่อ MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)
	db := mongoClient.Database("cinema")

	// เชื่อมต่อระบบจัดการกุญแจ Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	seatRepo := mongo_repo.NewMongoSeatRepository(db)
	auditLogRepo := NewMongoAuditLogRepository(db)
	lockRepo := redis_repo.NewRedisLockRepository(redisClient)
	pubSubRepo := redis_repo.NewPubSubRepository(redisClient)

	// (Async Audit Logging Message Queue)
	go pubSubRepo.SubscribeAuditLog(context.Background(), auditLogRepo)

	// Real-time Hub (WebSocket)
	wsHub := ws.NewHub()
	go wsHub.Run()

	// Business Logic (Usecase)
	lineNotify := service.NewLineNotifyService(cfg.LineChannelToken, cfg.LineTargetUserID)
	bookingUsecase := usecase.NewBookingUsecase(seatRepo, lockRepo, pubSubRepo, lineNotify)

	userRepo := mongo_repo.NewMongoUserRepository(db)

	// (HTTP Handlers)
	authHandler := v1.NewAuthHandler(cfg, userRepo, db.Collection("audit_logs"))
	bookingHandler := v1.NewBookingHandler(bookingUsecase, wsHub, db.Collection("bookings"), db.Collection("audit_logs"))
	adminHandler := v1.NewAdminHandler(db.Collection("seats"), db.Collection("bookings"), db.Collection("audit_logs"))

	r := gin.Default()

	// CORS Setup
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Authorization, Accept-Encoding")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	apiV1 := r.Group("/api/v1")
	{
		auth := apiV1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.GET("/google", authHandler.LoginRedirect)
			auth.GET("/google/callback", authHandler.GoogleCallback)
			
			// รูทจำลองสำหรับ Development
			auth.GET("/mock", authHandler.DevMockLogin)
			auth.GET("/mock-choice", func(c *gin.Context) {
				c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`
					<html>
					<head><title>Cinema Login Choice</title><link href="https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css" rel="stylesheet"></head>
					<body class="bg-gray-900 text-white flex items-center justify-center min-h-screen">
						<div class="bg-gray-800 p-8 rounded-lg shadow-xl text-center max-w-md w-full">
							<h1 class="text-2xl font-bold mb-6">🔑 Google Auth Not Setup</h1>
							<p class="text-gray-400 mb-6">สามารถจำลองสิทธิ์การเข้าใช้งานเพื่อทดสอบระบบ Distributed Lock ได้ทันทีผ่านปุ่มด้านล่าง:</p>
							<div class="flex flex-col gap-3">
								<a href="/api/v1/auth/mock?email=user@example.com&redirect=true" class="bg-blue-600 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded transition">👤 Mock: user@example.com</a>
								<p class="text-gray-500 text-xs">Admin ได้เฉพาะ email ใน ADMIN_EMAILS (.env)</p>
							</div>
						</div>
					</body>
					</html>
				`))
			})
		}

		// Real-time
		apiV1.GET("/ws", func(c *gin.Context) {
			ws.ServeWS(wsHub, c)
		})

		apiV1.GET("/seats", func(c *gin.Context) {
			showID := c.DefaultQuery("show_id", "default-show")
			seats, _ := seatRepo.GetSeatMap(c.Request.Context(), showID)
			c.JSON(http.StatusOK, gin.H{"show_id": showID, "seats": seats})
		})

		// (USER Role)
		userRoute := apiV1.Group("/")
		userRoute.Use(middleware.AuthMiddleware(cfg.JWTAccessSecret, "USER"))
		{
			userRoute.POST("/seats/lock", bookingHandler.SelectSeat)
			userRoute.POST("/seats/confirm", bookingHandler.ConfirmPayment)
		}

		// (ADMIN Role)
		adminRoute := apiV1.Group("/admin")
		adminRoute.Use(middleware.AdminAuthMiddleware(cfg.JWTAccessSecret, cfg))
		{
			adminRoute.GET("/bookings", adminHandler.GetBookings)
			adminRoute.GET("/audit-logs", adminHandler.GetAuditLogs)
			adminRoute.GET("/dashboard", adminHandler.Dashboard)
		}
	}

	log.Printf("[Backend] Cinema Engine safely operating on port %s", cfg.Port)
	_ = r.Run(":" + cfg.Port)
}