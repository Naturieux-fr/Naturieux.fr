# 02 — Reste à faire

*Roadmap priorisée. Mise à jour : 2026-06-11. Voir [01 — État du projet](01-etat-du-projet.md) pour le contexte.*

## Fait depuis le 09/06 (résumé)

Beaucoup de la liste P1/P2 a été livré : tous les **types de quiz** sauf le son
(Image, Éclair, Silhouette, Détail) + **réponse libre**, le **multijoueur**
temps réel (salons, modes Classique/Élimination, manches synchronisées,
WebSocket + reconnexion), les **vrais comptes** (mot de passe, inscription
libre ou sur invitation), la **gamification** (combos, toasts, série
quotidienne, **grades à paliers** par spécialité), l'**import photos en masse**,
la **conversion RAW→JPEG**, **MinIO** dans compose, l'auto-détection de la
source TAXREF, le **SRI** sur le CDN, le nettoyage des fichiers orphelins, et
l'**identité visuelle** (logo + favicon).

## P0 — Critique (conformité et fondations)

- [x] **Filtrage des licences photos** + filtre défensif client. (Sons : à faire avec le SoundQuiz.)
- [x] **Attribution des médias** affichée sous l'image. (Amélioration : lien vers l'observation d'origine.)
- [x] **Persistance SQLite** (`internal/adapters/sqlite`, driver pur Go). `DB_PATH`.
- [x] **Cache local des espèces** (`internal/adapters/cache`, préchauffage + TTL 7 j).

## P1 — Important (expérience de jeu)

- [x] **Comptes utilisateurs** : passés de simples pseudos à de **vrais comptes** — mot de passe bcrypt, jeton de session, **inscription libre ou sur invitation** (`REGISTRATION_MODE`, liens admin). `internal/application/account`.
- [x] **Leaderboard** (`GET /api/v1/leaderboard` + écran). (Évolution : par taxon / période.)
- [x] **Types de quiz** : Image, **Éclair**, **Silhouette**, **Détail** + **réponse libre** (3 essais). Sélecteurs sur l'accueil et dans les salons.
  - [x] **Son** : fait — mode **Chant** avec lecteur audio, sur des **enregistrements possédés** (table `taxref_sounds`, upload/URL dans l'admin, pas d'API externe). Reste à téléverser des chants.
- [~] **Filtres avancés** : **catégorie** faite, **mode famille** fait ; **lieu/saison** bloqués faute de données (phénologie + occurrence fine absentes de TAXREF — voir P2).
- [x] **Affichage des achievements et grades** : écran « Cabinet des hauts faits » (galerie débloqués/verrouillés) + **grades de spécialité à paliers** (Mammalogiste, Ornithologue… I/II/III à 100/500/2000) + toasts temps réel.

## Source de données TAXREF (indépendance iNaturalist)

- [x] **Référentiel TAXREF local** (`internal/adapters/taxref`, importeur natif `cmd/importtaxref`, 212k espèces). `SPECIES_SOURCE=taxref` (auto-détecté dès qu'une base est chargée).
- [x] **Photos « à nous »** (table `taxref_photos` liée par `cd_nom`).
- [x] **Back-office admin des photos** (`/admin`, auth rôle/bcrypt/HMAC, upload local ou S3/MinIO).
- [x] **Import en masse des photos** : `cmd/importphotos` (CSV `photo;groupe;nom_scientifique`, résolution cd_nom, conversion RAW).
- [x] **SRI sur les CDN** : `integrity`/`crossorigin` sur Alpine.js (index + admin).
- [x] **Nettoyage fichiers orphelins** : la suppression d'une photo retire aussi le fichier stocké.
- [x] **Conversion RAW → JPEG** (`internal/media`, aperçu embarqué, Go pur).
- [x] **Présence métropole** : colonne `FR` du TAXREF natif ; les distracteurs préfèrent les espèces présentes en métropole.

## Multijoueur

- [x] **Salons de duel** : création + code à 4 lettres, jointure, hôte qui lance (`internal/application/room`).
- [x] **Manches synchronisées** : tout le monde répond à la même question, avance automatique quand tous ont répondu, podium commun.
- [x] **Modes** : Classique et **Élimination** (mort subite).
- [x] **Temps réel WebSocket** (`coder/websocket`) + repli polling + **reconnexion** (reprise lobby/partie après rechargement).
- [x] **Sécurité** : jeton secret par joueur (anti-usurpation), timing serveur, nombre de questions plafonné.
- [ ] **Évolutions** : mode spectateur, salons publics/liste, keep-alive WebSocket mobile.

## Apprentissage & annotation (nouveau)

- [x] **Section apprentissage / articles** : rôle **rédacteur** (`writer`, promu depuis l'admin) écrivant des articles liés aux espèces ; **Bibliothèque** publique + **lien « En savoir plus »** sur la bonne réponse en quiz. *(fait)*
- [x] **Zones sur les images (multi-espèces)** : éditeur glisser-tracer dans l'admin pour associer une espèce à une zone (X ici, Y là). *(fait)*
- [x] **Zone de zoom (mode Détail)** : zone par photo utilisée pour le gros plan, à la place du recadrage aléatoire. *(fait)*
- [x] **Exploiter les zones espèces en jeu** : exercice « 📍 Où est l'espèce ? » (cliquer la bonne zone), validé côté serveur. *(fait)*

## P2 — Améliorations

- [x] **Quiz Son** — mode Chant sur enregistrements possédés (upload admin). *(fait)*
- [x] **Formats de jeu** : **Chrono** (90 s) et **Survie** (1 erreur). *(fait)*
- [x] **Mode révision** : revoir les espèces ratées (`player_misses`, décroissance au succès). *(fait)*
- [x] **PWA** : manifest + service worker, appli installable + shell hors-ligne. *(fait)*
- [ ] **Récupération de compte** : mot de passe oublié (nécessite e-mail).
- [x] **Mode famille** : épreuve « Famille » — trouver la famille de l'espèce photographiée (choix = familles). *(fait)*
- [ ] **Filtres lieu & saison** — **bloqué par les données** : TAXREF ne contient ni phénologie (saison) ni occurrence fine par région/département (seulement des codes de présence grossiers `fr`). Nécessite un jeu de données externe (occurrences INPN, phénologie) avant d'être faisable.
- [ ] **Récupération de compte** : mot de passe oublié (nécessite e-mail).
- [ ] **Migration API v2 iNaturalist** (pertinent seulement si on garde iNaturalist en appoint).
- [ ] **Tests E2E frontend** (Playwright) intégrés à la CI.
- [ ] **i18n** : déjà des noms français via TAXREF ; généraliser le fallback.

## P3 — Déploiement

- [x] **Conteneurisation** : Dockerfile multi-stage + `docker-compose.yml` (+ profil `minio`).
- [~] **Hébergement naturieux.fr** : outils prêts — reverse proxy **Caddy** (TLS auto), profil compose `tls`, en-têtes de sécurité, guide `05-mise-en-ligne.md`. Reste le déploiement effectif (géré par l'utilisateur sur son serveur).
- [x] **Page mentions légales / confidentialité** : page `/legal` (RGPD, licences, crédits). *(champs `[à compléter]` à remplir avant ouverture)*
- [x] **Monitoring** : endpoint `/metrics` (Prometheus) — requêtes totales/erreurs/par classe. *(fait)*

## Dette technique

- [x] Store de sessions hors du handler (service + repo SQLite).
- [x] Repo joueurs en mémoire supprimé (SQLite).
- [x] **`.gitattributes`** : fins de ligne harmonisées (LF). *(fait)*
- [ ] Couverture handlers HTTP : tester davantage les chemins succès avec service mocké.
- [ ] `golangci-lint` local : ne tourne pas (binaire go1.24 < cible 1.25) ; la CI l'exécute en `latest`.
