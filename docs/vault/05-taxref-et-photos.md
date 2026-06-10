# 05 — TAXREF, base de photos et choix de stockage

*Étude de faisabilité, 2026-06-10. Voir aussi [03 — Faisabilité](03-faisabilite.md).*

## TAXREF (référentiel taxonomique national)

- **Version** : TAXREF v18.0 (publiée le 2025-01-09) par UAR PatriNat (OFB-MNHN-CNRS-IRD).
- **Source retenue : archive Darwin Core sur GBIF** — `https://ipt.gbif.fr/archive.do?r=taxref` (`dwca-taxref-v4.17.zip`, ~32 Mo). Licence **CC-BY 4.0**, DOI `10.15468/vqueam`. Préférée à l'INPN car sa licence ouverte est sans ambiguïté (le formulaire INPN ajoute une clause « pas de mise en ligne sans autorisation »).
- Alternative : INPN (`inpn.mnhn.fr/telechargement/referentielEspece/taxref/18.0/menu`, fichier `TAXREFv18.txt`) et data.gouv.fr (Licence Ouverte/Etalab). L'API `taxref.mnhn.fr/api` existe (HAL/JSON) mais le **bulk est préférable** pour charger ~700k lignes ; réserver l'API aux compléments.

### Structure de l'archive (vérifiée)
- `taxon.txt` : **708 685 lignes** (188 Mo), TSV tabulé, UTF-8, en-tête Darwin Core.
  Colonnes utiles : `taxonID`, `acceptedNameUsageID` (synonymes → nom valide), `parentNameUsageID` (**pointeur parent = hiérarchie**), `scientificName`, `kingdom`/`phylum`/`class`/`order`/`family`/`genus` (rangs dénormalisés en texte), `taxonRank`, `scientificNameAuthorship`, `vernacularName` (noms FR, ex. « Renard roux, Renard, Goupil »).
- `vernacularname.txt` : 82 967 lignes (noms vernaculaires détaillés, multilingues).
- Répartition des rangs : **542 384 species**, 57 827 genus, 33 678 subspecies, 7 882 family…
- Filtrage pour le quiz : `taxonID == acceptedNameUsageID` (noms valides), `taxonRank == species`, segmenter par `class`/`order`/`family` ou par groupe grand public.

## Le problème des photos

**TAXREF ne contient aucune photo** — uniquement taxonomie + noms. Les photos INPN ont des **licences par cliché** (pas toutes réutilisables), donc pas de source libre clé en main.

Conséquence : « se séparer d'iNaturalist » pour les photos demande quand même une source d'images CC. Options (du plus léger au plus lourd) :

1. **Cache d'URLs** (déjà en place) : on garde iNaturalist/GBIF/Wikimedia comme source mais on met en cache les URLs + attribution dans notre SQLite. Léger, pas d'hébergement, mais dépendance au CDN externe subsiste.
2. **Base de photos « à nous » par pré-récupération** : pipeline qui, pour chaque taxon TAXREF, va chercher des photos CC (iNaturalist `photo_license=cc0,cc-by,cc-by-nc`, GBIF media, Wikimedia Commons), stocke URL + auteur + licence. Indépendance des requêtes au runtime, mais les fichiers restent hébergés ailleurs.
3. **Hébergement réel des fichiers** : on télécharge et stocke les images (disque/objet). Vraie indépendance, mais coût stockage + bande passante, et il faut **toujours** sourcer les images depuis des dépôts CC (on ne crée pas de photos ex nihilo).

Dans tous les cas, l'attribution par image reste obligatoire (cf. [03](03-faisabilite.md)).

## Base graphe (Neo4j) vs relationnel — tranché : **relationnel**

La taxonomie TAXREF est un **arbre à parent unique** (`parentNameUsageID`), profondeur ~7-30 niveaux. C'est le cas idéal du relationnel : **SQLite + CTE récursive** (`WITH RECURSIVE`) pour ancêtres/descendants, éventuellement une *closure table* ou colonne `path` matérialisée pour accélérer les sous-arbres.

Neo4j n'est justifié que pour un graphe **dense et multi-relationnel** : parents multiples, traversées profondes variables, relations porteuses de données, pathfinding complexe (réseaux trophiques, interactions inter-espèces). Pour un quiz (taxon → ancêtres, frères, filtres par groupe), **introduire Neo4j ajoute de l'infra sans bénéfice**. On reste sur SQLite, déjà embarqué dans le binaire Go.

## Recommandations

1. **Stockage TAXREF** : table SQLite `taxref(cd_nom PK, cd_ref, cd_taxsup, rang, scientific_name, vernacular_name, class, order, family, genus, …)`, index sur `cd_ref`, `cd_taxsup`, `rang`. Hiérarchie via CTE récursive. **Pas de Neo4j.**
2. **Import robuste** : lire l'en-tête pour mapper les colonnes par nom (jamais par index), valider UTF-8 + tabulation, importer en une transaction, stocker la version TAXREF en métadonnée pour les mises à jour annuelles.
3. **Quiz** : filtrer noms valides + rang species + (à terme) présence métropole ; segmenter par groupe taxonomique.
4. **Photos** : étape à décider (cf. 3 options ci-dessus). Quelle que soit l'option, sourcer en CC et conserver l'attribution par image.
5. **Citation** : afficher « Gargominy O. (2025). TAXREF v18. PatriNat (OFB-MNHN-CNRS-IRD). DOI 10.15468/vqueam » dans la page Crédits.
