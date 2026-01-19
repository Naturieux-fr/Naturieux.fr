package gamification_test

import (
	"testing"

	"github.com/fieve/naturieux/internal/domain/gamification"
)

func TestNewPlayer(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		username string
		wantErr  bool
	}{
		{
			name:     "valid player",
			id:       "p1",
			username: "naturelover",
			wantErr:  false,
		},
		{
			name:     "missing id",
			id:       "",
			username: "test",
			wantErr:  true,
		},
		{
			name:     "missing username",
			id:       "p1",
			username: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := gamification.NewPlayer(tt.id, tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPlayer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if p.Level() != 1 {
					t.Errorf("Initial level = %d, want 1", p.Level())
				}
				if p.TotalXP() != 0 {
					t.Errorf("Initial XP = %d, want 0", p.TotalXP())
				}
			}
		})
	}
}

func TestPlayer_AddXP(t *testing.T) {
	p, _ := gamification.NewPlayer("p1", "test")

	// Level 1 requires 100 XP
	events := p.AddXP(50)
	if len(events) != 0 {
		t.Error("Should not level up with 50 XP")
	}
	if p.Level() != 1 {
		t.Errorf("Level = %d, want 1", p.Level())
	}

	// Add more XP to reach level 2
	events = p.AddXP(60) // Total 110 XP, level 1 needs 100
	if len(events) != 1 {
		t.Errorf("Expected 1 level up event, got %d", len(events))
	}
	if p.Level() != 2 {
		t.Errorf("Level = %d, want 2", p.Level())
	}

	// Verify total XP
	if p.TotalXP() != 110 {
		t.Errorf("TotalXP = %d, want 110", p.TotalXP())
	}
}

func TestPlayer_AddXP_MultipleLevelUps(t *testing.T) {
	p, _ := gamification.NewPlayer("p1", "test")

	// Add enough XP for multiple levels
	// Level 1: 100, Level 2: 150, Level 3: 225 (total 475)
	events := p.AddXP(500)

	if len(events) < 3 {
		t.Errorf("Expected at least 3 level ups, got %d", len(events))
	}

	if p.Level() < 4 {
		t.Errorf("Level = %d, want at least 4", p.Level())
	}
}

func TestPlayer_RecordGame(t *testing.T) {
	p, _ := gamification.NewPlayer("p1", "test")

	achievements := p.RecordGame(8, 10, 5)

	if p.TotalGames() != 1 {
		t.Errorf("TotalGames = %d, want 1", p.TotalGames())
	}

	// Should unlock FirstGame achievement
	hasFirstGame := false
	for _, a := range achievements {
		if a == gamification.FirstGame {
			hasFirstGame = true
			break
		}
	}
	if !hasFirstGame {
		t.Error("Should unlock FirstGame achievement")
	}

	// Check accuracy
	if p.Accuracy() != 80 {
		t.Errorf("Accuracy = %f, want 80", p.Accuracy())
	}
}

func TestPlayer_BestStreak(t *testing.T) {
	p, _ := gamification.NewPlayer("p1", "test")

	p.RecordGame(5, 10, 5)
	if p.BestStreak() != 5 {
		t.Errorf("BestStreak = %d, want 5", p.BestStreak())
	}

	p.RecordGame(3, 10, 3) // Lower streak
	if p.BestStreak() != 5 {
		t.Errorf("BestStreak = %d, want 5 (should keep max)", p.BestStreak())
	}

	p.RecordGame(8, 10, 8) // Higher streak
	if p.BestStreak() != 8 {
		t.Errorf("BestStreak = %d, want 8", p.BestStreak())
	}
}

func TestPlayer_StreakMasterAchievement(t *testing.T) {
	p, _ := gamification.NewPlayer("p1", "test")

	// Record game with 10+ streak
	achievements := p.RecordGame(10, 10, 10)

	hasStreakMaster := false
	for _, a := range achievements {
		if a == gamification.StreakMaster {
			hasStreakMaster = true
			break
		}
	}
	if !hasStreakMaster {
		t.Error("Should unlock StreakMaster achievement with 10 streak")
	}
}

func TestXPForLevel(t *testing.T) {
	tests := []struct {
		level    int
		expected int
	}{
		{1, 100},
		{2, 150}, // 100 * 1.5
		{3, 225}, // 100 * 1.5^2
		{4, 337}, // 100 * 1.5^3
	}

	for _, tt := range tests {
		xp := gamification.XPForLevel(tt.level)
		// Allow small rounding differences
		if xp < tt.expected-1 || xp > tt.expected+1 {
			t.Errorf("XPForLevel(%d) = %d, want ~%d", tt.level, xp, tt.expected)
		}
	}
}

func TestPlayer_XPProgress(t *testing.T) {
	p, _ := gamification.NewPlayer("p1", "test")

	// At 0 XP, progress should be 0%
	if p.XPProgress() != 0 {
		t.Errorf("XPProgress = %f, want 0", p.XPProgress())
	}

	// Add 50 XP (50% of 100 needed for level 2)
	p.AddXP(50)
	progress := p.XPProgress()
	if progress < 49 || progress > 51 {
		t.Errorf("XPProgress = %f, want ~50", progress)
	}
}

func TestGetAchievementInfo(t *testing.T) {
	info := gamification.GetAchievementInfo(gamification.FirstGame)

	if info.Name == "" {
		t.Error("Achievement name should not be empty")
	}
	if info.Description == "" {
		t.Error("Achievement description should not be empty")
	}
	if info.XPReward <= 0 {
		t.Error("Achievement XP reward should be positive")
	}
}
