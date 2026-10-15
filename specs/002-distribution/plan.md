# Implementation Plan: Distribution du plugin sans Go

**Branch**: `002-distribution` | **Date**: 2026-10-15 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/002-distribution/spec.md`

## Summary

Rendre TELEMACH installable sans Go, par deux voies qui distribuent **les mêmes binaires** :

1. une **Release GitHub** par tag `vX.Y.Z` : quatre binaires (linux/darwin × amd64/arm64),
   sommes SHA-256 signées en keyless, SBOM, attestation de provenance SLSA, notes en
   français, Release immuable ;
2. une **image Docker Hub** `telemachlearning/telemach` (`FROM scratch`, non-root,
   multi-architecture, signée) qui sert à essayer `check` sans installation et à installer le
   plugin sur la machine hôte (`install`).

Choix techniques (voir [research.md](research.md)) : GoReleaser v2.18.3 (binaires, sommes,
SBOM, signature, image `dockers_v2`), `actions/attest` pour la provenance, cosign v3.1.3 pour
les signatures, Grype pour la gate de scan de l'image, un petit point d'entrée Go
`cmd/telemach-image` pour l'image, un workflow `release.yml` qui rejoue les gates de la CI
avant de publier depuis un environnement protégé « release ».

## Technical Context

**Language/Version**: Go 1.27.2 (inchangé) ; YAML GitHub Actions ; Dockerfile.

**Primary Dependencies**:
- code Go : **aucune nouvelle dépendance** ; `cmd/telemach-image` utilise la bibliothèque
  standard (`debug/buildinfo`, `syscall`, `os`) et réutilise `internal/render` (couleurs et
  symboles identiques au plugin, contrat [image-cli](contracts/image-cli.md)) ;
- outils de publication (CI seulement, jamais embarqués) : GoReleaser OSS v2.18.3, cosign
  v3.1.3, Syft v1.54.1, Grype (via `anchore/scan-action` v7.4.2), buildx/QEMU ;
- actions : voir le tableau des versions et SHA dans [research.md](research.md).

**Storage**: N/A (artefacts publiés sur GitHub Releases et Docker Hub).

**Testing**:
- `go test` pour `cmd/telemach-image` : copie, refus (FR-014), lecture de version d'un
  binaire existant, cibles `HOST_OS`, aide ; dossiers temporaires et faux binaires ;
- test d'intégration (étiquette `integration`) : construction de l'image en local, puis
  `docker run … install` et `docker run … check` exactement comme documenté, sans raccourci
  (LL-006) ;
- `goreleaser check` et `goreleaser release --snapshot` dans la CI des PR ;
- validation manuelle des quatre plateformes (SC-007) avant la première Release.

**Target Platform**: Linux (dont WSL2) et macOS, amd64 et arm64 ; image linux/amd64 et
linux/arm64.

**Project Type**: CLI (plugin Docker) + chaîne de publication.

**Performance Goals**: Release et image disponibles moins de 30 min après le push du tag
(SC-004) ; installation par l'image en moins de 1 min hors téléchargement (SC-002).

**Constraints**:
- image ≤ 15 Mo compressés par architecture (SC-006 ; ~10 Mo mesurés) ;
- aucun shell dans l'image, utilisateur non-root, aucun secret ;
- le jeton Docker Hub n'est accessible qu'au job `release`, dans l'environnement protégé ;
- releases et tags d'image jamais écrasés ;
- toutes les actions épinglées par SHA complet, permissions minimales par job.

**Scale/Scope**: 4 binaires et 2 images par version ; ~1 nouveau programme Go (~250 lignes
avec tests), 1 Dockerfile, 1 configuration GoReleaser, 1 nouveau workflow, ajouts à `ci.yml`,
README, CONTRIBUTING.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principe | Vérification | Statut |
|----------|--------------|--------|
| **I. Vocation pédagogique** | Les messages de `install` (succès et refus) sont en français et nomment le concept en jeu (bind mount, identité du processus, architecture, système cible) avec la commande corrigée à copier (FR-013, FR-014, [contrat](contracts/image-cli.md)). L'image elle-même illustre les bonnes pratiques enseignées : `scratch`, non-root, tag explicite, lecture seule pour l'essai. | ✅ |
| **II. Langue** | Code et identifiants en anglais (`cmd/telemach-image`, clés YAML). Messages, notes de Release, README, CONTRIBUTING, commentaires en français. Chemins visibles par l'apprenant en français (`/projet`, `/sortie`), choisis par la spec. | ✅ |
| **III. Sécurité** | Aucune nouvelle dépendance Go. Image sans shell ni paquet, non-root, sans secret, `.dockerignore` présent. Signatures keyless et provenance (pas de clé à protéger). Jeton Docker Hub limité au dépôt, créé par le mainteneur, visible du seul job `release`. Pas d'interpolation `${{ }}` d'entrées dans les scripts. Trivy écarté après sa compromission (R8). `install` refuse d'écraser un fichier inconnu ou un lien symbolique, écrit de façon atomique. Base `scratch` : rien à épingler par digest. | ✅ |
| **IV. Gates** | Nouvelle gate `image-scan` (Grype, sévérité haute bloquante) et `goreleaser check` dans les PR. `dockerfile-lint` couvre le nouveau `Dockerfile`. La publication rejoue toutes les gates (`ci.yml` en `workflow_call`). Actions épinglées par SHA, permissions minimales (écriture uniquement dans `release`). Ajouter `image-scan` (et `build`) aux checks obligatoires de `main`. | ✅ (protection de `main` à mettre à jour : tâche dédiée) |
| **V. LESSON-LEARNED** | Relu avant ce plan. **LL-001** : GoReleaser n'analyse pas le code, mais tout outil Go lancé par `go install`/`go run` reçoit `GOTOOLCHAIN=go1.27.2`. **LL-004** : versions et SHA vérifiés le 2026-10-15 (research). **LL-005** : prérequis côté dépôt et Docker Hub (environnement, secret, releases immuables) documentés dans CONTRIBUTING, et PR de test des gates avant le code. **LL-006** : test d'intégration qui suit à la lettre les commandes du README (dossier créé, `--user`, tag explicite). LL-002/LL-003 : sans objet (pas de nouveau code shell). | ✅ |

**Après la phase 1** : la conception (contrats, modèle de données) ne crée aucune dérogation.
Le seul point de vigilance est le repli prévu en R6 si `dockers_v2` ne peut pas embarquer les
binaires darwin : il ne change ni les exigences ni le Constitution Check.

## Project Structure

### Documentation (this feature)

```text
specs/002-distribution/
├── plan.md              # Ce fichier
├── research.md          # Phase 0 : versions vérifiées et décisions R1–R11
├── data-model.md        # Phase 1 : version, artefacts, image, environnement
├── quickstart.md        # Phase 1 : scénarios de validation de bout en bout
├── contracts/
│   ├── release-assets.md    # Noms et contenu des artefacts de la Release
│   ├── image-cli.md         # Interface de l'image (install, check, aide, codes de sortie)
│   └── image-tags.md        # Tags Docker Hub et règles de publication
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 (/speckit-tasks)
```

### Source Code (repository root)

```text
cmd/
├── docker-telemach/          # plugin existant (inchangé, sauf éventuel ajustement R5)
└── telemach-image/           # NOUVEAU : point d'entrée de l'image
    ├── main.go               # aiguillage : install / aide / exec du binaire linux
    ├── install.go            # copie atomique, refus pédagogiques (FR-013, FR-014)
    ├── messages.go           # textes en français
    └── *_test.go

Dockerfile                    # NOUVEAU : FROM scratch, USER 65532, copie des binaires
.dockerignore                 # NOUVEAU
.goreleaser.yaml              # NOUVEAU : builds, sommes, SBOM, signature, dockers_v2, release

.github/workflows/
├── ci.yml                    # + workflow_call, + goreleaser check/snapshot, + image-scan ;
│                             #   retrait du déclencheur tags v*
└── release.yml               # NOUVEAU : verify → gates → release (environment: release)

tests/integration/
└── image_test.go             # NOUVEAU : install/check via l'image, commandes du README

docs/
└── dockerhub.md              # NOUVEAU : description de la page Docker Hub (copiée à la main)

README.md                     # installation : Release, image, essai, sources (FR-020)
CONTRIBUTING.md               # procédure de publication et prérequis (FR-022)
```

**Structure Decision**: projet Go unique existant ; un second `main` (`cmd/telemach-image`)
est ajouté pour ne pas mêler au plugin une commande propre à l'image. La configuration de
publication est à la racine (conventions GoReleaser et Docker).

## Complexity Tracking

Aucune dérogation à la constitution.
