package auth

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"chat-app/internal/models"
	"chat-app/pkg/logger"
	"chat-app/pkg/validator"
)

// Handler handles auth-related HTTP requests
type Handler struct {
	service   Service
	logger    logger.Logger
	validator validator.Validator
}

// NewHandler creates a new auth handler
func NewHandler(service Service, logger logger.Logger, validator validator.Validator) *Handler {
	return &Handler{
		service:   service,
		logger:    logger,
		validator: validator,
	}
}

// Register handles user registration
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// Parse and validate request
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode register request", "error", err)
		sendJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    1000,
			Message: "Invalid request format",
		})
		return
	}

	// Validate request
	if err := h.validator.Validate(req); err != nil {
		h.logger.Info("Invalid register request", "error", err)
		sendJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    1000,
			Message: err.Error(),
		})
		return
	}

	// Call service
	resp, err := h.service.Register(r.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			sendJSON(w, http.StatusConflict, models.ErrorResponse{
				Code:    1000,
				Message: "Email or username already exists",
			})
			return
		}
		h.logger.Error("Failed to register user", "error", err)
		sendJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    1009,
			Message: "Failed to register user",
		})
		return
	}

	// Send response
	sendJSON(w, http.StatusCreated, resp)
}

// Login handles user login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode login request", "error", err)
		sendJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    1000,
			Message: "Invalid request format",
		})
		return
	}

	// Validate request
	if err := h.validator.Validate(req); err != nil {
		h.logger.Info("Invalid login request", "error", err)
		sendJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    1000,
			Message: err.Error(),
		})
		return
	}

	// Get client IP and user agent
	userAgent := r.UserAgent()
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		clientIP = r.RemoteAddr
	}

	// Call service
	resp, err := h.service.Login(r.Context(), &req, userAgent, clientIP)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			h.logger.Info("Invalid credentials", "email", req.Email)
			sendJSON(w, http.StatusUnauthorized, models.ErrorResponse{
				Code:    1008,
				Message: "Invalid email or password",
			})
			return
		}
		h.logger.Error("Failed to login user", "error", err)
		sendJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    1009,
			Message: "Failed to login user",
		})
		return
	}

	// Send response
	sendJSON(w, http.StatusOK, resp)
}

// Refresh handles token refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req models.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode refresh request", "error", err)
		sendJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    1000,
			Message: "Invalid request format",
		})
		return
	}

	// Validate request
	if err := h.validator.Validate(req); err != nil {
		h.logger.Info("Invalid refresh request", "error", err)
		sendJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    1000,
			Message: err.Error(),
		})
		return
	}

	// Get client IP and user agent
	userAgent := r.UserAgent()
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		clientIP = r.RemoteAddr
	}

	// Call service
	resp, err := h.service.Refresh(r.Context(), &req, userAgent, clientIP)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrTokenExpired) {
			sendJSON(w, http.StatusUnauthorized, models.ErrorResponse{
				Code:    1008,
				Message: err.Error(),
			})
			return
		}
		h.logger.Error("Failed to refresh token", "error", err)
		sendJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    1009,
			Message: "Failed to refresh token",
		})
		return
	}

	// Send response
	sendJSON(w, http.StatusOK, resp)
}

// Logout handles user logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		sendJSON(w, http.StatusUnauthorized, models.ErrorResponse{
			Code:    1008,
			Message: "Authentication required",
		})
		return
	}

	// Check header format
	fields := strings.Fields(authHeader)
	if len(fields) != 2 || fields[0] != "Bearer" {
		sendJSON(w, http.StatusUnauthorized, models.ErrorResponse{
			Code:    1008,
			Message: "Invalid authorization header format",
		})
		return
	}

	// Call service
	err := h.service.Logout(r.Context(), fields[1])
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			sendJSON(w, http.StatusUnauthorized, models.ErrorResponse{
				Code:    1008,
				Message: "Invalid token",
			})
			return
		}
		h.logger.Error("Failed to logout user", "error", err)
		sendJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    1009,
			Message: "Failed to logout user",
		})
		return
	}

	// Send response
	w.WriteHeader(http.StatusNoContent)