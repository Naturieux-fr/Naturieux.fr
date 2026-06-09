# Vault de connaissances — Naturieux.fr

Base de connaissances du projet. Chaque note est autonome et reliée aux autres.

## Notes

| Note | Contenu |
|------|---------|
| [01 — État du projet](01-etat-du-projet.md) | Ce qui est fait, ce qui fonctionne, métriques |
| [02 — Reste à faire](02-reste-a-faire.md) | Roadmap priorisée des fonctionnalités et dettes techniques |
| [03 — Faisabilité](03-faisabilite.md) | Étude de faisabilité : API iNaturalist, licences, quotas, alternatives |
| [04 — Normes de code](04-normes-de-code.md) | Conventions Go, architecture, qualité, workflow git |

## Documentation technique existante

- [Architecture](../architecture.md) — architecture hexagonale, design patterns, types de quiz
- [API iNaturalist](../api_inaturalist.md) — endpoints, paramètres, structures de réponse

## Démarrage rapide

```bash
go run ./cmd/server          # serveur sur http://localhost:8080
DEV_MODE=1 go run ./cmd/server  # mode dev avec données mock (pas d'appel API)
go test ./... -cover         # tests
golangci-lint run            # lint
```
