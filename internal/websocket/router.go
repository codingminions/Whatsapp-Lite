package websocket

import (
	"context"
	"encoding/json"
	"time"

	"github.com/codingminions/Whatsapp-Lite/internal/models"
	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/google/uuid"
)

// MessageHandler defines a function that handles a specific type of message
type MessageHandler func(client *Client, message *models.WebSocketMessage)

// Router routes WebSocket messages to appropriate handlers
type Router struct {
	handlers map[string]MessageHandler
	hub      *Hub
	logger   logger.Logger
}

// NewRouter creates a new router
func NewRouter(hub *Hub, logger logger.Logger) *Router {
	r := &Router{
		handlers: make(map[string]MessageHandler),
		hub:      hub,
		logger:   logger,
	}

	// Register the message handlers
	r.handlers["direct_message"] = r.handleDirectMessage
	r.handlers["typing_indicator"] = r.handleTypingIndicator
	r.handlers["read_receipt"] = r.handleReadReceipt
	r.handlers["presence"] = r.handlePresenceUpdate

	return r
}

// RouteMessage routes a message to its appropriate handler
func (r *Router) RouteMessage(client *Client, message *models.WebSocketMessage) {
	handler, ok := r.handlers[message.Type]
	if !ok {
		r.logger.Error("Unknown message type received", "type", message.Type)
		client.sendError(1001, "Invalid message type", message.Type)
		return
	}

	handler(client, message)
}

// Helper min function for string truncation
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleDirectMessage handles a direct message
func (r *Router) handleDirectMessage(client *Client, message *models.WebSocketMessage) {
	// Convert to a proper map if needed
	data, ok := message.Data.(map[string]interface{})
	if !ok {
		// If data is not a map, try to marshal and unmarshal to convert to the right format
		dataBytes, err := json.Marshal(message.Data)
		if err != nil {
			client.sendError(1000, "Invalid message format", message.Type)
			return
		}

		err = json.Unmarshal(dataBytes, &data)
		if err != nil {
			client.sendError(1000, "Invalid message format", message.Type)
			return
		}
	}

	// Extract recipient ID and content
	recipientIDStr, ok := data["recipient_id"].(string)
	if !ok {
		client.sendError(1000, "Missing recipient_id", message.Type)
		return
	}

	content, ok := data["content"].(string)
	if !ok {
		client.sendError(1000, "Missing message content", message.Type)
		return
	}

	clientMsgID, ok := data["message_id"].(string)
	if !ok {
		client.sendError(1000, "Missing client message_id", message.Type)
		return
	}

	// Parse recipient ID
	recipientID, err := uuid.Parse(recipientIDStr)
	if err != nil {
		client.sendError(1002, "Invalid recipient ID", message.Type)
		return
	}

	// Generate a server message ID
	serverMsgID := uuid.New()

	// Create conversation ID (smaller UUID first)
	conversationID := ""
	if client.userID.String() < recipientIDStr {
		conversationID = client.userID.String() + "-" + recipientIDStr
	} else {
		conversationID = recipientIDStr + "-" + client.userID.String()
	}

	// Send acknowledgment to sender with sent status
	ack := &models.WebSocketMessage{
		Type: "message_ack",
		Data: models.MessageAckData{
			ClientMessageID: clientMsgID,
			ServerMessageID: serverMsgID.String(),
			Status:          "sent",
			Timestamp:       time.Now(),
		},
	}
	client.SendMessage(ack)

	// Create message
	now := time.Now()
	msg := &models.DirectMessage{
		ID:          serverMsgID,
		SenderID:    client.userID,
		RecipientID: recipientID,
		Content:     content,
		Delivered:   false,
		Read:        false,
		CreatedAt:   now,
	}

	// Log message details for debugging
	r.logger.Info("Attempting to save direct message",
		"message_id", serverMsgID,
		"sender_id", client.userID,
		"recipient_id", recipientID,
		"content_preview", content[:min(20, len(content))])

	// Save to database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if r.hub.conversationRepo == nil {
		r.logger.Error("Conversation repository is not available")
		client.sendError(1009, "Server error: repository unavailable", message.Type)
		return
	}

	err = r.hub.conversationRepo.SaveMessage(ctx, msg)
	if err != nil {
		r.logger.Error("Failed to save message to database", "error", err)
		client.sendError(1009, "Failed to save message: "+err.Error(), message.Type)
		return
	}

	r.logger.Info("Message saved successfully", "message_id", serverMsgID)

	// Send delivered acknowledgment
	deliveredAck := &models.WebSocketMessage{
		Type: "message_ack",
		Data: models.MessageAckData{
			ClientMessageID: clientMsgID,
			ServerMessageID: serverMsgID.String(),
			Status:          "delivered",
			Timestamp:       time.Now(),
		},
	}
	client.SendMessage(deliveredAck)

	// Forward the message to the recipient if they're online
	recipientConnected := r.hub.IsUserConnected(recipientID)
	if recipientConnected {
		forwardMsg := &models.WebSocketMessage{
			Type: "direct_message",
			Data: models.DirectMessageData{
				MessageID:      serverMsgID.String(),
				ConversationID: conversationID,
				SenderID:       client.userID.String(),
				SenderUsername: client.username,
				Content:        content,
				Timestamp:      now,
			},
		}
		r.hub.SendToUser(recipientID, forwardMsg)
	}
}

// handleTypingIndicator handles a typing indicator
func (r *Router) handleTypingIndicator(client *Client, message *models.WebSocketMessage) {
	data, ok := message.Data.(map[string]interface{})
	if !ok {
		client.sendError(1000, "Invalid message format", message.Type)
		return
	}

	// Extract recipient ID and status
	recipientIDStr, ok := data["recipient_id"].(string)
	if !ok {
		client.sendError(1000, "Missing recipient_id", message.Type)
		return
	}

	status, ok := data["status"].(string)
	if !ok {
		client.sendError(1000, "Missing status", message.Type)
		return
	}

	// Parse recipient ID
	recipientID, err := uuid.Parse(recipientIDStr)
	if err != nil {
		client.sendError(1002, "Invalid recipient ID", message.Type)
		return
	}

	// Forward typing indicator to recipient
	msg := &models.WebSocketMessage{
		Type: "typing_indicator",
		Data: models.TypingIndicatorData{
			UserID:   client.userID.String(),
			Username: client.username,
			Status:   status,
		},
	}
	r.hub.SendToUser(recipientID, msg)
}

// handleReadReceipt handles a read receipt
func (r *Router) handleReadReceipt(client *Client, message *models.WebSocketMessage) {
	data, ok := message.Data.(map[string]interface{})
	if !ok {
		client.sendError(1000, "Invalid message format", message.Type)
		return
	}

	// Extract conversation ID and last read message ID
	conversationIDStr, ok := data["conversation_id"].(string)
	if !ok {
		client.sendError(1000, "Missing conversation_id", message.Type)
		return
	}

	lastReadMsgIDStr, ok := data["last_read_message_id"].(string)
	if !ok {
		client.sendError(1000, "Missing last_read_message_id", message.Type)
		return
	}

	// TODO: Update read status in database
	// This should be done through a service call

	// Forward read receipt to the other user in the conversation
	// For direct messages, the conversation ID is a combination of the two user IDs
	// TODO: Get the other user ID from the conversation ID
	otherUserID, err := uuid.Parse("00000000-0000-0000-0000-000000000000") // Placeholder
	if err != nil {
		client.sendError(1003, "Invalid conversation ID", message.Type)
		return
	}

	msg := &models.WebSocketMessage{
		Type: "read_receipt",
		Data: models.ReadReceiptData{
			UserID:            client.userID.String(),
			Username:          client.username,
			ConversationID:    conversationIDStr,
			LastReadMessageID: lastReadMsgIDStr,
		},
	}
	r.hub.SendToUser(otherUserID, msg)
}

// handlePresenceUpdate handles a presence update
func (r *Router) handlePresenceUpdate(client *Client, message *models.WebSocketMessage) {
	data, ok := message.Data.(map[string]interface{})
	if !ok {
		client.sendError(1000, "Invalid message format", message.Type)
		return
	}

	// Extract status
	status, ok := data["status"].(string)
	if !ok {
		client.sendError(1000, "Missing status", message.Type)
		return
	}

	// Validate status
	if status != "online" && status != "away" && status != "offline" {
		client.sendError(1000, "Invalid status value", message.Type)
		return
	}

	// TODO: Update user status in database
	// This should be done through a service call

	// Broadcast presence update to all connected clients
	r.hub.broadcastPresenceUpdate(client.userID, client.username, status)
}
