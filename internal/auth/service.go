package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"chat-app/internal/models"
	"chat-app/pkg/logger"
	"chat-app/pkg/token"
)

// Service errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
)

// Service handles auth business logic
type Service interface {
	Register(ctx context.Context, req *models.RegisterRequest) (*models.UserResponse, error)
	Login(ctx context.Context, req *models.LoginRequest, userAgent, clientIP string) (*models.LoginResponse, error)
	Refresh(ctx context.Context, req *models.RefreshRequest, userAgent, clientIP string) (*models.RefreshResponse, error)
	Logout(ctx context.Context, token string) error
	UpdateStatus(ctx context.Context, userID uuid.UUID, status string) error
}

// AuthService implements Service interface
type AuthService struct {
	repo            Repository
	tokenMaker      token.Maker
	logger          logger.Logger
	accessDuration  time.Duration
	refreshDuration time.Duration
}

// NewAuthService creates a new auth service
func NewAuthService(repo Repository, tokenMaker token.Maker, logger logger.Logger, accessDuration, refreshDuration time.Duration) *AuthService {
	return &AuthService{
		repo:            repo,
		tokenMaker:      tokenMaker,
		logger:          logger,
		accessDuration:  accessDuration,
		refreshDuration: refreshDuration,
	}
}

// Register handles user registration
func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.UserResponse, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash password", "error", err)
		return nil, err
	}

	// Create user
	now := time.Now()
	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Status:       "offline",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Save to database
	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			s.logger.Info("User already exists", "email", req.Email)
			return nil, ErrUserAlreadyExists
		}
		s.logger.Error("Failed to create user", "error", err)
		return nil, err
	}

	// Return user response
	return &models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}

// Login handles user login
func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest, userAgent, clientIP string) (*models.LoginResponse, error) {
	// Find user
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			s.logger.Info("User not found during login", "email", req.Email)
			return nil, ErrInvalidCredentials
		}
		s.logger.Error("Failed to get user by email", "error", err)
		return nil, err
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		s.logger.Info("Invalid password", "email", req.Email)
		return nil, ErrInvalidCredentials
	}

	// Create access token
	accessToken, accessPayload, err := s.tokenMaker.CreateToken(user.ID.String(), user.Username, s.accessDuration)
	if err != nil {
		s.logger.Error("Failed to create access token", "error", err)
		return nil, err
	}

	// Create refresh token
	refreshToken, err := s.createRefreshToken(ctx, user.ID, userAgent, clientIP)
	if err != nil {
		s.logger.Error("Failed to create refresh token", "error", err)
		return nil, err
	}

	// Update user status to online
	err = s.repo.UpdateUserStatus(ctx, user.ID, "online")
	if err != nil {
		s.logger.Error("Failed to update user status", "error", err)
		// Continue anyway, this shouldn't fail the login process
	}

	return &models.LoginResponse{
		UserID:       user.ID,
		Username:     user.Username,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessPayload.ExpiredAt,
	}, nil
}

// createRefreshToken creates a new refresh token
func (s *AuthService) createRefreshToken(ctx context.Context, userID uuid.UUID, userAgent, clientIP string) (string, error) {
	refreshToken, err := token.GenerateRandomString(32)
	if err != nil {
		return "", err
	}

	// Save session
	session := &models.Session{
		UserID:       userID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIP:     clientIP,
		ExpiresAt:    time.Now().Add(s.refreshDuration),
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}

	err = s.repo.CreateSession(ctx, session)
	if err != nil {
		return "", err
	}

	return refreshToken, nil
}

// Refresh handles token refresh
func (s *AuthService) Refresh(ctx context.Context, req *models.RefreshRequest, userAgent, clientIP string) (*models.RefreshResponse, error) {
	// Find session
	session, err := s.repo.GetSessionByRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			s.logger.Info("Session not found during refresh", "refresh_token", req.RefreshToken)
			return nil, ErrInvalidToken
		}
		s.logger.Error("Failed to get session by refresh token", "error", err)
		return nil, err
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		s.logger.Info("Refresh token expired", "user_id", session.UserID)
		return nil, ErrTokenExpired
	}

	// Get user
	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		s.logger.Error("Failed to get user by ID", "error", err)
		return nil, err
	}

	// Create new access token
	accessToken, accessPayload, err := s.tokenMaker.CreateToken(user.ID.String(), user.Username, s.accessDuration)
	if err != nil {
		s.logger.Error("Failed to create new access token", "error", err)
		return nil, err
	}

	// Delete old session
	err = s.repo.DeleteSession(ctx, req.RefreshToken)
	if err != nil {
		s.logger.Error("Failed to delete old session", "error", err)
		// Continue anyway
	}

	// Create new refresh token
	refreshToken, err := s.createRefreshToken(ctx, user.ID, userAgent, clientIP)
	if err != nil {
		s.logger.Error("Failed to create new refresh token", "error", err)
		return nil, err
	}

	return &models.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessPayload.ExpiredAt,
	}, nil
}

// Logout handles user logout
func (s *AuthService) Logout(ctx context.Context, tokenStr string) error {
	// Verify token
	payload, err := s.tokenMaker.VerifyToken(tokenStr)
	if err != nil {
		s.logger.Info("Invalid token during logout", "error", err)
		return ErrInvalidToken
	}

	// Parse user ID
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		s.logger.Error("Failed to parse user ID from token", "error", err)
		return err
	}

	// Update user status to offline
	err = s.repo.UpdateUserStatus(ctx, userID, "offline")
	if err != nil {
		s.logger.Error("Failed to update user status", "error", err)
		// Continue anyway
	}

	// Delete all user sessions
	err = s.repo.DeleteUserSessions(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to delete user sessions", "error", err)
		return err
	}

	return nil
}

// UpdateStatus updates a user's status
func (s *AuthService) UpdateStatus(ctx context.Context, userID uuid.UUID, status string) error {
	return s.repo.UpdateUserStatus(ctx, userID, status)
}
