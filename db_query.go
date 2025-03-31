// Place this file in the root of your project and run: go run database_query.go

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/codingminions/Whatsapp-Lite/configs"
	"github.com/codingminions/Whatsapp-Lite/pkg/database"
	"github.com/jmoiron/sqlx"
)

func main() {
	// Load configuration
	config, err := configs.LoadConfig("./configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
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

	db, err := connectToDBDirectly(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	fmt.Println("Connected to database")

	// Get table structure
	var tableInfo []struct {
		Column        string `db:"column_name"`
		DataType      string `db:"data_type"`
		IsNullable    string `db:"is_nullable"`
		DefaultValue  string `db:"column_default"`
	}

	err = db.Select(&tableInfo, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = 'direct_messages'
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Fatalf("Failed to get table structure: %v", err)
	}

	fmt.Println("\nDirect Messages Table Structure:")
	fmt.Println("--------------------------------")
	for _, col := range tableInfo {
		nullable := "NOT NULL"
		if col.IsNullable == "YES" {
			nullable = "NULL"
		}
		defaultVal := ""
		if col.DefaultValue != "" {
			defaultVal = fmt.Sprintf("DEFAULT %s", col.DefaultValue)
		}
		fmt.Printf("%-15s %-15s %-10s %s\n", col.Column, col.DataType, nullable, defaultVal)
	}

	// Display any constraints
	var constraints []struct {
		ConstraintName string `db:"constraint_name"`
		ConstraintType string `db:"constraint_type"`
	}

	err = db.Select(&constraints, `
		SELECT con.conname as constraint_name, 
		       CASE con.contype
		           WHEN 'p' THEN 'PRIMARY KEY'
		           WHEN 'f' THEN 'FOREIGN KEY'
		           WHEN 'u' THEN 'UNIQUE'
		           WHEN 'c' THEN 'CHECK'
		           ELSE con.contype::text
		       END as constraint_type
		FROM pg_constraint con
		JOIN pg_class rel ON rel.oid = con.conrelid
		JOIN pg_namespace nsp ON nsp.oid = rel.relnamespace
		WHERE rel.relname = 'direct_messages'
	`)
	if err != nil {
		log.Fatalf("Failed to get constraints: %v", err)
	}

	fmt.Println("\nConstraints:")
	fmt.Println("-----------")
	for _, con := range constraints {
		fmt.Printf("%-30s %s\n", con.ConstraintName, con.ConstraintType)
	}

	// Count rows
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM direct_messages")
	if err != nil {
		log.Fatalf("Failed to count rows: %v", err)
	}
	fmt.Printf("\nTotal rows in direct_messages: %d\n", count)
}

// connectToDBDirectly connects directly to the PostgreSQL database
func connectToDBDirectly(config database.PostgresConfig) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
