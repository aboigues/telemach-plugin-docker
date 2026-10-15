# Contribuer à TELEMACH PLUGIN DOCKER

Merci de ton intérêt ! Ce projet est un outil pédagogique : la qualité des explications compte
autant que celle du code. Les règles ci-dessous découlent de la
[constitution](.specify/memory/constitution.md).

## Avant de commencer

- Relis [`LESSON-LEARNED.md`](LESSON-LEARNED.md) : chaque incident passé y laisse une règle
  préventive à appliquer.
- Toute nouvelle fonctionnalité passe par Spec Kit (`/speckit-specify` → `/speckit-plan` →
  `/speckit-tasks` → `/speckit-implement`).

## Langue

- **En français** : spécifications, documentation, commentaires du code, messages affichés par
  le plugin, catalogue d'erreurs, `LESSON-LEARNED.md`. Accents obligatoires.
- **En anglais** : le code (identifiants, noms de fichiers de code), les noms de branches, les
  clés de configuration.
- Les termes et commandes Docker restent dans leur forme d'origine.

## Sécurité

- Ne versionne jamais de secret. Les seuls faux secrets autorisés sont dans `testdata/secrets/`.
- Toute nouvelle dépendance est justifiée dans le plan de la feature et épinglée sur la dernière
  version stable vérifiée.
- Le plugin n'exécute aucune commande et ne modifie aucun fichier de l'apprenant.

## Gates et fusion

La branche `main` est protégée :

- pas de push direct, pas de force-push, pas de suppression ;
- fusion uniquement par pull request, historique linéaire, conversations résolues ;
- aucune exception pour les administrateurs (`enforce_admins`) ;
- checks obligatoires : `lint`, `test`, `vuln`, `secrets`, `dependency-review`,
  `dockerfile-lint`, `goreleaser-check`, `image-scan`, `build`, `fuzz`, `integration`.

**Prérequis du dépôt** : le Dependency graph et les alertes Dependabot doivent être activés
(Settings › Code security), sans quoi la gate `dependency-review` échoue (voir LL-005).

**Workflow de fusion** : Claude Code commit, pousse et ouvre la pull request ; le mainteneur
relit la PR puis la fusionne lui-même. Ce clic sur « Merge » vaut validation humaine : aucune
approbation formelle n'est exigée, GitHub interdisant d'approuver sa propre PR.

## Publier une version

La publication est automatisée par `.github/workflows/release.yml` : elle produit la Release
GitHub (quatre binaires, sommes signées, SBOM, provenance) et l'image Docker Hub
`telemachlearning/telemach` (signée, multi-architecture). Voir
`specs/002-distribution/` pour le détail.

### Prérequis (une seule fois, par le mainteneur — LL-005)

- **Docker Hub** : dépôt public `telemachlearning/telemach` ; jeton d'accès personnel
  « Read & Write », limité à ce dépôt si le compte le permet. **Noter sa date d'expiration**
  et le renouveler avant.
- **GitHub, environnement `release`** (Settings › Environments) : déploiement réservé aux
  tags `v*` ; variable `DOCKERHUB_USERNAME=telemachlearning` ; secret `DOCKERHUB_TOKEN`
  (le jeton ci-dessus). Le jeton n'existe que dans cet environnement : aucun autre workflow,
  ni aucune branche, ne peut le lire.
- **Releases immuables** activées (Settings › General › Releases) : une Release publiée ne
  peut plus être modifiée et son tag ne peut plus être déplacé.

Claude Code ne manipule jamais le jeton Docker Hub.

### Procédure

1. Fusionner sur `main` tout ce qui doit être publié (gates vertes).
2. Poser un tag annoté sur `main` et le pousser :

   ```bash
   git checkout main && git pull
   git tag -a vX.Y.Z -m "TELEMACH X.Y.Z — résumé"
   git push origin vX.Y.Z
   ```

3. Suivre l'exécution (`gh run watch`). Le workflow vérifie le tag (format, commit de `main`,
   Release et image absentes), rejoue toutes les gates, puis publie.
4. Vérifier la Release et l'image (quickstart de la feature 002, V1 à V5).

Une **pré-version** (`vX.Y.Z-rc.N`) est publiée comme pré-version : elle ne déplace ni la
« dernière version » des Releases, ni les tags `X.Y` et `latest` de l'image.

### En cas d'échec

- **Avant la publication** (vérifications, gates, construction) : rien n'est publié, ou il
  reste un brouillon de Release. Supprimer le brouillon (`gh release delete vX.Y.Z`), corriger
  sur `main` si nécessaire, puis relancer **sur le tag** :
  `gh workflow run release.yml --ref vX.Y.Z`. Ne jamais lancer depuis une branche :
  l'identité des signatures serait celle de la branche et les vérifications documentées
  échoueraient. Si l'image a déjà été poussée, la relance est refusée : publier une nouvelle
  version.
- **Après la publication** : une Release publiée ne se corrige pas. Publier une nouvelle
  version (par exemple `vX.Y.(Z+1)`).
- Tout échec inattendu est consigné dans `LESSON-LEARNED.md`.

### Revue trimestrielle

Dependabot suit les actions GitHub et les modules Go, mais **pas** les versions d'outils
passées en paramètre des actions : GoReleaser (`version`), cosign (`cosign-release`), Syft
(`syft-version`), Grype (`grype-version`), golangci-lint et govulncheck. Les vérifier chaque
trimestre (dernière version stable réelle), en même temps que la revue de
`LESSON-LEARNED.md` et de la constitution, ainsi que la date d'expiration du jeton Docker Hub.

## Licence des contributions

Le projet est distribué sous la [licence Apache 2.0](LICENSE). En proposant une contribution,
tu acceptes qu'elle soit publiée sous cette même licence (section 5 de la licence). Toute
nouvelle dépendance doit avoir une licence compatible (MIT, BSD, Apache 2.0…).

## Incidents

Tout incident (CI rouge inattendue, faille, régression, gate contournée) est consigné dans
`LESSON-LEARNED.md`, dans la pull request qui le corrige.
