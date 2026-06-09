package quiz

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// SessionStatus represents the current status of a quiz session.
type SessionStatus string

const (
	SessionPending    SessionStatus = "pending"
	SessionInProgress SessionStatus = "in_progress"
	SessionCompleted  SessionStatus = "completed"
	SessionAbandoned  SessionStatus = "abandoned"
)

// Answer represents a player's answer to a question.
type Answer struct {
	QuestionID string
	SpeciesID  int
	TimeTaken  time.Duration
	IsCorrect  bool
	Score      int
	AnsweredAt time.Time
}

// Session represents a quiz session.
type Session struct {
	id           string
	userID       string
	difficulty   Difficulty
	quizTypes    []QuizType
	taxonFilter  string
	questions    []*Question
	answers      []Answer
	currentIndex int
	totalScore   int
	streak       int
	maxStreak    int
	status       SessionStatus
	startedAt    time.Time
	completedAt  *time.Time
}

// SessionBuilder helps construct quiz sessions.
type SessionBuilder struct {
	userID      string
	difficulty  Difficulty
	quizTypes   []QuizType
	taxonFilter string
	questions   []*Question
}

// NewSessionBuilder creates a new session builder.
func NewSessionBuilder() *SessionBuilder {
	return &SessionBuilder{
		difficulty: Beginner,
		quizTypes:  []QuizType{ImageQuiz},
	}
}

// WithUserID sets the user ID.
func (b *SessionBuilder) WithUserID(userID string) *SessionBuilder {
	b.userID = userID
	return b
}

// WithDifficulty sets the difficulty level.
func (b *SessionBuilder) WithDifficulty(d Difficulty) *SessionBuilder {
	b.difficulty = d
	return b
}

// WithQuizTypes sets the allowed quiz types.
func (b *SessionBuilder) WithQuizTypes(types ...QuizType) *SessionBuilder {
	b.quizTypes = types
	return b
}

// WithTaxonFilter sets the taxon filter.
func (b *SessionBuilder) WithTaxonFilter(taxon string) *SessionBuilder {
	b.taxonFilter = taxon
	return b
}

// WithQuestions sets the questions.
func (b *SessionBuilder) WithQuestions(questions []*Question) *SessionBuilder {
	b.questions = questions
	return b
}

// Build creates the session.
func (b *SessionBuilder) Build() (*Session, error) {
	if b.userID == "" {
		return nil, errors.New("user ID is required")
	}
	if len(b.questions) == 0 {
		return nil, errors.New("at least one question is required")
	}

	return &Session{
		id:           uuid.New().String(),
		userID:       b.userID,
		difficulty:   b.difficulty,
		quizTypes:    b.quizTypes,
		taxonFilter:  b.taxonFilter,
		questions:    b.questions,
		answers:      make([]Answer, 0, len(b.questions)),
		currentIndex: 0,
		status:       SessionPending,
	}, nil
}

// ID returns the session ID.
func (s *Session) ID() string {
	return s.id
}

// UserID returns the user ID.
func (s *Session) UserID() string {
	return s.userID
}

// Difficulty returns the difficulty level.
func (s *Session) Difficulty() Difficulty {
	return s.difficulty
}

// Status returns the current status.
func (s *Session) Status() SessionStatus {
	return s.status
}

// TotalScore returns the total score.
func (s *Session) TotalScore() int {
	return s.totalScore
}

// CurrentStreak returns the current streak.
func (s *Session) CurrentStreak() int {
	return s.streak
}

// MaxStreak returns the max streak achieved.
func (s *Session) MaxStreak() int {
	return s.maxStreak
}

// QuestionsCount returns the total number of questions.
func (s *Session) QuestionsCount() int {
	return len(s.questions)
}

// AnsweredCount returns the number of answered questions.
func (s *Session) AnsweredCount() int {
	return len(s.answers)
}

// CorrectCount returns the number of correct answers.
func (s *Session) CorrectCount() int {
	count := 0
	for _, a := range s.answers {
		if a.IsCorrect {
			count++
		}
	}
	return count
}

// CurrentQuestion returns the current question or nil if finished.
func (s *Session) CurrentQuestion() *Question {
	if s.currentIndex >= len(s.questions) {
		return nil
	}
	return s.questions[s.currentIndex]
}

// Start begins the session.
func (s *Session) Start() error {
	if s.status != SessionPending {
		return errors.New("session already started")
	}
	s.status = SessionInProgress
	s.startedAt = time.Now()
	return nil
}

// SubmitAnswer records an answer for the current question.
func (s *Session) SubmitAnswer(speciesID int, timeTaken time.Duration) (*Answer, error) {
	if s.status != SessionInProgress {
		return nil, errors.New("session not in progress")
	}

	question := s.CurrentQuestion()
	if question == nil {
		return nil, errors.New("no more questions")
	}

	isCorrect := question.CheckAnswer(speciesID)
	score := question.CalculateScore(timeTaken, isCorrect)

	// Update streak
	if isCorrect {
		s.streak++
		if s.streak > s.maxStreak {
			s.maxStreak = s.streak
		}
		// Streak bonus
		if s.streak >= 3 {
			score += s.streak * 10
		}
	} else {
		s.streak = 0
	}

	answer := Answer{
		QuestionID: question.ID(),
		SpeciesID:  speciesID,
		TimeTaken:  timeTaken,
		IsCorrect:  isCorrect,
		Score:      score,
		AnsweredAt: time.Now(),
	}

	s.answers = append(s.answers, answer)
	s.totalScore += score
	s.currentIndex++

	// Check if session is completed
	if s.currentIndex >= len(s.questions) {
		s.Complete()
	}

	return &answer, nil
}

// Complete marks the session as completed.
func (s *Session) Complete() {
	s.status = SessionCompleted
	now := time.Now()
	s.completedAt = &now
}

// Abandon marks the session as abandoned.
func (s *Session) Abandon() {
	s.status = SessionAbandoned
	now := time.Now()
	s.completedAt = &now
}

// Accuracy returns the accuracy percentage.
func (s *Session) Accuracy() float64 {
	if len(s.answers) == 0 {
		return 0
	}
	return float64(s.CorrectCount()) / float64(len(s.answers)) * 100
}

// Answers returns all recorded answers.
func (s *Session) Answers() []Answer {
	return s.answers
}

// Duration returns the session duration.
func (s *Session) Duration() time.Duration {
	if s.startedAt.IsZero() {
		return 0
	}
	if s.completedAt != nil {
		return s.completedAt.Sub(s.startedAt)
	}
	return time.Since(s.startedAt)
}

// TaxonFilter returns the taxon filter for the session.
func (s *Session) TaxonFilter() string {
	return s.taxonFilter
}

// StartedAt returns when the session started.
func (s *Session) StartedAt() time.Time {
	return s.startedAt
}

// CompletedAt returns when the session ended, or nil if still running.
func (s *Session) CompletedAt() *time.Time {
	return s.completedAt
}

// SessionSnapshot is a serializable representation of a Session, used for
// persistence and reconstruction.
type SessionSnapshot struct {
	ID          string             `json:"id"`
	UserID      string             `json:"user_id"`
	Difficulty  Difficulty         `json:"difficulty"`
	QuizTypes   []QuizType         `json:"quiz_types"`
	TaxonFilter string             `json:"taxon_filter,omitempty"`
	Questions   []QuestionSnapshot `json:"questions"`
	Answers     []Answer           `json:"answers"`
	TotalScore  int                `json:"total_score"`
	Streak      int                `json:"streak"`
	MaxStreak   int                `json:"max_streak"`
	Status      SessionStatus      `json:"status"`
	StartedAt   time.Time          `json:"started_at"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
}

// Snapshot returns a serializable representation of the session.
func (s *Session) Snapshot() SessionSnapshot {
	questions := make([]QuestionSnapshot, len(s.questions))
	for i, q := range s.questions {
		questions[i] = q.Snapshot()
	}
	return SessionSnapshot{
		ID:          s.id,
		UserID:      s.userID,
		Difficulty:  s.difficulty,
		QuizTypes:   s.quizTypes,
		TaxonFilter: s.taxonFilter,
		Questions:   questions,
		Answers:     s.answers,
		TotalScore:  s.totalScore,
		Streak:      s.streak,
		MaxStreak:   s.maxStreak,
		Status:      s.status,
		StartedAt:   s.startedAt,
		CompletedAt: s.completedAt,
	}
}

// RestoreSession reconstructs a Session from a snapshot. The current
// question index is derived from the number of recorded answers.
func RestoreSession(snap SessionSnapshot) (*Session, error) {
	if snap.ID == "" {
		return nil, errors.New("session id is required")
	}
	if snap.UserID == "" {
		return nil, errors.New("user ID is required")
	}
	if len(snap.Questions) == 0 {
		return nil, errors.New("at least one question is required")
	}

	questions := make([]*Question, len(snap.Questions))
	for i, qs := range snap.Questions {
		q, err := RestoreQuestion(qs)
		if err != nil {
			return nil, err
		}
		questions[i] = q
	}

	answers := snap.Answers
	if answers == nil {
		answers = make([]Answer, 0, len(questions))
	}

	return &Session{
		id:           snap.ID,
		userID:       snap.UserID,
		difficulty:   snap.Difficulty,
		quizTypes:    snap.QuizTypes,
		taxonFilter:  snap.TaxonFilter,
		questions:    questions,
		answers:      answers,
		currentIndex: len(answers),
		totalScore:   snap.TotalScore,
		streak:       snap.Streak,
		maxStreak:    snap.MaxStreak,
		status:       snap.Status,
		startedAt:    snap.StartedAt,
		completedAt:  snap.CompletedAt,
	}, nil
}
