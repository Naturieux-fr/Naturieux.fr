// Package room implements ephemeral multiplayer quiz rooms: a host creates a
// room, players join with a code, everyone answers the same shared questions,
// and a live scoreboard is exposed for polling. Rooms live in memory and are
// reclaimed when idle.
package room

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
)

// Errors returned by the manager.
var (
	ErrNotFound      = errors.New("room not found")
	ErrForbidden     = errors.New("forbidden")
	ErrBadState      = errors.New("invalid room state")
	ErrNoQuestions   = errors.New("could not generate questions")
	ErrPlayerUnknown = errors.New("player not in room")
)

// Status is a room's lifecycle phase.
type Status string

const (
	Lobby   Status = "lobby"
	Playing Status = "playing"
	Done    Status = "finished"
)

// Settings configures the shared quiz of a room.
type Settings struct {
	Difficulty  quiz.Difficulty
	QuizType    quiz.QuizType
	TaxonFilter string
	Count       int
}

// QuestionMaker produces quiz questions; satisfied by the quiz factory.
type QuestionMaker interface {
	CreateQuestion(ctx context.Context, quizType quiz.QuizType, difficulty quiz.Difficulty, taxonFilter string) (*quiz.Question, error)
}

// player holds one participant's progress.
type player struct {
	id       string
	name     string
	score    int
	streak   int // current consecutive correct answers
	best     int // best streak this game
	correct  int // number of correct answers
	index    int // index of the question to answer next
	done     bool
	lastSeen time.Time
}

// streakBonusThreshold is the streak length from which a combo bonus applies.
const streakBonusThreshold = 3

// Room is a single multiplayer game.
type Room struct {
	mu        sync.Mutex
	code      string
	hostID    string
	settings  Settings
	status    Status
	questions []*quiz.Question
	players   []*player
	created   time.Time
	activity  time.Time
}

// Manager owns all live rooms.
type Manager struct {
	mu    sync.Mutex
	rooms map[string]*Room
	maker QuestionMaker
	now   func() time.Time
}

// NewManager creates a room manager that builds questions with maker.
func NewManager(maker QuestionMaker) *Manager {
	return &Manager{rooms: make(map[string]*Room), maker: maker, now: time.Now}
}

// Code returns the room's join code.
func (r *Room) Code() string { return r.code }

// Create opens a new room hosted by the given player and returns it.
func (m *Manager) Create(hostID, hostName string, s Settings) (*Room, error) {
	if hostID == "" || hostName == "" {
		return nil, errors.New("host id and name are required")
	}
	if s.Count <= 0 {
		s.Count = 10
	}
	if s.QuizType == "" {
		s.QuizType = quiz.ImageQuiz
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	code, err := m.uniqueCode()
	if err != nil {
		return nil, err
	}
	r := &Room{
		code:     code,
		hostID:   hostID,
		settings: s,
		status:   Lobby,
		players:  []*player{{id: hostID, name: hostName, lastSeen: m.now()}},
		created:  m.now(),
		activity: m.now(),
	}
	m.rooms[code] = r
	return r, nil
}

// Get returns a room by code.
func (m *Manager) Get(code string) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[code]
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

// Join adds a player to a room still in the lobby.
func (m *Manager) Join(code, playerID, playerName string) (*Room, error) {
	r, err := m.Get(code)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status != Lobby {
		return nil, ErrBadState
	}
	r.touch(m.now())
	if p := r.find(playerID); p != nil {
		p.name = playerName
		return r, nil
	}
	r.players = append(r.players, &player{id: playerID, name: playerName, lastSeen: m.now()})
	return r, nil
}

// Start generates the shared questions and begins the game. Only the host may
// start, and only from the lobby.
func (m *Manager) Start(ctx context.Context, code, hostID string) error {
	r, err := m.Get(code)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if hostID != r.hostID {
		return ErrForbidden
	}
	if r.status != Lobby {
		return ErrBadState
	}

	questions := make([]*quiz.Question, 0, r.settings.Count)
	for i := 0; i < r.settings.Count; i++ {
		q, err := m.maker.CreateQuestion(ctx, r.settings.QuizType, r.settings.Difficulty, r.settings.TaxonFilter)
		if err != nil {
			continue
		}
		questions = append(questions, q)
	}
	if len(questions) == 0 {
		return ErrNoQuestions
	}

	r.questions = questions
	r.status = Playing
	r.touch(m.now())
	return nil
}

// AnswerResult is the outcome of answering one question in a room.
type AnswerResult struct {
	IsCorrect        bool
	CorrectName      string
	CorrectSpeciesID int
	Score            int
	TotalScore       int
	Streak           int
	Done             bool
	NextQuestion     *quiz.Question
}

// Answer records a player's answer to their current question.
func (m *Manager) Answer(code, playerID string, speciesID int, taken time.Duration) (*AnswerResult, error) {
	r, err := m.Get(code)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status != Playing {
		return nil, ErrBadState
	}
	p := r.find(playerID)
	if p == nil {
		return nil, ErrPlayerUnknown
	}
	if p.done || p.index >= len(r.questions) {
		return nil, ErrBadState
	}
	r.touch(m.now())

	q := r.questions[p.index]
	correct := q.CheckAnswer(speciesID)
	score := q.CalculateScore(taken, correct)

	// Combo: a running streak adds an escalating bonus, rewarding momentum.
	if correct {
		p.streak++
		p.correct++
		if p.streak > p.best {
			p.best = p.streak
		}
		if p.streak >= streakBonusThreshold {
			score += p.streak * 10
		}
	} else {
		p.streak = 0
	}

	p.score += score
	p.index++
	if p.index >= len(r.questions) {
		p.done = true
	}

	res := &AnswerResult{
		IsCorrect:        correct,
		CorrectName:      q.CorrectSpecies().DisplayName(),
		CorrectSpeciesID: q.CorrectSpecies().ID(),
		Score:            score,
		TotalScore:       p.score,
		Streak:           p.streak,
		Done:             p.done,
	}
	if !p.done {
		res.NextQuestion = r.questions[p.index]
	}

	// The room finishes once everyone is done.
	if r.allDone() {
		r.status = Done
	}
	return res, nil
}

// PlayerState is a participant's public standing in a room.
type PlayerState struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Score    int    `json:"score"`
	Streak   int    `json:"streak"`
	Best     int    `json:"best_streak"`
	Correct  int    `json:"correct"`
	Answered int    `json:"answered"`
	Done     bool   `json:"done"`
	Rank     int    `json:"rank"`
	IsHost   bool   `json:"is_host"`
}

// State is a snapshot of a room for polling clients.
type State struct {
	Code     string        `json:"code"`
	HostID   string        `json:"host_id"`
	Status   Status        `json:"status"`
	Total    int           `json:"total_questions"`
	Players  []PlayerState `json:"players"`
	Settings StateSettings `json:"settings"`
}

// StateSettings is the client-facing view of a room's configuration.
type StateSettings struct {
	Difficulty  string `json:"difficulty"`
	QuizType    string `json:"quiz_type"`
	TaxonFilter string `json:"taxon_filter"`
	Count       int    `json:"count"`
}

// Snapshot returns the current room state with players ranked by score.
func (m *Manager) Snapshot(code string) (State, error) {
	r, err := m.Get(code)
	if err != nil {
		return State{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	players := make([]PlayerState, len(r.players))
	for i, p := range r.players {
		players[i] = PlayerState{
			ID: p.id, Name: p.name, Score: p.score, Streak: p.streak,
			Best: p.best, Correct: p.correct, Answered: p.index, Done: p.done,
			IsHost: p.id == r.hostID,
		}
	}
	// Rank by score (desc); ties keep their order.
	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			if players[j].Score > players[i].Score {
				players[i], players[j] = players[j], players[i]
			}
		}
	}
	for i := range players {
		players[i].Rank = i + 1
	}

	return State{
		Code:    r.code,
		HostID:  r.hostID,
		Status:  r.status,
		Total:   len(r.questions),
		Players: players,
		Settings: StateSettings{
			Difficulty:  string(r.settings.Difficulty),
			QuizType:    string(r.settings.QuizType),
			TaxonFilter: r.settings.TaxonFilter,
			Count:       r.settings.Count,
		},
	}, nil
}

// CurrentQuestion returns the question a player is currently on (for image
// streaming), or nil.
func (m *Manager) CurrentQuestion(code, playerID string) (*quiz.Question, error) {
	r, err := m.Get(code)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.find(playerID)
	if p == nil || r.status != Playing || p.index >= len(r.questions) {
		return nil, ErrNotFound
	}
	return r.questions[p.index], nil
}

// Cleanup removes rooms idle for longer than ttl. Call periodically.
func (m *Manager) Cleanup(ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff := m.now().Add(-ttl)
	for code, r := range m.rooms {
		r.mu.Lock()
		idle := r.activity.Before(cutoff)
		r.mu.Unlock()
		if idle {
			delete(m.rooms, code)
		}
	}
}

// uniqueCode generates a room code not already in use. Caller holds m.mu.
func (m *Manager) uniqueCode() (string, error) {
	for attempts := 0; attempts < 20; attempts++ {
		c, err := randomCode(4)
		if err != nil {
			return "", err
		}
		if _, taken := m.rooms[c]; !taken {
			return c, nil
		}
	}
	return "", errors.New("could not allocate a room code")
}

// codeAlphabet excludes visually ambiguous characters.
const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randomCode(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating room code: %w", err)
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = codeAlphabet[int(b)%len(codeAlphabet)]
	}
	return string(out), nil
}

// --- Room helpers (caller holds r.mu) ---

func (r *Room) touch(t time.Time) { r.activity = t }

func (r *Room) find(id string) *player {
	for _, p := range r.players {
		if p.id == id {
			return p
		}
	}
	return nil
}

func (r *Room) allDone() bool {
	for _, p := range r.players {
		if !p.done {
			return false
		}
	}
	return true
}
