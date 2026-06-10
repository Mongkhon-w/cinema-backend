package v1

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"cinema-backend/config"
	"cinema-backend/internal/domain" 

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler struct {
	oauthConfig *oauth2.Config
	cfg         *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		cfg: cfg,
		oauthConfig: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		},
	}
}

// หน้าลงชื่อเข้าใช้ของ Google
func (h *AuthHandler) LoginRedirect(c *gin.Context) {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	
	url := h.oauthConfig.AuthCodeURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// รับกลับจาก Google เพื่อเปลี่ยนข้อมูลและออก tokens
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code not found"})
		return
	}

	token, err := h.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	// เรียกขอโปรไฟล์ผู้ใช้จาก Google API
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}
	defer resp.Body.Close()

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode user info"})
		return
	}

	// นำ googleUser.ID มาผูกเป็น user_id เพื่อใช้สิทธิ์ (USER Role)
	accessToken, refreshToken, err := h.generateTokens(googleUser.ID, "USER")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate system tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user_id":       googleUser.ID, 
	})
}

func (h *AuthHandler) generateTokens(userID, role string) (string, string, error) {
	// Access Token (อายุ 15 นาที)
	accessClaims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	aToken, err := accessToken.SignedString([]byte(h.cfg.JWTAccessSecret))
	if err != nil {
		return "", "", err
	}

	// Refresh Token (อายุ 7 วัน)
	refreshClaims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	rToken, err := refreshToken.SignedString([]byte(h.cfg.JWTRefreshSecret))
	if err != nil {
		return "", "", err
	}

	return aToken, rToken, nil
}