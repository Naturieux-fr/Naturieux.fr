# 01 — État du projet

*Dernière mise à jour : 2026-06-09*

## Résumé

Quiz naturaliste gamifié en Go 1.22, architecture hexagonale (ports & adapters) + DDD.
Backend fonctionnel, frontend Alpine.js livré, données espèces via l'API iNaturalist (v1).

## Ce qui est fait

### Backend
- **Domaine** : `species` (Espèce/Taxon), `quiz` (Question, Session, types et difficultés), `gamification` (Joueur, XP, niveaux, achievements, streaks).
- **Application** : `quiz.Service` (start/answer/abandon), `QuestionFactory` (génération de questions avec distracteurs).
- **Adapters** :
  - `inaturalist` : client API v1 (observations avec photos, taxons, filtre lieu France `place_id=6753`).
  - `http` : handlers REST (`/api/v1/quiz/start`, `/answer`, `/abandon`, `/api/v1/config`, `/health`).
  - `mock` : dépôt d'espèces en mémoire pour le mode dev (`DEV_MODE=1`), évite les appels API.
- **Serveur** : graceful shutdown, CORS dev, fichiers statiques, timeouts adaptés.

### Frontend (`web/`)
- `index.html` : écrans accueil / quiz / résultats.
- `static/js/app.js` : composant Alpine.js (logique de partie, timer, feedback).
- `static/css/style.css` : thème sombre responsive.

### Qualité / CI
- Workflows GitHub Actions : CI, quality-gate (type SonarQube), security.
- Couverture tests : species 100 %, inaturalist ~93 %, application/quiz ~82 %, gamification ~81 %, domaine quiz ~77 %, handlers HTTP ~58 %.

## Limites connues (état actuel)

- **Aucune persistance** : sessions et joueurs en mémoire (map), tout est perdu au redémarrage.
- **Sessions stockées dans le handler HTTP** (`Handler.sessions`) — pas de `SessionRepository` branché (passé `nil` au service).
- **Pas de filtrage de licence** sur les photos iNaturalist ni d'affichage d'attribution — point juridique critique, voir [03 — Faisabilité](03-faisabilite.md).
- Quiz types Flash/Partial/Silhouette/Sound définis dans le domaine mais l'expérience frontend n'existe que pour Image (et Flash partiellement).
- Joueur unique « demo » créé au démarrage, pas de comptes utilisateurs.
- API iNaturalist appelée en direct à chaque partie (pas de cache local), v1 utilisée alors que la v2 est recommandée.

## Liens

- [02 — Reste à faire](02-reste-a-faire.md)
- [03 — Faisabilité](03-faisabilite.md)
- [Architecture](../architecture.md)
