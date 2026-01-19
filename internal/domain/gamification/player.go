// Package gamification contains domain entities for game progression.
package gamification

import (
	"errors"
	"math"
	"time"
)

// Player represents a game player with progression.
type Player struct {
	id             string
	username       string
	totalXP        int
	level          int
	totalGames     int
	totalCorrect   int
	totalQuestions int
	bestStreak     int
	achievements   []Achievement
	dailyStreak    int
	lastPlayedAt   *time.Time
	createdAt      time.Time
}

// NewPlayer creates a new player.
func NewPlayer(id, username string) (*Player, error) {
	if id == "" {
		return nil, errors.New("player id is required")
	}
	if username == "" {
		return nil, errors.New("username is required")
	}

	return &Player{
		id:           id,
		username:     username,
		level:        1,
		achievements: make([]Achievement, 0),
		createdAt:    time.Now(),
	}, nil
}

// ID returns the player ID.
func (p *Player) ID() string {
	return p.id
}

// Username returns the username.
func (p *Player) Username() string {
	return p.username
}

// TotalXP returns total experience points.
func (p *Player) TotalXP() int {
	return p.totalXP
}

// Level returns the current level.
func (p *Player) Level() int {
	return p.level
}

// TotalGames returns the total games played.
func (p *Player) TotalGames() int {
	return p.totalGames
}

// Accuracy returns the overall accuracy percentage.
func (p *Player) Accuracy() float64 {
	if p.totalQuestions == 0 {
		return 0
	}
	return float64(p.totalCorrect) / float64(p.totalQuestions) * 100
}

// BestStreak returns the best streak achieved.
func (p *Player) BestStreak() int {
	return p.bestStreak
}

// DailyStreak returns consecutive days played.
func (p *Player) DailyStreak() int {
	return p.dailyStreak
}

// Achievements returns all unlocked achievements.
func (p *Player) Achievements() []Achievement {
	return p.achievements
}

// XPForLevel calculates XP required for a given level.
func XPForLevel(level int) int {
	// Exponential growth: 100 * 1.5^(level-1)
	return int(100 * math.Pow(1.5, float64(level-1)))
}

// XPToNextLevel returns XP needed for next level.
func (p *Player) XPToNextLevel() int {
	currentLevelXP := 0
	for i := 1; i < p.level; i++ {
		currentLevelXP += XPForLevel(i)
	}
	return XPForLevel(p.level) - (p.totalXP - currentLevelXP)
}

// XPProgress returns progress percentage to next level.
func (p *Player) XPProgress() float64 {
	currentLevelXP := 0
	for i := 1; i < p.level; i++ {
		currentLevelXP += XPForLevel(i)
	}
	progressXP := p.totalXP - currentLevelXP
	required := XPForLevel(p.level)
	return float64(progressXP) / float64(required) * 100
}

// AddXP adds experience points and handles level ups.
func (p *Player) AddXP(xp int) []LevelUpEvent {
	if xp <= 0 {
		return nil
	}

	p.totalXP += xp
	events := make([]LevelUpEvent, 0)

	// Check for level ups
	for {
		required := 0
		for i := 1; i <= p.level; i++ {
			required += XPForLevel(i)
		}
		if p.totalXP >= required {
			p.level++
			events = append(events, LevelUpEvent{
				NewLevel:   p.level,
				TotalXP:    p.totalXP,
				OccurredAt: time.Now(),
			})
		} else {
			break
		}
		// Cap at level 100
		if p.level >= 100 {
			break
		}
	}

	return events
}

// RecordGame records a completed game session.
func (p *Player) RecordGame(correct, total, maxStreak int) []Achievement {
	p.totalGames++
	p.totalCorrect += correct
	p.totalQuestions += total

	if maxStreak > p.bestStreak {
		p.bestStreak = maxStreak
	}

	// Update daily streak
	now := time.Now()
	if p.lastPlayedAt != nil {
		daysSince := int(now.Sub(*p.lastPlayedAt).Hours() / 24)
		if daysSince == 1 {
			p.dailyStreak++
		} else if daysSince > 1 {
			p.dailyStreak = 1
		}
	} else {
		p.dailyStreak = 1
	}
	p.lastPlayedAt = &now

	// Check for new achievements
	return p.checkAchievements()
}

func (p *Player) checkAchievements() []Achievement {
	newAchievements := make([]Achievement, 0)

	achievementChecks := []struct {
		achievement Achievement
		condition   func() bool
	}{
		{FirstGame, func() bool { return p.totalGames >= 1 }},
		{Veteran, func() bool { return p.totalGames >= 100 }},
		{StreakMaster, func() bool { return p.bestStreak >= 10 }},
		{PerfectScore, func() bool { return p.Accuracy() == 100 && p.totalQuestions >= 10 }},
		{Dedicated, func() bool { return p.dailyStreak >= 7 }},
		{LevelTen, func() bool { return p.level >= 10 }},
		{LevelFifty, func() bool { return p.level >= 50 }},
	}

	for _, check := range achievementChecks {
		if check.condition() && !p.hasAchievement(check.achievement) {
			p.achievements = append(p.achievements, check.achievement)
			newAchievements = append(newAchievements, check.achievement)
		}
	}

	return newAchievements
}

func (p *Player) hasAchievement(a Achievement) bool {
	for _, existing := range p.achievements {
		if existing == a {
			return true
		}
	}
	return false
}

// LevelUpEvent represents a level up occurrence.
type LevelUpEvent struct {
	NewLevel   int
	TotalXP    int
	OccurredAt time.Time
}
