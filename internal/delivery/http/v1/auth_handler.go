package v1

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cinema-backend/config"
	"cinema-backend/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler struct {
	cfg           *config.Config
	userRepo      domain.UserRepository
	auditLogsColl *mongo.Collection
	oauthConfig   *oauth2.Config
}

func NewAuthHandler(cfg *config.Config, userRepo domain.UserRepository, auditLogsColl *mongo.Collection) *AuthHandler {
	return &AuthHandler{
		cfg:           cfg,
		userRepo:      userRepo,
		auditLogsColl: auditLogsColl,
		oauthConfig: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		},
	}
}

func (h *AuthHandler) LoginRedirect(c *gin.Context) {
	if h.cfg.GoogleClientID == "" || h.cfg.GoogleClientID == "your_google_client_id" {
		if h.cfg.Env != "development" {
			c.JSON(http.StatusForbidden, gin.H{"error": "OAuth client is not configured and Mock login is disabled in production"})
			return
		}
		c.Redirect(http.StatusTemporaryRedirect, "/api/v1/auth/mock-choice")
		return
	}

	b := make([]byte, 16)
	_, _ = rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	url := h.oauthConfig.AuthCodeURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		h.redirectWithError(c, "missing_oauth_code")
		return
	}

	token, err := h.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		h.redirectWithError(c, "token_exchange_failed")
		return
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		h.redirectWithError(c, "userinfo_failed")
		return
	}
	defer resp.Body.Close()

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		h.redirectWithError(c, "userinfo_decode_failed")
		return
	}

	// Google OAuth = user ธรรมดา ADMIN เฉพาะ email ADMIN
	role := h.cfg.ResolveRole(googleUser.Email)
	accessToken, refreshToken, err := h.generateTokens(googleUser.ID, googleUser.Email, role)
	if err != nil {
		h.redirectWithError(c, "token_generation_failed")
		return
	}

	h.redirectWithTokens(c, accessToken, refreshToken, googleUser.ID, googleUser.Email, role)
}

func (h *AuthHandler) DevMockLogin(c *gin.Context) {
	if h.cfg.Env != "development" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied: Mock login is strictly prohibited in production environment",
		})
		return
	}

	email := strings.TrimSpace(c.Query("email"))
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}

	// role ADMIN เท่านั้น
	role := h.cfg.ResolveRole(email)

	password := c.Query("password")
	if role == "ADMIN" && password != "123456" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid admin password"})
		return
	}

	// เช็คว่ามีผู้ใช้ในระบบหรือไม่
	user, err := h.userRepo.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if user == nil && role != "ADMIN" {
		// ถ้าไม่ใช่แอดมิน และไม่มีชื่อในระบบ ให้ล็อคอินไม่ได้ และบันทึก Log
		logEntry := map[string]interface{}{
			"event":     "LOGIN_FAILED_USER_NOT_FOUND",
			"user_id":   email,
			"details":   "Login attempt failed for unregistered user: " + email,
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		}
		_, _ = h.auditLogsColl.InsertOne(c.Request.Context(), logEntry)

		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not registered"})
		return
	}

	// Check normal user password if provided
	if user != nil && role != "ADMIN" {
		if user.Password != password {
			logEntry := map[string]interface{}{
				"event":     "LOGIN_FAILED_INVALID_PASSWORD",
				"user_id":   email,
				"details":   "Login attempt failed due to invalid password: " + email,
				"timestamp": time.Now().Format("2006-01-02 15:04:05"),
			}
			_, _ = h.auditLogsColl.InsertOne(c.Request.Context(), logEntry)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
			return
		}
	}

	var userID string
	if user != nil {
		userID = user.ID
	} else {
		// Admin fallback if not in DB
		hash := md5.Sum([]byte(email))
		userID = fmt.Sprintf("USR-%x", hash[:6])
	}

	accessToken, refreshToken, err := h.generateTokens(userID, email, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate mock tokens"})
		return
	}

	if c.Query("redirect") == "true" {
		h.redirectWithTokens(c, accessToken, refreshToken, userID, email, role)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user_id":       userID,
		"email":         email,
		"role":          role,
		"message":       "Logged in with mock developer credentials",
	})
}

func (h *AuthHandler) redirectWithTokens(c *gin.Context, accessToken, refreshToken, userID, email, role string) {
	callbackURL := fmt.Sprintf(
		"%s/auth/callback?access_token=%s&refresh_token=%s&user_id=%s&email=%s&role=%s",
		strings.TrimRight(h.cfg.FrontendURL, "/"),
		url.QueryEscape(accessToken),
		url.QueryEscape(refreshToken),
		url.QueryEscape(userID),
		url.QueryEscape(email),
		url.QueryEscape(role),
	)
	c.Redirect(http.StatusTemporaryRedirect, callbackURL)
}

type RegisterInput struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email already exists
	existingUser, _ := h.userRepo.GetUserByEmail(c.Request.Context(), input.Email)
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already in use"})
		return
	}

	hash := md5.Sum([]byte(input.Email + time.Now().String()))
	newID := fmt.Sprintf("USR-%x", hash[:6])

	newUser := &domain.User{
		ID:       newID,
		Name:     input.Name,
		Email:    input.Email,
		Password: input.Password,
	}

	err := h.userRepo.CreateUser(c.Request.Context(), newUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Registration successful", "user": newUser})
}

func (h *AuthHandler) redirectWithError(c *gin.Context, reason string) {
	callbackURL := fmt.Sprintf(
		"%s/auth/callback?error=%s",
		strings.TrimRight(h.cfg.FrontendURL, "/"),
		url.QueryEscape(reason),
	)
	c.Redirect(http.StatusTemporaryRedirect, callbackURL)
}

func (h *AuthHandler) generateTokens(userID, email, role string) (string, string, error) {
	accessClaims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	aToken, err := accessToken.SignedString([]byte(h.cfg.JWTAccessSecret))
	if err != nil {
		return "", "", err
	}

	refreshClaims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	rToken, err := refreshToken.SignedString([]byte(h.cfg.JWTRefreshSecret))
	if err != nil {
		return "", "", err
	}

	return aToken, rToken, nil
}
