package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"chat-app/internal/models"
)

// Repository errors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrSessionNotFound   = errors.New("session not found")
)

// Repository interface for auth operations
type Repository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	CreateSession(ctx context.Context, session *models.Session) error
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.Session, error)
	DeleteSession(ctx context.Context, refreshToken string) error
	DeleteUserSessions(ctx context.Context, userID uuid.UUID) error
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, status string) error
}

// PostgresRepository implements Repository interface with PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateUser creates a new user in the database
func (r *PostgresRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Status,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		// Check if the error is due to unique constraint violation
		if err.Error() == "pq: duplicate key value violates unique constraint" {
			return ErrUserAlreadyExists
		}
		return err
	}

	return nil
}

// GetUserByEmail retrieves a user by email
func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, status, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *PostgresRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, status, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

// CreateSession creates a new session in the database
func (r *PostgresRepository) CreateSession(ctx context.Context, session *models.Session) error {
	query := `
		INSERT INTO sessions (user_id, refresh_token, user_agent, client_ip, expires_at, created_at, last_active_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		session.UserID,
		session.RefreshToken,
		session.UserAgent,
		session.ClientIP,
		session.ExpiresAt,
		session.CreatedAt,
		session.LastActiveAt,
	).Scan(&session.ID)

	if err != nil {
		return err
	}

	return nil
}

// GetSessionByRefreshToken retrieves a session by refresh token
func (r *PostgresRepository) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.Session, error) {
	query := `
		SELECT id, user_id, refresh_token, user_agent, client_ip, expires_at, created_at, last_active_at
		FROM sessions
		WHERE refresh_token = $1
	`

	var session models.Session
	err := r.db.GetContext(ctx, &session, query, refreshToken)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	return &session, nil
}

// DeleteSession deletes a session by refresh token
func (r *PostgresRepository) DeleteSession(ctx context.Context, refreshToken string) error {
	query := `
		DELETE FROM sessions
		WHERE refresh_token = $1
	`

	_, err := r.db.ExecContext(ctx, query, refreshToken)
	return err
}

// DeleteUserSessions deletes all sessions for a user
func (r *PostgresRepository) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	query := `
		DELETE FROM sessions
		WHERE user_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// UpdateUserStatus updates a user's status
func (r *PostgresRepository) UpdateUserStatus(ctx context.Context, userID uuid.UUID, status string) error {
	query := `
		UPDATE users
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, time.Now(), userID)
	return err
}
