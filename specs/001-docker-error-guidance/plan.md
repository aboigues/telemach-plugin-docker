# Implementation Plan: Guidage pédagogique à partir des erreurs Docker

**Branch**: `001-docker-error-guidance` | **Date**: 2026-10-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-docker-error-guidance/spec.md`

## Summary

Le plugin TELEMACH capte automatiquement les erreurs des commandes `docker` et
`docker compose` tapées par l'apprenant. Il les explique en français : concept Docker, cause
probable, corrections et approfondissement à la demande. Il sait aussi analyser les fichiers
Compose, ligne par ligne, après une erreur ou à la demande.

Choix techniques (voir [research.md](research.md)) :
- **Forme** : plugin Docker CLI autonome écrit en Go, binaire unique `docker-telemach`, qui
  fonctionne entièrement en local, sans réseau ;
- **Capture** : une intégration shell bash/zsh qui duplique stderr de la commande `docker` ;
- **Reconnaissance** : un catalogue d'erreurs déclaratif, embarqué dans le binaire ;
- **Analyse Compose** : `compose-go` (le parseur de référence) et yaml.v3 pour les positions ;
- **Affichage** : sortie largement colorée (lipgloss pour la mise en page, chroma pour la
  coloration syntaxique), accessible et désactivable.

## Technical Context

**Language/Version**: Go 1.27.2 (`go.mod` : `go 1.27`, `toolchain go1.27.2`)

**Primary Dependencies**:
- `github.com/spf13/cobra` v1.10.2 : sous-commandes ;
- `github.com/compose-spec/compose-go/v2` v2.16.1 : parseur Compose ;
- `go.yaml.in/yaml/v3` v3.0.5 : positions YAML ;
- `charm.land/lipgloss/v2` v2.0.6 : mise en page colorée ;
- `github.com/alecthomas/chroma/v2` v2.27.0 : coloration syntaxique ;
- `golang.org/x/term` v0.47.0 : détection du terminal.

Toutes les versions ont été vérifiées le 2026-10-14.

**Storage**: un fichier d'état local `$XDG_STATE_HOME/telemach/last.json` (droits `0600`,
contenu masqué). Le catalogue est embarqué dans le binaire (`go:embed`). Pas de base de
données.

**Testing**:
- `go test` avec `go-cmp` v0.7.0 et des golden files ;
- corpus de référence `testdata/` pour les critères SC-001, SC-002 et SC-006 ;
- fuzzing du matcher et du masquage ;
- tests d'intégration avec un vrai démon Docker (étiquette de build `integration`) ;
- tests du script shell sous bash et zsh.

**Target Platform**: Linux et macOS (bash, zsh), y compris WSL2. Windows natif (PowerShell) est
hors périmètre de cette feature.

**Project Type**: CLI, sous forme de plugin Docker CLI.

**Performance Goals**: explication affichée en moins de 2 s pour 95 % des cas (SC-003).
Objectif interne : moins de 200 ms, car le démarrage d'un binaire Go est quasi immédiat.

**Constraints**:
- 100 % hors ligne, aucun appel réseau ;
- lecture seule sur les fichiers de l'apprenant ;
- aucune commande n'est exécutée, sauf `docker --version`, lu une fois par le script shell (FR-001b) ;
- le code de sortie et le stdout de `docker` restent intacts ;
- stderr capturé limité à 1 Mio ;
- binaire statique (`CGO_ENABLED=0`).

**Scale/Scope**:
- catalogue initial d'au moins 30 erreurs couvrant les sous-commandes de FR-003 ;
- au moins 20 cas Compose ;
- un seul utilisateur local.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principe | Vérification | Statut |
|----------|--------------|--------|
| **I. Vocation pédagogique** | Chaque entrée du catalogue est obligée de fournir un concept défini, une cause, au moins une correction et un approfondissement (validé par les tests). Les erreurs inconnues sont traitées honnêtement (FR-008). Les anti-patterns sont signalés (FR-010). Les exemples sont exécutables (scénarios du quickstart). | ✅ |
| **II. Langue** | Code et identifiants en anglais. Catalogue, messages, commentaires et documentation en français. Les termes Docker restent tels quels. Les golden files vérifient que les accents s'affichent correctement. | ✅ |
| **III. Sécurité** | Masquage systématique des secrets (R7). Neutralisation des séquences ANSI (R6). Expressions régulières RE2 en temps linéaire. Lecture seule. Aucune exécution. Aucun réseau. Fichiers temporaires et d'état en `0600`. Dépendances épinglées sur des versions vérifiées. `docker/cli` non importé, pour réduire la surface d'attaque. | ✅ |
| **IV. Gates GitHub Actions** | La CI n'existe pas encore. `tasks.md` commence par la créer : `golangci-lint`+`gosec`, `go test -race`, `govulncheck`, `gitleaks`, revue des dépendances, `dockerfile-lint` (hadolint, `testdata/` exclu), actions épinglées par SHA, `permissions` minimales. Puis T010 protège `main` : checks obligatoires, `enforce_admins`, PR obligatoire ; la fusion par l'utilisateur vaut validation humaine. | ⚠️ à faire en premier |
| **V. LESSON-LEARNED** | Relu avant ce plan : aucun incident consigné, donc aucune règle préventive à appliquer. | ✅ |

**Point d'attention (pas une violation)** : compose-go dépend transitivement de
`go.yaml.in/yaml/v4` v4.0.0-rc.4, qui est une pré-version. Ce choix est justifié dans
research.md (R5) et surveillé par `govulncheck`.

**Re-check post-design** : le modèle de données et les contrats respectent les cinq principes.
Le contrat de `shell-init` garantit le code de sortie, le stdout intact et le nettoyage des
fichiers temporaires. Le contrat de `explain` interdit toute cause inventée. Aucune violation
à justifier.

## Project Structure

### Documentation (this feature)

```text
specs/001-docker-error-guidance/
├── plan.md              # Ce fichier
├── research.md          # Phase 0 — décisions R1 à R10
├── data-model.md        # Phase 1 — entités
├── quickstart.md        # Phase 1 — scénarios de validation V1 à V15
├── contracts/
│   └── cli.md           # Phase 1 — contrat des sous-commandes
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 (/speckit-tasks — pas encore créé)
```

### Source Code (repository root)

```text
cmd/
└── docker-telemach/
    └── main.go                 # Point d'entrée, câblage cobra

internal/
├── plugin/                     # Protocole Docker CLI (docker-cli-plugin-metadata)
├── shellinit/                  # Génération des scripts bash/zsh (gabarits embarqués)
│   └── scripts/{bash,zsh}.sh
├── command/                    # Analyse des arguments → Command
├── catalog/                    # Chargement + validation du catalogue
│   └── data/*.yaml             # Catalogue d'erreurs, en français, un fichier par catégorie
├── diagnose/                   # Matcher, codes de sortie (R3) → Diagnosis
├── compose/                    # Analyse Compose (yaml.v3 + compose-go) → ComposeIssue,
│                               # et son rendu (render.go : placé ici pour éviter un cycle d'imports)
├── redact/                     # Masquage des secrets (R7)
├── sanitize/                   # Neutralisation des séquences ANSI/contrôle
├── render/                     # lipgloss + chroma, gestion couleur/NO_COLOR
└── state/                      # LastDiagnosis, empreinte, fenêtre de répétition

testdata/
├── errors/                     # Corpus ≥ 30 erreurs Docker réelles (+ attendu)
├── compose/                    # ≥ 20 répertoires Compose erronés + valid/
├── secrets/                    # Corpus de fuite (SC-006)
└── golden/                     # Sorties de référence (couleur / sans couleur)

tests/
├── integration/                # Build tag `integration`, vrai démon Docker
└── shell/                      # Tests bash/zsh de la fonction docker

.github/workflows/
└── ci.yml                      # Gates (principe IV)

go.mod
.golangci.yml
```

**Structure Decision**: un seul projet Go, sous la forme recommandée pour un binaire CLI
(`cmd/` + `internal/`). Tout le code est dans `internal/`, ce qui n'expose aucune API publique
et garde l'outil autonome. Le catalogue est séparé dans `internal/catalog/data/` pour être
relu et enrichi sans toucher au moteur (principe I).

## Complexity Tracking

Aucune violation de la constitution à justifier.
