# Documentation API iNaturalist

## Sources
- Documentation officielle: https://api.inaturalist.org/v1/docs/
- Pratiques recommandees: https://www.inaturalist.org/pages/api+recommended+practices

## Rate Limits
- **Frequence**: ~1 requete par seconde
- **Limite journaliere**: ~10,000 requetes/jour
- **Media**: Max 5 GB/heure ou 24 GB/jour
- **Pagination**: Max 10,000 resultats totaux

## Endpoints Principaux

### GET /observations
Recherche d'observations avec photos.

**Parametres utiles:**
- `taxon_id`: Filtrer par ID taxonomique
- `photos=true`: Seulement les observations avec photos
- `quality_grade=research`: Donnees de qualite recherche
- `place_id`: Filtrer par lieu geographique
- `per_page`: Jusqu'a 200 resultats par requete
- `page`: Pagination
- `order_by`: Tri (created_at, observed_on, etc.)
- `identified=true`: Seulement les observations identifiees

**Exemple:**
```
GET https://api.inaturalist.org/v1/observations?taxon_id=3&photos=true&quality_grade=research&per_page=50
```

### GET /taxa
Recherche de taxons (especes).

**Parametres:**
- `q`: Recherche textuelle
- `id`: ID specifique du taxon
- `rank`: Rang taxonomique (species, genus, family, etc.)
- `is_active=true`: Taxons actifs seulement
- `per_page`: Max 30 resultats

**Exemple:**
```
GET https://api.inaturalist.org/v1/taxa?q=vulpes&rank=species
```

### GET /taxa/autocomplete
Autocompletion pour la recherche de taxons.

**Exemple:**
```
GET https://api.inaturalist.org/v1/taxa/autocomplete?q=renard
```

## Structure des Reponses

### Observation
```json
{
  "id": 123456,
  "species_guess": "Red Fox",
  "taxon": {
    "id": 42069,
    "name": "Vulpes vulpes",
    "preferred_common_name": "Red Fox",
    "iconic_taxon_name": "Mammalia",
    "default_photo": {
      "medium_url": "https://...",
      "square_url": "https://..."
    }
  },
  "photos": [
    {
      "id": 789,
      "url": "https://static.inaturalist.org/photos/789/medium.jpg",
      "medium_url": "https://...",
      "large_url": "https://...",
      "original_url": "https://..."
    }
  ],
  "location": "48.8566,2.3522",
  "place_guess": "Paris, France"
}
```

### Taxon
```json
{
  "id": 42069,
  "name": "Vulpes vulpes",
  "rank": "species",
  "preferred_common_name": "Red Fox",
  "iconic_taxon_name": "Mammalia",
  "ancestor_ids": [1, 2, 3, ...],
  "default_photo": {
    "medium_url": "https://..."
  }
}
```

## URLs des Photos

### Domaines
- `static.inaturalist.org` - Licences restreintes
- `inaturalist-open-data.s3.amazonaws.com` - Licences ouvertes

### Tailles disponibles
- `original` - Taille originale
- `large` - Grande (1024px)
- `medium` - Moyenne (500px)
- `small` - Petite (240px)
- `thumb` - Miniature (100px)
- `square` - Carree (75x75px)

### Modification des URLs
Remplacer la taille dans l'URL:
```
https://static.inaturalist.org/photos/123/medium.jpg
-> https://static.inaturalist.org/photos/123/large.jpg
```

## Headers Recommandes
```
User-Agent: Naturieux/1.0 (contact@naturieux.fr)
Accept: application/json
```

## Groupes Taxonomiques pour le Quiz

### Iconic Taxa (groupes principaux)
- Mammalia (Mammiferes)
- Aves (Oiseaux)
- Reptilia (Reptiles)
- Amphibia (Amphibiens)
- Actinopterygii (Poissons)
- Insecta (Insectes)
- Arachnida (Araignees)
- Mollusca (Mollusques)
- Plantae (Plantes)
- Fungi (Champignons)

## Requetes Utiles pour le Quiz

### Obtenir des especes aleatoires avec photos
```
GET /observations?
  photos=true&
  quality_grade=research&
  iconic_taxa=Mammalia&
  place_id=6753&  # France
  per_page=200&
  order_by=random
```

### Obtenir des especes similaires (pour les mauvaises reponses)
```
GET /taxa?
  taxon_id=PARENT_ID&
  rank=species&
  per_page=10
```
