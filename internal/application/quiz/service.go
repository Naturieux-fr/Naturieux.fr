package quiz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// Default values for session configuration.
const (
	defaultQuestionCount = 10
	accuracyBonusHigh    = 100
	accuracyBonusMedium  = 50
	accuracyThresholdHigh   = 90
	accuracyThresholdMedium = 80
	streakXPMultiplier   = 10
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

// normalizeRequest applies default values to the request.
func (req *StartSessionRequest) normalize() {
	if req.QuestionCount <= 0 {
		req.QuestionCount = defaultQuestionCount
	}
	if len(req.QuizTypes) == 0 {
		req.QuizTypes = []quiz.QuizType{quiz.ImageQuiz}
	}
	if !quiz.IsValidDifficulty(req.Difficulty) {
		req.Difficulty = quiz.Beginner
	}
}

// StartSession creates and starts a new quiz session.
func (s *Service) StartSession(ctx context.Context, req StartSessionRequest) (*StartSessionResponse, error) {
	if req.UserID == "" {
		return nil, errors.New("user ID is required")
	}
	req.normalize()

	if _, err := s.playerRepo.GetByID(ctx, req.UserID); err != nil {
		return nil, fmt.Errorf("player not found: %w", err)
	}

	questions, err := s.generateQuestions(ctx, req)
	if err != nil {
		return nil, err
	}

	session, err := s.buildAndStartSession(req, questions)
	if err != nil {
		return nil, err
	}

	if err := s.saveSession(ctx, session); err != nil {
		return nil, err
	}

	return &StartSessionResponse{
		SessionID:      session.ID(),
		FirstQuestion:  session.CurrentQuestion(),
		TotalQuestions: len(questions),
	}, nil
}

// generateQuestions creates questions for the session.
func (s *Service) generateQuestions(ctx context.Context, req StartSessionRequest) ([]*quiz.Question, error) {
	questions := make([]*quiz.Question, 0, req.QuestionCount)

	for i := 0; i < req.QuestionCount; i++ {
		quizType := req.QuizTypes[i%len(req.QuizTypes)]
		question, err := s.questionFactory.CreateQuestion(ctx, quizType, req.Difficulty)
		if err != nil {
			continue
		}
		questions = append(questions, question)
	}

	if len(questions) == 0 {
		return nil, errors.New("failed to generate any questions")
	}
	return questions, nil
}

// buildAndStartSession creates and starts a new session.
func (s *Service) buildAndStartSession(req StartSessionRequest, questions []*quiz.Question) (*quiz.Session, error) {
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

	if err := session.Start(); err != nil {
		return nil, fmt.Errorf("starting session: %w", err)
	}
	return session, nil
}

// saveSession persists the session if repository is configured.
func (s *Service) saveSession(ctx context.Context, session *quiz.Session) error {
	if s.sessionRepo == nil {
		return nil
	}
	if err := s.sessionRepo.Save(ctx, session); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}
	return nil
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

	answer, err := session.SubmitAnswer(req.SpeciesID, req.TimeTaken)
	if err != nil {
		return nil, fmt.Errorf("submitting answer: %w", err)
	}

	if err := s.saveSession(ctx, session); err != nil {
		return nil, err
	}

	response := s.buildAnswerResponse(session, currentQuestion, answer)

	if response.SessionComplete {
		if err := s.handleSessionComplete(ctx, session); err != nil {
			fmt.Printf("error handling session complete: %v\n", err)
		}
	}

	return response, nil
}

// buildAnswerResponse creates the response for an answer submission.
func (s *Service) buildAnswerResponse(
	session *quiz.Session,
	question *quiz.Question,
	answer *quiz.Answer,
) *SubmitAnswerResponse {
	return &SubmitAnswerResponse{
		IsCorrect:        answer.IsCorrect,
		Score:            answer.Score,
		CorrectSpeciesID: question.CorrectSpecies().ID(),
		CorrectName:      question.CorrectSpecies().DisplayName(),
		CurrentStreak:    session.CurrentStreak(),
		NextQuestion:     session.CurrentQuestion(),
		SessionComplete:  session.Status() == quiz.SessionCompleted,
		TotalScore:       session.TotalScore(),
		Accuracy:         session.Accuracy(),
	}
}

// handleSessionComplete processes gamification when a session completes.
func (s *Service) handleSessionComplete(ctx context.Context, session *quiz.Session) error {
	player, err := s.playerRepo.GetByID(ctx, session.UserID())
	if err != nil {
		return fmt.Errorf("getting player: %w", err)
	}

	xp := s.calculateSessionXP(session)
	s.processLevelUps(player, xp)
	s.processAchievements(ctx, player, session)

	if err := s.playerRepo.Update(ctx, player); err != nil {
		return fmt.Errorf("updating player: %w", err)
	}

	s.publishSessionCompleted(session, player)
	return nil
}

// calculateSessionXP calculates XP earned from a session.
func (s *Service) calculateSessionXP(session *quiz.Session) int {
	xp := session.TotalScore()

	accuracy := session.Accuracy()
	if accuracy >= accuracyThresholdHigh {
		xp += accuracyBonusHigh
	} else if accuracy >= accuracyThresholdMedium {
		xp += accuracyBonusMedium
	}

	xp += session.MaxStreak() * streakXPMultiplier
	return xp
}

// processLevelUps handles level up events.
func (s *Service) processLevelUps(player *gamification.Player, xp int) {
	levelUps := player.AddXP(xp)
	for _, event := range levelUps {
		if s.eventPublisher != nil {
			s.eventPublisher.PublishLevelUp(player, event)
		}
	}
}

// processAchievements handles achievement unlocks.
func (s *Service) processAchievements(_ context.Context, player *gamification.Player, session *quiz.Session) {
	achievements := player.RecordGame(
		session.CorrectCount(),
		session.QuestionsCount(),
		session.MaxStreak(),
	)

	for _, achievement := range achievements {
		if s.eventPublisher != nil {
			s.eventPublisher.PublishAchievementUnlocked(player, achievement)
		}
		info := gamification.GetAchievementInfo(achievement)
		player.AddXP(info.XPReward)
	}
}

// publishSessionCompleted publishes the session completed event.
func (s *Service) publishSessionCompleted(session *quiz.Session, player *gamification.Player) {
	if s.eventPublisher != nil {
		s.eventPublisher.PublishSessionCompleted(session, player)
	}
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
	return s.saveSession(ctx, session)
}
