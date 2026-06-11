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
	CreateQuestionFor(ctx context.Context, quizType quiz.QuizType, difficulty quiz.Difficulty, speciesID int) (*quiz.Question, error)
}

// CuratedSource supplies an admin-scheduled curated quiz for a period, if any.
type CuratedSource interface {
	Scheduled(ctx context.Context, period, key string) (quizID string, ok bool, err error)
	QuizSpecies(ctx context.Context, quizID string) ([]int, error)
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
	curated CuratedSource // optional; when set, admin-scheduled quizzes win
	mu      sync.Mutex
	cache   map[string][]*quiz.Question // "daily:2026-06-11" -> questions
	binding map[string]string           // sessionID -> "daily:2026-06-11"
	now     func() time.Time
}

// NewManager creates a challenge manager. curated may be nil (auto only).
func NewManager(maker QuestionMaker, curated CuratedSource) *Manager {
	return &Manager{
		maker:   maker,
		curated: curated,
		cache:   make(map[string][]*quiz.Question),
		binding: make(map[string]string),
		now:     time.Now,
	}
}

// Invalidate drops the cached questions for a period key (e.g. after an admin
// schedules a curated quiz), so the next player gets the new set.
func (m *Manager) Invalidate(period Period, key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, string(period)+":"+key)
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

	// Generate outside the lock. A scheduled curated quiz wins; otherwise the
	// questions are auto-generated for the period.
	qs := m.curatedQuestions(ctx, p, key)
	if len(qs) == 0 {
		for i := 0; i < defaultCount; i++ {
			q, err := m.maker.CreateQuestion(ctx, quiz.ImageQuiz, defaultDifficulty, "")
			if err != nil {
				continue
			}
			qs = append(qs, q)
		}
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

// curatedQuestions returns the questions of the curated quiz scheduled for this
// period, or nil when none is scheduled.
func (m *Manager) curatedQuestions(ctx context.Context, p Period, key string) []*quiz.Question {
	if m.curated == nil {
		return nil
	}
	quizID, ok, err := m.curated.Scheduled(ctx, string(p), key)
	if err != nil || !ok {
		return nil
	}
	species, err := m.curated.QuizSpecies(ctx, quizID)
	if err != nil {
		return nil
	}
	qs := make([]*quiz.Question, 0, len(species))
	for _, sp := range species {
		q, err := m.maker.CreateQuestionFor(ctx, quiz.ImageQuiz, defaultDifficulty, sp)
		if err != nil {
			continue
		}
		qs = append(qs, q)
	}
	return qs
}

// Bind records that a quiz session belongs to a challenge.
func (m *Manager) Bind(sessionID string, p Period, key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.binding[sessionID] = string(p) + ":" + key
}

// Release forgets a session→challenge binding once the run is recorded, so the
// binding map does not grow without bound.
func (m *Manager) Release(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.binding, sessionID)
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
