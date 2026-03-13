package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Juan-Castaneda-A/transcriptor/internal/api"
	"github.com/Juan-Castaneda-A/transcriptor/internal/config"
	"github.com/Juan-Castaneda-A/transcriptor/internal/queue"
	"github.com/Juan-Castaneda-A/transcriptor/internal/storage"
	"github.com/Juan-Castaneda-A/transcriptor/internal/ws"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("🚀 Transcriptor API v0.1.0-alpha starting...")

	// Load .env file
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("ℹ️ No .env file found or error loading it, relying on system env vars")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize Supabase client
	supabaseClient := storage.NewClient(cfg.SupabaseURL, cfg.SupabaseAnonKey, cfg.SupabaseServiceKey)
	log.Println("✅ Supabase client initialized")

	// Initialize Redis queue
	queueClient, err := queue.NewClient(cfg.RedisURL)
	if err != nil {
		log.Printf("⚠️  Redis connection failed: %v (will retry on demand)", err)
		// Don't crash - allow the API to start even without Redis for health checks
	}

	// Initialize WebSocket hub
	wsHub := ws.NewHub()

	// Start listening for Redis pub/sub status updates (forwarded to WebSockets)
	if queueClient != nil {
		go wsHub.ListenToRedis(context.Background(), queueClient.RawClient(), queue.StatusChannel)
	}

	// Initialize API handlers
	handler := api.NewHandler(supabaseClient, queueClient)

	// Setup router
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("GET /api/health", handler.HealthCheck)

	// Protected routes (require authentication)
	authMw := api.AuthMiddleware(cfg.SupabaseURL, cfg.SupabaseAnonKey)

	mux.Handle("POST /api/upload", authMw(http.HandlerFunc(handler.UploadFile)))
	mux.Handle("POST /api/upload/confirm", authMw(http.HandlerFunc(handler.ConfirmUpload)))
	mux.Handle("GET /api/projects", authMw(http.HandlerFunc(handler.GetProjects)))
	mux.Handle("GET /api/transcriptions/{id}", authMw(http.HandlerFunc(handler.GetTranscription)))
	mux.Handle("GET /api/transcriptions/{id}/export", authMw(http.HandlerFunc(handler.GetTranscription)))

	// WebSocket (authenticated via query param for now)
	mux.HandleFunc("GET /ws", wsHub.HandleWebSocket)

	// Apply CORS middleware
	corsMw := api.CORSMiddleware(cfg.AllowedOrigins)
	finalHandler := corsMw(mux)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      finalHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("🛑 Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		server.Shutdown(ctx)
		if queueClient != nil {
			queueClient.Close()
		}
	}()

	log.Printf("🌐 Server listening on port %s", cfg.Port)
	log.Printf("📡 WebSocket available at ws://localhost:%s/ws", cfg.Port)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("❌ Server error: %v", err)
	}

	log.Println("✅ Server stopped gracefully")
}
