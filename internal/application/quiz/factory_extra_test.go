package quiz_test

import (
	"context"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/mock"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
)

func TestFactory_Variants(t *testing.T) {
	f := appquiz.NewQuestionFactory(mock.NewSpeciesRepository())
	ctx := context.Background()

	// Standard image question.
	if q, err := f.CreateQuestion(ctx, quiz.ImageQuiz, quiz.Beginner, ""); err != nil || q == nil {
		t.Errorf("CreateQuestion: %v", err)
	}
	// Flash, silhouette, partial all build.
	for _, qt := range []quiz.QuizType{quiz.FlashQuiz, quiz.SilhouetteQuiz, quiz.PartialQuiz} {
		if _, err := f.CreateQuestion(ctx, qt, quiz.Intermediate, ""); err != nil {
			t.Errorf("CreateQuestion %s: %v", qt, err)
		}
	}
	// CreateQuestionFor a specific species.
	pick, _ := f.CreateQuestion(ctx, quiz.ImageQuiz, quiz.Beginner, "")
	cd := pick.CorrectSpecies().ID()
	if _, err := f.CreateQuestionFor(ctx, quiz.ImageQuiz, quiz.Beginner, cd); err != nil {
		t.Errorf("CreateQuestionFor: %v", err)
	}
	// Sound quiz without an audio source -> error.
	if _, err := f.CreateQuestion(ctx, quiz.SoundQuiz, quiz.Beginner, ""); err == nil {
		t.Error("sound without source should error")
	}
	// Family quiz without a family source (mock) -> error.
	if _, err := f.CreateQuestion(ctx, quiz.FamilyQuiz, quiz.Beginner, ""); err == nil {
		t.Error("family quiz on mock should error")
	}
}

func TestFactory_FamilyAndSoundWithTaxref(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := taxref.EnsureSchema(db); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO taxref_species
		(cd_nom, cd_ref, rang, scientific_name, vernacular_name, kingdom, class, ordre, family, genus, taxa_group, fr) VALUES
		(1,1,'ES','Erithacus rubecula','Rougegorge','Animalia','Aves','Passeriformes','Muscicapidae','Erithacus','Oiseaux','P'),
		(2,2,'ES','Turdus merula','Merle','Animalia','Aves','Passeriformes','Turdidae','Turdus','Oiseaux','P'),
		(3,3,'ES','Parus major','Mesange','Animalia','Aves','Passeriformes','Paridae','Parus','Oiseaux','P'),
		(4,4,'ES','Fringilla coelebs','Pinson','Animalia','Aves','Passeriformes','Fringillidae','Fringilla','Oiseaux','P')`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	repo := taxref.NewRepository(db)
	ctx := context.Background()
	if _, err := repo.AddPhoto(ctx, 1, "https://x/p.jpg", "a", "cc-by", "beginner"); err != nil {
		t.Fatalf("add photo: %v", err)
	}
	if _, err := repo.AddSound(ctx, 1, "https://x/s.mp3", "rec", "cc-by"); err != nil {
		t.Fatalf("add sound: %v", err)
	}

	f := appquiz.NewQuestionFactory(repo, appquiz.WithAudioSource(repo))
	if q, err := f.CreateQuestion(ctx, quiz.FamilyQuiz, quiz.Beginner, ""); err != nil || q == nil {
		t.Errorf("family quiz: %v", err)
	} else if q.QuizType() != quiz.FamilyQuiz {
		t.Errorf("quiz type = %s", q.QuizType())
	}
	if q, err := f.CreateQuestion(ctx, quiz.SoundQuiz, quiz.Beginner, ""); err != nil || q == nil {
		t.Errorf("sound quiz: %v", err)
	}
}
