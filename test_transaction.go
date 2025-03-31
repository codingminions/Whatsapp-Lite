// Place this file in the root of your project and run: go run test_transaction.go

package main

import (
	"fmt"

	"github.com/codingminions/Whatsapp-Lite/configs"
	"github.com/codingminions/Whatsapp-Lite/internal/conversation"
	"github.com/codingminions/Whatsapp-Lite/pkg/database"
	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/google/uuid"
)

func main() {
	// Initialize logger
	log := logger.NewZapLogger(true)
	log.Info("Starting test with transaction")

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

	// Use the database connection function from postgres.go
	db, err := database.ConnectPostgres(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()
	log.Info("Connected to database")

	// Create transaction repository
	repo := conversation.NewTransactionRepository(db, log)

	// Create test message
	sender := uuid.New()
	recipient := uuid.New()
	content := "Test messages using transaction - " + uuid.New().String()

	// Print the test data
	fmt.Printf("Sending message:\n")
	fmt.Printf("  Sender: %s\n", sender)
	fmt.Printf("  Recipient: %s\n", recipient)
	fmt.Printf("  Content: %s\n", content)

	// Save message
	err = repo.SaveMessageDirect(sender, recipient, content)
	if err != nil {
		log.Error("Failed to save message with transaction", "error", err)
	} else {
		log.Info("Message saved successfully with transaction")
	}

	// Verify message was saved
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM direct_messages WHERE content = $1", content)
	if err != nil {
		log.Error("Failed to verify message", "error", err)
		return
	}

	log.Info("Message verification", "found_count", count)
}
