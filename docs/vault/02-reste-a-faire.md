# 02 — Reste à faire

*Roadmap priorisée. Mise à jour : 2026-06-09. Voir [01 — État du projet](01-etat-du-projet.md) pour le contexte.*

## P0 — Critique (conformité et fondations)

- [x] **Filtrage des licences photos** : `photo_license=cc0,cc-by,cc-by-nc` sur les requêtes observations + filtre défensif côté client (photo par défaut du taxon exclue de l'affichage). Reste à faire pour les **sons** (`sound_license`) quand le SoundQuiz arrivera.
- [x] **Attribution des médias** : `license_code` + `attribution` portés jusqu'à l'API (`media_attribution`, `media_license`) et affichés sous l'image du quiz. Amélioration possible : lien vers l'observation d'origine.
- [x] **Persistance SQLite** : `internal/adapters/sqlite` (driver pur Go `modernc.org/sqlite`), sessions stockées en snapshot JSON + colonnes dénormalisées pour les stats, joueurs persistés (XP, niveaux, achievements). Le store en mémoire du handler est supprimé : les parties survivent à un redémarrage du serveur. Variable `DB_PATH` (défaut `naturieux.db`).
- [x] **Cache local des espèces** : `internal/adapters/cache` (décorateur SQLite autour du client iNaturalist). Préchauffage ~60 espèces/taxon au démarrage + toutes les 12 h (1 requête/taxon), TTL 7 jours, dégradation gracieuse vers l'API. Démarrage d'une partie : ~8-10 s → < 25 ms. Reste possible : back-off explicite sur 429 côté client.

## P1 — Important (expérience de jeu)

- [x] **Comptes utilisateurs simples** : `POST /api/v1/players` (pseudo unique 2-20 caractères) + `GET /api/v1/players/{id}`, identifiant conservé en localStorage, formulaire de bienvenue au premier lancement. Le serveur est la source de vérité de l'XP/niveaux. Évolution possible : récupération de compte (code secret ou e-mail).
- [x] **Leaderboard** : endpoint `GET /api/v1/leaderboard` (classement par XP, paramètre `limit`) + écran frontend (médailles top 3, niveau, parties, précision). Évolution possible : classement par taxon ou par période.
- [ ] **Quiz types restants côté frontend** :
  - Flash : déjà côté API (durée d'affichage), finaliser l'UI.
  - Partial : recadrage CSS (zoom sur une zone de la photo).
  - Silhouette : filtre CSS (`brightness(0)` sur fond clair) — à valider visuellement.
  - Sound : nécessite `sounds=true` côté client iNaturalist + lecteur audio ; compléter avec Xeno-canto pour les oiseaux (clé API requise depuis oct. 2025).
- [ ] **Filtres avancés** : choix du groupe taxonomique (oiseaux, mammifères, plantes, champignons…), du lieu (région/département), de la saison.
- [ ] **Affichage des achievements et niveaux** dans l'UI (le domaine gamification existe déjà).

## Source de données TAXREF (indépendance iNaturalist)

- [x] **Référentiel TAXREF local** : `internal/adapters/taxref` (dépôt SQLite implémentant `ports.SpeciesRepository`). Importeur Darwin Core (`cmd/importtaxref`), 212k espèces valides chargées en ~2,7 s. Sélectionnable par `SPECIES_SOURCE=taxref`. Voir [05 — TAXREF & photos](05-taxref-et-photos.md).
- [x] **Photos « à nous »** : table `taxref_photos` liée par `cd_nom`, alimentée par notre propre collection (`Repository.AddPhoto`). Plus de dépendance iNaturalist pour les images en mode taxref.
- [x] **Back-office admin des photos** : page `/admin` (auth par rôle, mots de passe bcrypt, jeton HMAC) pour chercher une espèce, ajouter/supprimer des photos liées par `cd_nom` et régler leur **difficulté** (utilisée par le tirage du quiz). Upload de fichier (stockage **local** ou **S3/MinIO** via `STORAGE`) ou URL externe. Admin seedé par `ADMIN_USERNAME`/`ADMIN_PASSWORD`.
- [ ] **Import en masse des photos** : commande/écran pour charger une collection entière (CSV cd_nom→fichier/url) d'un coup, au lieu d'une par une.
- [ ] **SRI sur les CDN** : ajouter `integrity`/`crossorigin` sur Alpine.js (index.html + admin.html) ou auto-héberger la lib.
- [ ] **Nettoyage fichiers orphelins** : supprimer le fichier stocké quand on supprime une photo locale (aujourd'hui seul l'enregistrement part).
- [x] **Conversion RAW → JPEG** : `internal/media` extrait l'aperçu JPEG pleine taille embarqué dans les RAW (.RW2…) en Go pur, sans dépendance externe ; `cmd/importphotos` convertit à la volée (4 `.RW2` → JPEG 1920×1440).
- [ ] **Présence métropole** : la version GBIF de TAXREF ne porte pas la colonne de présence territoriale (`FR`) ; pour filtrer strictement la métropole, croiser avec l'extrait INPN ou les statuts. Actuellement tous les taxons valides sont inclus.

## P2 — Améliorations

- [ ] **Migration API v2 iNaturalist** : sélection de champs (`fields`), UUID — la v1 fonctionne mais la v2 est la version pérenne (pertinent seulement si on garde iNaturalist en source d'appoint).
- [ ] **Tests E2E frontend** automatisés (Playwright) intégrés à la CI.
- [ ] **WebSocket** : mode duel / multijoueur en temps réel.
- [ ] **Mode révision** : revoir les espèces ratées (répétition espacée).
- [ ] **i18n** : noms vernaculaires français via `locale=fr` (paramètre API), fallback nom scientifique.
- [ ] **PWA** : manifest + service worker pour usage mobile/terrain.

## P3 — Déploiement

- [x] **Conteneurisation** : Dockerfile multi-stage (binaire statique CGO-free sur alpine, non-root, healthcheck `/health`, volume `/data` pour la base) + `docker-compose.yml`. `docker compose up` lance l'app.
- [ ] **Hébergement naturieux.fr** : reverse proxy + TLS, variable `PORT`, logs.
- [ ] **Page mentions légales** : crédits iNaturalist, licences des médias, politique de confidentialité.
- [ ] **Monitoring** : métriques basiques (sessions/jour, taux d'erreur API iNaturalist).

## Dette technique

- [x] Store de sessions déplacé hors du handler HTTP (service + repository SQLite).
- [x] Repo joueurs en mémoire de `cmd/server` supprimé (remplacé par SQLite).
- [ ] Couverture handlers HTTP < 60 % : tester les chemins succès avec un service mocké (injecter une interface au lieu de `*appquiz.Service`).
- [ ] Harmoniser fins de ligne (warnings LF/CRLF sous Windows) via `.gitattributes`.
