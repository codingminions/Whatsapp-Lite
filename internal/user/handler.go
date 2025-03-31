package user

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/codingminions/Whatsapp-Lite/internal/auth"
	"github.com/codingminions/Whatsapp-Lite/internal/models"
	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/google/uuid"
)

// Handler handles user-related HTTP requests
type Handler struct {
	service Service
	logger  logger.Logger
}

// NewHandler creates a new user handler
func NewHandler(service Service, logger logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// GetUsers handles requests to get a list of users
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	// Get the authenticated user ID from context
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

	// Parse query parameters
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 {
		limit = 20
	}

	search := query.Get("search")

	// Call service
	resp, err := h.service.GetUsers(r.Context(), userID, page, limit, search)
	if err != nil {
		h.logger.Error("Failed to get users", "error", err)
		sendJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    1009,
			Message: "Failed to get users",
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
