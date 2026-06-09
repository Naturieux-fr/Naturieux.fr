# 02 — Reste à faire

*Roadmap priorisée. Mise à jour : 2026-06-09. Voir [01 — État du projet](01-etat-du-projet.md) pour le contexte.*

## P0 — Critique (conformité et fondations)

- [x] **Filtrage des licences photos** : `photo_license=cc0,cc-by,cc-by-nc` sur les requêtes observations + filtre défensif côté client (photo par défaut du taxon exclue de l'affichage). Reste à faire pour les **sons** (`sound_license`) quand le SoundQuiz arrivera.
- [x] **Attribution des médias** : `license_code` + `attribution` portés jusqu'à l'API (`media_attribution`, `media_license`) et affichés sous l'image du quiz. Amélioration possible : lien vers l'observation d'origine.
- [x] **Persistance SQLite** : `internal/adapters/sqlite` (driver pur Go `modernc.org/sqlite`), sessions stockées en snapshot JSON + colonnes dénormalisées pour les stats, joueurs persistés (XP, niveaux, achievements). Le store en mémoire du handler est supprimé : les parties survivent à un redémarrage du serveur. Variable `DB_PATH` (défaut `naturieux.db`).
- [x] **Cache local des espèces** : `internal/adapters/cache` (décorateur SQLite autour du client iNaturalist). Préchauffage ~60 espèces/taxon au démarrage + toutes les 12 h (1 requête/taxon), TTL 7 jours, dégradation gracieuse vers l'API. Démarrage d'une partie : ~8-10 s → < 25 ms. Reste possible : back-off explicite sur 429 côté client.

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

- [x] Store de sessions déplacé hors du handler HTTP (service + repository SQLite).
- [x] Repo joueurs en mémoire de `cmd/server` supprimé (remplacé par SQLite).
- [ ] Couverture handlers HTTP < 60 % : tester les chemins succès avec un service mocké (injecter une interface au lieu de `*appquiz.Service`).
- [ ] Harmoniser fins de ligne (warnings LF/CRLF sous Windows) via `.gitattributes`.
