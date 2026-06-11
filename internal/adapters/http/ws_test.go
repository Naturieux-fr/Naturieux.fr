package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	httphandler "github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/mock"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/room"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
)

func TestRoomWebSocket(t *testing.T) {
	mgr := room.NewManager(appquiz.NewQuestionFactory(mock.NewSpeciesRepository()))
	rm, _, err := mgr.Create("h1", "Host", room.Settings{Difficulty: quiz.Beginner, Count: 1})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	mux := http.NewServeMux()
	httphandler.NewRoomHandler(mgr, httphandler.NewHandler(nil, false)).RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/rooms/" + rm.Code() + "/ws"
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = c.Close(websocket.StatusNormalClosure, "bye") }()

	// The server pushes the current state immediately.
	typ, data, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if typ != websocket.MessageText || !strings.Contains(string(data), rm.Code()) {
		t.Errorf("unexpected first message: %s", data)
	}

	// A WebSocket on an unknown room is refused.
	if _, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/api/v1/rooms/ZZZZ/ws", nil); err == nil {
		t.Error("ws on unknown room should fail")
	}
}
