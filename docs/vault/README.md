# Vault de connaissances — Naturieux.fr

Base de connaissances du projet. Chaque note est autonome et reliée aux autres.

## Notes

| Note | Contenu |
|------|---------|
| [01 — État du projet](01-etat-du-projet.md) | Ce qui est fait, ce qui fonctionne, métriques |
| [02 — Reste à faire](02-reste-a-faire.md) | Roadmap priorisée des fonctionnalités et dettes techniques |
| [03 — Faisabilité](03-faisabilite.md) | Étude de faisabilité : API iNaturalist, licences, quotas, alternatives |
| [04 — Normes de code](04-normes-de-code.md) | Conventions Go, architecture, qualité, workflow git |
| [05 — TAXREF & photos](05-taxref-et-photos.md) | Référentiel TAXREF (source, structure, licence), stratégie photos, choix relationnel vs graphe |

## Documentation technique existante

- [Architecture](../architecture.md) — architecture hexagonale, design patterns, types de quiz
- [API iNaturalist](../api_inaturalist.md) — endpoints, paramètres, structures de réponse

## Démarrage rapide

```bash
go run ./cmd/server              # iNaturalist + cache (défaut)
DEV_MODE=1 go run ./cmd/server   # données mock (pas d'appel réseau)
go test ./... -cover             # tests
golangci-lint run                # lint
```

### Source d'espèces (`SPECIES_SOURCE`)

```bash
SPECIES_SOURCE=taxref go run ./cmd/server   # TAXREF local + nos photos
SPECIES_SOURCE=inat   go run ./cmd/server   # iNaturalist + cache (défaut)
SPECIES_SOURCE=mock   go run ./cmd/server   # données de démo
```

### Importer TAXREF

```bash
# 1. Télécharger l'archive Darwin Core (CC-BY) et extraire taxon.txt
curl -L "https://ipt.gbif.fr/archive.do?r=taxref" -o taxref.zip && unzip taxref.zip taxon.txt
# 2. Charger dans la base (~2,7 s pour 212k espèces)
go run ./cmd/importtaxref -file taxon.txt -version v18.0
# 3. Lancer en mode TAXREF
SPECIES_SOURCE=taxref go run ./cmd/server
```

Les photos maison se lient par `cd_nom` (table `taxref_photos`).

### Docker

```bash
docker compose up --build        # http://localhost:8080, base persistée dans un volume
DEV_MODE=1 docker compose up      # en mode mock
```
