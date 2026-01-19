package gamification

// Achievement represents an unlockable achievement.
type Achievement string

const (
	// Game count achievements
	FirstGame Achievement = "first_game" // Complete first game
	Veteran   Achievement = "veteran"    // Complete 100 games
	Dedicated Achievement = "dedicated"  // Play 7 days in a row

	// Performance achievements
	PerfectScore Achievement = "perfect_score" // 100% accuracy (min 10 questions)
	StreakMaster Achievement = "streak_master" // 10 correct answers in a row

	// Level achievements
	LevelTen   Achievement = "level_ten"   // Reach level 10
	LevelFifty Achievement = "level_fifty" // Reach level 50

	// Category achievements
	MammalExpert Achievement = "mammal_expert" // 100 correct mammal identifications
	BirdWatcher  Achievement = "bird_watcher"  // 100 correct bird identifications
	BugHunter    Achievement = "bug_hunter"    // 100 correct insect identifications
	Botanist     Achievement = "botanist"      // 100 correct plant identifications

	// Difficulty achievements
	ExpertMode    Achievement = "expert_mode"    // Complete an expert quiz
	MasterNatural Achievement = "master_natural" // Complete a master quiz with 80%+
)

// AchievementInfo contains display information for an achievement.
type AchievementInfo struct {
	ID          Achievement
	Name        string
	Description string
	Icon        string
	XPReward    int
}

// GetAchievementInfo returns display info for an achievement.
func GetAchievementInfo(a Achievement) AchievementInfo {
	info := map[Achievement]AchievementInfo{
		FirstGame: {
			ID:          FirstGame,
			Name:        "Premier Pas",
			Description: "Completez votre premiere partie",
			Icon:        "🎮",
			XPReward:    50,
		},
		Veteran: {
			ID:          Veteran,
			Name:        "Veteran",
			Description: "Completez 100 parties",
			Icon:        "🏆",
			XPReward:    500,
		},
		Dedicated: {
			ID:          Dedicated,
			Name:        "Dedie",
			Description: "Jouez 7 jours consecutifs",
			Icon:        "📅",
			XPReward:    200,
		},
		PerfectScore: {
			ID:          PerfectScore,
			Name:        "Sans Faute",
			Description: "Obtenez 100% sur au moins 10 questions",
			Icon:        "💯",
			XPReward:    300,
		},
		StreakMaster: {
			ID:          StreakMaster,
			Name:        "Serie Parfaite",
			Description: "10 bonnes reponses consecutives",
			Icon:        "🔥",
			XPReward:    150,
		},
		LevelTen: {
			ID:          LevelTen,
			Name:        "Naturaliste",
			Description: "Atteignez le niveau 10",
			Icon:        "🌿",
			XPReward:    100,
		},
		LevelFifty: {
			ID:          LevelFifty,
			Name:        "Expert Nature",
			Description: "Atteignez le niveau 50",
			Icon:        "🌳",
			XPReward:    1000,
		},
		MammalExpert: {
			ID:          MammalExpert,
			Name:        "Expert Mammiferes",
			Description: "Identifiez 100 mammiferes correctement",
			Icon:        "🦊",
			XPReward:    250,
		},
		BirdWatcher: {
			ID:          BirdWatcher,
			Name:        "Ornithologue",
			Description: "Identifiez 100 oiseaux correctement",
			Icon:        "🦅",
			XPReward:    250,
		},
		BugHunter: {
			ID:          BugHunter,
			Name:        "Entomologiste",
			Description: "Identifiez 100 insectes correctement",
			Icon:        "🦋",
			XPReward:    250,
		},
		Botanist: {
			ID:          Botanist,
			Name:        "Botaniste",
			Description: "Identifiez 100 plantes correctement",
			Icon:        "🌸",
			XPReward:    250,
		},
		ExpertMode: {
			ID:          ExpertMode,
			Name:        "Mode Expert",
			Description: "Completez un quiz en difficulte Expert",
			Icon:        "⭐",
			XPReward:    200,
		},
		MasterNatural: {
			ID:          MasterNatural,
			Name:        "Maitre Naturaliste",
			Description: "Completez un quiz Maitre avec 80%+",
			Icon:        "👑",
			XPReward:    500,
		},
	}

	if i, ok := info[a]; ok {
		return i
	}
	return AchievementInfo{ID: a, Name: string(a)}
}
