package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AdminHandler struct {
	seatsColl     *mongo.Collection
	bookingsColl  *mongo.Collection
	auditLogsColl *mongo.Collection
}

func NewAdminHandler(seatsColl, bookingsColl, auditLogsColl *mongo.Collection) *AdminHandler {
	return &AdminHandler{
		seatsColl:     seatsColl,
		bookingsColl:  bookingsColl,
		auditLogsColl: auditLogsColl,
	}
}

func (h *AdminHandler) GetBookings(c *gin.Context) {
	// Default filter: ignore AVAILABLE seats
	filter := bson.M{"status": bson.M{"$ne": "AVAILABLE"}}

	if userID := c.Query("user_id"); userID != "" {
		filter["user_id"] = bson.M{"$regex": userID, "$options": "i"}
	}
	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}
	if showID := c.Query("show_id"); showID != "" {
		filter["show_id"] = showID
	}

	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}})
	cursor, err := h.seatsColl.Find(c.Request.Context(), filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookings"})
		return
	}
	defer cursor.Close(c.Request.Context())

	var bookings []map[string]interface{}
	if err = cursor.All(c.Request.Context(), &bookings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode bookings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bookings": bookings, "total": len(bookings)})
}

func (h *AdminHandler) GetAuditLogs(c *gin.Context) {
	filter := bson.M{}

	if event := c.Query("event"); event != "" {
		filter["event"] = bson.M{"$regex": event, "$options": "i"}
	}
	if userID := c.Query("user_id"); userID != "" {
		filter["user_id"] = userID
	}

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(100)
	cursor, err := h.auditLogsColl.Find(c.Request.Context(), filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch logs"})
		return
	}
	defer cursor.Close(c.Request.Context())

	var logs []map[string]interface{}
	if err = cursor.All(c.Request.Context(), &logs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": len(logs)})
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	bookingCount, _ := h.bookingsColl.CountDocuments(c.Request.Context(), bson.M{})
	logCount, _ := h.auditLogsColl.CountDocuments(c.Request.Context(), bson.M{})

	c.JSON(http.StatusOK, gin.H{
		"status":          "ok",
		"total_bookings":  bookingCount,
		"total_audit_logs": logCount,
	})
}
