package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httphandler "github.com/fieve/naturieux/internal/adapters/http"
)

func TestHandler_HandleHealthCheck(t *testing.T) {
	handler := httphandler.NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HandleHealthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HandleHealthCheck() status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response httphandler.Response
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("HandleHealthCheck() Success = false, want true")
	}
}

func TestHandler_HandleHealthCheck_WrongMethod(t *testing.T) {
	handler := httphandler.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HandleHealthCheck(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("HandleHealthCheck() status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandler_HandleStartSession_WrongMethod(t *testing.T) {
	handler := httphandler.NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quiz/start", nil)
	rec := httptest.NewRecorder()

	handler.HandleStartSession(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("HandleStartSession() status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandler_HandleStartSession_InvalidJSON(t *testing.T) {
	handler := httphandler.NewHandler(nil)

	body := bytes.NewBufferString("invalid json")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/start", body)
	rec := httptest.NewRecorder()

	handler.HandleStartSession(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleStartSession() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandler_HandleStartSession_MissingUserID(t *testing.T) {
	handler := httphandler.NewHandler(nil)

	reqBody := httphandler.StartSessionRequest{
		Difficulty:    "beginner",
		QuestionCount: 5,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/start", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleStartSession(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleStartSession() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandler_HandleSubmitAnswer_WrongMethod(t *testing.T) {
	handler := httphandler.NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quiz/answer", nil)
	rec := httptest.NewRecorder()

	handler.HandleSubmitAnswer(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("HandleSubmitAnswer() status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandler_HandleSubmitAnswer_SessionNotFound(t *testing.T) {
	handler := httphandler.NewHandler(nil)

	reqBody := httphandler.SubmitAnswerRequest{
		SessionID:   "nonexistent",
		SpeciesID:   1,
		TimeTakenMs: 5000,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/answer", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleSubmitAnswer(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("HandleSubmitAnswer() status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandler_HandleAbandonSession_WrongMethod(t *testing.T) {
	handler := httphandler.NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quiz/abandon", nil)
	rec := httptest.NewRecorder()

	handler.HandleAbandonSession(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("HandleAbandonSession() status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandler_HandleAbandonSession_SessionNotFound(t *testing.T) {
	handler := httphandler.NewHandler(nil)

	reqBody := map[string]string{"session_id": "nonexistent"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/abandon", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleAbandonSession(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("HandleAbandonSession() status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandler_RegisterRoutes(t *testing.T) {
	handler := httphandler.NewHandler(nil)
	mux := http.NewServeMux()

	handler.RegisterRoutes(mux)

	// Test that routes are registered by checking health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("RegisterRoutes() health check status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_HandleSubmitAnswer_MissingSessionID(t *testing.T) {
	handler := httphandler.NewHandler(nil)

	reqBody := httphandler.SubmitAnswerRequest{
		SpeciesID:   1,
		TimeTakenMs: 5000,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/answer", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleSubmitAnswer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleSubmitAnswer() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandler_HandleAbandonSession_InvalidJSON(t *testing.T) {
	handler := httphandler.NewHandler(nil)

	body := bytes.NewBufferString("invalid json")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/abandon", body)
	rec := httptest.NewRecorder()

	handler.HandleAbandonSession(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleAbandonSession() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandler_HandleSubmitAnswer_InvalidJSON(t *testing.T) {
	handler := httphandler.NewHandler(nil)

	body := bytes.NewBufferString("invalid json")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/answer", body)
	rec := httptest.NewRecorder()

	handler.HandleSubmitAnswer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleSubmitAnswer() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
