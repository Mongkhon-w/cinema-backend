package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// ตรวจ JWT Access Token และตรวจสอบสิทธิ์ตาม Role 
func AuthMiddleware(secretKey string, requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// ตรวจสอบสิทธิ์ของ Role 
		userRole := claims["role"].(string)
		if requiredRole != "" && userRole != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"}) 
			c.Abort()
			return
		}

		// บันทึก Context เพื่อนำไปใช้งานต่อใน Layer  
		c.Set("user_id", claims["user_id"].(string))
		c.Set("role", userRole)
		c.Next()
	}
}