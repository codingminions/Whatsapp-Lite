package user

import (
	"context"

	"github.com/codingminions/Whatsapp-Lite/internal/models"
	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/google/uuid"
)

// Service handles user business logic
type Service interface {
	GetUsers(ctx context.Context, userID uuid.UUID, page, limit int, search string) (*models.UserListResponse, error)
}

// UserService implements Service interface
type UserService struct {
	repo   Repository
	logger logger.Logger
}

// NewUserService creates a new user service
func NewUserService(repo Repository, logger logger.Logger) *UserService {
	return &UserService{
		repo:   repo,
		logger: logger,
	}
}

// GetUsers returns a list of users with pagination
func (s *UserService) GetUsers(ctx context.Context, userID uuid.UUID, page, limit int, search string) (*models.UserListResponse, error) {
	// Get users from repository
	users, total, err := s.repo.GetUsers(ctx, userID, page, limit, search)
	if err != nil {
		s.logger.Error("Failed to get users", "error", err)
		return nil, err
	}

	// Calculate next page
	var nextPage int
	if (page * limit) < total {
		nextPage = page + 1
	} else {
		nextPage = 0
	}

	return &models.UserListResponse{
		Users: users,
		Pagination: models.Pagination{
			Total:    total,
			Page:     page,
			Limit:    limit,
			NextPage: nextPage,
		},
	}, nil
}
