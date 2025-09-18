package middleware

import (
	"auth/internal/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"log/slog"
)

type JWTMiddleware struct {
	jwtService  *service.JWTService
	userService *service.UserService
}

func NewJWTMiddleware(jwtService *service.JWTService, userService *service.UserService) *JWTMiddleware {
	return &JWTMiddleware{
		jwtService:  jwtService,
		userService: userService,
	}
}

func (m *JWTMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			c.Abort()
			return
		}

		claims, err := m.jwtService.ValidateToken(token)
		if err != nil {
			slog.Warn("Invalid token", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Проверяем что пользователь существует
		user, err := m.userService.GetUserByID(c, claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		// Добавляем данные в контекст
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user", user)
		c.Set("claims", claims)

		c.Next()
	}
}

func (m *JWTMiddleware) extractToken(c *gin.Context) string {
	bearerToken := c.GetHeader("Authorization")
	if bearerToken != "" {
		tokenParts := strings.Split(bearerToken, " ")
		if len(tokenParts) == 2 && strings.ToLower(tokenParts[0]) == "bearer" {
			return tokenParts[1]
		}
	}
	return ""
}
