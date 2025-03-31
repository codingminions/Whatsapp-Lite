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
	"github.com/codingminions/Whatsapp-Lite/internal/conversation"
	"github.com/codingminions/Whatsapp-Lite/internal/user"
	"github.com/codingminions/Whatsapp-Lite/internal/websocket"
	"github.com/codingminions/Whatsapp-Lite/pkg/database"
	"github.com/codingminions/Whatsapp-Lite/pkg/logger"
	"github.com/codingminions/Whatsapp-Lite/pkg/token"
	"github.com/codingminions/Whatsapp-Lite/pkg/validator"
	"github.com/gorilla/mux"
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

	// Initialize user components
	userRepo := user.NewPostgresRepository(db)
	userService := user.NewUserService(userRepo, log)
	userHandler := user.NewHandler(userService, log)

	// Initialize conversation components
	convRepo := conversation.NewPostgresRepository(db, log)
	convService := conversation.NewConversationService(convRepo, log)
	convHandler := conversation.NewHandler(convService, log)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(log, convRepo)
	wsHub.InitRouter() // Initialize the router after hub is created
	wsHandler := websocket.NewHandler(wsHub, tokenMaker, log)

	// Start WebSocket hub
	go wsHub.Run()

	// Initialize router
	router := mux.NewRouter()

	// Static files
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))

	// Public routes
	router.HandleFunc("/", serveTemplate("./web/templates/index.html")).Methods("GET")
	router.HandleFunc("/login", serveTemplate("./web/templates/login.html")).Methods("GET")
	router.HandleFunc("/register", serveTemplate("./web/templates/register.html")).Methods("GET")
	router.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		// Simple auth check, redirect to login if not authenticated
		cookie, err := r.Cookie("auth_token")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		serveTemplate("./web/templates/chat.html")(w, r)
	}).Methods("GET")

	// Auth API routes
	router.HandleFunc("/auth/register", authHandler.Register).Methods("POST")
	router.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
	router.HandleFunc("/auth/refresh", authHandler.Refresh).Methods("POST")
	router.Handle("/auth/logout", authMiddleware.Authenticate(http.HandlerFunc(authHandler.Logout))).Methods("POST")

	// User API routes
	router.Handle("/users", authMiddleware.Authenticate(http.HandlerFunc(userHandler.GetUsers))).Methods("GET")

	// Conversation API routes
	router.Handle("/conversations", authMiddleware.Authenticate(http.HandlerFunc(convHandler.GetConversations))).Methods("GET")
	router.Handle("/conversations/{conversation_id}/messages", authMiddleware.Authenticate(http.HandlerFunc(convHandler.GetMessages))).Methods("GET")

	// WebSocket route
	router.HandleFunc("/ws", wsHandler.ServeWS)

	// Configure CORS if needed
	// Uncomment and configure if needed for frontend development
	/*
		corsMiddleware := cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: true,
		})

		// Apply CORS middleware
		routerWithMiddleware := corsMiddleware.Handler(router)
	*/

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Server.Port),
		Handler:      router, // Change to routerWithMiddleware if using CORS
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
