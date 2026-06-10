# Naturieux.fr

[![CI](https://github.com/Naturieux-fr/Naturieux.fr/actions/workflows/ci.yml/badge.svg)](https://github.com/Naturieux-fr/Naturieux.fr/actions/workflows/ci.yml)
[![Quality Gate](https://github.com/Naturieux-fr/Naturieux.fr/actions/workflows/quality-gate.yml/badge.svg)](https://github.com/Naturieux-fr/Naturieux.fr/actions/workflows/quality-gate.yml)
[![Security](https://github.com/Naturieux-fr/Naturieux.fr/actions/workflows/security.yml/badge.svg)](https://github.com/Naturieux-fr/Naturieux.fr/actions/workflows/security.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Naturieux-fr/Naturieux.fr)](https://goreportcard.com/report/github.com/Naturieux-fr/Naturieux.fr)

Quiz d'identification d'espèces, gamifié, adossé au **référentiel taxonomique national TAXREF** et à une collection de photos maison. Interface « carnet de naturaliste » (Go + Alpine.js), base SQLite embarquée, déployable en un conteneur.

## Fonctionnalités

- **Quiz par image** : identifier l'espèce d'après une photo, avec des distracteurs taxonomiquement proches (même genre, puis famille, puis ordre) pour des questions réellement difficiles.
- **Catégories françaises** issues de TAXREF (Mammifères, Oiseaux, Reptiles, Amphibiens, Insectes, Plantes, Champignons) et 4 niveaux d'épreuve (Apprenti → Maître).
- **Comptes joueurs** : pseudo, progression (XP, niveaux, séries, achievements), classement au mérite.
- **Back-office `/admin`** : recherche d'une espèce, gestion des photos par `cd_nom` et réglage de leur difficulté ; upload de fichier (stockage local ou S3/MinIO).
- **Trois sources d'espèces** interchangeables : TAXREF local, API iNaturalist (avec cache), ou données de démonstration.

## Architecture

Architecture hexagonale (ports & adapters) + DDD.

```
cmd/
├── server/        # serveur HTTP
├── importtaxref/  # import du référentiel TAXREF
└── importphotos/  # import en masse d'une collection de photos
internal/
├── domain/        # entités métier (species, quiz, gamification)
├── ports/         # interfaces (contrats)
├── application/   # services (quiz, admin)
├── adapters/      # taxref, inaturalist, cache, sqlite, storage, http
├── auth/          # mots de passe (bcrypt) + jetons de session
└── media/         # conversion RAW → JPEG
web/               # frontend (HTML/CSS/Alpine.js)
docs/vault/        # base de connaissances (état, roadmap, faisabilité, normes)
```

## Démarrage rapide

```bash
go run ./cmd/server          # source par défaut : iNaturalist + cache
DEV_MODE=1 go run ./cmd/server  # données de démonstration (sans réseau)
go test ./... -cover
```

### Mode TAXREF (recommandé)

```bash
# 1. Référentiel TAXREF natif (Licence Ouverte) — fichier TAXREFvNN.txt
go run ./cmd/importtaxref -file TAXREFv18.txt -version v18.0
# 2. Collection de photos (CSV: photo;groupe_taxonomique;nom_scientifique)
#    Les RAW (.RW2…) sont convertis en JPEG, les noms résolus en cd_nom.
go run ./cmd/importphotos -csv collection.csv -dir ./photos -attribution "(c) Naturieux" -license cc-by
# 3. Lancer
SPECIES_SOURCE=taxref go run ./cmd/server
```

### Docker

```bash
docker compose up --build       # http://localhost:8080
```

`docker-compose.yml` documente toutes les variables : `SPECIES_SOURCE`, `STORAGE` (local|s3) et `S3_*`, `ADMIN_USERNAME` / `ADMIN_PASSWORD` / `AUTH_SECRET`.

## API (extrait)

| Méthode | Route | Rôle |
|---|---|---|
| POST | `/api/v1/players` | inscription (pseudo) |
| GET | `/api/v1/players/{id}` | profil |
| POST | `/api/v1/quiz/start` | démarrer une partie |
| POST | `/api/v1/quiz/answer` | répondre |
| GET | `/api/v1/leaderboard` | classement |
| POST | `/api/v1/auth/login` | connexion admin |
| GET | `/health` | sonde de santé |

## Sources & licences

- **TAXREF** v18 (INPN / PatriNat — OFB-MNHN-CNRS-IRD), Licence Ouverte / CC-BY.
- **Photos** : collection propre, ou sources Creative Commons via iNaturalist (filtrage `photo_license`, attribution conservée).

Détails et étude de faisabilité dans [`docs/vault`](docs/vault).

## Licence

MIT
