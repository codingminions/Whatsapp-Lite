package conversation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/codingminions/Whatsapp-Lite/internal/auth"
	"github.com/codingminions/Whatsapp-Lite/internal/models"
	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler handles conversation-related HTTP requests
type Handler struct {
	service Service
	logger  logger.Logger
}

// NewHandler creates a new conversation handler
func NewHandler(service Service, logger logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// GetConversations handles requests to get a list of user's conversations
func (h *Handler) GetConversations(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userIDStr, err := auth.GetUserID(r.Context())
	if err != nil {
		h.logger.Error("Failed to get user ID from context", "error", err)
		sendJSON(w, http.StatusUnauthorized, models.ErrorResponse{
			Code:    1008,
			Message: "Authentication required",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID format", "error", err)
		sendJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    1000,
			Message: "Invalid user ID format",
		})
		return
	}

	// Call service
	resp, err := h.service.GetConversations(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get conversations", "error", err)
		sendJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    1009,
			Message: "Failed to get conversations",
		})
		return
	}

	// Send response
	sendJSON(w, http.StatusOK, resp)
}

// GetMessages handles requests to get messages in a conversation
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userIDStr, err := auth.GetUserID(r.Context())
	if err != nil {
		h.logger.Error("Failed to get user ID from context", "error", err)
		sendJSON(w, http.StatusUnauthorized, models.ErrorResponse{
			Code:    1008,
			Message: "Authentication required",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID format", "error", err)
		sendJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    1000,
			Message: "Invalid user ID format",
		})
		return
	}

	// Get conversation ID from URL
	vars := mux.Vars(r)
	conversationID := vars["conversation_id"]
	if conversationID == "" {
		sendJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    1000,
			Message: "Missing conversation ID",
		})
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	before := query.Get("before") // Cursor for pagination

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 {
		limit = 50 // Default limit
	}

	// Call service
	resp, err := h.service.GetMessages(r.Context(), conversationID, userID, before, limit)
	if err != nil {
		h.logger.Error("Failed to get messages", "error", err)
		sendJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    1009,
			Message: "Failed to get messages",
		})
		return
	}

	// Send response
	sendJSON(w, http.StatusOK, resp)
}

// sendJSON sends a JSON response
func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "Error encoding JSON response", http.StatusInternalServerError)
		}
	}
}
