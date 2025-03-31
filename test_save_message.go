package main

import (
	"fmt"
	"log"

	"github.com/codingminions/Whatsapp-Lite/configs"
	"github.com/codingminions/Whatsapp-Lite/pkg/database"
)

func main() {
	// Load configuration
	config, err := configs.LoadConfig("./configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Prepare database connection config
	dbConfig := database.PostgresConfig{
		Host:     config.Database.Host,
		Port:     config.Database.Port,
		User:     config.Database.User,
		Password: config.Database.Password,
		DBName:   config.Database.DBName,
		SSLMode:  config.Database.SSLMode,
	}

	// Attempt to connect
	db, err := database.ConnectToPostgres(dbConfig)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer database.SafeClose(db)

	// Verify table existence
	var tableExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'direct_messages'
		)
	`).Scan(&tableExists)
	if err != nil {
		log.Fatalf("Failed to check table existence: %v", err)
	}

	fmt.Printf("Connected successfully to database: %s\n", config.Database.DBName)
	fmt.Printf("Table 'direct_messages' exists: %v\n", tableExists)

	// List tables in public schema
	rows, err := db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public'
	`)
	if err != nil {
		log.Fatalf("Failed to list tables: %v", err)
	}
	defer rows.Close()

	fmt.Println("\nTables in public schema:")
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Printf("Error scanning table name: %v", err)
			continue
		}
		fmt.Println(tableName)
	}
}
