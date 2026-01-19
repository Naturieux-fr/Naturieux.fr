// Package quiz contains domain entities for quiz functionality.
package quiz

import (
	"time"
)

// QuizType represents different types of quiz questions.
type QuizType string

const (
	ImageQuiz      QuizType = "image"      // Full image visible
	FlashQuiz      QuizType = "flash"      // Image visible briefly (1-3s)
	PartialQuiz    QuizType = "partial"    // Only part of image visible
	SilhouetteQuiz QuizType = "silhouette" // Silhouette only
	SoundQuiz      QuizType = "sound"      // Audio only
)

// Difficulty represents quiz difficulty levels.
type Difficulty string

const (
	Beginner     Difficulty = "beginner"
	Intermediate Difficulty = "intermediate"
	Expert       Difficulty = "expert"
	Master       Difficulty = "master"
)

// DifficultyConfig holds configuration for each difficulty level.
type DifficultyConfig struct {
	Difficulty      Difficulty
	ChoicesCount    int
	TimeLimit       time.Duration
	ScoreMultiplier float64
	FlashDuration   time.Duration // For FlashQuiz
}

// DefaultDifficultyConfigs returns the default configurations.
func DefaultDifficultyConfigs() map[Difficulty]DifficultyConfig {
	return map[Difficulty]DifficultyConfig{
		Beginner: {
			Difficulty:      Beginner,
			ChoicesCount:    4,
			TimeLimit:       30 * time.Second,
			ScoreMultiplier: 1.0,
			FlashDuration:   5 * time.Second,
		},
		Intermediate: {
			Difficulty:      Intermediate,
			ChoicesCount:    6,
			TimeLimit:       20 * time.Second,
			ScoreMultiplier: 1.5,
			FlashDuration:   3 * time.Second,
		},
		Expert: {
			Difficulty:      Expert,
			ChoicesCount:    8,
			TimeLimit:       15 * time.Second,
			ScoreMultiplier: 2.0,
			FlashDuration:   2 * time.Second,
		},
		Master: {
			Difficulty:      Master,
			ChoicesCount:    10,
			TimeLimit:       10 * time.Second,
			ScoreMultiplier: 3.0,
			FlashDuration:   1 * time.Second,
		},
	}
}

// IsValidQuizType checks if a quiz type is valid.
func IsValidQuizType(qt QuizType) bool {
	switch qt {
	case ImageQuiz, FlashQuiz, PartialQuiz, SilhouetteQuiz, SoundQuiz:
		return true
	}
	return false
}

// IsValidDifficulty checks if a difficulty is valid.
func IsValidDifficulty(d Difficulty) bool {
	switch d {
	case Beginner, Intermediate, Expert, Master:
		return true
	}
	return false
}
