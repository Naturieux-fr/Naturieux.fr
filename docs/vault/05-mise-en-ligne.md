# 05 — Mise en ligne (HTTPS)

Guide pour publier Naturieux sur Internet avec un certificat TLS automatique.
Le serveur Go reste en HTTP ; un reverse proxy **Caddy** termine le TLS et
obtient/renouvelle le certificat Let's Encrypt tout seul.

## Prérequis

- Un nom de domaine (ex. `naturieux.fr`) dont l'enregistrement **A/AAAA**
  pointe vers l'IP publique du serveur.
- Les ports **80** et **443** ouverts et redirigés vers la machine.
- Docker + Docker Compose installés.

## Lancement

```bash
# Secret de session (obligatoire en prod — sinon les jetons changent à chaque
# redémarrage et tout le monde est déconnecté). Générer une valeur stable :
export AUTH_SECRET="$(openssl rand -hex 32)"

# Compte administrateur initial (créé au premier démarrage)
export ADMIN_USERNAME="admin"
export ADMIN_PASSWORD="…un mot de passe fort…"

# Domaine pour Caddy
export NATURIEUX_DOMAIN="naturieux.fr"

# Inscription : "open" (tout le monde) ou "invite" (sur lien admin)
export REGISTRATION_MODE="open"

docker compose --profile tls up -d --build
```

Caddy obtient le certificat au premier accès HTTPS sur le domaine. Le site est
alors disponible sur `https://naturieux.fr` (le HTTP est redirigé vers HTTPS).

> Sans vrai domaine, `NATURIEUX_DOMAIN=localhost` fait émettre un certificat
> auto-signé par Caddy pour tester la chaîne TLS en local.

## Variables utiles

| Variable | Rôle |
|----------|------|
| `AUTH_SECRET` | Clé de signature des sessions. **À fixer** en prod. |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | Compte admin initial. |
| `REGISTRATION_MODE` | `open` ou `invite`. |
| `NATURIEUX_DOMAIN` | Domaine servi par Caddy. |
| `SPECIES_SOURCE` | Vide = auto (TAXREF si importé, sinon iNaturalist). |
| `STORAGE` | `local` (volume `/data`) ou `s3` (MinIO/S3, voir compose). |

## En-têtes de sécurité

L'application ajoute `X-Content-Type-Options`, `X-Frame-Options`,
`Referrer-Policy`, et `Strict-Transport-Security` (HSTS) dès que la requête
arrive en HTTPS (directement ou via `X-Forwarded-Proto` que Caddy positionne).

## Conformité (RGPD)

La page **/legal** présente les mentions légales et la politique de
confidentialité. **Avant la mise en ligne publique**, compléter les champs
`[à compléter]` de `web/legal.html` : éditeur, contact e-mail, hébergeur.

## Sauvegardes

Tout l'état (base SQLite + photos locales) est dans le volume Docker
`naturieux-data` (`/data`). Sauvegarder ce volume régulièrement :

```bash
docker run --rm -v naturieux_naturieux-data:/data -v "$PWD":/backup alpine \
  tar czf /backup/naturieux-backup.tar.gz -C /data .
```
