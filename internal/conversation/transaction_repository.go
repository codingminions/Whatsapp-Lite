package conversation

import (
	"context"
	"fmt"
	"time"

	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// TransactionRepository provides a simplified repository implementation focused on transactions
type TransactionRepository struct {
	db     *sqlx.DB
	logger logger.Logger
}

// NewTransactionRepository creates a new transaction-focused repository
func NewTransactionRepository(db *sqlx.DB, logger logger.Logger) *TransactionRepository {
	return &TransactionRepository{
		db:     db,
		logger: logger,
	}
}

// SaveMessageDirect saves a message directly to the database
func (r *TransactionRepository) SaveMessageDirect(senderID, recipientID uuid.UUID, content string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to begin transaction", "error", err)
		return err
	}

	// Ensure transaction is rolled back if there's an error
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.logger.Error("Failed to rollback transaction", "error", rollbackErr)
			}
		}
	}()

	// Create message
	messageID := uuid.New()
	now := time.Now()

	r.logger.Info("Saving message with transaction",
		"message_id", messageID,
		"sender_id", senderID,
		"recipient_id", recipientID)

	// Insert the message
	query := `
		INSERT INTO direct_messages (id, sender_id, recipient_id, content, delivered, read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = tx.ExecContext(
		ctx,
		query,
		messageID,
		senderID,
		recipientID,
		content,
		false, // delivered
		false, // read
		now,   // created_at
	)

	if err != nil {
		r.logger.Error("Failed to insert message in transaction", "error", err)
		return fmt.Errorf("failed to insert message: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		r.logger.Error("Failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("Message saved successfully with transaction", "message_id", messageID)
	return nil
}
