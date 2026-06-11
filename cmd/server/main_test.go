package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	adminapp "github.com/Naturieux-fr/Naturieux.fr/internal/application/admin"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestSecurityHeaders(t *testing.T) {
	h := securityHeaders(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	h.ServeHTTP(rec, req)
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing nosniff")
	}
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Error("HSTS expected over https")
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil)) // plain http
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Error("no HSTS over plain http")
	}
}

func TestCorsMiddleware(t *testing.T) {
	h := corsMiddleware(okHandler())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/", nil))
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS origin")
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), "DELETE") {
		t.Error("DELETE should be allowed")
	}
}

func TestMetrics(t *testing.T) {
	m := newMetrics()
	h := m.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/boom" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/boom", nil))

	rec := httptest.NewRecorder()
	m.handler(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "naturieux_requests_total") || !strings.Contains(body, "naturieux_requests_errors_total 1") {
		t.Errorf("metrics output: %s", body)
	}
}

func TestAuthSecretAndSeed(t *testing.T) {
	t.Setenv("AUTH_SECRET", "fixed-secret")
	if authSecret() != "fixed-secret" {
		t.Error("authSecret should use env")
	}

	db, _ := sqlite.Open(":memory:")
	defer func() { _ = db.Close() }()
	playerRepo := sqlite.NewPlayerRepository(db)
	if err := ensureDemoPlayer(playerRepo); err != nil {
		t.Fatalf("ensureDemoPlayer: %v", err)
	}
	if taxrefLoaded(db) {
		t.Error("no taxref species yet")
	}

	t.Setenv("ADMIN_USERNAME", "boss")
	t.Setenv("ADMIN_PASSWORD", "bosspass1")
	svc := adminapp.NewService(playerRepo, "secret")
	seedAdminFromEnv(svc)
	if _, err := svc.Login(context.Background(), "boss", "bosspass1"); err != nil {
		t.Errorf("seeded admin login: %v", err)
	}
}
