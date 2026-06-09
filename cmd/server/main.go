// Package main is the entry point for the Naturieux quiz server.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httphandler "github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/inaturalist"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/mock"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

const (
	defaultPort = "8080"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Check for dev mode
	devMode := os.Getenv("DEV_MODE") == "true" || os.Getenv("DEV_MODE") == "1"

	// Initialize species repository based on mode
	var speciesRepo ports.SpeciesRepository
	if devMode {
		log.Println("🔧 DEV MODE ENABLED - Using mock data")
		speciesRepo = mock.NewSpeciesRepository()
	} else {
		log.Println("🌿 Production mode - Using iNaturalist API")
		speciesRepo = inaturalist.NewClient()
	}

	// In-memory player repository (use proper database in production)
	playerRepo := newInMemoryPlayerRepository()

	// Create a demo player
	demoPlayer, err := gamification.NewPlayer("demo", "demo_user")
	if err != nil {
		log.Fatalf("Failed to create demo player: %v", err)
	}
	if err := playerRepo.Create(context.Background(), demoPlayer); err != nil {
		log.Fatalf("Failed to store demo player: %v", err)
	}

	// Create question factory
	questionFactory := appquiz.NewQuestionFactory(
		speciesRepo,
		appquiz.WithTaxonFilter(""),   // All taxa
		appquiz.WithPlaceFilter(6753), // France
	)

	// Create quiz service
	quizService := appquiz.NewService(
		questionFactory,
		nil, // No session persistence for now
		playerRepo,
		nil, // No event publisher for now
	)

	// Create HTTP handler
	handler := httphandler.NewHandler(quizService, devMode)

	// Create HTTP server
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Serve static files
	staticFS := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", staticFS))

	// Serve index.html for root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "web/index.html")
	})

	// Add CORS middleware for development
	corsHandler := corsMiddleware(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      corsHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second, // Increased for iNaturalist API calls
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting Naturieux server on port %s", port)
		log.Printf("Frontend: http://localhost:%s/", port)
		log.Printf("Health check: http://localhost:%s/health", port)
		log.Printf("API: http://localhost:%s/api/v1/", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	if err := server.Shutdown(ctx); err != nil {
		cancel()
		log.Printf("Server forced to shutdown: %v", err)
		return
	}
	cancel()

	log.Println("Server stopped")
}

// corsMiddleware adds CORS headers for development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// inMemoryPlayerRepository is a simple in-memory player repository for demo.
type inMemoryPlayerRepository struct {
	players map[string]*gamification.Player
}

func newInMemoryPlayerRepository() *inMemoryPlayerRepository {
	return &inMemoryPlayerRepository{
		players: make(map[string]*gamification.Player),
	}
}

func (r *inMemoryPlayerRepository) Create(_ context.Context, player *gamification.Player) error {
	r.players[player.ID()] = player
	return nil
}

func (r *inMemoryPlayerRepository) GetByID(_ context.Context, id string) (*gamification.Player, error) {
	if p, ok := r.players[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("player not found: %s", id)
}

func (r *inMemoryPlayerRepository) GetByUsername(_ context.Context, username string) (*gamification.Player, error) {
	for _, p := range r.players {
		if p.Username() == username {
			return p, nil
		}
	}
	return nil, fmt.Errorf("player not found: %s", username)
}

func (r *inMemoryPlayerRepository) Update(_ context.Context, player *gamification.Player) error {
	r.players[player.ID()] = player
	return nil
}

func (r *inMemoryPlayerRepository) GetLeaderboard(_ context.Context, limit int) ([]*gamification.Player, error) {
	result := make([]*gamification.Player, 0, limit)
	for _, p := range r.players {
		result = append(result, p)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

// Ensure interface compliance
var _ ports.PlayerRepository = (*inMemoryPlayerRepository)(nil)
