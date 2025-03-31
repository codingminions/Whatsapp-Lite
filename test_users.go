// Place this file in the root of your project and run: go run test_real_users.go

package main

import (
	"context"
	"time"

	"github.com/codingminions/Whatsapp-Lite/configs"
	"github.com/codingminions/Whatsapp-Lite/internal/models"
	"github.com/codingminions/Whatsapp-Lite/pkg/database"
	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func main() {
	// Initialize logger
	log := logger.NewZapLogger(true)
	log.Info("Starting test with real users")

	// Load configuration
	config, err := configs.LoadConfig("./configs/config.yaml")
	if err != nil {
		log.Fatal("Failed to load configuration", "error", err)
	}

	// Connect to database
	dbConfig := database.PostgresConfig{
		Host:     config.Database.Host,
		Port:     config.Database.Port,
		User:     config.Database.User,
		Password: config.Database.Password,
		DBName:   config.Database.DBName,
		SSLMode:  config.Database.SSLMode,
	}

	db, err := database.ConnectPostgres(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()
	log.Info("Connected to database")

	// Get two real users from the database
	var users []struct {
		ID       uuid.UUID `db:"id"`
		Username string    `db:"username"`
	}

	err = db.Select(&users, "SELECT id, username FROM users LIMIT 2")
	if err != nil {
		log.Fatal("Failed to get users", "error", err)
	}

	if len(users) < 2 {
		log.Fatal("Need at least 2 users in the database to run this test")
	}

	senderID := users[0].ID
	recipientID := users[1].ID

	log.Info("Using real users for test", 
		"sender", users[0].Username, "sender_id", senderID,
		"recipient", users[1].Username, "recipient_id", recipientID)

	// Create test message
	messageID := uuid.New()
	content := "Test message between real users - " + uuid.New().String()
	now := time.Now()

	// Create message struct
	message := &models.DirectMessage{
		ID:          messageID,
		SenderID:    senderID,
		RecipientID: recipientID,
		Content:     content,
		Delivered:   false,
		Read:        false,
		CreatedAt:   now,
	}

	// Save message to database
	log.Info("Saving message to database", 
		"message_id", messageID, 
		"content", content)

	err = saveMessage(db, message)
	if err != nil {
		log.Error("Failed to save message", "error", err)
	} else {
		log.Info("Message saved successfully!")
	}

	// Verify message was saved
	var savedMessages []struct {
		ID        uuid.UUID `db:"id"`
		Content   string    `db:"content"`
		CreatedAt time.Time `db:"created_at"`
	}

	err = db.Select(&savedMessages, "SELECT id, content, created_at FROM direct_messages WHERE id = $1", messageID)
	if err != nil {
		log.Error("Failed to verify message", "error", err)
	} else if len(savedMessages) > 0 {
		log.Info("Message verified in database", 
			"id", savedMessages[0].ID,
			"content", savedMessages[0].Content,
			"created_at", savedMessages[0].CreatedAt)
	} else {
		log.Error("Message not found in database!")
	}

	// Print the conversation ID format that should be used
	conversationID := ""
	if senderID.String() < recipientID.String() {
		conversationID = senderID.String() + "-" + recipientID.String()
	} else {
		conversationID = recipientID.String() + "-" + senderID.String()
	}
	log.Info("Conversation ID for these users", "conversation_id", conversationID)
}

// saveMessage saves a message to the database
func saveMessage(db *sqlx.DB, message *models.DirectMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO direct_messages (id, sender_id, recipient_id, content, delivered, read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := db.ExecContext(
		ctx,
		query,
		message.ID,
		message.SenderID,
		message.RecipientID,
		message.Content,
		message.Delivered,
		message.Read,
		message.CreatedAt,
	)

	return err
}
