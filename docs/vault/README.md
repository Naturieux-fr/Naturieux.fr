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

### Importer TAXREF (fichier natif INPN)

On utilise le **fichier natif** `TAXREFvNN.txt` (et non l'export GBIF réduit) car il
porte les colonnes `GROUP1/2/3_INPN` (catégories françaises : Mammifères, Oiseaux,
Reptiles, Amphibiens…) et `FR` (présence métropole).

```bash
# 1. Récupérer l'archive TAXREF native (Licence Ouverte) et extraire TAXREFv18.txt
#    Source officielle : https://inpn.mnhn.fr/telechargement/referentielEspece/taxref/18.0/menu
#    Miroir : https://geonature.fr/data/inpn/taxonomie/TAXREF_v18_2025.zip
unzip TAXREF_v18_2025.zip TAXREFv18.txt
# 2. Charger dans la base (~3 s pour 212k espèces)
go run ./cmd/importtaxref -file TAXREFv18.txt -version v18.0
# 3. Lancer en mode TAXREF
SPECIES_SOURCE=taxref go run ./cmd/server
```

Les catégories du quiz filtrent sur `GROUP2_INPN` (ou `REGNE` pour Plantes/Champignons).
Les photos maison se lient par `cd_nom` (table `taxref_photos`).

### Importer une collection de photos en masse

```bash
# CSV: photo;groupe_taxonomique;nom_scientifique  (séparateur ;)
# Les noms scientifiques sont résolus en cd_nom ; les RAW (.RW2) sont ignorés.
STORAGE=local go run ./cmd/importphotos \
  -csv BDD_test.csv -dir ./photos \
  -attribution "(c) Naturieux" -license cc-by
```

### Docker

```bash
docker compose up --build        # http://localhost:8080, base persistée dans un volume
DEV_MODE=1 docker compose up      # en mode mock
```
