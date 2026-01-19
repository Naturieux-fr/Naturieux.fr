# Architecture Naturieux.fr

## Vue d'ensemble

Architecture Hexagonale (Clean Architecture) avec Domain-Driven Design.

```
naturieux/
├── cmd/
│   └── server/           # Point d'entree
├── internal/
│   ├── domain/           # Entites et logique metier
│   │   ├── species/      # Espece, Taxon
│   │   ├── quiz/         # Question, Session, Types
│   │   ├── user/         # Joueur, Profil
│   │   └── gamification/ # Score, Niveau, Achievement
│   ├── ports/            # Interfaces (contrats)
│   │   ├── inbound/      # API vers domaine
│   │   └── outbound/     # Domaine vers externe
│   ├── adapters/         # Implementations
│   │   ├── inaturalist/  # Client API iNaturalist
│   │   ├── http/         # Handlers HTTP
│   │   └── persistence/  # Base de donnees
│   └── application/      # Services applicatifs
├── pkg/                  # Utilitaires partages
├── docs/                 # Documentation
└── test/                 # Tests d'integration
```

## Design Patterns Utilises

### 1. Repository Pattern
Abstraction de l'acces aux donnees.
```go
type SpeciesRepository interface {
    GetByID(ctx context.Context, id int) (*Species, error)
    GetRandom(ctx context.Context, filter SpeciesFilter) ([]*Species, error)
}
```

### 2. Factory Pattern
Creation des differents types de quiz.
```go
type QuizFactory interface {
    Create(quizType QuizType, difficulty Difficulty) (Quiz, error)
}
```

### 3. Strategy Pattern
Strategies de difficulte et d'affichage.
```go
type DifficultyStrategy interface {
    GetTimeLimit() time.Duration
    GetChoicesCount() int
    GetScoreMultiplier() float64
}
```

### 4. Observer Pattern
Notifications de gamification.
```go
type GameEventPublisher interface {
    Subscribe(subscriber GameEventSubscriber)
    Publish(event GameEvent)
}
```

### 5. Builder Pattern
Construction de sessions de quiz.
```go
session := NewQuizSessionBuilder().
    WithDifficulty(Expert).
    WithQuizTypes(ImageQuiz, FlashQuiz).
    WithTaxonFilter(Mammalia).
    WithQuestionCount(10).
    Build()
```

## Types de Quiz

| Type | Description | Difficulte |
|------|-------------|------------|
| ImageQuiz | Image complete visible | Toutes |
| FlashQuiz | Image visible 1-3s | Intermediaire+ |
| PartialQuiz | Partie de l'image visible | Expert+ |
| SilhouetteQuiz | Silhouette de l'animal | Expert+ |
| SoundQuiz | Son de l'animal | Toutes |

## Niveaux de Difficulte

| Niveau | Choix | Temps | Multiplicateur |
|--------|-------|-------|----------------|
| Debutant | 4 | 30s | x1.0 |
| Intermediaire | 6 | 20s | x1.5 |
| Expert | 8 | 15s | x2.0 |
| Maitre | 10 | 10s | x3.0 |

## Gamification

- **XP**: Points d'experience par bonne reponse
- **Niveaux**: Progression du joueur (1-100)
- **Achievements**: Badges pour accomplissements
- **Streaks**: Bonus pour series de bonnes reponses
- **Classement**: Leaderboard global et par categorie
