# 03 — Étude de faisabilité

*Quiz d'identification d'espèces basé sur l'API iNaturalist. Sources officielles iNaturalist, GBIF, Wikimedia, Xeno-canto. Date : 2026-06-09.*

## 1. Limites de débit (rate limits) et authentification

**Limites de débit** (source : « API Recommended Practices ») :
- Plafond technique : **100 requêtes/minute**, mais iNaturalist demande de **rester sous 60 req/min** (~1 req/s).
- Plafond journalier recommandé : **~10 000 requêtes/jour**.
- Dépassement = **HTTP 429** ; prévoir délais et back-off.
- **Médias** (photos/sons) : ne pas dépasser **5 Go/heure** ni **24 Go/jour**, sous peine de **blocage permanent**.
- Une seule IP ; multiplier les IP pour contourner les limites peut entraîner un blocage.

**Authentification** :
- Les endpoints publics en lecture (`GET /v1/taxa`, `GET /v1/observations`) **ne nécessitent pas de token**.
- iNaturalist recommande de **n'authentifier que si nécessaire** : les requêtes authentifiées ne sont pas mises en cache et coûtent plus cher côté serveur. Pour un quiz public : requêtes anonymes.

**Bonnes pratiques** : filtres précis, `per_page` élevé (jusqu'à 200), **User-Agent personnalisé** identifiant l'application.

Sources :
- https://www.inaturalist.org/pages/api+recommended+practices
- https://forum.inaturalist.org/t/429-too-many-requests/19907

## 2. API v2 vs v1

- **L'API v2 existe** et est la version recommandée (https://api.inaturalist.org/v2/docs/) — utilisée par le site et les applis mobiles iNaturalist, donc activement maintenue.
- v0 dépréciée ; **v1 fonctionnelle et stable** mais destinée à être remplacée par la v2.
- Différences clés v2 : champs à retourner **explicites** (`fields`) → réponses plus légères ; **UUID** plutôt qu'ID numériques ; pas de réimplémentation des endpoints OAuth.

**Recommandation** : viser la **v2** pour les nouveaux développements ; la v1 reste un repli acceptable à court terme.

Sources :
- https://forum.inaturalist.org/t/is-v2-the-official-api-now/77258
- https://forum.inaturalist.org/t/which-version-of-the-api-should-apps-use/79919

## 3. Licences photos, affichage légal, attribution, filtrage

**Licences présentes** : CC0, CC BY, CC BY-SA, CC BY-NC, CC BY-ND, CC BY-NC-SA, CC BY-NC-ND, **ou « tous droits réservés »**. Licence par défaut : **CC BY-NC**.

**Affichage légal dans un quiz public gratuit : oui**, à condition de :
1. **N'utiliser que les photos sous licence CC** (exclure « all rights reserved »).
2. **Fournir l'attribution** : auteur, licence (ex. « CC BY-NC »), lien vers la licence et vers l'observation d'origine.
3. Rester **non commercial** avec du CC BY-NC ; pour garder une option commerciale : se limiter à CC0 / CC BY / CC BY-SA.

**Filtrage par licence via l'API : oui.** Paramètres `photo_license`, `license`, `sound_license` (ex. `photo_license=cc0,cc-by,cc-by-nc`). L'objet photo expose `license_code` et `attribution` — à stocker et afficher.

Sources :
- https://help.inaturalist.org/en/support/solutions/articles/151000169918-can-i-use-the-photos-and-sounds-that-are-posted-on-inaturalist-
- https://www.inaturalist.org/pages/search+urls

## 4. Enregistrements sonores

- `GET /v1/observations?sounds=true` filtre les observations avec son ; l'objet `sounds` fournit URL, format et licence.
- Formats : WAV, MP3, M4A.
- Licences : mêmes règles que les photos, **licence sonore indépendante**, filtrable via `sound_license`.
- Volume de sons plus limité que les photos → compléter avec Xeno-canto (oiseaux), cf. §6.

Sources :
- https://help.inaturalist.org/en/support/solutions/articles/151000169939-how-do-i-add-sounds-
- https://www.inaturalist.org/blog/99871-1-000-000-observations-with-sounds-on-inaturalist

## 5. Conditions d'utilisation de l'API

- **Usage commercial** : le contenu CC BY-NC ne peut pas servir à un usage commercial. Pour du commercial : CC0 / CC BY / CC BY-SA uniquement, ou accord des auteurs.
- **Cache** : pratique acceptée et encouragée (les réponses taxa/lieux sont déjà servies avec un cache long). Voir §7.
- **Exigences** : User-Agent identifiant l'app ; attribution des médias ; API destinée aux applications, pas au scraping massif ; iNaturalist peut bloquer sans préavis en cas d'abus.

Sources :
- https://www.inaturalist.org/pages/terms
- https://forum.inaturalist.org/t/is-this-use-of-inaturalist-content-a-breach-of-the-commercial-purpose-terms/9841

## 6. Alternatives / compléments

- **GBIF** (https://techdocs.gbif.org/en/openapi/) : agrégateur mondial (inclut les données research-grade iNaturalist, export hebdomadaire). Licences par jeu de données : CC0, CC BY, CC BY-NC. Rate-limiting variable (429) ; gros volumes via l'API de download. Bon pour les métadonnées, moins pour les images.
- **Wikimedia Commons** : licences libres (CC BY-SA, CC BY, domaine public). **User-Agent identifiant obligatoire** (contact e-mail/URL) ; limites de débit renforcées en 2026. Crédit : auteur + licence + lien source + modifications. Qualité/cadrage hétérogènes.
- **Xeno-canto** (chants d'oiseaux) : quasi tout en Creative Commons. **1000 req/heure** ; **clé API requise depuis le 10 octobre 2025** pour télécharger. Excellent pour un quiz sonore ornithologique.

## 7. Cache local vs appels live

- iNaturalist **recommande le cache** : appeler l'API à chaque action « ajoute une charge inutile aux serveurs ».
- Position des développeurs (forum) : **cache serveur rafraîchi périodiquement** plutôt qu'appels live ; l'API sert à « récupérer de petits/moyens lots, pas au bulk download ». Gros volumes → exports ou dataset GBIF hebdomadaire.

**Architecture conseillée pour Naturieux.fr** :
1. Pré-construire un référentiel local de taxons + pool de métadonnées photos/sons (URL, `license_code`, `attribution`) filtrés par licence.
2. **Ne jamais ré-héberger** les médias : pointer vers les URL iNaturalist, en respectant les limites de bande passante.
3. Rafraîchir le cache périodiquement (cron), pas à chaque partie.
4. Afficher systématiquement auteur + licence avec chaque média.

Sources :
- https://forum.inaturalist.org/t/are-you-allowed-to-store-inaturalist-data-from-the-api-on-your-own-server/75969

---

## Verdict de faisabilité

**FAISABLE** — Un quiz public et gratuit d'identification d'espèces sur l'API iNaturalist est réalisable et conforme, sous conditions.

**Conditions impératives** :
1. **Filtrer les médias par licence CC** (`photo_license` / `sound_license`), exclure « tous droits réservés ».
2. **Attribution complète** sur chaque média (auteur, licence, lien). Non négociable.
3. **Modèle non commercial** tant que du CC BY-NC est utilisé.
4. **Respect des quotas** (≤ 60 req/min, ~10 000 req/jour ; médias ≤ 5 Go/h et 24 Go/j), back-off sur 429.
5. **User-Agent identifiant**, requêtes anonymes, **cache local rafraîchi périodiquement**.
6. Cibler l'**API v2** à terme.

**Risques** :
- Juridique si le filtrage de licence ou l'attribution est mal implémenté — principal point de défaillance à sécuriser en premier (cf. P0 dans [02 — Reste à faire](02-reste-a-faire.md)).
- Blocage en cas de dépassement de quotas ou de ré-hébergement massif.
- Évolution des CGU tierces (Xeno-canto : clé API ; Wikimedia : durcissement 2026) — à surveiller.
- Dépendance externe : prévoir une dégradation gracieuse si l'API est indisponible (d'où le cache local et le mode mock).

**Note** : revérifier les chiffres exacts de quotas et les CGU sur les pages officielles avant mise en production, ces valeurs pouvant évoluer.
