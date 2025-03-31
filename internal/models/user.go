package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Status       string    `json:"status" db:"status"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// UserResponse is the API response for a user
type UserResponse struct {
	ID        uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// UserInfo represents user information with online status
type UserInfo struct {
	ID           uuid.UUID `json:"user_id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Status       string    `json:"-" db:"status"`
	OnlineStatus bool      `json:"online_status"`
	LastSeen     time.Time `json:"last_seen" db:"updated_at"`
}

// UserListResponse is the response for the user list endpoint
type UserListResponse struct {
	Users      []UserInfo `json:"users"`
	Pagination Pagination `json:"pagination"`
}

// Pagination contains pagination information
type Pagination struct {
	Total    int `json:"total"`
	Page     int `json:"page"`
	Limit    int `json:"limit"`
	NextPage int `json:"next_page"`
}

// Conversation represents a conversation in the API
type Conversation struct {
	ConversationID string   `json:"conversation_id"`
	OtherUser      UserInfo `json:"other_user"`
	LastMessage    Message  `json:"last_message"`
	UnreadCount    int      `json:"unread_count"`
}

// ConversationListResponse is the response for the conversation list endpoint
type ConversationListResponse struct {
	Conversations []Conversation `json:"conversations"`
}

// RegisterRequest is the request body for user registration
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Username string `json:"username" validate:"required,min=3,max=50"`
}

// LoginRequest is the request body for user login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse is the API response for a successful login
type LoginResponse struct {
	UserID       uuid.UUID `json:"user_id"`
	Username     string    `json:"username"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// RefreshRequest is the request body for token refresh
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshResponse is the API response for a successful token refresh
type RefreshResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// ErrorResponse is the API response for errors
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
