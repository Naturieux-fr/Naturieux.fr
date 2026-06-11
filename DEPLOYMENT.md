# Déploiement de Naturieux

Guide complet pour héberger Naturieux en production (HTTPS) ou en réseau local.
L'application est **autonome** : en fonctionnement elle n'appelle aucun service
externe — toutes les données vivent dans sa base SQLite locale.

> Architecture : un **serveur Go** (back-end + base SQLite) servi derrière un
> reverse proxy **Caddy** qui gère le TLS automatiquement. Les navigateurs des
> joueurs ne parlent qu'à **ton** serveur.

---

## 1. Prérequis

- Une machine (VPS ou serveur perso) avec **Docker** + **Docker Compose**.
- Pour l'accès public en HTTPS :
  - un **nom de domaine** (ex. `naturieux.fr`) dont l'enregistrement **A/AAAA**
    pointe vers l'IP publique de la machine ;
  - les ports **80** et **443** ouverts et redirigés vers la machine.
- Pour un usage **réseau local uniquement**, ni domaine ni Internet ne sont
  nécessaires (voir §8).

---

## 2. Récupérer le code

```bash
git clone https://github.com/Naturieux-fr/Naturieux.fr.git
cd Naturieux.fr
```

---

## 3. Configuration

Crée un fichier `.env` à la racine (Docker Compose le lit automatiquement) :

```dotenv
# Domaine servi par Caddy (HTTPS auto)
NATURIEUX_DOMAIN=naturieux.fr

# Clé de signature des sessions — OBLIGATOIRE et STABLE en prod.
# Générer une fois : openssl rand -hex 32
AUTH_SECRET=colle-ici-une-longue-valeur-aleatoire

# Compte administrateur initial (créé au premier démarrage)
ADMIN_USERNAME=admin
ADMIN_PASSWORD=un-mot-de-passe-fort

# Inscription : open (tout le monde) ou invite (sur lien admin)
REGISTRATION_MODE=open

# Source des espèces : vide = auto (TAXREF si importé, sinon iNaturalist).
# Forcer avec : taxref | inat | mock
SPECIES_SOURCE=

# Stockage des médias : local (volume Docker) ou s3 (MinIO/S3)
STORAGE=local
```

> ⚠️ **`AUTH_SECRET` doit rester identique** entre deux redémarrages : sinon
> tous les jetons de session deviennent invalides et les joueurs sont
> déconnectés.

---

## 4. Lancement en HTTPS

```bash
docker compose --profile tls up -d --build
```

Caddy obtient le certificat Let's Encrypt au premier accès HTTPS sur le domaine.
Le site est alors disponible sur `https://naturieux.fr` (le HTTP est redirigé).

Vérifier :

```bash
curl -sf https://naturieux.fr/health     # doit répondre
docker compose logs -f naturieux caddy   # suivre les logs
```

> Test local sans vrai domaine : `NATURIEUX_DOMAIN=localhost` (Caddy émet un
> certificat auto-signé).

---

## 5. DNS (exemple OVH)

Dans la **zone DNS** de `naturieux.fr` :

1. Ajouter un enregistrement **A** : `naturieux.fr` → **IP publique** du serveur
   (+ un `A` ou `CNAME` pour `www`).
2. **Ne pas toucher aux enregistrements MX** (tes e-mails continuent de marcher).
3. Ignorer l'« hébergement web gratuit » d'OVH : il ne peut **pas** exécuter une
   application Go/Docker.

La propagation DNS peut prendre quelques minutes à quelques heures.

---

## 6. Premier démarrage & administration

- L'**admin** est créé à partir de `ADMIN_USERNAME` / `ADMIN_PASSWORD`.
- Console d'administration : **`https://naturieux.fr/admin`**.
- Page mentions légales / RGPD : **`/legal`** — **complète les champs
  `[à compléter]`** de `web/legal.html` (éditeur, e-mail de contact, hébergeur)
  avant l'ouverture publique.

---

## 7. Charger les données (une seule fois)

Naturieux fonctionne sans API externe : on **importe les données une fois** dans
sa base, puis plus aucun appel sortant.

### 0. Bootstrap automatique (optionnel)

Pour que `docker compose up` télécharge **et importe TAXREF** tout seul au
**premier** démarrage (puis plus jamais), renseigne dans `.env` :

```dotenv
# URL du fichier TAXREF natif (.txt) ou de son archive .zip (open data INPN)
BOOTSTRAP_TAXREF_URL=https://…/TAXREFvNN.zip
# Optionnel — export d'occurrences pour lieu/saison (fichier potentiellement
# très volumineux : à activer en connaissance de cause)
BOOTSTRAP_OCCURRENCES_URL=
```

Le téléchargement n'a lieu que si la base est **vide**. **Tes photos et sons ne
sont jamais téléchargés** (ce sont tes médias) : ajoute-les via l'admin (§7b).
Si tu préfères tout maîtriser à la main, laisse ces variables vides et utilise
les imports ci-dessous.

Les imports écrivent dans le **même fichier SQLite** que le serveur (volume
`naturieux-data`). Le plus simple est de les lancer dans un conteneur jetable
`golang` qui monte ce volume — **aucun Go requis sur l'hôte**. Fais-les de
préférence **avant le premier démarrage** ou pendant une courte maintenance
(`docker compose stop naturieux`).

### a. Référentiel des espèces (TAXREF)

Télécharger le fichier natif TAXREF (INPN, Licence Ouverte) dans le dossier
courant, puis :

```bash
docker run --rm \
  -v naturieux_naturieux-data:/data \
  -v "$PWD":/w -w /w golang:1.25 \
  go run ./cmd/importtaxref -file /w/TAXREFv18.txt -db /data/naturieux.db -version v18.0
```

### b. Photos & sons

Depuis **`/admin` → Espèces & photos** :
- ajouter des **photos** par espèce (upload ou URL) ;
- ajouter des **chants/sons** (section « Chants & sons ») pour le quiz Son.

### c. Lieu & saison (optionnel)

Pour activer les filtres **région** et **mois**, importer **une fois** un export
d'occurrences (GBIF « Simple CSV » filtré sur la France, ou INPN/OpenObs keyé par
CD_REF) :

```bash
docker run --rm \
  -v naturieux_naturieux-data:/data \
  -v "$PWD":/w -w /w golang:1.25 \
  go run ./cmd/importoccurrences -file /w/occurrence.txt -db /data/naturieux.db
```

Détails : `docs/vault/06-import-lieu-saison.md`. L'appariement se fait **par
identifiant TAXREF** (CD_REF) si présent, sinon par nom scientifique.

---

## 8. Réseau local (sans Internet)

Pour jouer en LAN sans domaine ni HTTPS, lancer sans le profil `tls` :

```bash
docker compose up -d --build         # le serveur écoute sur :8080
```

Les joueurs accèdent via `http://IP-LOCALE:8080`. Pense à ouvrir le port dans le
pare-feu de la machine.

---

## 9. Sauvegardes

Tout l'état (base SQLite + photos/sons locaux) est dans le volume Docker
`naturieux-data` (`/data`). À sauvegarder régulièrement :

```bash
docker run --rm \
  -v naturieux_naturieux-data:/data \
  -v "$PWD":/backup alpine \
  tar czf /backup/naturieux-backup.tar.gz -C /data .
```

Restauration : décompresser l'archive dans le volume avant de démarrer.

---

## 10. Mises à jour

```bash
git pull
docker compose --profile tls up -d --build
```

La base et les médias (volume `/data`) sont conservés. Le schéma SQLite est
migré automatiquement au démarrage.

---

## 11. Dépannage

| Symptôme | Piste |
|----------|-------|
| Le certificat HTTPS ne se crée pas | Ports **80/443** doivent être accessibles depuis Internet. Certains FAI **bloquent le port 80** en auto-hébergement → me demander la méthode **DNS-01** (Caddy + module OVH). |
| `/health` ne répond pas | `docker compose logs naturieux` ; vérifier que le conteneur tourne. |
| Joueurs déconnectés après redémarrage | `AUTH_SECRET` a changé → le fixer dans `.env`. |
| Métriques / supervision | `GET /metrics` (format Prometheus) : requêtes, erreurs, par classe. |
| Pas de questions | Source d'espèces vide : importer TAXREF, ou `SPECIES_SOURCE=mock` pour un test. |

---

## 12. Récapitulatif des variables

| Variable | Rôle | Défaut |
|----------|------|--------|
| `NATURIEUX_DOMAIN` | Domaine servi par Caddy | `localhost` |
| `AUTH_SECRET` | Clé de signature des sessions (**à fixer**) | aléatoire éphémère |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | Compte admin initial | — |
| `REGISTRATION_MODE` | `open` ou `invite` | `open` |
| `SPECIES_SOURCE` | `taxref` / `inat` / `mock` (vide = auto) | auto |
| `STORAGE` | `local` ou `s3` | `local` |
| `DB_PATH` | Chemin de la base SQLite | `/data/naturieux.db` |
| `MEDIA_DIR` | Dossier des médias locaux | `/data/media` |
| `PORT` | Port d'écoute du serveur | `8080` |

Stockage S3/MinIO : voir les variables `S3_*` dans `docker-compose.yml`.

---

*Pour aller plus loin : `docs/vault/05-mise-en-ligne.md` (mise en ligne) et
`docs/vault/06-import-lieu-saison.md` (import des occurrences).*
