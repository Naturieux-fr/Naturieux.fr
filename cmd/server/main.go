// Package main is the entry point for the Naturieux quiz server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/cache"
	httphandler "github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/inaturalist"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/mock"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

const (
	defaultPort = "8080"
	// francePlaceID is the iNaturalist place ID used to focus the quiz
	// on species observable in France.
	francePlaceID = 6753
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Check for dev mode
	devMode := os.Getenv("DEV_MODE") == "true" || os.Getenv("DEV_MODE") == "1"

	// Open SQLite database for persistence
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "naturieux.db"
	}
	db, err := sqlite.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()
	log.Printf("Database: %s", dbPath)

	// Background tasks stop when the server shuts down
	backgroundCtx, stopBackground := context.WithCancel(context.Background())
	defer stopBackground()

	// Initialize species repository based on mode
	var speciesRepo ports.SpeciesRepository
	if devMode {
		log.Println("🔧 DEV MODE ENABLED - Using mock data")
		speciesRepo = mock.NewSpeciesRepository()
	} else {
		log.Println("🌿 Production mode - Using iNaturalist API with local cache")
		speciesCache, err := cache.New(db, inaturalist.NewClient(),
			cache.WithPlaceID(francePlaceID))
		if err != nil {
			log.Fatalf("Failed to initialize species cache: %v", err)
		}
		go speciesCache.StartAutoWarm(backgroundCtx, cache.DefaultWarmInterval,
			cache.WarmTaxa, cache.DefaultWarmTarget)
		speciesRepo = speciesCache
	}

	playerRepo := sqlite.NewPlayerRepository(db)
	sessionRepo := sqlite.NewSessionRepository(db)

	// Ensure the demo player exists
	if err := ensureDemoPlayer(playerRepo); err != nil {
		log.Fatalf("Failed to create demo player: %v", err)
	}

	// Create question factory
	questionFactory := appquiz.NewQuestionFactory(
		speciesRepo,
		appquiz.WithTaxonFilter(""),            // All taxa
		appquiz.WithPlaceFilter(francePlaceID), // France
	)

	// Create quiz service
	quizService := appquiz.NewService(
		questionFactory,
		sessionRepo,
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

// ensureDemoPlayer creates the demo player if it does not exist yet.
func ensureDemoPlayer(playerRepo ports.PlayerRepository) error {
	ctx := context.Background()
	if _, err := playerRepo.GetByID(ctx, "demo"); err == nil {
		return nil
	}

	demoPlayer, err := gamification.NewPlayer("demo", "demo_user")
	if err != nil {
		return err
	}
	return playerRepo.Create(ctx, demoPlayer)
}
