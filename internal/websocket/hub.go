package websocket

import (
	"context"
	"sync"

	"github.com/codingminions/Whatsapp-Lite/internal/models"
	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/google/uuid"
)

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// User ID to client mapping
	userClients map[string]*Client

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Router for handling messages
	router *Router

	// Mutex for thread safety
	mu sync.RWMutex

	// Logger
	logger logger.Logger

	// Conversation repository for saving messages
	conversationRepo ConversationRepository
}

// ConversationRepository defines the methods needed by the websocket hub
type ConversationRepository interface {
	SaveMessage(ctx context.Context, message *models.DirectMessage) error
}

// NewHub creates a new Hub
func NewHub(logger logger.Logger, conversationRepo ConversationRepository) *Hub {
	hub := &Hub{
		register:         make(chan *Client),
		unregister:       make(chan *Client),
		clients:          make(map[*Client]bool),
		userClients:      make(map[string]*Client),
		logger:           logger,
		conversationRepo: conversationRepo,
	}
	// We'll wait to initialize the router until after the hub is created
	// to avoid circular references
	return hub
}

// InitRouter initializes the message router
func (h *Hub) InitRouter() {
	h.router = NewRouter(h, h.logger)
}

// Run starts the hub's event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.logger.Info("Client connected",
		"user_id", client.userID.String(),
		"username", client.username)

	h.clients[client] = true
	h.userClients[client.userID.String()] = client

	// Notify other users that this user is online
	h.broadcastPresenceUpdate(client.userID, client.username, "online")
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		delete(h.userClients, client.userID.String())
		close(client.send)

		// Notify other users that this user is offline
		h.broadcastPresenceUpdate(client.userID, client.username, "offline")
	}
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID uuid.UUID, message *models.WebSocketMessage) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, ok := h.userClients[userID.String()]
	if !ok {
		return false
	}

	client.SendMessage(message)
	return true
}

// broadcastPresenceUpdate notifies all clients about a user's presence update
func (h *Hub) broadcastPresenceUpdate(userID uuid.UUID, username, status string) {
	message := &models.WebSocketMessage{
		Type: "presence_update",
		Data: models.PresenceData{
			UserID:   userID.String(),
			Username: username,
			Status:   status,
		},
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		// Don't send presence update to the user themselves
		if client.userID != userID {
			client.SendMessage(message)
		}
	}
}

// GetConnectedUserCount returns the number of connected users
func (h *Hub) GetConnectedUserCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.userClients)
}

// IsUserConnected checks if a user is connected
func (h *Hub) IsUserConnected(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.userClients[userID.String()]
	return ok
}
