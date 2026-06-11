# 06 — Import lieu & saison (occurrences)

Les filtres **lieu (région)** et **saison (mois)** s'appuient sur un jeu de
données d'occurrences **importé une seule fois** — comme TAXREF. **Aucun appel
à une API en fonctionnement.**

## 1. Obtenir le fichier (une fois)

Deux sources possibles (licences ouvertes) :

- **GBIF** — https://www.gbif.org/occurrence/search : filtrer **Country = France**,
  puis *Download* → format **« Simple »** (CSV tabulé). Dézipper l'archive.
- **INPN / OpenObs** — https://openobs.mnhn.fr : export par territoire (clé CD_REF).

Le fichier est volumineux (plusieurs Go) mais on ne le traite qu'une fois.

## 2. Importer (agrégation locale)

TAXREF doit déjà être importé. Puis :

```bash
go run ./cmd/importoccurrences -file path/to/occurrence.txt -db naturieux.db
# ou le binaire : ./importoccurrences -file occurrence.txt -db naturieux.db
```

L'outil lit le fichier tabulé (colonnes repérées par leur nom : `species`/
`scientificName`, `month`/`mois`, `level1Name`/`stateProvince`, `countryCode`),
agrège par espèce les **mois** et **régions** où elle est observée, et remplit
`species_months` / `species_regions`. Les espèces sont reliées à TAXREF par leur
nom scientifique. Un couple (espèce, mois) ou (espèce, région) n'est gardé qu'à
partir de quelques observations, pour écarter le bruit.

## 3. Utilisation

Une fois importé, l'accueil affiche les sélecteurs **Lieu** et **Saison** (cachés
tant qu'aucune donnée n'est importée). Le quiz ne propose alors que des espèces
observées dans la région et/ou au mois choisis (et qui ont une photo). Réimporter
remplace les données précédentes.
