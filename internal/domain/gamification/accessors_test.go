package gamification_test

import (
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification"
)

func TestPlayer_CategoryAndXP(t *testing.T) {
	p, err := gamification.NewPlayer("p1", "Bob")
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	p.AddCategoryCorrect("Aves", 5)
	if p.CategoryCorrect()["Aves"] != 5 {
		t.Errorf("category = %v", p.CategoryCorrect())
	}
	if p.XPToNextLevel() <= 0 {
		t.Errorf("XPToNextLevel = %d", p.XPToNextLevel())
	}
}
