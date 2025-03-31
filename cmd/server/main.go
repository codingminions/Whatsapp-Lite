package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codingminions/Whatsapp-Lite/configs"
	"github.com/codingminions/Whatsapp-Lite/internal/auth"
	"github.com/codingminions/Whatsapp-Lite/pkg/database"
	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/codingminions/Whatsapp-Lite/pkg/token"
	"github.com/codingminions/Whatsapp-Lite/pkg/validator"
)

func main() {
	// Define command line flags
	configPath := flag.String("config", "./configs/config.yaml", "path to config file")
	dev := flag.Bool("dev", false, "run in development mode")
	flag.Parse()

	// Initialize logger
	log := logger.NewZapLogger(*dev)
	log.Info("Starting chat application server")

	// Load configuration
	config, err := configs.LoadConfig(*configPath)
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

	// Initialize validator
	validate := validator.NewCustomValidator()

	// Initialize JWT token maker
	tokenMaker, err := token.NewJWTMaker(config.JWT.SecretKey)
	if err != nil {
		log.Fatal("Failed to create token maker", "error", err)
	}

	// Initialize auth components
	authRepo := auth.NewPostgresRepository(db)
	authService := auth.NewAuthService(
		authRepo,
		tokenMaker,
		log,
		config.JWT.AccessExpiry,
		config.JWT.RefreshExpiry,
	)
	authHandler := auth.NewHandler(authService, log, validate)
	authMiddleware := auth.NewAuthMiddleware(tokenMaker, log)

	// Initialize router
	router := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("./web/static"))
	router.Handle("/static/", http.StripPrefix("/static/", fs))

	// Public routes
	router.HandleFunc("/", serveTemplate("./web/templates/index.html"))
	router.HandleFunc("/login", serveTemplate("./web/templates/login.html"))
	router.HandleFunc("/register", serveTemplate("./web/templates/register.html"))
	router.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		// Simple auth check, redirect to login if not authenticated
		cookie, err := r.Cookie("auth_token")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		serveTemplate("./web/templates/chat.html")(w, r)
	})

	// Auth API routes - Using method check for Go 1.21 compatibility
	router.HandleFunc("/auth/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		authHandler.Register(w, r)
	})

	router.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		authHandler.Login(w, r)
	})

	router.HandleFunc("/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		authHandler.Refresh(w, r)
	})

	router.HandleFunc("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		authMiddleware.Authenticate(http.HandlerFunc(authHandler.Logout)).ServeHTTP(w, r)
	})

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Server.Port),
		Handler:      router,
		ReadTimeout:  config.Server.ReadTimeout,
		WriteTimeout: config.Server.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("Server listening", "port", config.Server.Port)
		serverErrors <- server.ListenAndServe()
	}()

	// Listen for signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or an error
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Error("Server error", "error", err)
		}
	case <-shutdown:
		log.Info("Shutting down server")

		// Create context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), config.Server.ShutdownTimeout)
		defer cancel()

		// Shut down server
		if err := server.Shutdown(ctx); err != nil {
			log.Error("Server shutdown error", "error", err)
			server.Close()
		}
	}

	log.Info("Server stopped")
}

// serveTemplate serves an HTML template
func serveTemplate(filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filename)
	}
}
