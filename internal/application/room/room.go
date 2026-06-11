// Package room implements ephemeral multiplayer quiz rooms: a host creates a
// room, players join with a code, everyone answers the same shared questions,
// and a live scoreboard is exposed for polling. Rooms live in memory and are
// reclaimed when idle.
package room

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// Mode is a multiplayer game format.
type Mode string

const (
	// Classic: everyone answers every question; highest score wins.
	Classic Mode = "classic"
	// Elimination: a single wrong answer knocks a player out (sudden death);
	// the last survivors rank highest.
	Elimination Mode = "elimination"
)

// Settings configures the shared quiz of a room.
type Settings struct {
	Difficulty  quiz.Difficulty
	QuizType    quiz.QuizType
	TaxonFilter string
	Count       int
	Mode        Mode
	AnswerMode  string // "choices" or "free" (rendered client-side)
	Public      bool   // listed publicly so anyone can join without a code
}

// RoomSummary is a public room's listing entry.
type RoomSummary struct {
	Code       string        `json:"code"`
	Host       string        `json:"host"`
	Players    int           `json:"players"`
	Mode       Mode          `json:"mode"`
	QuizType   quiz.QuizType `json:"quiz_type"`
	AnswerMode string        `json:"answer_mode"`
}

// QuestionMaker produces quiz questions; satisfied by the quiz factory.
type QuestionMaker interface {
	CreateQuestion(ctx context.Context, quizType quiz.QuizType, difficulty quiz.Difficulty, taxonFilter string) (*quiz.Question, error)
}

// player holds one participant's progress.
type player struct {
	id         string
	name       string
	token      string // secret, required to act as this player
	score      int
	streak     int  // current consecutive correct answers
	best       int  // best streak this game
	correct    int  // number of correct answers
	answers    int  // number of questions answered
	answered   bool // has answered the current round
	done       bool
	eliminated bool      // knocked out in elimination mode
	since      time.Time // when the current round was delivered (server-side timing)
	lastSeen   time.Time
}

// streakBonusThreshold is the streak length from which a combo bonus applies.
const streakBonusThreshold = 3

// maxRoomQuestions caps a room's question count to bound work per room.
const maxRoomQuestions = 30

// Room is a single multiplayer game. All players answer the same question
// (round) and the game advances only once everyone has answered it.
type Room struct {
	mu        sync.Mutex
	code      string
	hostID    string
	settings  Settings
	status    Status
	questions []*quiz.Question
	round     int // index of the question everyone is currently answering
	players   []*player
	created   time.Time
	activity  time.Time
	subs      map[int]chan State // live subscribers (WebSocket)
	subSeq    int
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

// Create opens a new room hosted by the given player and returns the room with
// the host's secret token (needed to start and to answer).
func (m *Manager) Create(hostID, hostName string, s Settings) (*Room, string, error) {
	if hostID == "" || hostName == "" {
		return nil, "", errors.New("host id and name are required")
	}
	if s.Count <= 0 {
		s.Count = 10
	}
	if s.Count > maxRoomQuestions {
		s.Count = maxRoomQuestions
	}
	if s.QuizType == "" {
		s.QuizType = quiz.ImageQuiz
	}
	if s.Mode == "" {
		s.Mode = Classic
	}
	if s.AnswerMode == "" {
		s.AnswerMode = "choices"
	}

	token, err := randomToken()
	if err != nil {
		return nil, "", err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	code, err := m.uniqueCode()
	if err != nil {
		return nil, "", err
	}
	r := &Room{
		code:     code,
		hostID:   hostID,
		settings: s,
		status:   Lobby,
		players:  []*player{{id: hostID, name: hostName, token: token, lastSeen: m.now()}},
		created:  m.now(),
		activity: m.now(),
	}
	m.rooms[code] = r
	return r, token, nil
}

// ListOpen returns the public rooms still in the lobby (joinable by anyone),
// most recent first.
func (m *Manager) ListOpen() []RoomSummary {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []RoomSummary{}
	for _, r := range m.rooms {
		r.mu.Lock()
		if r.settings.Public && r.status == Lobby {
			host := ""
			if len(r.players) > 0 {
				host = r.players[0].name
			}
			out = append(out, RoomSummary{
				Code: r.code, Host: host, Players: len(r.players),
				Mode: r.settings.Mode, QuizType: r.settings.QuizType, AnswerMode: r.settings.AnswerMode,
			})
		}
		r.mu.Unlock()
	}
	return out
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

// Join adds a player to a room still in the lobby and returns their secret
// token. Re-joining with the same id returns a fresh token.
func (m *Manager) Join(code, playerID, playerName string) (*Room, string, error) {
	if playerID == "" || playerName == "" {
		return nil, "", errors.New("player id and name are required")
	}
	token, err := randomToken()
	if err != nil {
		return nil, "", err
	}
	r, err := m.Get(code)
	if err != nil {
		return nil, "", err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status != Lobby {
		return nil, "", ErrBadState
	}
	r.touch(m.now())
	if p := r.find(playerID); p != nil {
		p.name = playerName
		p.token = token
		r.broadcastLocked()
		return r, token, nil
	}
	r.players = append(r.players, &player{id: playerID, name: playerName, token: token, lastSeen: m.now()})
	r.broadcastLocked()
	return r, token, nil
}

// Start generates the shared questions and begins the game. Only the host
// (authenticated by token) may start, and only from the lobby. Question
// generation happens without holding the room lock.
func (m *Manager) Start(ctx context.Context, code, token string) error {
	r, err := m.Get(code)
	if err != nil {
		return err
	}

	// Validate the request and snapshot the settings under a brief lock.
	r.mu.Lock()
	host := r.find(r.hostID)
	if host == nil || host.token != token {
		r.mu.Unlock()
		return ErrForbidden
	}
	if r.status != Lobby {
		r.mu.Unlock()
		return ErrBadState
	}
	settings := r.settings
	r.mu.Unlock()

	// Build the questions outside the lock (bounded by maxRoomQuestions).
	questions := make([]*quiz.Question, 0, settings.Count)
	for i := 0; i < settings.Count; i++ {
		q, err := m.maker.CreateQuestion(ctx, settings.QuizType, settings.Difficulty, settings.TaxonFilter)
		if err != nil {
			continue
		}
		questions = append(questions, q)
	}
	if len(questions) == 0 {
		return ErrNoQuestions
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status != Lobby { // a concurrent Start already ran
		return ErrBadState
	}
	r.questions = questions
	r.status = Playing
	now := m.now()
	for _, p := range r.players {
		p.since = now
	}
	r.touch(now)
	r.broadcastLocked()
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
	Done             bool // the game is over for this player (finished or eliminated)
	Eliminated       bool
	Waiting          bool // answered, now waiting for the others this round
}

// Answer records, for the player identified by token, an answer to the current
// round's question. Timing is measured server-side; the species id may be 0 (a
// deliberate "no answer", e.g. timeout or exhausted free-text attempts).
func (m *Manager) Answer(code, token string, speciesID int) (*AnswerResult, error) {
	r, err := m.Get(code)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status != Playing {
		return nil, ErrBadState
	}
	p := r.findByToken(token)
	if p == nil {
		return nil, ErrPlayerUnknown
	}
	if p.done || p.answered || r.round >= len(r.questions) {
		return nil, ErrBadState
	}
	q := r.questions[r.round]
	correct := q.CheckAnswer(speciesID)
	return r.record(p, q, correct, m.now()), nil
}

// GuessName checks a free-text guess against the current round's question
// without recording it. It returns whether the guess is correct and, only when
// correct, the species id so the caller can record the answer.
func (m *Manager) GuessName(code, token, guess string) (bool, int, error) {
	r, err := m.Get(code)
	if err != nil {
		return false, 0, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status != Playing {
		return false, 0, ErrBadState
	}
	p := r.findByToken(token)
	if p == nil {
		return false, 0, ErrPlayerUnknown
	}
	if p.done || p.answered || r.round >= len(r.questions) {
		return false, 0, ErrBadState
	}
	q := r.questions[r.round]
	if q.CorrectSpecies().MatchesName(guess) {
		return true, q.CorrectSpecies().ID(), nil
	}
	return false, 0, nil
}

// record scores a player's answer to the current round, then advances the whole
// room once everyone has answered. Caller holds r.mu.
func (r *Room) record(p *player, q *quiz.Question, correct bool, now time.Time) *AnswerResult {
	r.touch(now)
	taken := now.Sub(p.since)
	if taken < 0 {
		taken = 0
	}
	score := q.CalculateScore(taken, correct)

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
	p.answers++
	p.answered = true
	// Sudden death: a wrong answer ends this player's game immediately.
	if !correct && r.settings.Mode == Elimination {
		p.eliminated = true
		p.done = true
	}

	r.advanceIfRoundComplete(now)

	res := &AnswerResult{
		IsCorrect:        correct,
		CorrectName:      q.CorrectSpecies().DisplayName(),
		CorrectSpeciesID: q.CorrectSpecies().ID(),
		Score:            score,
		TotalScore:       p.score,
		Streak:           p.streak,
		Eliminated:       p.eliminated,
		Done:             p.eliminated || r.status == Done,
	}
	res.Waiting = !res.Done
	r.broadcastLocked()
	return res
}

// advanceIfRoundComplete moves the whole room to the next question once every
// still-active player has answered the current round. Caller holds r.mu.
func (r *Room) advanceIfRoundComplete(now time.Time) {
	for _, p := range r.players {
		if !p.done && !p.answered {
			return // someone still has to answer
		}
	}
	if r.allDone() {
		r.status = Done
		return
	}
	r.round++
	if r.round >= len(r.questions) {
		r.status = Done
		return
	}
	for _, p := range r.players {
		if !p.done {
			p.answered = false
			p.since = now
		}
	}
}

// PlayerState is a participant's public standing in a room.
type PlayerState struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Score       int    `json:"score"`
	Streak      int    `json:"streak"`
	Best        int    `json:"best_streak"`
	Correct     int    `json:"correct"`
	Answered    int    `json:"answered"`
	HasAnswered bool   `json:"has_answered"` // answered the current round
	Done        bool   `json:"done"`
	Eliminated  bool   `json:"eliminated"`
	Rank        int    `json:"rank"`
	IsHost      bool   `json:"is_host"`
}

// State is a snapshot of a room for polling clients.
type State struct {
	Code     string        `json:"code"`
	HostID   string        `json:"host_id"`
	Status   Status        `json:"status"`
	Total    int           `json:"total_questions"`
	Round    int           `json:"round"` // current shared question index
	Players  []PlayerState `json:"players"`
	Settings StateSettings `json:"settings"`
}

// StateSettings is the client-facing view of a room's configuration.
type StateSettings struct {
	Difficulty  string `json:"difficulty"`
	QuizType    string `json:"quiz_type"`
	TaxonFilter string `json:"taxon_filter"`
	Count       int    `json:"count"`
	Mode        string `json:"mode"`
	AnswerMode  string `json:"answer_mode"`
}

// Snapshot returns the current room state with players ranked by score.
func (m *Manager) Snapshot(code string) (State, error) {
	r, err := m.Get(code)
	if err != nil {
		return State{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshotLocked(), nil
}

// snapshotLocked builds the public room state. Caller holds r.mu.
func (r *Room) snapshotLocked() State {
	players := make([]PlayerState, len(r.players))
	for i, p := range r.players {
		players[i] = PlayerState{
			ID: p.id, Name: p.name, Score: p.score, Streak: p.streak,
			Best: p.best, Correct: p.correct, Answered: p.answers,
			HasAnswered: p.answered, Done: p.done,
			Eliminated: p.eliminated, IsHost: p.id == r.hostID,
		}
	}
	// Rank survivors above the eliminated, then by score (desc).
	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			if rankBefore(players[j], players[i]) {
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
		Round:   r.round,
		Players: players,
		Settings: StateSettings{
			Difficulty:  string(r.settings.Difficulty),
			QuizType:    string(r.settings.QuizType),
			TaxonFilter: r.settings.TaxonFilter,
			Count:       r.settings.Count,
			Mode:        string(r.settings.Mode),
			AnswerMode:  r.settings.AnswerMode,
		},
	}
}

// Subscribe registers a live listener for a room and returns its id and a
// channel of state snapshots. Call Unsubscribe with the id when done.
func (m *Manager) Subscribe(code string) (int, <-chan State, error) {
	r, err := m.Get(code)
	if err != nil {
		return 0, nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.subs == nil {
		r.subs = make(map[int]chan State)
	}
	r.subSeq++
	id := r.subSeq
	ch := make(chan State, 1)
	r.subs[id] = ch
	return id, ch, nil
}

// Unsubscribe removes a live listener.
func (m *Manager) Unsubscribe(code string, id int) {
	r, err := m.Get(code)
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if ch, ok := r.subs[id]; ok {
		delete(r.subs, id)
		close(ch)
	}
}

// broadcastLocked pushes the current state to every subscriber, keeping only
// the latest snapshot per slow consumer. Caller holds r.mu.
func (r *Room) broadcastLocked() {
	if len(r.subs) == 0 {
		return
	}
	st := r.snapshotLocked()
	for _, ch := range r.subs {
		select {
		case ch <- st:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- st:
			default:
			}
		}
	}
}

// rankBefore reports whether a should rank above b: survivors first, then by
// score.
func rankBefore(a, b PlayerState) bool {
	if a.Eliminated != b.Eliminated {
		return !a.Eliminated
	}
	return a.Score > b.Score
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
	if p == nil || r.status != Playing || p.done || p.answered || r.round >= len(r.questions) {
		return nil, ErrNotFound
	}
	return r.questions[r.round], nil
}

// Cleanup removes rooms idle for longer than ttl. Call periodically.
func (m *Manager) Cleanup(ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff := m.now().Add(-ttl)
	for code, r := range m.rooms {
		r.mu.Lock()
		idle := r.activity.Before(cutoff)
		if idle {
			for id, ch := range r.subs {
				delete(r.subs, id)
				close(ch)
			}
		}
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

func (r *Room) findByToken(token string) *player {
	if token == "" {
		return nil
	}
	for _, p := range r.players {
		if p.token == token {
			return p
		}
	}
	return nil
}

// randomToken returns an unguessable per-player secret.
func randomToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating room token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func (r *Room) allDone() bool {
	for _, p := range r.players {
		if !p.done {
			return false
		}
	}
	return true
}

// Count returns the number of live rooms.
func (m *Manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rooms)
}
