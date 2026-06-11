package middleware

import (
	"net/http"
	"strings"

	"cinema-backend/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func AuthMiddleware(secretKey string, requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := parseAndValidateToken(c, secretKey)
		if !ok {
			return
		}

		userRole, _ := claims["role"].(string)
		if requiredRole != "" && userRole != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			c.Abort()
			return
		}

		setClaimsContext(c, claims)
		c.Next()
	}
}

// อนุญาตเฉพาะ email ADMIN เท่านั้น
func AdminAuthMiddleware(secretKey string, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := parseAndValidateToken(c, secretKey)
		if !ok {
			return
		}

		userRole, _ := claims["role"].(string)
		email, _ := claims["email"].(string)

		if userRole != "ADMIN" || !cfg.IsAdminEmail(email) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: admin email required"})
			c.Abort()
			return
		}

		setClaimsContext(c, claims)
		c.Next()
	}
}

func parseAndValidateToken(c *gin.Context, secretKey string) (jwt.MapClaims, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		c.Abort()
		return nil, false
	}

	tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		c.Abort()
		return nil, false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		c.Abort()
		return nil, false
	}

	return claims, true
}

func setClaimsContext(c *gin.Context, claims jwt.MapClaims) {
	if userID, ok := claims["user_id"].(string); ok {
		c.Set("user_id", userID)
	}
	if role, ok := claims["role"].(string); ok {
		c.Set("role", role)
	}
	if email, ok := claims["email"].(string); ok {
		c.Set("email", email)
	}
}
