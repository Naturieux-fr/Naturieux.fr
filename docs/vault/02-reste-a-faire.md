# 02 — Reste à faire

*Roadmap priorisée. Mise à jour : 2026-06-09. Voir [01 — État du projet](01-etat-du-projet.md) pour le contexte.*

## P0 — Critique (conformité et fondations)

- [x] **Filtrage des licences photos** : `photo_license=cc0,cc-by,cc-by-nc` sur les requêtes observations + filtre défensif côté client (photo par défaut du taxon exclue de l'affichage). Reste à faire pour les **sons** (`sound_license`) quand le SoundQuiz arrivera.
- [x] **Attribution des médias** : `license_code` + `attribution` portés jusqu'à l'API (`media_attribution`, `media_license`) et affichés sous l'image du quiz. Amélioration possible : lien vers l'observation d'origine.
- [ ] **Persistance SQLite** : implémenter `ports.QuizSessionRepository` et `ports.PlayerRepository` sur SQLite (`modernc.org/sqlite` pour rester sans CGO), retirer le store en mémoire du handler HTTP.
- [ ] **Cache local des espèces** : pré-charger un pool de taxons/photos (métadonnées seulement, jamais les fichiers) rafraîchi périodiquement, au lieu d'appeler l'API à chaque partie. Respecter ≤ 60 req/min, back-off sur 429.

## P1 — Important (expérience de jeu)

- [ ] **Comptes utilisateurs simples** : pseudo + identifiant local (pas d'OAuth dans un premier temps), remplacement du joueur « demo ».
- [ ] **Leaderboard** : classement global et par taxon, endpoint `GET /api/v1/leaderboard`.
- [ ] **Quiz types restants côté frontend** :
  - Flash : déjà côté API (durée d'affichage), finaliser l'UI.
  - Partial : recadrage CSS (zoom sur une zone de la photo).
  - Silhouette : filtre CSS (`brightness(0)` sur fond clair) — à valider visuellement.
  - Sound : nécessite `sounds=true` côté client iNaturalist + lecteur audio ; compléter avec Xeno-canto pour les oiseaux (clé API requise depuis oct. 2025).
- [ ] **Filtres avancés** : choix du groupe taxonomique (oiseaux, mammifères, plantes, champignons…), du lieu (région/département), de la saison.
- [ ] **Affichage des achievements et niveaux** dans l'UI (le domaine gamification existe déjà).

## P2 — Améliorations

- [ ] **Migration API v2 iNaturalist** : sélection de champs (`fields`), UUID — la v1 fonctionne mais la v2 est la version pérenne.
- [ ] **Tests E2E frontend** automatisés (Playwright) intégrés à la CI.
- [ ] **WebSocket** : mode duel / multijoueur en temps réel.
- [ ] **Mode révision** : revoir les espèces ratées (répétition espacée).
- [ ] **i18n** : noms vernaculaires français via `locale=fr` (paramètre API), fallback nom scientifique.
- [ ] **PWA** : manifest + service worker pour usage mobile/terrain.

## P3 — Déploiement

- [ ] **Conteneurisation** : Dockerfile multi-stage (binaire statique Go).
- [ ] **Hébergement naturieux.fr** : reverse proxy + TLS, variable `PORT`, logs.
- [ ] **Page mentions légales** : crédits iNaturalist, licences des médias, politique de confidentialité.
- [ ] **Monitoring** : métriques basiques (sessions/jour, taux d'erreur API iNaturalist).

## Dette technique

- [ ] Déplacer le store de sessions hors du handler HTTP (vers le service + repository).
- [ ] `cmd/server` : extraire `newInMemoryPlayerRepository` dans `internal/adapters/memory`.
- [ ] Couverture handlers HTTP < 60 % : tester les chemins succès avec un service mocké (injecter une interface au lieu de `*appquiz.Service`).
- [ ] Harmoniser fins de ligne (warnings LF/CRLF sous Windows) via `.gitattributes`.
