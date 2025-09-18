// internal/handlers/auth.go
package handlers

import (
	"auth/internal/models"
	"auth/internal/service"
	"context"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
)

type UserHandlers struct {
	s          *service.UserService
	jwtService *service.JWTService
}

func NewAuthHandlers(userService *service.UserService, jwtService *service.JWTService) *UserHandlers {
	return &UserHandlers{
		s:          userService,
		jwtService: jwtService,
	}
}

// POST /api/v1/auth/register
func (h *UserHandlers) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("Invalid registration request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authResponse, err := h.s.RegisterWithTokens(context.TODO(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Устанавливаем refresh token в HTTP-only cookie
	h.setRefreshTokenCookie(c, authResponse.RefreshToken)

	c.JSON(http.StatusCreated, gin.H{
		"message":      "User registered successfully",
		"user":         authResponse.User,
		"access_token": authResponse.AccessToken,
		"token_type":   authResponse.TokenType,
		"expires_in":   authResponse.ExpiresIn,
	})
}

// POST /api/v1/auth/login
func (h *UserHandlers) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authResponse, err := h.s.LoginWithTokens(context.TODO(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Устанавливаем refresh token в HTTP-only cookie
	h.setRefreshTokenCookie(c, authResponse.RefreshToken)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Login successful",
		"user":         authResponse.User,
		"access_token": authResponse.AccessToken,
		"token_type":   authResponse.TokenType,
		"expires_in":   authResponse.ExpiresIn,
	})
}

// POST /api/v1/auth/refresh
func (h *UserHandlers) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token required"})
		return
	}

	authResponse, err := h.s.RefreshToken(refreshToken)
	if err != nil {
		c.SetCookie("refresh_token", "", -1, "/", "", false, true)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.setRefreshTokenCookie(c, authResponse.RefreshToken)

	c.JSON(http.StatusOK, gin.H{
		"access_token": authResponse.AccessToken,
		"token_type":   authResponse.TokenType,
		"expires_in":   authResponse.ExpiresIn,
	})
}

// POST /api/v1/auth/logout
func (h *UserHandlers) Logout(c *gin.Context) {
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GET /api/v1/auth/me
func (h *UserHandlers) GetProfile(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	currentUser := user.(*models.User)
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":        currentUser.ID,
			"name":      currentUser.Name,
			"last_name": currentUser.LastName,
			"email":     currentUser.Email,
		},
	})
}

// Установка refresh token в HTTP-only cookie
func (h *UserHandlers) setRefreshTokenCookie(c *gin.Context, refreshToken string) {
	c.SetCookie(
		"refresh_token",
		refreshToken,
		7*24*60*60, // 7 дней
		"/",
		"",
		false, // secure (true для HTTPS)
		true,  // httpOnly
	)
}
