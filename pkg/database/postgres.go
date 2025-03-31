package database

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// PostgresConfig contains the configuration for a PostgreSQL connection
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// ConnectPostgres connects to a PostgreSQL database
func ConnectPostgres(config PostgresConfig) (*sqlx.DB, error) {
	// dsn := fmt.Sprintf(
	// 	"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
	// 	config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	// )

	db, err := sqlx.Connect("postgres", "host=localhost port=5432 user=prateekkumar password='' dbname=chat_app sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
