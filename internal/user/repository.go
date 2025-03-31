package user

import (
	"context"
	"fmt"
	"time"

	"github.com/codingminions/Whatsapp-Lite/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository interface for user operations
type Repository interface {
	GetUsers(ctx context.Context, currentUserID uuid.UUID, page, limit int, search string) ([]models.UserInfo, int, error)
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, status string, lastSeen time.Time) error
}

// PostgresRepository implements Repository interface with PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// GetUsers retrieves a list of users with pagination
// GetUsers retrieves a list of users with pagination
func (r *PostgresRepository) GetUsers(ctx context.Context, currentUserID uuid.UUID, page, limit int, search string) ([]models.UserInfo, int, error) {
	offset := (page - 1) * limit

	var params []interface{}
	var whereClause string

	// Base query to get all users except the current user
	whereClause = "id != $1"
	params = append(params, currentUserID)

	// Add search filter if provided
	if search != "" {
		whereClause += " AND (username ILIKE $2 OR email ILIKE $2)"
		params = append(params, "%"+search+"%")
	}

	// Count total users matching the criteria
	countQuery := fmt.Sprintf(`
        SELECT COUNT(*) 
        FROM users 
        WHERE %s
    `, whereClause)

	var total int
	err := r.db.GetContext(ctx, &total, countQuery, params...)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated user list
	usersQuery := fmt.Sprintf(`
        SELECT id, username, status, updated_at
        FROM users
        WHERE %s
        ORDER BY username ASC
        LIMIT $%d OFFSET $%d
    `, whereClause, len(params)+1, len(params)+2)

	params = append(params, limit, offset)

	rows, err := r.db.QueryContext(ctx, usersQuery, params...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []models.UserInfo
	for rows.Next() {
		var user models.UserInfo
		err := rows.Scan(&user.ID, &user.Username, &user.Status, &user.LastSeen)
		if err != nil {
			return nil, 0, err
		}

		// Set online status based on user's status field
		user.OnlineStatus = user.Status == "online"

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// UpdateUserStatus updates a user's status and last seen timestamp
func (r *PostgresRepository) UpdateUserStatus(ctx context.Context, userID uuid.UUID, status string, lastSeen time.Time) error {
	query := `
		UPDATE users
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, lastSeen, userID)
	return err
}
