# Implementation Plan: Analyse de la ligne de commande, même en cas de succès

**Branch**: `003-command-line-analysis` | **Date**: 2026-10-15 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/003-command-line-analysis/spec.md`

## Summary

TELEMACH analyse désormais la ligne `docker run|create|exec` tapée par l'apprenant, **même
quand Docker réussit**, pour signaler les écritures que Docker accepte sans broncher : options
placées après l'image ou le conteneur (issue #9) et sept pièges silencieux (volume nommé au lieu
du dossier local, port aléatoire, variable non définie, image sans tag, ports ignorés en réseau
`host`, secret en clair, privilèges excessifs). L'avertissement, en français et au format des
explications existantes, s'affiche **après** la commande, sans modifier ses sorties ni son code.

Choix techniques (voir [research.md](research.md)) : un paquet pur `internal/lint` (une règle
par piège, signaux forts/faibles contre les faux positifs), les textes dans le catalogue
embarqué (`cmdline.yaml`), une sous-commande `docker telemach lint` appelée par l'intégration
shell pour les seules commandes `run|create|exec` réussies, l'intégration à `explain` en cas
d'échec, une petite mémoire `hints.json` pour les répétitions.

## Technical Context

**Language/Version**: Go 1.27.2 (inchangé) ; scripts bash et zsh de `shell-init`.

**Primary Dependencies**: aucune nouvelle dépendance. Réutilise `internal/command`,
`internal/catalog`, `internal/render`, `internal/redact`, `internal/sanitize`, `internal/state`,
cobra.

**Storage**: `last.json` (dernier diagnostic, champ `Findings` ajouté) et nouveau `hints.json`
(empreintes des pièges affichés, 24 h, 200 entrées max), dans `$XDG_STATE_HOME/telemach`,
`0600`.

**Testing**:
- tests par tables de `internal/lint` : corpus fautif (≥ 40 lignes, ≥ 10 options mal placées,
  ≥ 3 par piège) et corpus légitime (≥ 50 lignes dont ≥ 20 arguments d'entrypoint réels) dans
  `testdata/cmdline/` (SC-001, SC-002) ;
- golden files du rendu (avertissement complet, rappel, information, intégration à `explain`) ;
- tests du script shell sous bash **et** zsh (LL-002) : appel de `lint` seulement pour
  `run|create|exec` réussies, stdout/stderr/code inchangés, `TELEMACH_LINT=0` ;
- tests d'intégration avec un vrai démon : exemple de l'issue #9 et commandes corrigées
  exécutées (SC-001), mesure du délai (SC-003).

**Target Platform**: Linux, macOS, WSL2 (bash, zsh) — inchangé.

**Project Type**: CLI (plugin Docker).

**Performance Goals**: délai ajouté < 50 ms au p95 pour `run|create|exec` (mesuré : 25 ms de
démarrage, 32 ms pour un rendu complet) ; aucun appel du plugin pour les autres commandes.

**Constraints**: aucune exécution de commande, aucun accès réseau ; seules lectures locales :
présence d'une variable d'environnement (jamais sa valeur), existence d'un dossier dans le
dossier courant ; secrets masqués partout ; sorties et code de Docker inchangés.

**Scale/Scope**: 8 pièges (options mal placées + 7), 8 règles Go, 9 entrées de catalogue (options mal placées : une entrée pour `run`/`create`, une pour `exec`), ~120 lignes de corpus.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principe | Vérification | Statut |
|----------|--------------|--------|
| **I. Vocation pédagogique** | Chaque piège a une entrée de catalogue complète (concept défini, cause, correction exécutable, approfondissement) ; le piège 7 est présenté comme anti-pattern avec ses alternatives ; formulation honnête (« probablement ») et silence en cas de doute ; `lint` utilisable à la main pour vérifier avant d'exécuter. | ✅ |
| **II. Langue** | Identifiants en anglais (`internal/lint`, `cmdline.*`, `TELEMACH_LINT`) ; textes, messages, commentaires et documentation en français. | ✅ |
| **III. Sécurité** | Aucune exécution, aucun réseau ; la valeur d'une variable d'environnement n'est jamais lue ; arguments nettoyés puis masqués avant affichage et avant écriture sur disque (`hints.json` ne contient aucune valeur) ; nouveau piège qui **enseigne** la sécurité (secret en clair, `--privileged`, socket Docker) ; exemples d'images avec tag explicite, non-root (tests de sécurité du catalogue). | ✅ |
| **IV. Gates** | Aucune nouvelle gate nécessaire : tests unitaires, shell (bash/zsh) et d'intégration dans les jobs existants `test` et `integration`, tous obligatoires ; lint couvrant les étiquettes de build (LL-007). | ✅ |
| **V. LESSON-LEARNED** | Relu (LL-001 à LL-009). **LL-002** : script testé sous bash et zsh, assertions sur stdout **et** stderr. **LL-003** : `lint` ne crée aucun fichier temporaire ; le chemin d'échec existant est inchangé. **LL-006** : tests shell sans raccourci de PATH (chemin absolu du binaire). **LL-008** : validations enchaînées par `&&`. **LL-009** : vérifications ad hoc en `bash -c`, échec si valeur vide. | ✅ |

**Après la phase 1** : aucune dérogation. Point d'attention documenté : la décision de la 001
« le plugin n'exécute rien » est respectée (les deux lectures locales ne sont pas des
exécutions), ce qui ne sera pas le cas de l'issue #8.

## Project Structure

### Documentation (this feature)

```text
specs/003-command-line-analysis/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-lint.md      # sous-commande lint, intégration shell, intégration à explain, last
├── checklists/requirements.md
└── tasks.md             # Phase 2 (/speckit-tasks)
```

### Source Code (repository root)

```text
internal/lint/                 # NOUVEAU : analyse pure de la ligne de commande
├── doc.go
├── lint.go                    # Analyze, Finding, Env, ordre des résultats
├── misplaced.go               # options après l'image / le conteneur (signaux forts/faibles)
├── pitfalls.go                # pièges 1 à 7
├── fix.go                     # reconstruction de la commande corrigée, citation shell
└── *_test.go                  # tables + corpus testdata/cmdline/

internal/catalog/
├── schema.go                  # + champ Severity, fallback « cmdline »
└── data/cmdline.yaml          # NOUVEAU : 9 entrées cmdline.*

internal/render/               # + rendu avertissement / information / rappel / section « Dans ta commande »
internal/state/                # + Findings dans LastDiagnosis, + hints.json (HintStore)
internal/command/command.go    # + options à valeur de « create » (= run)
internal/shellinit/scripts/    # bash.sh et zsh.sh : détection run|create|exec, appel de lint

cmd/docker-telemach/
├── lint.go                    # NOUVEAU : sous-commande lint
├── explain.go                 # + pièges intégrés (FR-010)
└── last.go                    # + réaffichage d'un avertissement

testdata/cmdline/              # NOUVEAU : misplaced-*.tsv, pitfalls-*.tsv (fautifs / légitimes)
tests/shell/                   # + cas lint (bash et zsh)
tests/integration/             # + issue #9 et commandes corrigées exécutées, délai
README.md, docs/catalogue.md   # documentation (avertissements, TELEMACH_LINT, lint à la main)
```

**Structure Decision**: projet Go unique existant ; un nouveau paquet `internal/lint` isole la
détection (pure, testée par tables) ; tout le reste étend des paquets existants.

## Complexity Tracking

Aucune dérogation à la constitution.
