package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Env                string
	Port               string
	MongoURI           string
	RedisAddr          string
	RedisPassword      string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	FrontendURL        string
	AdminEmails        string
	JWTAccessSecret    string
	JWTRefreshSecret   string
	LineChannelToken   string
	LineTargetUserID   string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Info: .env file not found, using system environment variables instead")
	}

	return &Config{
		Env:                getEnv("ENV", "production"), 
		Port:               getEnv("PORT", "8080"),
		MongoURI:           getEnv("MONGO_URI", "mongodb://localhost:27017"),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
		AdminEmails:        getEnv("ADMIN_EMAILS", ""),
		JWTAccessSecret:    getEnv("JWT_ACCESS_SECRET", ""),
		JWTRefreshSecret:   getEnv("JWT_REFRESH_SECRET", ""),
		LineChannelToken:   getEnv("LINE_CHANNEL_ACCESS_TOKEN", ""),
		LineTargetUserID:   getEnv("LINE_TARGET_USER_ID", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// IsAdminEmail ตรวจว่า email อยู่ใน ADMIN_EMAILS (รองรับหลาย email คั่นด้วย comma)
func (c *Config) IsAdminEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || c.AdminEmails == "" {
		return false
	}
	for _, adminEmail := range strings.Split(c.AdminEmails, ",") {
		if strings.ToLower(strings.TrimSpace(adminEmail)) == email {
			return true
		}
	}
	return false
}

func (c *Config) ResolveRole(email string) string {
	if c.IsAdminEmail(email) {
		return "ADMIN"
	}
	return "USER"
}