package gamification_test

import (
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification"
)

func TestPlayer_SnapshotRoundTrip(t *testing.T) {
	player, err := gamification.NewPlayer("p1", "alice")
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	player.AddXP(250)
	player.RecordGame(9, 10, 6)

	restored, err := gamification.RestorePlayer(player.Snapshot())
	if err != nil {
		t.Fatalf("RestorePlayer() error = %v", err)
	}

	if restored.ID() != "p1" || restored.Username() != "alice" {
		t.Errorf("Identity = %s/%s, want p1/alice", restored.ID(), restored.Username())
	}
	if restored.TotalXP() != player.TotalXP() {
		t.Errorf("TotalXP = %d, want %d", restored.TotalXP(), player.TotalXP())
	}
	if restored.Level() != player.Level() {
		t.Errorf("Level = %d, want %d", restored.Level(), player.Level())
	}
	if restored.TotalGames() != 1 {
		t.Errorf("TotalGames = %d, want 1", restored.TotalGames())
	}
	if restored.BestStreak() != 6 {
		t.Errorf("BestStreak = %d, want 6", restored.BestStreak())
	}
	if restored.DailyStreak() != player.DailyStreak() {
		t.Errorf("DailyStreak = %d, want %d", restored.DailyStreak(), player.DailyStreak())
	}
	if len(restored.Achievements()) != len(player.Achievements()) {
		t.Errorf("Achievements = %d, want %d", len(restored.Achievements()), len(player.Achievements()))
	}
	if restored.Accuracy() != player.Accuracy() {
		t.Errorf("Accuracy = %f, want %f", restored.Accuracy(), player.Accuracy())
	}
}

func TestRestorePlayer_Invalid(t *testing.T) {
	if _, err := gamification.RestorePlayer(gamification.PlayerSnapshot{}); err == nil {
		t.Error("RestorePlayer() with empty snapshot should return an error")
	}
}
