package models

import (
	"time"

	"github.com/google/uuid"
)

// DirectMessage represents a direct message in the database
type DirectMessage struct {
	ID          uuid.UUID `json:"id" db:"id"`
	SenderID    uuid.UUID `json:"sender_id" db:"sender_id"`
	RecipientID uuid.UUID `json:"recipient_id" db:"recipient_id"`
	Content     string    `json:"content" db:"content"`
	Delivered   bool      `json:"delivered" db:"delivered"`
	Read        bool      `json:"read" db:"read"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Message represents a message in the API
type Message struct {
	ID             uuid.UUID             `json:"message_id" db:"message_id"`
	Content        string                `json:"content" db:"content"`
	SenderID       string                `json:"sender_id" db:"sender_id"`
	SenderUsername string                `json:"sender_username" db:"sender_username"`
	Timestamp      time.Time             `json:"timestamp" db:"timestamp"`
	DeliveryStatus MessageDeliveryStatus `json:"delivery_status"`
}

// MessageDeliveryStatus represents the delivery status of a message
type MessageDeliveryStatus struct {
	Delivered bool `json:"delivered"`
	Read      bool `json:"read"`
}

// MessageListResponse is the response for message history
type MessageListResponse struct {
	ConversationID string    `json:"conversation_id"`
	Messages       []Message `json:"messages"`
	HasMore        bool      `json:"has_more"`
	NextCursor     string    `json:"next_cursor,omitempty"`
}

// WebSocketMessage is the message format for WebSocket communication
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// DirectMessageData is the data for a direct message WebSocket message
type DirectMessageData struct {
	MessageID      string    `json:"message_id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	SenderUsername string    `json:"sender_username"`
	Content        string    `json:"content"`
	Timestamp      time.Time `json:"timestamp"`
}

// MessageAckData is the data for a message acknowledgment WebSocket message
type MessageAckData struct {
	ClientMessageID string    `json:"client_message_id"`
	ServerMessageID string    `json:"server_message_id,omitempty"`
	Status          string    `json:"status"`
	Timestamp       time.Time `json:"timestamp,omitempty"`
}

// TypingIndicatorData is the data for a typing indicator WebSocket message
type TypingIndicatorData struct {
	UserID         string `json:"user_id"`
	Username       string `json:"username"`
	ConversationID string `json:"conversation_id,omitempty"`
	Status         string `json:"status"`
}

// ReadReceiptData is the data for a read receipt WebSocket message
type ReadReceiptData struct {
	UserID            string    `json:"user_id"`
	Username          string    `json:"username"`
	ConversationID    string    `json:"conversation_id"`
	LastReadMessageID string    `json:"last_read_message_id"`
	Timestamp         time.Time `json:"timestamp,omitempty"`
}

// PresenceData is the data for a presence update WebSocket message
type PresenceData struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Status   string    `json:"status"`
	LastSeen time.Time `json:"last_seen,omitempty"`
}

// ErrorData is the data for an error WebSocket message
type ErrorData struct {
	Code                int    `json:"code"`
	Message             string `json:"message"`
	OriginalMessageType string `json:"original_message_type,omitempty"`
}
