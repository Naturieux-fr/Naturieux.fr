// Package challenge runs the daily and weekly challenges: a quiz that is the
// same for every player over a period, with a dedicated leaderboard. Questions
// are generated once per period and cached, so everyone faces the same set.
package challenge

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
)

// Period is a challenge cadence.
type Period string

const (
	Daily  Period = "daily"
	Weekly Period = "weekly"
)

// IsValid reports whether p is a known period.
func (p Period) IsValid() bool { return p == Daily || p == Weekly }

// QuestionMaker produces quiz questions; satisfied by the quiz factory.
type QuestionMaker interface {
	CreateQuestion(ctx context.Context, quizType quiz.QuizType, difficulty quiz.Difficulty, taxonFilter string) (*quiz.Question, error)
}

// defaults for an auto-generated challenge.
const (
	defaultCount = 10
)

var defaultDifficulty = quiz.Intermediate

// Manager generates and caches the questions for each period, and maps quiz
// sessions to the challenge they belong to.
type Manager struct {
	maker   QuestionMaker
	mu      sync.Mutex
	cache   map[string][]*quiz.Question // "daily:2026-06-11" -> questions
	binding map[string]string           // sessionID -> "daily:2026-06-11"
	now     func() time.Time
}

// NewManager creates a challenge manager.
func NewManager(maker QuestionMaker) *Manager {
	return &Manager{
		maker:   maker,
		cache:   make(map[string][]*quiz.Question),
		binding: make(map[string]string),
		now:     time.Now,
	}
}

// Key returns the period key for the current time (e.g. "2026-06-11" for daily,
// "2026-W24" for weekly).
func (m *Manager) Key(p Period) string {
	t := m.now()
	if p == Weekly {
		y, w := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, w)
	}
	return t.Format("2006-01-02")
}

// Questions returns the cached question set for the current period, generating
// it on first use so that every player gets the same quiz.
func (m *Manager) Questions(ctx context.Context, p Period) (string, []*quiz.Question, error) {
	if !p.IsValid() {
		return "", nil, errors.New("invalid challenge period")
	}
	key := m.Key(p)
	cacheKey := string(p) + ":" + key

	m.mu.Lock()
	if qs, ok := m.cache[cacheKey]; ok {
		m.mu.Unlock()
		return key, qs, nil
	}
	m.mu.Unlock()

	// Generate outside the lock.
	qs := make([]*quiz.Question, 0, defaultCount)
	for i := 0; i < defaultCount; i++ {
		q, err := m.maker.CreateQuestion(ctx, quiz.ImageQuiz, defaultDifficulty, "")
		if err != nil {
			continue
		}
		qs = append(qs, q)
	}
	if len(qs) == 0 {
		return "", nil, errors.New("could not prepare challenge questions")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.cache[cacheKey]; ok { // a concurrent caller won
		return key, existing, nil
	}
	m.cache[cacheKey] = qs
	return key, qs, nil
}

// Bind records that a quiz session belongs to a challenge.
func (m *Manager) Bind(sessionID string, p Period, key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.binding[sessionID] = string(p) + ":" + key
}

// Lookup returns the challenge a session belongs to.
func (m *Manager) Lookup(sessionID string) (Period, string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.binding[sessionID]
	if !ok {
		return "", "", false
	}
	for i := 0; i < len(v); i++ {
		if v[i] == ':' {
			return Period(v[:i]), v[i+1:], true
		}
	}
	return "", "", false
}
