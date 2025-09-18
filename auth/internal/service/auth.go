package service

import (
	"auth/internal/models"
	"auth/internal/repository"
	"auth/internal/utils"
	"context"
	"errors"
	"gorm.io/gorm"
	"log/slog"
)

type AuthResponse struct {
	User         *UserResponse `json:"user"`
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	TokenType    string        `json:"token_type"`
	ExpiresIn    int64         `json:"expires_in"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type UserService struct {
	r          *repository.UserRepository
	jwtService *JWTService
}

func NewUserService(r *repository.UserRepository, jwt *JWTService) *UserService {
	return &UserService{
		r:          r,
		jwtService: jwt,
	}

}

func (s *UserService) Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {

	existingUser, _ := s.r.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	hashPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		slog.Error("Failed to hash password", "error", err)
		return nil, errors.New("failed to process password")
	}

	user := &models.User{
		Name:          req.Name,
		LastName:      req.LastName,
		PasswordHash:  hashPassword,
		Email:         req.Email,
		CountPictures: 0,
	}

	if err = s.r.Create(ctx, user); err != nil {
		slog.Error("Failed to create user", "error", err, "email", req.Email)
		return nil, errors.New("failed to create user")
	}

	slog.Info("User registered successfully",
		"user_id", user.ID,
		"email", user.Email,
	)

	return user, nil
}

func (s *UserService) Login(ctx context.Context, req *models.LoginRequest) (*models.User, error) {
	user, err := s.r.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Warn("Login attempt with non-existent email", "email", req.Email)
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	if err := utils.CheckPassword(user.PasswordHash, req.Password); err != nil {
		slog.Warn("Invalid password attempt",
			"email", req.Email,
			"user_id", user.ID,
		)
		return nil, errors.New("invalid credentials")
	}

	slog.Info("User logged in successfully",
		"user_id", user.ID,
		"email", user.Email,
	)

	return user, nil
}

func (s *UserService) RegisterWithTokens(ctx context.Context, req *models.RegisterRequest) (*AuthResponse, error) {
	user, err := s.Register(ctx, req)
	if err != nil {
		return nil, err
	}

	return s.generateAuthResponse(user)
}

func (s *UserService) LoginWithTokens(ctx context.Context, req *models.LoginRequest) (*AuthResponse, error) {
	user, err := s.Login(ctx, req)
	if err != nil {
		return nil, err
	}

	return s.generateAuthResponse(user)
}

func (s *UserService) generateAuthResponse(user *models.User) (*AuthResponse, error) {
	accessToken, err := s.jwtService.GenerateAccessToken(user)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(user)
	if err != nil {
		return nil, errors.New("failed to generate refresh token")
	}

	return &AuthResponse{
		User: &UserResponse{
			ID:       user.ID,
			Name:     user.Name,
			LastName: user.LastName,
			Email:    user.Email,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    15 * 60, // 15 минут
	}, nil
}

func (s *UserService) RefreshToken(refreshToken string) (*AuthResponse, error) {
	claims, err := s.jwtService.ValidateToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	user, err := s.r.GetByID(context.TODO(), claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	return s.generateAuthResponse(user)
}

func (s *UserService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return s.r.GetByID(ctx, userID)
}
