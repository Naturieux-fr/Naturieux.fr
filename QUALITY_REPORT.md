# Quality Report

> Last updated: 2026-01-19 19:32:08 UTC

## Quality Gate: ✅ PASSED

| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| Test Coverage | 71.6% | ≥ 70% | ✅ |
| Critical Issues | 0 | ≤ 0 | ✅ |
| Security Issues | 0 | = 0 | ✅ |
| Lint Issues | 0 | - | ℹ️ |
| High Complexity | 0 | ≤ 10 | ✅ |
| Code Duplicates | 13 | ≤ 5 | ⚠️ |

## Coverage by Package

```
github.com/fieve/naturieux/cmd/server/main.go:25:				main				0.0%
github.com/fieve/naturieux/cmd/server/main.go:110:				corsMiddleware			0.0%
github.com/fieve/naturieux/cmd/server/main.go:130:				newInMemoryPlayerRepository	0.0%
github.com/fieve/naturieux/cmd/server/main.go:136:				Create				0.0%
github.com/fieve/naturieux/cmd/server/main.go:141:				GetByID				0.0%
github.com/fieve/naturieux/cmd/server/main.go:148:				GetByUsername			0.0%
github.com/fieve/naturieux/cmd/server/main.go:157:				Update				0.0%
github.com/fieve/naturieux/cmd/server/main.go:162:				GetLeaderboard			0.0%
github.com/fieve/naturieux/internal/adapters/http/handlers.go:20:		NewHandler			100.0%
github.com/fieve/naturieux/internal/adapters/http/handlers.go:35:		writeJSON			100.0%
github.com/fieve/naturieux/internal/adapters/http/handlers.go:42:		writeError			100.0%
github.com/fieve/naturieux/internal/adapters/http/handlers.go:50:		writeSuccess			100.0%
github.com/fieve/naturieux/internal/adapters/http/handlers.go:111:		HandleStartSession		50.0%
github.com/fieve/naturieux/internal/adapters/http/handlers.go:160:		HandleSubmitAnswer		58.3%
github.com/fieve/naturieux/internal/adapters/http/handlers.go:216:		HandleAbandonSession		68.8%
github.com/fieve/naturieux/internal/adapters/http/handlers.go:246:		HandleHealthCheck		100.0%
github.com/fieve/naturieux/internal/adapters/http/handlers.go:259:		questionToDTO			0.0%
github.com/fieve/naturieux/internal/adapters/http/handlers.go:285:		RegisterRoutes			100.0%
github.com/fieve/naturieux/internal/adapters/inaturalist/client.go:36:		WithBaseURL			100.0%
github.com/fieve/naturieux/internal/adapters/inaturalist/client.go:43:		WithHTTPClient			100.0%
github.com/fieve/naturieux/internal/adapters/inaturalist/client.go:50:		WithUserAgent			100.0%
github.com/fieve/naturieux/internal/adapters/inaturalist/client.go:57:		NewClient			100.0%
github.com/fieve/naturieux/internal/adapters/inaturalist/client.go:80:		newRateLimiter			100.0%
github.com/fieve/naturieux/internal/adapters/inaturalist/client.go:84:		wait				100.0%
github.com/fieve/naturieux/internal/adapters/inaturalist/client.go:134:		doRequest			93.8%
github.com/fieve/naturieux/internal/adapters/inaturalist/client.go:164:		GetByID				92.3%
github.com/fieve/naturieux/internal/adapters/inaturalist/client.go:187:		GetRandom			91.7%
github.com/fieve/naturieux/internal/adapters/inaturalist/client.go:249:		GetSimilar			88.9%
github.com/fieve/naturieux/internal/adapters/inaturalist/client.go:295:		Search				88.2%
github.com/fieve/naturieux/internal/adapters/inaturalist/client.go:322:		taxonToSpecies			100.0%
github.com/fieve/naturieux/internal/adapters/inaturalist/client.go:335:		photoToSpeciesPhoto		100.0%
github.com/fieve/naturieux/internal/application/quiz/factory.go:34:		WithTaxonFilter			100.0%
github.com/fieve/naturieux/internal/application/quiz/factory.go:41:		WithPlaceFilter			100.0%
github.com/fieve/naturieux/internal/application/quiz/factory.go:48:		NewQuestionFactory		100.0%
github.com/fieve/naturieux/internal/application/quiz/factory.go:59:		CreateQuestion			90.5%
github.com/fieve/naturieux/internal/application/quiz/factory.go:125:		getWrongChoices			75.0%
github.com/fieve/naturieux/internal/application/quiz/factory.go:177:		selectMediaURL			42.9%
github.com/fieve/naturieux/internal/application/quiz/service.go:30:		NewService			100.0%
github.com/fieve/naturieux/internal/application/quiz/service.go:61:		StartSession			75.9%
github.com/fieve/naturieux/internal/application/quiz/service.go:152:		SubmitAnswer			62.5%
github.com/fieve/naturieux/internal/application/quiz/service.go:203:		handleSessionComplete		87.5%
github.com/fieve/naturieux/internal/application/quiz/service.go:259:		GetSessionStats			0.0%
github.com/fieve/naturieux/internal/application/quiz/service.go:267:		AbandonSession			71.4%
github.com/fieve/naturieux/internal/domain/gamification/achievements.go:41:	GetAchievementInfo		75.0%
github.com/fieve/naturieux/internal/domain/gamification/player.go:27:		NewPlayer			100.0%
github.com/fieve/naturieux/internal/domain/gamification/player.go:45:		ID				0.0%
github.com/fieve/naturieux/internal/domain/gamification/player.go:50:		Username			0.0%
github.com/fieve/naturieux/internal/domain/gamification/player.go:55:		TotalXP				100.0%
github.com/fieve/naturieux/internal/domain/gamification/player.go:60:		Level				100.0%
github.com/fieve/naturieux/internal/domain/gamification/player.go:65:		TotalGames			100.0%
github.com/fieve/naturieux/internal/domain/gamification/player.go:70:		Accuracy			66.7%
github.com/fieve/naturieux/internal/domain/gamification/player.go:78:		BestStreak			100.0%
github.com/fieve/naturieux/internal/domain/gamification/player.go:83:		DailyStreak			0.0%
github.com/fieve/naturieux/internal/domain/gamification/player.go:88:		Achievements			0.0%
github.com/fieve/naturieux/internal/domain/gamification/player.go:93:		XPForLevel			100.0%
github.com/fieve/naturieux/internal/domain/gamification/player.go:99:		XPToNextLevel			0.0%
github.com/fieve/naturieux/internal/domain/gamification/player.go:108:		XPProgress			83.3%
github.com/fieve/naturieux/internal/domain/gamification/player.go:119:		AddXP				86.7%
github.com/fieve/naturieux/internal/domain/gamification/player.go:153:		RecordGame			86.7%
github.com/fieve/naturieux/internal/domain/gamification/player.go:180:		checkAchievements		100.0%
github.com/fieve/naturieux/internal/domain/gamification/player.go:206:		hasAchievement			100.0%
github.com/fieve/naturieux/internal/domain/quiz/question.go:30:			NewQuestion			90.5%
github.com/fieve/naturieux/internal/domain/quiz/question.go:85:			ID				100.0%
github.com/fieve/naturieux/internal/domain/quiz/question.go:90:			QuizType			0.0%
github.com/fieve/naturieux/internal/domain/quiz/question.go:95:			Difficulty			0.0%
github.com/fieve/naturieux/internal/domain/quiz/question.go:100:		CorrectSpecies			0.0%
github.com/fieve/naturieux/internal/domain/quiz/question.go:105:		Choices				0.0%
github.com/fieve/naturieux/internal/domain/quiz/question.go:110:		MediaURL			0.0%
github.com/fieve/naturieux/internal/domain/quiz/question.go:115:		TimeLimit			0.0%
github.com/fieve/naturieux/internal/domain/quiz/question.go:120:		FlashDuration			0.0%
github.com/fieve/naturieux/internal/domain/quiz/question.go:125:		CheckAnswer			100.0%
github.com/fieve/naturieux/internal/domain/quiz/question.go:130:		CalculateScore			88.9%
github.com/fieve/naturieux/internal/domain/quiz/session.go:58:			NewSessionBuilder		100.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:66:			WithUserID			100.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:72:			WithDifficulty			100.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:78:			WithQuizTypes			100.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:84:			WithTaxonFilter			0.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:90:			WithQuestions			100.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:96:			Build				100.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:118:			ID				0.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:123:			UserID				0.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:128:			Difficulty			0.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:133:			Status				100.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:138:			TotalScore			0.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:143:			CurrentStreak			100.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:148:			MaxStreak			100.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:153:			QuestionsCount			0.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:158:			AnsweredCount			0.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:163:			CorrectCount			100.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:174:			CurrentQuestion			66.7%
github.com/fieve/naturieux/internal/domain/quiz/session.go:182:			Start				100.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:192:			SubmitAnswer			95.2%
github.com/fieve/naturieux/internal/domain/quiz/session.go:241:			Complete			100.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:248:			Abandon				100.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:255:			Accuracy			66.7%
github.com/fieve/naturieux/internal/domain/quiz/session.go:263:			Answers				0.0%
github.com/fieve/naturieux/internal/domain/quiz/session.go:268:			Duration			0.0%
github.com/fieve/naturieux/internal/domain/quiz/types.go:39:			DefaultDifficultyConfigs	100.0%
github.com/fieve/naturieux/internal/domain/quiz/types.go:73:			IsValidQuizType			100.0%
github.com/fieve/naturieux/internal/domain/quiz/types.go:82:			IsValidDifficulty		100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:23:		IsValidIconicTaxon		100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:50:		New				100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:68:		ID				100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:73:		ScientificName			100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:78:		CommonName			100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:83:		IconicTaxon			100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:88:		DisplayName			100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:96:		Photos				100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:101:		AddPhoto			100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:106:		HasPhotos			100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:111:		SetAncestorIDs			100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:116:		AncestorIDs			100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:121:		SetRank				100.0%
github.com/fieve/naturieux/internal/domain/species/species.go:126:		Rank				100.0%
total:										(statements)			71.6%
```

## Thresholds

- **Minimum Coverage**: 70%
- **Max Critical Issues**: 0
- **Max Security Issues**: 0
- **Max Cyclomatic Complexity**: 15

---
*Generated by Quality Gate workflow*
