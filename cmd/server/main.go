// Package main is the entry point for the Naturieux quiz server.
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
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
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/storage"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/account"
	adminapp "github.com/Naturieux-fr/Naturieux.fr/internal/application/admin"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/challenge"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/room"
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

	// Initialize the species repository. SPECIES_SOURCE selects the backend:
	//   mock    - in-memory sample data (also enabled by DEV_MODE)
	//   taxref  - local TAXREF reference + our own photo collection
	//   inat    - iNaturalist API with a local cache (default)
	speciesRepo, err := buildSpeciesRepo(backgroundCtx, db, devMode)
	if err != nil {
		log.Fatalf("Failed to initialize species source: %v", err)
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

	// Admin back-office: auth + photo management (photos only when TAXREF)
	secret := authSecret()
	authService := adminapp.NewService(playerRepo, secret)
	seedAdminFromEnv(authService)
	taxrefRepo, _ := speciesRepo.(*taxref.Repository) // nil unless SPECIES_SOURCE=taxref
	mediaStore, err := storage.FromEnv(backgroundCtx)
	if err != nil {
		log.Fatalf("Failed to initialize media storage: %v", err)
	}
	log.Printf("🗄️  Media storage ready")

	// Player accounts: registration (open or invite-only) + login.
	regMode := account.Open
	if os.Getenv("REGISTRATION_MODE") == "invite" {
		regMode = account.Invite
	}
	inviteRepo := sqlite.NewInviteRepository(db)
	accountService := account.NewService(playerRepo, playerRepo, inviteRepo, secret, regMode)
	accountHandler := httphandler.NewAccountHandler(accountService)
	handler.SetRegistrationMode(string(regMode))
	handler.SetAuthenticator(accountService) // gameplay endpoints derive the player from the session token
	log.Printf("👤 Account registration: %s", regMode)

	adminHandler := httphandler.NewAdminHandler(authService, taxrefRepo, mediaStore, accountService)

	// Multiplayer rooms (in-memory, polled by clients; reclaimed when idle)
	roomManager := room.NewManager(questionFactory)
	adminHandler.SetAdminData(playerRepo, roomManager)

	// Daily/weekly challenges (shared question set + dedicated leaderboard);
	// admin-curated quizzes can be scheduled as the défi.
	curatedRepo := sqlite.NewCuratedRepository(db)
	challengeManager := challenge.NewManager(questionFactory, curatedRepo)
	challengeHandler := httphandler.NewChallengeHandler(challengeManager, quizService, sqlite.NewChallengeRepository(db), playerRepo)
	challengeHandler.SetAuthenticator(accountService)
	adminHandler.SetCuratedData(curatedRepo, challengeManager)
	go reapRooms(backgroundCtx, roomManager)
	roomHandler := httphandler.NewRoomHandler(roomManager, handler)

	// Create HTTP server
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	adminHandler.RegisterRoutes(mux)
	roomHandler.RegisterRoutes(mux)
	accountHandler.RegisterRoutes(mux)
	challengeHandler.RegisterRoutes(mux)

	// Serve the admin page
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/admin.html")
	})

	// Locally stored uploads are read by the quiz-image proxy (the raw /media
	// path is not exposed to players; only the admin file server uses it).
	if local, ok := mediaStore.(*storage.Local); ok {
		handler.SetLocalMediaDir(local.Dir())
		mux.Handle("/media/", http.StripPrefix("/media/", http.FileServer(http.Dir(local.Dir()))))
	}

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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// buildSpeciesRepo selects and constructs the species data source from
// the DEV_MODE and SPECIES_SOURCE environment variables.
func buildSpeciesRepo(ctx context.Context, db *sql.DB, devMode bool) (ports.SpeciesRepository, error) {
	source := os.Getenv("SPECIES_SOURCE")
	if source == "" {
		// Auto-detect: demo data in dev mode, otherwise the local TAXREF as
		// soon as it is loaded, and only fall back to iNaturalist when there
		// is no local reference yet.
		switch {
		case devMode:
			source = "mock"
		case taxrefLoaded(db):
			source = "taxref"
		default:
			source = "inat"
		}
	}

	switch source {
	case "mock":
		log.Println("🔧 Species source: mock data")
		return mock.NewSpeciesRepository(), nil

	case "taxref":
		log.Println("🇫🇷 Species source: local TAXREF + owned photos")
		if err := taxref.EnsureSchema(db); err != nil {
			return nil, err
		}
		repo := taxref.NewRepository(db)
		count, err := repo.CountSpecies(ctx)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			log.Println("⚠️  TAXREF is empty — import it with: go run ./cmd/importtaxref -file taxon.txt")
		} else {
			log.Printf("TAXREF loaded: %d species (version %q)", count, repo.Version(ctx))
		}
		return repo, nil

	default:
		log.Println("🌿 Species source: iNaturalist API with local cache")
		speciesCache, err := cache.New(db, inaturalist.NewClient(), cache.WithPlaceID(francePlaceID))
		if err != nil {
			return nil, err
		}
		go speciesCache.StartAutoWarm(ctx, cache.DefaultWarmInterval, cache.WarmTaxa, cache.DefaultWarmTarget)
		return speciesCache, nil
	}
}

// reapRooms periodically discards multiplayer rooms idle for over an hour.
func reapRooms(ctx context.Context, mgr *room.Manager) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mgr.Cleanup(time.Hour)
		}
	}
}

// taxrefLoaded reports whether a populated TAXREF reference is present.
func taxrefLoaded(db *sql.DB) bool {
	if err := taxref.EnsureSchema(db); err != nil {
		return false
	}
	count, err := taxref.NewRepository(db).CountSpecies(context.Background())
	return err == nil && count > 0
}

// authSecret returns the token-signing secret from AUTH_SECRET, or a random
// one (admin sessions then reset on restart, which is acceptable).
func authSecret() string {
	if s := os.Getenv("AUTH_SECRET"); s != "" {
		return s
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		log.Fatalf("generating auth secret: %v", err)
	}
	log.Println("⚠️  AUTH_SECRET not set — using a random secret (admin sessions reset on restart)")
	return hex.EncodeToString(buf)
}

// seedAdminFromEnv creates/updates the admin account when ADMIN_USERNAME and
// ADMIN_PASSWORD are set.
func seedAdminFromEnv(svc *adminapp.Service) {
	user, pass := os.Getenv("ADMIN_USERNAME"), os.Getenv("ADMIN_PASSWORD")
	if user == "" || pass == "" {
		log.Println("ℹ️  ADMIN_USERNAME/ADMIN_PASSWORD not set — no admin account seeded")
		return
	}
	if err := svc.SeedAdmin(context.Background(), user, pass); err != nil {
		log.Fatalf("seeding admin: %v", err)
	}
	log.Printf("Admin account ready: %s", user)
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
