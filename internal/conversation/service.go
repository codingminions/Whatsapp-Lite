package conversation

import (
	"context"
	"errors"

	"github.com/codingminions/Whatsapp-Lite/internal/models"
	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/google/uuid"
)

// Service errors
var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrUnauthorized         = errors.New("user not authorized to access this conversation")
)

// Service handles conversation business logic
type Service interface {
	GetConversations(ctx context.Context, userID uuid.UUID) (*models.ConversationListResponse, error)
	GetMessages(ctx context.Context, conversationID string, userID uuid.UUID, before string, limit int) (*models.MessageListResponse, error)
}

// ConversationService implements Service interface
type ConversationService struct {
	repo   Repository
	logger logger.Logger
}

// NewConversationService creates a new conversation service
func NewConversationService(repo Repository, logger logger.Logger) *ConversationService {
	return &ConversationService{
		repo:   repo,
		logger: logger,
	}
}

// GetConversations returns a list of conversations for a user
func (s *ConversationService) GetConversations(ctx context.Context, userID uuid.UUID) (*models.ConversationListResponse, error) {
	conversations, err := s.repo.GetConversations(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get conversations", "error", err)
		return nil, err
	}

	return &models.ConversationListResponse{
		Conversations: conversations,
	}, nil
}

// GetMessages returns messages in a conversation
func (s *ConversationService) GetMessages(ctx context.Context, conversationID string, userID uuid.UUID, before string, limit int) (*models.MessageListResponse, error) {
	// Check if user is part of the conversation
	isParticipant, err := s.repo.IsUserInConversation(ctx, conversationID, userID)
	if err != nil {
		s.logger.Error("Failed to check if user is in conversation", "error", err)
		return nil, err
	}

	if !isParticipant {
		s.logger.Info("User attempted to access unauthorized conversation", "user_id", userID, "conversation_id", conversationID)
		return nil, ErrUnauthorized
	}

	// Get messages
	messages, hasMore, nextCursor, err := s.repo.GetMessages(ctx, conversationID, before, limit)
	if err != nil {
		if errors.Is(err, ErrConversationNotFound) {
			return nil, ErrConversationNotFound
		}
		s.logger.Error("Failed to get messages", "error", err)
		return nil, err
	}

	// Update read status for messages
	if len(messages) > 0 {
		lastMsgID := messages[0].ID.String() // Messages should be sorted newest first
		err = s.repo.MarkMessagesAsRead(ctx, conversationID, userID, lastMsgID)
		if err != nil {
			s.logger.Error("Failed to mark messages as read", "error", err)
			// Continue anyway, this shouldn't fail the main request
		}
	}

	return &models.MessageListResponse{
		ConversationID: conversationID,
		Messages:       messages,
		HasMore:        hasMore,
		NextCursor:     nextCursor,
	}, nil
}
