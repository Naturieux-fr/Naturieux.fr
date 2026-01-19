# Naturieux.fr

[![CI](https://github.com/Naturieux-fr/Naturieux.fr/actions/workflows/ci.yml/badge.svg)](https://github.com/Naturieux-fr/Naturieux.fr/actions/workflows/ci.yml)
[![Quality Gate](https://github.com/Naturieux-fr/Naturieux.fr/actions/workflows/quality-gate.yml/badge.svg)](https://github.com/Naturieux-fr/Naturieux.fr/actions/workflows/quality-gate.yml)
[![Security](https://github.com/Naturieux-fr/Naturieux.fr/actions/workflows/security.yml/badge.svg)](https://github.com/Naturieux-fr/Naturieux.fr/actions/workflows/security.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Naturieux-fr/Naturieux.fr)](https://goreportcard.com/report/github.com/Naturieux-fr/Naturieux.fr)

Quiz naturaliste gamifie utilisant l'API iNaturalist pour l'identification d'especes.

## Fonctionnalites

- **Types de quiz varies**:
  - `ImageQuiz` - Image complete visible
  - `FlashQuiz` - Image visible brievement (1-3s)
  - `PartialQuiz` - Partie de l'image visible
  - `SilhouetteQuiz` - Silhouette uniquement
  - `SoundQuiz` - Audio uniquement

- **Niveaux de difficulte**:
  - Debutant (4 choix, 30s)
  - Intermediaire (6 choix, 20s)
  - Expert (8 choix, 15s)
  - Maitre (10 choix, 10s)

- **Gamification**:
  - Systeme de XP et niveaux
  - Achievements/badges
  - Streaks et bonus
  - Leaderboard

## Architecture

```
naturieux/
├── cmd/server/           # Point d'entree
├── internal/
│   ├── domain/           # Entites metier (DDD)
│   │   ├── species/      # Espece, Taxon
│   │   ├── quiz/         # Question, Session
│   │   └── gamification/ # Score, Niveau, Achievement
│   ├── ports/            # Interfaces (contrats)
│   ├── adapters/         # Implementations
│   │   ├── inaturalist/  # Client API iNaturalist
│   │   └── http/         # Handlers HTTP
│   └── application/      # Services applicatifs
└── docs/                 # Documentation
```

### Design Patterns

- **Repository Pattern** - Abstraction de l'acces aux donnees
- **Factory Pattern** - Creation des differents types de quiz
- **Strategy Pattern** - Strategies de difficulte
- **Builder Pattern** - Construction de sessions de quiz
- **Observer Pattern** - Notifications de gamification

## Installation

```bash
# Cloner le projet
git clone https://github.com/Naturieux-fr/Naturieux.fr.git
cd Naturieux.fr

# Installer les dependances
go mod download

# Lancer les tests
go test ./... -cover

# Compiler
go build -o bin/server ./cmd/server

# Lancer le serveur
./bin/server
```

## API

### Demarrer une session

```bash
POST /api/v1/quiz/start
Content-Type: application/json

{
  "user_id": "demo",
  "difficulty": "beginner",
  "quiz_types": ["image"],
  "taxon_filter": "Mammalia",
  "question_count": 10
}
```

### Soumettre une reponse

```bash
POST /api/v1/quiz/answer
Content-Type: application/json

{
  "session_id": "abc123",
  "species_id": 42069,
  "time_taken_ms": 5000
}
```

### Health check

```bash
GET /health
```

## Couverture de Tests

| Module | Couverture |
|--------|------------|
| species | 100% |
| inaturalist | 92.7% |
| gamification | 81% |
| application/quiz | 81.9% |
| quiz domain | 77.5% |
| http handlers | 60.5% |
| **Total** | **73.8%** |

## API iNaturalist

Ce projet utilise l'[API iNaturalist](https://api.inaturalist.org/v1/docs/) pour recuperer les donnees sur les especes.

### Limites

- ~1 requete/seconde
- ~10,000 requetes/jour
- User-Agent requis

### Endpoints utilises

- `GET /observations` - Observations avec photos
- `GET /taxa` - Recherche de taxons
- `GET /taxa/autocomplete` - Autocompletion

## Licence

MIT
