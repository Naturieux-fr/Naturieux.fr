# 04 — Normes de code

*Conventions à respecter sur le projet. Mise à jour : 2026-06-09.*

## Go

- **Version** : Go 1.22 (cf. `go.mod`).
- **Formatage** : `gofmt` obligatoire (vérifié en CI). Lancer `gofmt -w .` avant commit.
- **Lint** : `golangci-lint run` doit passer sans erreur.
- **Vet** : `go vet ./...` doit passer (inclut la cohérence des tests avec les signatures).
- **Commentaires de doc** : chaque package a un commentaire `// Package xxx ...` ; chaque symbole exporté est documenté (`// NomDuSymbole ...`).
- **Erreurs** : retournées, jamais ignorées silencieusement (sauf cas justifié et commenté, ex. encode de réponse déjà envoyée).
- **Contexte** : `context.Context` en premier paramètre de toute fonction faisant de l'I/O.

## Architecture (hexagonale + DDD)

- **`internal/domain`** : aucune dépendance externe (ni HTTP, ni API, ni DB). Logique métier pure, invariants validés dans les constructeurs (`NewXxx`).
- **`internal/ports`** : interfaces uniquement (contrats consommés par l'application).
- **`internal/adapters`** : implémentations concrètes des ports (iNaturalist, HTTP, mock, persistance à venir). Un adapter ne dépend jamais d'un autre adapter.
- **`internal/application`** : orchestration des cas d'usage, dépend du domaine et des ports, jamais des adapters.
- Sens des dépendances : `adapters → application → domain`. Jamais l'inverse.

## Tests

- Tests unitaires à côté du code (`xxx_test.go`), package `xxx_test` pour tester l'API publique.
- Objectif de couverture : **≥ 75 % global**, 100 % sur le domaine pur.
- Pas de tests dépendant du réseau : utiliser `httptest` et le dépôt mock.
- Nommage : `TestType_Méthode_Cas` (ex. `TestHandler_HandleStartSession_InvalidJSON`).

## Git

- **Branches** : travail sur `dev`, merge vers `main`.
- **Commits conventionnels** : `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:` — message à l'impératif, concis, en anglais.
- **Aucun trailer ni signature automatique** dans les messages de commit.
- Ne jamais commiter : binaires, fichiers de couverture, dossiers de configuration d'outils locaux, secrets (utiliser `.git/info/exclude` pour les exclusions propres au poste).
- CI (GitHub Actions) : build + tests + lint + quality-gate + security doivent être verts avant merge.

## API / frontend

- Réponses API au format `Response{success, data, error}` uniforme.
- Validation des entrées dans les handlers (méthode HTTP, JSON, champs requis) avant l'appel au service.
- Frontend sans framework lourd : Alpine.js + CSS vanilla, pas de build step.
- Toute photo/son affiché doit porter son attribution (voir [03 — Faisabilité](03-faisabilite.md)).
