package quiz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fieve/naturieux/internal/domain/gamification"
	"github.com/fieve/naturieux/internal/domain/quiz"
	"github.com/fieve/naturieux/internal/ports"
)

// Service handles quiz game logic and orchestration.
type Service struct {
	questionFactory QuestionFactory
	sessionRepo     ports.QuizSessionRepository
	playerRepo      ports.PlayerRepository
	eventPublisher  GameEventPublisher
}

// GameEventPublisher publishes game events for gamification.
type GameEventPublisher interface {
	PublishSessionCompleted(session *quiz.Session, player *gamification.Player)
	PublishLevelUp(player *gamification.Player, event gamification.LevelUpEvent)
	PublishAchievementUnlocked(player *gamification.Player, achievement gamification.Achievement)
}

// NewService creates a new quiz service.
func NewService(
	factory QuestionFactory,
	sessionRepo ports.QuizSessionRepository,
	playerRepo ports.PlayerRepository,
	eventPublisher GameEventPublisher,
) *Service {
	return &Service{
		questionFactory: factory,
		sessionRepo:     sessionRepo,
		playerRepo:      playerRepo,
		eventPublisher:  eventPublisher,
	}
}

// StartSessionRequest contains parameters for starting a new quiz session.
type StartSessionRequest struct {
	UserID        string
	Difficulty    quiz.Difficulty
	QuizTypes     []quiz.QuizType
	TaxonFilter   string
	QuestionCount int
}

// StartSessionResponse contains the result of starting a session.
type StartSessionResponse struct {
	SessionID      string
	FirstQuestion  *quiz.Question
	TotalQuestions int
}

// StartSession creates and starts a new quiz session.
func (s *Service) StartSession(ctx context.Context, req StartSessionRequest) (*StartSessionResponse, error) {
	// Validate request
	if req.UserID == "" {
		return nil, errors.New("user ID is required")
	}
	if req.QuestionCount <= 0 {
		req.QuestionCount = 10 // Default
	}
	if len(req.QuizTypes) == 0 {
		req.QuizTypes = []quiz.QuizType{quiz.ImageQuiz}
	}
	if !quiz.IsValidDifficulty(req.Difficulty) {
		req.Difficulty = quiz.Beginner
	}

	// Verify player exists
	_, err := s.playerRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("player not found: %w", err)
	}

	// Generate questions
	questions := make([]*quiz.Question, 0, req.QuestionCount)
	for i := 0; i < req.QuestionCount; i++ {
		// Rotate through quiz types
		quizType := req.QuizTypes[i%len(req.QuizTypes)]

		question, qErr := s.questionFactory.CreateQuestion(ctx, quizType, req.Difficulty)
		if qErr != nil {
			// Log and continue trying
			continue
		}
		questions = append(questions, question)
	}

	if len(questions) == 0 {
		return nil, errors.New("failed to generate any questions")
	}

	// Create session
	session, err := quiz.NewSessionBuilder().
		WithUserID(req.UserID).
		WithDifficulty(req.Difficulty).
		WithQuizTypes(req.QuizTypes...).
		WithTaxonFilter(req.TaxonFilter).
		WithQuestions(questions).
		Build()
	if err != nil {
		return nil, fmt.Errorf("building session: %w", err)
	}

	// Start the session
	if err := session.Start(); err != nil {
		return nil, fmt.Errorf("starting session: %w", err)
	}

	// Persist session
	if s.sessionRepo != nil {
		if err := s.sessionRepo.Save(ctx, session); err != nil {
			return nil, fmt.Errorf("saving session: %w", err)
		}
	}

	return &StartSessionResponse{
		SessionID:      session.ID(),
		FirstQuestion:  session.CurrentQuestion(),
		TotalQuestions: len(questions),
	}, nil
}

// SubmitAnswerRequest contains parameters for submitting an answer.
type SubmitAnswerRequest struct {
	SessionID string
	SpeciesID int
	TimeTaken time.Duration
}

// SubmitAnswerResponse contains the result of submitting an answer.
type SubmitAnswerResponse struct {
	IsCorrect        bool
	Score            int
	CorrectSpeciesID int
	CorrectName      string
	CurrentStreak    int
	NextQuestion     *quiz.Question
	SessionComplete  bool
	TotalScore       int
	Accuracy         float64
}

// SubmitAnswer processes an answer submission.
func (s *Service) SubmitAnswer(
	ctx context.Context,
	session *quiz.Session,
	req SubmitAnswerRequest,
) (*SubmitAnswerResponse, error) {
	if session == nil {
		return nil, errors.New("session is required")
	}

	currentQuestion := session.CurrentQuestion()
	if currentQuestion == nil {
		return nil, errors.New("no current question")
	}

	// Submit the answer
	answer, err := session.SubmitAnswer(req.SpeciesID, req.TimeTaken)
	if err != nil {
		return nil, fmt.Errorf("submitting answer: %w", err)
	}

	// Persist updated session
	if s.sessionRepo != nil {
		if err := s.sessionRepo.Save(ctx, session); err != nil {
			return nil, fmt.Errorf("saving session: %w", err)
		}
	}

	response := &SubmitAnswerResponse{
		IsCorrect:        answer.IsCorrect,
		Score:            answer.Score,
		CorrectSpeciesID: currentQuestion.CorrectSpecies().ID(),
		CorrectName:      currentQuestion.CorrectSpecies().DisplayName(),
		CurrentStreak:    session.CurrentStreak(),
		NextQuestion:     session.CurrentQuestion(),
		SessionComplete:  session.Status() == quiz.SessionCompleted,
		TotalScore:       session.TotalScore(),
		Accuracy:         session.Accuracy(),
	}

	// Handle session completion
	if response.SessionComplete {
		if err := s.handleSessionComplete(ctx, session); err != nil {
			// Log error but don't fail the response
			fmt.Printf("error handling session complete: %v\n", err)
		}
	}

	return response, nil
}

// handleSessionComplete processes gamification when a session completes.
func (s *Service) handleSessionComplete(ctx context.Context, session *quiz.Session) error {
	player, err := s.playerRepo.GetByID(ctx, session.UserID())
	if err != nil {
		return fmt.Errorf("getting player: %w", err)
	}

	// Calculate XP earned
	xp := session.TotalScore()

	// Bonus XP for accuracy
	if session.Accuracy() >= 90 {
		xp += 100
	} else if session.Accuracy() >= 80 {
		xp += 50
	}

	// Bonus XP for streak
	xp += session.MaxStreak() * 10

	// Add XP and check for level ups
	levelUps := player.AddXP(xp)
	for _, event := range levelUps {
		if s.eventPublisher != nil {
			s.eventPublisher.PublishLevelUp(player, event)
		}
	}

	// Record game and check for achievements
	achievements := player.RecordGame(
		session.CorrectCount(),
		session.QuestionsCount(),
		session.MaxStreak(),
	)
	for _, achievement := range achievements {
		if s.eventPublisher != nil {
			s.eventPublisher.PublishAchievementUnlocked(player, achievement)
		}
		// Award achievement XP
		info := gamification.GetAchievementInfo(achievement)
		player.AddXP(info.XPReward)
	}

	// Persist player updates
	if err := s.playerRepo.Update(ctx, player); err != nil {
		return fmt.Errorf("updating player: %w", err)
	}

	// Publish session completed event
	if s.eventPublisher != nil {
		s.eventPublisher.PublishSessionCompleted(session, player)
	}

	return nil
}

// GetSessionStats returns statistics for a user's sessions.
func (s *Service) GetSessionStats(ctx context.Context, userID string) (*ports.UserQuizStats, error) {
	if s.sessionRepo == nil {
		return nil, errors.New("session repository not configured")
	}
	return s.sessionRepo.GetStats(ctx, userID)
}

// AbandonSession marks a session as abandoned.
func (s *Service) AbandonSession(ctx context.Context, session *quiz.Session) error {
	if session == nil {
		return errors.New("session is required")
	}

	session.Abandon()

	if s.sessionRepo != nil {
		if err := s.sessionRepo.Save(ctx, session); err != nil {
			return fmt.Errorf("saving abandoned session: %w", err)
		}
	}

	return nil
}
