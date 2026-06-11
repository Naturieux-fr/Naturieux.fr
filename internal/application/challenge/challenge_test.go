package challenge

import (
	"context"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/mock"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
)

func newManager() *Manager {
	factory := appquiz.NewQuestionFactory(mock.NewSpeciesRepository())
	return NewManager(factory, nil)
}

func TestPeriod_IsValid(t *testing.T) {
	if !Daily.IsValid() || !Weekly.IsValid() {
		t.Error("Daily/Weekly should be valid")
	}
	if Period("yearly").IsValid() {
		t.Error("unknown period should be invalid")
	}
}

func TestManager_Key(t *testing.T) {
	m := newManager()
	day := m.Key(Daily)
	if len(day) != 10 || day[4] != '-' { // YYYY-MM-DD
		t.Errorf("daily key %q is not a date", day)
	}
	week := m.Key(Weekly)
	if len(week) < 7 || week[4] != '-' || week[5] != 'W' { // YYYY-Www
		t.Errorf("weekly key %q is not an ISO week", week)
	}
}

func TestManager_QuestionsCachedAndBound(t *testing.T) {
	m := newManager()
	ctx := context.Background()

	key, qs, err := m.Questions(ctx, Daily)
	if err != nil {
		t.Fatalf("Questions: %v", err)
	}
	if key == "" || len(qs) == 0 {
		t.Fatalf("expected a key and questions, got %q / %d", key, len(qs))
	}

	// A second call returns the same cached set.
	_, qs2, _ := m.Questions(ctx, Daily)
	if &qs[0] != &qs2[0] && qs[0].ID() != qs2[0].ID() {
		t.Error("expected the same cached questions on the second call")
	}

	// Bind / Lookup / Release a session.
	m.Bind("sess-1", Daily, key)
	if p, k, ok := m.Lookup("sess-1"); !ok || p != Daily || k != key {
		t.Errorf("Lookup = (%q,%q,%v), want (daily,%q,true)", p, k, ok, key)
	}
	m.Release("sess-1")
	if _, _, ok := m.Lookup("sess-1"); ok {
		t.Error("Lookup should fail after Release")
	}

	// Invalidate drops the cache so a fresh set is generated.
	m.Invalidate(Daily, key)
	if _, _, ok := m.Lookup("missing"); ok {
		t.Error("unknown session should not be found")
	}
}

func TestManager_InvalidPeriod(t *testing.T) {
	if _, _, err := newManager().Questions(context.Background(), Period("bad")); err == nil {
		t.Error("expected an error for an invalid period")
	}
}
