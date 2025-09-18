// internal/service/jwt.go
package service

import (
	"auth/internal/models"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"log/slog"
)

type JWTService struct {
	secretKey       []byte
	accessTokenExp  time.Duration
	refreshTokenExp time.Duration
}

type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	LastName string `json:"last_name"`
	jwt.RegisteredClaims
}

func NewJWTService() *JWTService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-super-secret-jwt-key-change-in-production"
		slog.Warn("Using default JWT secret - change in production!")
	}

	return &JWTService{
		secretKey:       []byte(secret),
		accessTokenExp:  15 * time.Minute,
		refreshTokenExp: 7 * 24 * time.Hour,
	}
}

func (s *JWTService) GenerateAccessToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID:   user.ID,
		Email:    user.Email,
		Name:     user.Name,
		LastName: user.LastName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenExp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "plant-analyzer-auth",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

func (s *JWTService) GenerateRefreshToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshTokenExp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "plant-analyzer-auth",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid token signing method")
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
