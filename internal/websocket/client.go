package websocket

import (
	"encoding/json"
	"time"

	"github.com/codingminions/Whatsapp-Lite/internal/models"
	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 10000
)

// Client represents a single websocket connection
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   uuid.UUID
	username string
	logger   logger.Logger
}

// NewClient creates a new websocket client
func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID, username string, logger logger.Logger) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		username: username,
		logger:   logger,
	}
}

// readPump pumps messages from the websocket connection to the hub
// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("Unexpected websocket close", "error", err)
			}
			break
		}

		// Log received message for debugging
		c.logger.Debug("Received WebSocket message",
			"user_id", c.userID.String(),
			"username", c.username,
			"message", string(message))

		// Parse the message
		var wsMessage models.WebSocketMessage
		if err := json.Unmarshal(message, &wsMessage); err != nil {
			c.logger.Error("Failed to parse websocket message", "error", err)
			c.sendError(1000, "Invalid message format", "unknown")
			continue
		}

		// Handle the message by its type
		c.hub.router.RouteMessage(c, &wsMessage)
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(message *models.WebSocketMessage) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		c.logger.Error("Failed to marshal websocket message", "error", err)
		return
	}

	c.send <- messageBytes
}

// sendError sends an error message to the client
func (c *Client) sendError(code int, message, originalType string) {
	errorMsg := &models.WebSocketMessage{
		Type: "error",
		Data: models.ErrorData{
			Code:                code,
			Message:             message,
			OriginalMessageType: originalType,
		},
	}

	c.SendMessage(errorMsg)
}
