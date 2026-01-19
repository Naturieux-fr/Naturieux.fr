package quiz_test

import (
	"context"
	"testing"
	"time"

	appquiz "github.com/fieve/naturieux/internal/application/quiz"
	"github.com/fieve/naturieux/internal/domain/gamification"
	"github.com/fieve/naturieux/internal/domain/quiz"
	"github.com/fieve/naturieux/internal/domain/species"
)

// mockPlayerRepository is a test double for PlayerRepository.
type mockPlayerRepository struct {
	players map[string]*gamification.Player
}

func newMockPlayerRepository() *mockPlayerRepository {
	return &mockPlayerRepository{
		players: make(map[string]*gamification.Player),
	}
}

func (m *mockPlayerRepository) Create(ctx context.Context, player *gamification.Player) error {
	m.players[player.ID()] = player
	return nil
}

func (m *mockPlayerRepository) GetByID(ctx context.Context, id string) (*gamification.Player, error) {
	if p, ok := m.players[id]; ok {
		return p, nil
	}
	return nil, context.DeadlineExceeded // Simulate not found
}

func (m *mockPlayerRepository) GetByUsername(ctx context.Context, username string) (*gamification.Player, error) {
	for _, p := range m.players {
		if p.Username() == username {
			return p, nil
		}
	}
	return nil, context.DeadlineExceeded
}

func (m *mockPlayerRepository) Update(ctx context.Context, player *gamification.Player) error {
	m.players[player.ID()] = player
	return nil
}

func (m *mockPlayerRepository) GetLeaderboard(ctx context.Context, limit int) ([]*gamification.Player, error) {
	result := make([]*gamification.Player, 0, limit)
	for _, p := range m.players {
		result = append(result, p)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

// mockQuestionFactory for testing
type mockQuestionFactory struct {
	questions []*quiz.Question
	index     int
}

func newMockQuestionFactory() *mockQuestionFactory {
	return &mockQuestionFactory{
		questions: make([]*quiz.Question, 0),
	}
}

func (m *mockQuestionFactory) CreateQuestion(ctx context.Context, quizType quiz.QuizType, difficulty quiz.Difficulty) (*quiz.Question, error) {
	if m.index >= len(m.questions) {
		// Create a default question
		sp, _ := species.New(m.index+1, "Test Species", "Test Common", "Mammalia")
		sp.AddPhoto(species.Photo{ID: 1, URL: "https://example.com/photo.jpg", MediumURL: "https://example.com/photo_medium.jpg"})

		wrong, _ := species.New(m.index+100, "Wrong Species", "Wrong", "Mammalia")

		choices := []quiz.Choice{
			{Species: sp, IsCorrect: true},
			{Species: wrong, IsCorrect: false},
		}

		q, _ := quiz.NewQuestion("q-default", quizType, difficulty, sp, choices, "https://example.com/img.jpg")
		m.index++
		return q, nil
	}
	q := m.questions[m.index]
	m.index++
	return q, nil
}

// mockEventPublisher for testing
type mockEventPublisher struct {
	sessionCompletedCount    int
	levelUpCount             int
	achievementUnlockedCount int
}

func (m *mockEventPublisher) PublishSessionCompleted(session *quiz.Session, player *gamification.Player) {
	m.sessionCompletedCount++
}

func (m *mockEventPublisher) PublishLevelUp(player *gamification.Player, event gamification.LevelUpEvent) {
	m.levelUpCount++
}

func (m *mockEventPublisher) PublishAchievementUnlocked(player *gamification.Player, achievement gamification.Achievement) {
	m.achievementUnlockedCount++
}

func TestService_StartSession(t *testing.T) {
	playerRepo := newMockPlayerRepository()
	player, _ := gamification.NewPlayer("user1", "testuser")
	playerRepo.Create(context.Background(), player)

	factory := newMockQuestionFactory()
	eventPub := &mockEventPublisher{}

	service := appquiz.NewService(factory, nil, playerRepo, eventPub)

	req := appquiz.StartSessionRequest{
		UserID:        "user1",
		Difficulty:    quiz.Beginner,
		QuizTypes:     []quiz.QuizType{quiz.ImageQuiz},
		QuestionCount: 5,
	}

	resp, err := service.StartSession(context.Background(), req)
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}

	if resp.SessionID == "" {
		t.Error("SessionID should not be empty")
	}

	if resp.FirstQuestion == nil {
		t.Error("FirstQuestion should not be nil")
	}

	if resp.TotalQuestions != 5 {
		t.Errorf("TotalQuestions = %d, want 5", resp.TotalQuestions)
	}
}

func TestService_StartSession_UserNotFound(t *testing.T) {
	playerRepo := newMockPlayerRepository()
	factory := newMockQuestionFactory()
	eventPub := &mockEventPublisher{}

	service := appquiz.NewService(factory, nil, playerRepo, eventPub)

	req := appquiz.StartSessionRequest{
		UserID:        "nonexistent",
		Difficulty:    quiz.Beginner,
		QuestionCount: 5,
	}

	_, err := service.StartSession(context.Background(), req)
	if err == nil {
		t.Error("StartSession() should return error for nonexistent user")
	}
}

func TestService_StartSession_DefaultValues(t *testing.T) {
	playerRepo := newMockPlayerRepository()
	player, _ := gamification.NewPlayer("user1", "testuser")
	playerRepo.Create(context.Background(), player)

	factory := newMockQuestionFactory()
	eventPub := &mockEventPublisher{}

	service := appquiz.NewService(factory, nil, playerRepo, eventPub)

	req := appquiz.StartSessionRequest{
		UserID: "user1",
		// No other fields set - should use defaults
	}

	resp, err := service.StartSession(context.Background(), req)
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}

	// Default question count should be 10
	if resp.TotalQuestions != 10 {
		t.Errorf("TotalQuestions = %d, want 10 (default)", resp.TotalQuestions)
	}
}

func TestService_SubmitAnswer(t *testing.T) {
	playerRepo := newMockPlayerRepository()
	player, _ := gamification.NewPlayer("user1", "testuser")
	playerRepo.Create(context.Background(), player)

	factory := newMockQuestionFactory()
	eventPub := &mockEventPublisher{}

	service := appquiz.NewService(factory, nil, playerRepo, eventPub)

	// Start a session
	req := appquiz.StartSessionRequest{
		UserID:        "user1",
		Difficulty:    quiz.Beginner,
		QuestionCount: 2,
	}

	startResp, err := service.StartSession(context.Background(), req)
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}

	// Get the session from the builder (we need to recreate it since we don't have session repo)
	session, _ := quiz.NewSessionBuilder().
		WithUserID("user1").
		WithDifficulty(quiz.Beginner).
		WithQuestions([]*quiz.Question{startResp.FirstQuestion}).
		Build()
	session.Start()

	// Submit correct answer
	correctID := startResp.FirstQuestion.CorrectSpecies().ID()
	submitReq := appquiz.SubmitAnswerRequest{
		SessionID: startResp.SessionID,
		SpeciesID: correctID,
		TimeTaken: 5 * time.Second,
	}

	submitResp, err := service.SubmitAnswer(context.Background(), session, submitReq)
	if err != nil {
		t.Fatalf("SubmitAnswer() error = %v", err)
	}

	if !submitResp.IsCorrect {
		t.Error("Answer should be correct")
	}

	if submitResp.Score <= 0 {
		t.Error("Score should be positive for correct answer")
	}

	if submitResp.CorrectName == "" {
		t.Error("CorrectName should not be empty")
	}
}

func TestService_SubmitAnswer_Wrong(t *testing.T) {
	playerRepo := newMockPlayerRepository()
	player, _ := gamification.NewPlayer("user1", "testuser")
	playerRepo.Create(context.Background(), player)

	factory := newMockQuestionFactory()
	service := appquiz.NewService(factory, nil, playerRepo, nil)

	// Start session
	req := appquiz.StartSessionRequest{
		UserID:        "user1",
		QuestionCount: 1,
	}
	startResp, _ := service.StartSession(context.Background(), req)

	session, _ := quiz.NewSessionBuilder().
		WithUserID("user1").
		WithQuestions([]*quiz.Question{startResp.FirstQuestion}).
		Build()
	session.Start()

	// Submit wrong answer
	submitReq := appquiz.SubmitAnswerRequest{
		SpeciesID: 99999, // Wrong ID
		TimeTaken: 5 * time.Second,
	}

	submitResp, err := service.SubmitAnswer(context.Background(), session, submitReq)
	if err != nil {
		t.Fatalf("SubmitAnswer() error = %v", err)
	}

	if submitResp.IsCorrect {
		t.Error("Answer should be wrong")
	}

	if submitResp.Score != 0 {
		t.Errorf("Score = %d, want 0 for wrong answer", submitResp.Score)
	}
}

func TestService_SubmitAnswer_SessionComplete(t *testing.T) {
	playerRepo := newMockPlayerRepository()
	player, _ := gamification.NewPlayer("user1", "testuser")
	playerRepo.Create(context.Background(), player)

	factory := newMockQuestionFactory()
	eventPub := &mockEventPublisher{}
	service := appquiz.NewService(factory, nil, playerRepo, eventPub)

	// Start session with 1 question
	req := appquiz.StartSessionRequest{
		UserID:        "user1",
		QuestionCount: 1,
	}
	startResp, _ := service.StartSession(context.Background(), req)

	session, _ := quiz.NewSessionBuilder().
		WithUserID("user1").
		WithQuestions([]*quiz.Question{startResp.FirstQuestion}).
		Build()
	session.Start()

	// Submit answer
	correctID := startResp.FirstQuestion.CorrectSpecies().ID()
	submitReq := appquiz.SubmitAnswerRequest{
		SpeciesID: correctID,
		TimeTaken: 5 * time.Second,
	}

	submitResp, err := service.SubmitAnswer(context.Background(), session, submitReq)
	if err != nil {
		t.Fatalf("SubmitAnswer() error = %v", err)
	}

	if !submitResp.SessionComplete {
		t.Error("Session should be complete after last question")
	}

	if submitResp.NextQuestion != nil {
		t.Error("NextQuestion should be nil when session complete")
	}
}

func TestService_AbandonSession(t *testing.T) {
	factory := newMockQuestionFactory()
	service := appquiz.NewService(factory, nil, nil, nil)

	sp, _ := species.New(1, "Test", "Test", "Mammalia")
	sp.AddPhoto(species.Photo{ID: 1, URL: "https://example.com/photo.jpg"})

	choices := []quiz.Choice{
		{Species: sp, IsCorrect: true},
	}
	// Need at least 2 choices
	wrong, _ := species.New(2, "Wrong", "Wrong", "Mammalia")
	choices = append(choices, quiz.Choice{Species: wrong, IsCorrect: false})

	q, _ := quiz.NewQuestion("q1", quiz.ImageQuiz, quiz.Beginner, sp, choices, "https://example.com/img.jpg")

	session, _ := quiz.NewSessionBuilder().
		WithUserID("user1").
		WithQuestions([]*quiz.Question{q}).
		Build()
	session.Start()

	err := service.AbandonSession(context.Background(), session)
	if err != nil {
		t.Errorf("AbandonSession() error = %v", err)
	}

	if session.Status() != quiz.SessionAbandoned {
		t.Errorf("Status = %v, want abandoned", session.Status())
	}
}

func TestService_AbandonSession_NilSession(t *testing.T) {
	service := appquiz.NewService(nil, nil, nil, nil)

	err := service.AbandonSession(context.Background(), nil)
	if err == nil {
		t.Error("AbandonSession() should return error for nil session")
	}
}
