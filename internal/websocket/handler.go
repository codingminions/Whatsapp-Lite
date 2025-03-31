package websocket

import (
	"net/http"

	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/codingminions/Whatsapp-Lite/pkg/token"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Handler manages WebSocket connections
type Handler struct {
	hub        *Hub
	upgrader   websocket.Upgrader
	tokenMaker token.Maker
	logger     logger.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, tokenMaker token.Maker, logger logger.Logger) *Handler {
	return &Handler{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now
				// In production, this should be more restrictive
				return true
			},
		},
		tokenMaker: tokenMaker,
		logger:     logger,
	}
}

// ServeWS handles WebSocket requests from clients
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Extract token from query string
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		h.logger.Error("Missing token in WebSocket connection request")
		http.Error(w, "Missing authentication token", http.StatusUnauthorized)
		return
	}

	// Verify token
	payload, err := h.tokenMaker.VerifyToken(tokenStr)
	if err != nil {
		h.logger.Error("Invalid token in WebSocket connection request", "error", err)
		http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
		return
	}

	// Parse user ID
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		h.logger.Error("Invalid user ID in token", "error", err)
		http.Error(w, "Invalid user ID", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection to WebSocket", "error", err)
		return
	}

	// Create client
	client := NewClient(h.hub, conn, userID, payload.Username, h.logger)

	// Register client in hub
	h.hub.register <- client

	// Start the client's read and write pumps in separate goroutines
	go client.writePump()
	go client.readPump()
}
