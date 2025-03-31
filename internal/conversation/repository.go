package conversation

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/codingminions/Whatsapp-Lite/internal/models"
	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository interface for conversation operations
type Repository interface {
	GetConversations(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error)
	GetMessages(ctx context.Context, conversationID string, before string, limit int) ([]models.Message, bool, string, error)
	IsUserInConversation(ctx context.Context, conversationID string, userID uuid.UUID) (bool, error)
	MarkMessagesAsRead(ctx context.Context, conversationID string, userID uuid.UUID, lastReadMessageID string) error
	SaveMessage(ctx context.Context, message *models.DirectMessage) error
	GetOrCreateConversation(ctx context.Context, userID1, userID2 uuid.UUID) (string, error)
}

// PostgresRepository implements Repository interface with PostgreSQL
type PostgresRepository struct {
	db     *sqlx.DB
	logger logger.Logger
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB, logger logger.Logger) *PostgresRepository {
	return &PostgresRepository{
		db:     db,
		logger: logger,
	}
}

// GetConversations retrieves a list of conversations for a user
func (r *PostgresRepository) GetConversations(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error) {
	// First check if the user has any messages at all
	checkQuery := `
        SELECT COUNT(*)
        FROM direct_messages
        WHERE sender_id = $1 OR recipient_id = $1
    `

	var count int
	err := r.db.GetContext(ctx, &count, checkQuery, userID)
	if err != nil {
		return nil, err
	}

	// If no messages, return empty slice
	if count == 0 {
		return []models.Conversation{}, nil
	}

	query := `
        WITH direct_conversations AS (
            -- Get all direct messages where user is sender or recipient
            SELECT
                CASE 
                    WHEN sender_id = $1 THEN recipient_id
                    WHEN recipient_id = $1 THEN sender_id
                END as other_user_id,
                id as last_message_id,
                content as last_message_content,
                created_at,
                CASE 
                    WHEN sender_id = $1 THEN TRUE
                    ELSE delivered
                END as delivered,
                CASE 
                    WHEN sender_id = $1 THEN TRUE
                    ELSE read
                END as read,
                ROW_NUMBER() OVER (
                    PARTITION BY 
                        CASE 
                            WHEN sender_id = $1 THEN recipient_id
                            WHEN recipient_id = $1 THEN sender_id
                        END
                    ORDER BY created_at DESC
                ) as row_num
            FROM direct_messages
            WHERE sender_id = $1 OR recipient_id = $1
        ),
        unread_counts AS (
            -- Count unread messages for each conversation
            SELECT 
                sender_id as other_user_id, 
                COUNT(*) as unread_count
            FROM direct_messages
            WHERE recipient_id = $1 AND read = FALSE
            GROUP BY sender_id
        )
        -- Join with users to get usernames
        SELECT 
            LEAST(dc.other_user_id, $1)::text || '-' || GREATEST(dc.other_user_id, $1)::text as conversation_id,
            dc.other_user_id as user_id, 
            u.username, 
            u.status,
            u.updated_at as last_seen,
            dc.last_message_id as message_id,
            dc.last_message_content as content,
            dc.created_at as timestamp,
            dc.delivered,
            dc.read,
            COALESCE(uc.unread_count, 0) as unread_count
        FROM direct_conversations dc
        JOIN users u ON dc.other_user_id = u.id
        LEFT JOIN unread_counts uc ON dc.other_user_id = uc.other_user_id
        WHERE dc.row_num = 1
        ORDER BY dc.created_at DESC
    `

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []models.Conversation
	for rows.Next() {
		var conversation models.Conversation
		var otherUser models.UserInfo
		var lastMessage models.Message
		var status string
		var lastSeen time.Time

		err := rows.Scan(
			&conversation.ConversationID,
			&otherUser.ID,
			&otherUser.Username,
			&status,
			&lastSeen,
			&lastMessage.ID,
			&lastMessage.Content,
			&lastMessage.Timestamp,
			&lastMessage.DeliveryStatus.Delivered,
			&lastMessage.DeliveryStatus.Read,
			&conversation.UnreadCount,
		)
		if err != nil {
			return nil, err
		}

		// Set relationship
		lastMessage.SenderID = otherUser.ID.String() // Assuming the last message is from the other user for simplicity

		// Set online status based on user status field
		otherUser.OnlineStatus = status == "online"
		otherUser.LastSeen = lastSeen

		// Populate the conversation struct
		conversation.OtherUser = otherUser
		conversation.LastMessage = lastMessage

		conversations = append(conversations, conversation)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return conversations, nil
}

// GetMessages retrieves messages for a conversation with pagination
func (r *PostgresRepository) GetMessages(ctx context.Context, conversationID string, before string, limit int) ([]models.Message, bool, string, error) {
	// Parse conversationID to get user IDs
	user1ID, user2ID, err := splitConversationID(conversationID)
	if err != nil {
		return nil, false, "", err
	}

	// Build query for direct messages
	query := `
        SELECT 
            dm.id as message_id,
            dm.content,
            dm.sender_id,
            u.username as sender_username,
            dm.created_at as timestamp,
            dm.delivered,
            dm.read
        FROM direct_messages dm
        JOIN users u ON dm.sender_id = u.id
        WHERE (dm.sender_id = $1 AND dm.recipient_id = $2)
           OR (dm.sender_id = $2 AND dm.recipient_id = $1)
    `

	args := []interface{}{user1ID, user2ID}

	// Add cursor condition if provided
	if before != "" {
		beforeID, err := uuid.Parse(before)
		if err != nil {
			return nil, false, "", errors.New("invalid before cursor")
		}
		query += " AND dm.id < $3"
		args = append(args, beforeID)
	}

	// Add ordering and limit
	query += " ORDER BY dm.created_at DESC LIMIT $" + strconv.Itoa(len(args)+1)
	args = append(args, limit+1) // Get one extra message to check if there are more

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, "", err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		var deliveryStatus models.MessageDeliveryStatus

		err := rows.Scan(
			&msg.ID,
			&msg.Content,
			&msg.SenderID,
			&msg.SenderUsername,
			&msg.Timestamp,
			&deliveryStatus.Delivered,
			&deliveryStatus.Read,
		)
		if err != nil {
			return nil, false, "", err
		}

		msg.DeliveryStatus = deliveryStatus
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, false, "", err
	}

	// Check if there are more messages
	hasMore := len(messages) > limit
	var nextCursor string

	if hasMore {
		// Remove the extra message
		nextCursor = messages[limit].ID.String()
		messages = messages[:limit]
	}

	return messages, hasMore, nextCursor, nil
}

// IsUserInConversation checks if a user is part of a conversation
func (r *PostgresRepository) IsUserInConversation(ctx context.Context, conversationID string, userID uuid.UUID) (bool, error) {
	// For direct conversations, the ID contains both user IDs
	user1ID, user2ID, err := splitConversationID(conversationID)
	if err != nil {
		return false, err
	}

	return userID == user1ID || userID == user2ID, nil
}

// MarkMessagesAsRead marks messages in a conversation as read
func (r *PostgresRepository) MarkMessagesAsRead(ctx context.Context, conversationID string, userID uuid.UUID, lastReadMessageID string) error {
	// Parse conversationID to get user IDs
	user1ID, user2ID, err := splitConversationID(conversationID)
	if err != nil {
		return err
	}

	// Determine the other user ID
	var otherUserID uuid.UUID
	if userID == user1ID {
		otherUserID = user2ID
	} else if userID == user2ID {
		otherUserID = user1ID
	} else {
		return errors.New("user is not part of this conversation")
	}

	// Update read status for messages from the other user
	query := `
        UPDATE direct_messages
        SET read = TRUE
        WHERE sender_id = $1 AND recipient_id = $2 AND read = FALSE
    `

	_, err = r.db.ExecContext(ctx, query, otherUserID, userID)
	return err
}

// SaveMessage saves a direct message to the database
func (r *PostgresRepository) SaveMessage(ctx context.Context, message *models.DirectMessage) error {
	query := `
        INSERT INTO direct_messages (id, sender_id, recipient_id, content, delivered, read, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

	// Log what we're trying to insert
	fmt.Println("Saving message to database",
		"message_id", message.ID,
		"sender_id", message.SenderID,
		"recipient_id", message.RecipientID)

	_, err := r.db.ExecContext(
		ctx,
		query,
		message.ID,
		message.SenderID,
		message.RecipientID,
		message.Content,
		message.Delivered,
		message.Read,
		message.CreatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to save message", "error", err)
		return err
	}

	r.logger.Info("Message saved successfully", "message_id", message.ID)
	return nil
}

// GetOrCreateConversation gets or creates a conversation between two users
func (r *PostgresRepository) GetOrCreateConversation(ctx context.Context, userID1, userID2 uuid.UUID) (string, error) {
	// For direct messages, the conversation ID is just the concatenation of the two user IDs (smaller UUID first)
	var smaller, larger uuid.UUID
	if userID1.String() < userID2.String() {
		smaller = userID1
		larger = userID2
	} else {
		smaller = userID2
		larger = userID1
	}

	return smaller.String() + "-" + larger.String(), nil
}

// Helper functions

// splitConversationID splits a conversation ID into its component UUID parts
func splitConversationID(conversationID string) (uuid.UUID, uuid.UUID, error) {
	// A standard UUID is 36 characters (including hyphens)
	if len(conversationID) < 73 { // 36 + 1 + 36 = 73
		return uuid.Nil, uuid.Nil, errors.New("invalid conversation ID format: too short")
	}

	// Extract the two UUIDs
	firstUuidStr := conversationID[:36]
	secondUuidStr := conversationID[37:] // Skip the separator hyphen

	// Parse the UUID strings
	firstUuid, err := uuid.Parse(firstUuidStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, errors.New("invalid first UUID in conversation ID")
	}

	secondUuid, err := uuid.Parse(secondUuidStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, errors.New("invalid second UUID in conversation ID")
	}

	return firstUuid, secondUuid, nil
}

// stringify converts an int to a string
func stringify(n int) string {
	return strconv.Itoa(n)
}
