---

description: "Liste des tâches pour la feature 001-docker-error-guidance"
---

# Tasks: Guidage pédagogique à partir des erreurs Docker

**Input**: Design documents from `/specs/001-docker-error-guidance/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, quickstart.md

**Tests**: inclus. Les critères de succès SC-001, SC-002, SC-006 et SC-007 de la spec exigent
des corpus de référence mesurés, et le principe IV de la constitution exige des tests bloquants
en CI. Dans chaque histoire, les tests sont écrits **avant** l'implémentation et doivent
d'abord échouer.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Conventions du projet (à respecter dans chaque tâche)

- Projet Go unique : `cmd/docker-telemach/`, `internal/…`, `testdata/`, `tests/` (voir plan.md).
- **Code et identifiants en anglais ; commentaires, messages, catalogue et documentation en
  français**, accents compris (principe II).
- Relire `LESSON-LEARNED.md` avant de commencer chaque phase (principe V).
- Constitution de référence : **v1.1.0** (une image sans tag dans un exemple = avertissement non bloquant ; non-root et images minimales restent obligatoires).
- Toute dépendance ou action GitHub ajoutée : vérifier la **dernière version stable réelle** au
  moment de l'ajout. Les versions de research.md ont été vérifiées le 2026-10-14.
- Toute chaîne affichée ou écrite sur disque passe par `sanitize` puis `redact`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: initialiser le dépôt et poser les gates CI **avant** tout code (principe IV)

- [X] T001 Initialiser le dépôt git (`git init -b main`, premier commit sur `main` contenant la documentation existante : `.specify/`, `.claude/`, `specs/`, `LESSON-LEARNED.md`, puis création de la branche `001-docker-error-guidance`) et créer `.gitignore` à la racine (binaires `docker-telemach`, `dist/`, fichiers de couverture `*.out`, `.idea/`, `.vscode/`)
- [X] T002 Créer `go.mod` à la racine avec `module github.com/aboigues/telemach-plugin-docker`, `go 1.27` et `toolchain go1.27.2`. Ajouter les dépendances directes aux versions de research.md : `github.com/spf13/cobra` v1.10.2, `github.com/compose-spec/compose-go/v2` v2.16.1, `go.yaml.in/yaml/v3` v3.0.5, `charm.land/lipgloss/v2` v2.0.6, `github.com/alecthomas/chroma/v2` v2.27.0, `golang.org/x/term` v0.47.0, `github.com/google/go-cmp` v0.7.0
- [X] T003 Créer l'arborescence du plan : `cmd/docker-telemach/`, `internal/{plugin,shellinit/scripts,command,catalog/data,diagnose,compose,redact,sanitize,render,state,textdist}/`, `testdata/{errors,compose,secrets,golden}/`, `tests/{integration,shell}/`, avec un fichier `doc.go` par paquet contenant le commentaire de paquet en français
- [X] T004 [P] Créer `.golangci.yml` au format golangci-lint v2 (v2.14.0) avec les linters `govet`, `staticcheck`, `errcheck`, `gosec`, `ineffassign`, `unused`, `revive`, `gocritic`, `bodyclose`, `misspell` (locale US, à appliquer aux identifiants seulement) et le formateur `gofumpt`
- [X] T005 [P] Créer `.github/workflows/ci.yml` (déclencheurs `pull_request` et `push` sur `main`, `permissions: contents: read` au niveau du workflow). Jobs :
  - `lint` (golangci-lint v2.14.0) ;
  - `test` (`go test -race -cover ./...` avec `go-version-file: go.mod`) ;
  - `vuln` (`govulncheck` de `golang.org/x/vuln` v1.9.0) ;
  - `secrets` (gitleaks v8.30.1) ;
  - `dependency-review` (sur `pull_request` uniquement) ;
  - `dockerfile-lint` : hadolint v2.15.1 sur tout `Dockerfile` ou `*.Dockerfile` **hors `testdata/`**, dont les Dockerfile volontairement erronés servent au corpus. Le job réussit s'il ne trouve aucun fichier, et le scan d'image s'ajoutera le jour où le projet livrera une image (principe IV).

  **Chaque action est épinglée par SHA de commit complet**, avec la version en commentaire (rechercher les SHA au moment de l'écriture). Interdire `pull_request_target`.
- [X] T006 [P] Créer `.github/dependabot.yml` : écosystèmes `gomod` et `github-actions`, fréquence hebdomadaire, mises à jour de sécurité groupées
- [X] T007 [P] Créer `.gitleaks.toml` qui étend la configuration par défaut et n'autorise (allowlist) **que** le répertoire `testdata/secrets/`, qui contient de faux secrets volontaires
- [X] T008 [P] Créer `SECURITY.md` (français) : procédure de signalement responsable, versions supportées, délai de réponse
- [X] T009 [P] Créer `CONTRIBUTING.md` (français) :
  - règles de langue ;
  - relecture de `LESSON-LEARNED.md` ;
  - gates requises ;
  - liste des checks déclarés « required » dans la protection de la branche `main` (`lint`, `test`, `vuln`, `secrets`, `dependency-review`, `dockerfile-lint`) et rappel des règles appliquées par la tâche T010 (pas de push direct, pas de contournement administrateur).
- [X] T010 Protéger la branche `main` du dépôt **public** `https://github.com/aboigues/telemach-plugin-docker` (principe IV). Le dépôt existe déjà et est vide ; `gh` est authentifié en tant que `aboigues`. Dépend de T001 et T005. **Action visible de l'extérieur : demander confirmation à l'utilisateur avant chaque étape.**
  1. Ajouter le remote : `git remote add origin https://github.com/aboigues/telemach-plugin-docker.git`, puis pousser `main` (`git push -u origin main`).
  2. Créer `.github/branch-protection.json` avec :
     - `required_status_checks` : `strict: true`, contextes `lint`, `test`, `vuln`, `secrets`, `dependency-review`, `dockerfile-lint` ;
     - `enforce_admins: true` ;
     - `required_pull_request_reviews` : `required_approving_review_count: 0` (PR obligatoire, pas d'approbation formelle : la fusion par l'utilisateur vaut validation humaine) ;
     - `required_linear_history: true`, `required_conversation_resolution: true` ;
     - `allow_force_pushes: false`, `allow_deletions: false`, `restrictions: null`.
  3. Appliquer : `gh api -X PUT repos/aboigues/telemach-plugin-docker/branches/main/protection --input .github/branch-protection.json`.
  4. Vérifier avec `gh api repos/aboigues/telemach-plugin-docker/branches/main/protection` que les six contextes, `enforce_admins` et la PR obligatoire sont actifs, puis tenter un `git push origin main` direct, qui doit être refusé.
  5. **Workflow de fusion retenu** : Claude Code commit, pousse et ouvre la PR avec le compte `aboigues` ; l'utilisateur relit la PR et la fusionne lui-même. Ce clic sur « Merge » vaut validation humaine. Aucune approbation formelle n'est exigée, car GitHub interdit d'approuver sa propre PR.
  6. Consigner le résultat dans `CONTRIBUTING.md` (workflow de fusion), et dans `LESSON-LEARNED.md` en cas d'incident.

**Checkpoint**: `go build ./...` passe sur un squelette vide ; le workflow CI est valide (`actionlint` si disponible) ; `main` est protégée et refuse un push direct.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: chaîne complète « capture → diagnostic → rendu » avec repli « erreur non reconnue », partagée par toutes les histoires

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Tests fondamentaux (écrire d'abord)

- [X] T011 [P] Écrire `internal/sanitize/sanitize_test.go` : retrait des séquences ANSI CSI/OSC et des caractères de contrôle sauf `\n` et `\t`, conservation des accents UTF-8, et cible `FuzzSanitize` (aucune séquence `\x1b` en sortie, jamais de panique)
- [X] T012 [P] Créer le corpus `testdata/secrets/` (au moins 15 cas `*.txt`, chacun avec son `*.expected.txt`). Il couvre :
  - `-e DB_PASSWORD=…`, `--env API_TOKEN=…`, `environment:` en liste et en dictionnaire ;
  - `postgres://user:pass@host` ;
  - en-tête `Authorization: Bearer …` ;
  - jetons `ghp_…`, `glpat-…`, `dckr_pat_…` ;
  - noms contenant `SECRET`, `KEY`, `CREDENTIAL`, `AUTH`, `PASSWD`, sans distinction de casse.
- [X] T013 [P] Écrire `internal/redact/redact_test.go` : vérifie chaque cas de `testdata/secrets/` (la valeur est remplacée par `••••••`, le nom de la variable est conservé) et la cible `FuzzRedact` (sortie idempotente, aucune panique)
- [X] T014 [P] Écrire `internal/command/command_test.go`, une table de cas. Cas couverts :
  - `run -p 8080:80 nginx` ;
  - `compose -f a.yaml -f b.yaml up` ;
  - `--context x run …`, `-H tcp://… ps` ;
  - `exec -it c sh` (`Interactive=true`), `attach`, `login` sans `--password-stdin` (interactif) ;
  - `build -f Dockerfile .`.

  Vérifier `Family`, `Subcommand`, `Flags`, `ComposeFiles`, `Interactive`, et le masquage des valeurs `-e`.
- [X] T015 [P] Écrire `internal/render/color_test.go` : résolution de `--color=auto|always|never`, `NO_COLOR` (toute valeur non vide → pas de couleur), `CLICOLOR_FORCE=1` (équivaut à `always`), sortie non-TTY (pas de couleur en `auto`)
- [X] T016 [P] Écrire `internal/state/state_test.go` :
  - répertoire `0700` et fichier `0600` ;
  - `$XDG_STATE_HOME/telemach/last.json`, avec repli sur `~/.local/state/telemach/` ;
  - `SchemaVersion` = `1` ;
  - fichier corrompu ou de schéma inconnu ignoré puis réécrit sans erreur ;
  - `IsRepeat` vrai pour la même empreinte à moins de 10 minutes, faux au-delà.
- [X] T017 [P] Écrire `tests/shell/shellinit_test.go`. Le test exécute les scripts générés sous `bash`, et sous `zsh` s'il est présent (sinon `t.Skip`), avec un faux binaire `docker` et un faux `docker-telemach` placés dans `PATH`. Il vérifie les **6 garanties** de `contracts/cli.md` § shell-init :
  1. le code de sortie d'origine est renvoyé ;
  2. le stdout reste identique octet pour octet ;
  3. le fichier temporaire est créé en `0600` puis supprimé, y compris après SIGINT ;
  4. `TELEMACH_DISABLE=1` désactive la capture ;
  5. si `docker-telemach` est absent, le comportement est celui de `docker` ;
  6. en mode interactif, stderr n'est pas redirigé et `--interactive` est transmis ;
  7. `docker --version` est appelé **une seule fois** à l'évaluation du script (compter les appels au faux `docker`), et sa valeur est transmise par `--docker-version` ; aucune autre commande n'est lancée par le script.
- [X] T018 [P] Écrire `internal/plugin/metadata_test.go` : la sortie JSON est exactement `{"SchemaVersion":"0.1.0","Vendor":"TELEMACH","Version":…,"ShortDescription":"Explique en français les erreurs Docker et Docker Compose"}`

### Implémentation fondamentale

- [X] T019 [P] Implémenter `internal/sanitize/sanitize.go` (`func Clean(s string) string`) pour faire passer T011
- [X] T020 [P] Implémenter `internal/redact/redact.go` (`func Redact(s string) string`, `func RedactArgs(args []string) []string`) selon research.md R7. Les motifs sont en RE2 (`regexp`) et le marqueur de remplacement est `••••••`. Faire passer T013.
- [X] T021 [P] Implémenter `internal/textdist/levenshtein.go` (`func Distance(a, b string) int`, `func Closest(word string, candidates []string, maxDist int) (string, bool)`) avec `internal/textdist/levenshtein_test.go`
- [X] T022 [P] Implémenter `internal/command/command.go` : type `Command` (`Family` enum `docker`|`compose`, `Subcommand`, `Flags map[string]string`, `ComposeFiles []string`, `Interactive bool`) et `func Parse(args []string) Command`, conformes à data-model.md. Faire passer T014.
- [X] T023 [P] Implémenter `internal/render/color.go` (`type ColorMode`, `func ResolveColor(flag string, env func(string) string, isTTY bool) bool`) avec `golang.org/x/term`. Faire passer T015.
- [X] T024 [P] Implémenter `internal/render/theme.go` : styles lipgloss v2 à couleurs adaptatives (fond clair et sombre), un par rôle (résumé, concept, cause, correction, anti-pattern, message d'origine atténué, succès). Chaque rôle a un symbole et un libellé, la couleur n'étant **jamais** le seul porteur d'information : `✖`, `📘 Concept`, `🔎 Cause probable`, `💡 Corrections`, `⚠ Anti-pattern`, `✔`, `➜ Pour aller plus loin`. Le thème doit aussi produire un rendu sans couleur.
- [X] T025 [P] Implémenter `internal/render/highlight.go` : `func Highlight(code, lexer string, color bool) string` avec chroma v2 (lexers `bash`, `yaml`, `docker`), et `func HighlightLines(lines []Line, focusLine, focusCol int, color bool) string`, qui surligne la ligne fautive et place un marqueur `^` sous la colonne. Sans couleur : texte brut avec numéros de ligne et marqueur. Ajouter un test dans `internal/render/highlight_test.go`.
- [X] T026 Implémenter `internal/catalog/schema.go` et `internal/catalog/loader.go` :
  - types `CatalogEntry`, `Fix`, `ConceptRef`, `Deepening`, `AntiPatternNote` (data-model.md) ;
  - chargement par `go:embed data/*.yaml` avec `yaml.v3` en mode `KnownFields(true)` ;
  - validation au chargement :
    - `ID` unique et conforme à `^[a-z]+(\.[a-z0-9-]+)+$` ;
    - `Category` ∈ {`network`, `image`, `container`, `volume`, `daemon`, `permission`, `build`, `compose`, `runtime`} ;
    - `Patterns` ≥ 1 sauf si `ExitCodes` est non vide, toutes compilées ;
    - `Summary` obligatoire et ≤ 160 caractères ;
    - `Cause` obligatoire ;
    - `Fixes` ≥ 1 ;
    - `Deepen` obligatoire ;
    - `DocURL` préfixé par `https://docs.docker.com/` ;
    - `Tests` ≥ 1.
- [X] T027 Écrire `internal/catalog/catalog_test.go` : charge le catalogue embarqué et vérifie toutes les règles de T026. Pour chaque entrée, chaque fichier listé dans `Tests` doit exister dans `testdata/errors/` et correspondre à au moins un de ses `Patterns`. Contrôle de langue (SC-005) : `Summary`, `Cause` et `Concept.Definition` contiennent au moins un mot-outil français (`le`, `la`, `les`, `de`, `du`, `des`, `un`, `une`, `est`, `pas`) (dépend de T026).
- [X] T028 Implémenter `internal/diagnose/diagnose.go` :
  - types `Diagnosis` (`Confidence` ∈ {`recognized`, `partial`, `unknown`}, `Origin` ∈ {`docker`, `application`, `unknown`}, `Entry`, `Captures`, `ComposeIssues`, `Fingerprint`) ;
  - `func Diagnose(cmd command.Command, exitCode int, stderr string, cat *catalog.Catalog) Diagnosis` : filtre sur `Applies` et `ExitCodes`, première entrée dont un motif correspond, groupes nommés extraits puis masqués ;
  - repli `unknown` sans cause (FR-008) ;
  - `Fingerprint` = SHA-256 de la commande et de l'erreur normalisées et masquées.

  Ajouter des tests dans `internal/diagnose/diagnose_test.go` (dépend de T022 et T026).
- [X] T029 [P] Implémenter `internal/state/state.go` : type `LastDiagnosis` (`SchemaVersion`, `Fingerprint`, `EntryID`, `Captures` déjà masquées, `RecordedAt`), `Load`, `Save` (écriture atomique, droits `0600`), `IsRepeat(fp string, now time.Time) bool` sur une fenêtre de 10 minutes. Faire passer T016.
- [X] T030 Implémenter `internal/render/explanation.go` :
  - `func RenderExplanation(w io.Writer, d diagnose.Diagnosis, original string, opts Options)` ;
  - blocs dans l'ordre de l'exemple de `contracts/cli.md` : résumé, concept et définition, cause, corrections numérotées avec commande colorée par `Highlight`, anti-pattern éventuel, message Docker d'origine **atténué avec les fragments reconnus (captures : port, image, clé, chemin) surlignés en couleur** (FR-007, clarification du 2026-10-14), indication `➜ Pour aller plus loin : docker telemach deepen` ;
  - bloc « Erreur non reconnue » sans cause, avec lien vers la documentation (FR-008) ;
  - rendu court quand `Repeated` est vrai.

  Dépend de T024 et T025.
- [X] T031 [P] Implémenter `internal/plugin/metadata.go` (sous-commande cachée `docker-cli-plugin-metadata`) pour faire passer T018. **Ne pas importer `github.com/docker/cli`**.
- [X] T032 [P] Écrire `internal/shellinit/scripts/bash.sh` et `internal/shellinit/scripts/zsh.sh` (commentaires en français). Chacun définit une fonction `docker` :
  - les variables internes sont préfixées `__telemach_` ;
  - le fichier temporaire est créé par `mktemp` dans `${XDG_RUNTIME_DIR:-${TMPDIR:-/tmp}}` avec `umask 077` ;
  - stderr est dupliqué via `2> >(tee "$tmp" >&2)`, sauf pour une commande interactive ;
  - `trap` assure le nettoyage ;
  - en cas d'échec, appel de `command docker-telemach explain --exit-code "$rc" --stderr-file "$tmp" -- "$@"`, dont le code de sortie est ignoré ;
  - la fonction renvoie toujours `$rc` ;
  - court-circuits si `TELEMACH_DISABLE=1` ou si `docker-telemach` est absent ;
  - à l'évaluation, lecture unique de la version par `__telemach_docker_version=$(command docker --version 2>/dev/null | sed -n 's/^Docker version \([^,]*\),.*/\1/p')`, puis transmission par `--docker-version "$__telemach_docker_version"` si elle n'est pas vide (FR-001b).

  Implémenter `internal/shellinit/shellinit.go` (`go:embed`, `func Script(shell string) (string, error)`, erreur en français si le shell n'est pas pris en charge). Faire passer T017.
- [X] T033 Implémenter `cmd/docker-telemach/main.go` : racine cobra `docker-telemach` avec les options globales `--color` (`auto`|`always`|`never`) et `--width` (entier ≥ 40). Brancher les sous-commandes `docker-cli-plugin-metadata` et `shell-init` (`cmd/docker-telemach/shellinit.go`, écriture sur stdout, code `2` si le shell est inconnu). Messages d'aide et d'erreur en français.
- [X] T034 Implémenter `cmd/docker-telemach/explain.go` selon `contracts/cli.md` § explain :
  - options `--exit-code` (obligatoire, 1 à 255), `--stderr-file`, `--interactive`, `--docker-version` (format `^[0-9]+\.[0-9]+\.[0-9]+([-+.][0-9A-Za-z.-]+)?$`, ignorée si invalide), `-- <args>` ;
  - en mode `--interactive`, diagnostic fondé sur le seul code de sortie, sinon « non reconnue » ;
  - version détectée mentionnée dans le bloc « Erreur non reconnue » ;
  - lecture de stderr limitée à 1 Mio (au-delà : on garde la **fin**), sans jamais modifier ni supprimer le fichier ;
  - pipeline `sanitize` → `redact` → `command.Parse` → `diagnose.Diagnose` → `state` (répétition et sauvegarde) → `render`, sortie sur **stderr** ;
  - code `0` dès qu'une explication est produite, code `2` en cas d'usage invalide.

  - **la sortie d'`explain` est colorée** (FR-007a) : le mode de couleur est résolu par `render.ResolveColor`, à partir de `--color`, de `NO_COLOR`, de `CLICOLOR_FORCE` et de la détection d'un terminal **sur stderr**.

  Ajouter `cmd/docker-telemach/explain_test.go`, qui couvre :
  - le scénario V5 du quickstart (erreur inventée → bloc « non reconnue », aucune cause) ;
  - `--color=always` : séquences de couleur présentes, message d'origine atténué et fragment reconnu surligné ;
  - `--color=never` et `NO_COLOR=1` : aucune séquence `\x1b`, mais symboles et libellés toujours présents (SC-006a).

**Checkpoint**: `go test ./...` passe ; `docker-telemach explain` affiche « Erreur non reconnue » pour toute erreur, et l'intégration shell fonctionne de bout en bout.

---

## Phase 3: User Story 1 - Comprendre l'échec d'une commande Docker simple (Priority: P1) 🎯 MVP

**Goal**: expliquer en français, en couleur, les erreurs des commandes Docker simples (FR-002 à FR-010), et distinguer une erreur de Docker d'une erreur de l'application.

**Independent Test**: provoquer un port déjà utilisé, une image mal orthographiée, un démon arrêté et un `exit 3` applicatif (quickstart V1 à V4). Chaque erreur doit recevoir une explication complète et correcte ; le corpus doit être reconnu à ≥ 90 % (SC-001).

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T035 [P] [US1] Constituer le corpus `testdata/errors/`. Il comporte au moins 30 erreurs **réelles** capturées avec Docker 29.x. Chaque cas est un triplet `<id>.stderr.txt`, `<id>.args` (une ligne d'arguments) et `<id>.expected.yaml` (`entry_id`, `exit_code`, `origin`, `captures`). Cas obligatoires :
  - **Réseau** : port déjà alloué, spécification de port invalide, réseau introuvable, réseau avec des endpoints actifs.
  - **Images** :
    - image introuvable, `pull access denied`, faute de frappe `ngnix` ;
    - `invalid reference format` (majuscules), `manifest unknown` (tag inexistant) ;
    - `rmi` d'une image utilisée, `push` refusé faute d'authentification ;
    - `exec format error` (architecture différente).
  - **Conteneurs** : conflit de nom, `No such container` (pour `logs`, `stop`, `start`, `exec`), conteneur arrêté pour `exec`, `rm` d'un conteneur en cours d'exécution.
  - **Démon et permissions** : démon arrêté, `permission denied` sur le socket, `no space left on device`, `unauthorized` au `login`.
  - **Volumes** : source de bind mount inexistante, volume utilisé lors de `volume rm`.
  - **Build** : Dockerfile introuvable, `COPY failed` (fichier introuvable), `RUN` qui renvoie un code non nul (sortie `build` sur plusieurs étapes).
  - **Exécution** : code 126, code 127 `executable file not found in $PATH`, code 137 (OOM), code 143, code applicatif `3`.
  - **Compose (commandes sans analyse de fichier)** : `docker compose ps` et `docker compose logs` lancés hors d'un projet (`no configuration file provided`), `docker compose down` sur un projet inexistant (`compose-*.stderr.txt`, utilisés par T063).
  - **Non reconnu** : un cas volontairement inconnu.
- [ ] T036 [P] [US1] Écrire `internal/diagnose/corpus_test.go`. Il parcourt `testdata/errors/`, compare chaque `Diagnosis` à `expected.yaml`, calcule et journalise le taux de reconnaissance (échec si < 90 %, SC-001), et vérifie qu'aucun cas `unknown` n'a de cause (SC-005).
- [ ] T037 [P] [US1] Écrire `tests/integration/docker_test.go` (`//go:build integration`). Il rejoue les scénarios V1 (port alloué, code 125 préservé), V2 (`ngnix`) et V4 (`exit 3` → origine applicative) contre un vrai démon, via la fonction shell, et nettoie les conteneurs créés.
- [ ] T038 [P] [US1] Écrire les tests golden `internal/render/explanation_golden_test.go`, qui compare à `testdata/golden/us1/*.golden` en `--color=never` **et** `--color=always`. L'option `-update` régénère les fichiers. Cas couverts : port alloué, image introuvable, démon arrêté, erreur applicative, erreur non reconnue, rendu court en cas de répétition.

### Implementation for User Story 1

Règle commune aux tâches de catalogue T039 à T045, appliquée à chaque entrée :
- concept défini ;
- `Summary` ≤ 160 caractères ;
- au moins une `Fix` ;
- `Deepen` complet (`Explanation`, `Example` exécutable, `Exercise` sans risque) ;
- `DocURL` vers docs.docker.com ;
- `Tests` pointant vers les cas de T035.

- [ ] T039 [P] [US1] Rédiger `internal/catalog/data/network.yaml` (français) : `run.port-already-allocated` (capture `port` ; corrections : `docker ps --filter publish=<port>` et un autre port hôte, en **reprenant l'image de la commande de l'apprenant** via la capture `image`. Le `Deepen.Example` utilise une image de la liste revue de T068, par exemple `nginxinc/nginx-unprivileged:<tag>-alpine`, avec un tag explicite vérifié au moment de la rédaction), `run.invalid-port-spec`, `network.not-found`, `network.active-endpoints`
- [ ] T040 [P] [US1] Rédiger `internal/catalog/data/image.yaml` : `image.not-found`, `image.pull-access-denied`, `image.invalid-reference-format`, `image.manifest-unknown`, `image.in-use` (`rmi`), `image.push-denied`, `image.exec-format-error` (option `--platform`)
- [ ] T041 [P] [US1] Rédiger `internal/catalog/data/container.yaml` : `container.name-conflict`, `container.not-found` (`logs`, `stop`, `start`, `exec`, `rm`), `container.not-running` (`exec`), `container.remove-running` (correction sûre : `docker stop` ; `rm -f` présenté avec mise en garde)
- [ ] T042 [P] [US1] Rédiger `internal/catalog/data/daemon.yaml` et `internal/catalog/data/permission.yaml` :
  - `daemon.not-running` (concept client/démon ; vérification selon Linux, macOS ou WSL2) ;
  - `daemon.no-space-left` ;
  - `permission.socket-denied` : `AntiPattern` pour `sudo chmod 666 /var/run/docker.sock` et pour `sudo docker` systématique. Corrections sûres : mode rootless, ou groupe `docker` en expliquant qu'il donne des droits équivalents à root.
  - `permission.registry-unauthorized`.
- [ ] T043 [P] [US1] Rédiger `internal/catalog/data/volume.yaml` : `volume.bind-source-missing`, `volume.in-use`
- [ ] T044 [P] [US1] Rédiger `internal/catalog/data/build.yaml` : `build.dockerfile-not-found`, `build.copy-source-missing` (concept de contexte de build et `.dockerignore`), `build.run-failed` (étape en échec et code renvoyé)
- [ ] T045 [P] [US1] Rédiger `internal/catalog/data/runtime.yaml`, entrées fondées sur `ExitCodes` (research.md R3) : `runtime.exit-126`, `runtime.exit-127`, `runtime.exit-137` (OOM, option `--memory`), `runtime.exit-143`, `runtime.application-error` (tout autre code de `run` → origine `application`, orienter vers `docker logs`, FR-004)
- [ ] T046 [US1] Implémenter `internal/diagnose/exitcode.go`. Pour `run`, `start` et `exec`, le code 125 donne l'origine `docker` ; 126, 127, 137 et 143 ont leur entrée dédiée ; tout autre code donne l'origine `application`. Pour les autres sous-commandes, le code de sortie n'indique pas l'origine. Intégrer cette logique dans `Diagnose` (`internal/diagnose/diagnose.go`), avant la recherche par motifs.
- [ ] T047 [US1] Implémenter `internal/diagnose/suggest.go` : suggestion d'image à partir d'une liste embarquée d'images officielles populaires (`internal/diagnose/data/official-images.txt` : nginx, alpine, ubuntu, debian, postgres, mysql, redis, node, python, golang, httpd, mongo, busybox, traefik…) avec `textdist.Closest` (distance ≤ 2). Injecter le résultat dans la capture `suggestion` de `image.not-found`. **Aucun appel à `docker images` ni au réseau.**
- [ ] T048 [US1] Implémenter `internal/diagnose/buildstep.go` : dans une sortie `docker build` sur plusieurs étapes, isoler l'étape en échec (`[n/m] RUN …`) et les lignes d'erreur associées, transmises comme captures `step` et `step_error` (edge case « erreur multi-lignes »)
- [ ] T049 [US1] Brancher la détection des répétitions dans `cmd/docker-telemach/explain.go` : `state.IsRepeat` → rendu court qui renvoie à `docker telemach deepen` ; `state.Save` après chaque diagnostic
- [ ] T050 [US1] Mettre au point le catalogue et le matcher jusqu'à ce que T036 (≥ 90 %), T038 et T027 passent, puis générer les fichiers golden de `testdata/golden/us1/`

**Checkpoint**: MVP. Les scénarios V1 à V5, V10 et V11 du quickstart passent ; SC-001 ≥ 90 %.

---

## Phase 4: User Story 2 - Diagnostiquer une erreur Docker Compose dans le fichier YAML (Priority: P2)

**Goal**: localiser et expliquer les erreurs des fichiers Compose (fichier, ligne, clé, service), après une erreur de `docker compose` ou à la demande avec `docker telemach check` (FR-011 à FR-014, FR-018).

**Independent Test**: un jeu de fichiers Compose erronés (une erreur par fichier) donne la bonne ligne, la bonne clé et la bonne correction dans ≥ 90 % des cas (SC-002). Un fichier valide donne `✔ Aucun problème détecté`.

### Tests for User Story 2 ⚠️

- [ ] T051 [P] [US2] Constituer `testdata/compose/` : au moins 20 répertoires, chacun avec son `compose.yaml` (ou ses fichiers multiples) et un `expected.yaml` (`kind`, `file`, `line`, `column`, `service`, `key_path`, `suggestion`). Cas obligatoires :
  - **Syntaxe YAML** :
    - `tab-indent` ;
    - `missing-colon` ;
    - `bad-indentation` ;
    - `duplicate-key`.
  - **Clés inconnues** :
    - `unknown-key-port` (`port` → `ports`) ;
    - `top-level-typo` (`service:` → `services:`) ;
    - `depends-on-typo`.
  - **Types et valeurs** :
    - `wrong-type-ports` (chaîne au lieu d'une liste) ;
    - `wrong-type-environment` ;
    - `invalid-port-format` ;
    - `invalid-restart-value`.
  - **Références et dépendances** :
    - `undefined-depends-on` ;
    - `undefined-network` ;
    - `undefined-volume` ;
    - `dependency-cycle` ;
    - `no-image-no-build`.
  - **Variables** : `unresolved-variable` (`${DB_HOST}`).
  - **Secrets** : `plaintext-password` (`environment: POSTGRES_PASSWORD: …`).
  - **Fichiers** :
    - `multi-file-override` (erreur dans le second fichier `-f`) ;
    - `empty-dir` (aucun fichier Compose) ;
    - `unreadable-encoding` (fichier en UTF-16 ou en Latin-1 avec des octets invalides → `file-unreadable`) ;
    - `unreadable-permissions` (fichier en `0000`, créé par le test lui-même dans un répertoire temporaire, car git ne conserve pas ces droits → `file-unreadable`) ;
    - `valid` (aucun problème).
- [ ] T052 [P] [US2] Écrire `internal/compose/corpus_test.go`. Il analyse chaque cas, compare les `ComposeIssue` à `expected.yaml` et calcule le taux de localisation exacte (ligne et clé) : échec si < 90 % (SC-002). Il vérifie aussi qu'aucune valeur de `plaintext-password` n'apparaît dans les `Excerpt` (SC-006).
- [ ] T053 [P] [US2] Écrire `internal/compose/readonly_test.go` : après analyse, le hash SHA-256 et la date de modification de chaque fichier du cas sont inchangés, et aucun nouveau fichier n'existe dans le répertoire (FR-018)
- [ ] T054 [P] [US2] Écrire `cmd/docker-telemach/check_test.go`, qui couvre les codes de sortie de `contracts/cli.md` § check :
  - `0` pour `valid` ;
  - `1` pour `tab-indent` ;
  - `2` pour `empty-dir`, avec l'explication des noms de fichiers recherchés ;
  - `-f` multiples, chaque problème indiquant son fichier d'origine.
- [ ] T055 [P] [US2] Écrire `tests/integration/compose_test.go` (`//go:build integration`) : scénario V6 (`docker compose up` sur `tab-indent` et `unknown-key-port` via la fonction shell → ligne exacte et correction affichées)

### Implementation for User Story 2

- [ ] T056 [P] [US2] Implémenter `internal/compose/issue.go`. Type `ComposeIssue` (data-model.md) :
  - `Kind` ∈ {`yaml-syntax`, `tab-indent`, `unknown-key`, `wrong-type`, `undefined-reference`, `dependency-cycle`, `unresolved-variable`, `plaintext-secret`, `file-not-found`, `file-unreadable`} ;
  - `File` ;
  - `Line` et `Column` (à partir de 1, 0 si inconnus) ;
  - `Service`, `KeyPath`, `Suggestion` ;
  - `Excerpt` et `Fixed` (`[]Line`) ;
  - `Blocking`.

  Ajouter `func Sort(issues []ComposeIssue)` : bloquants d'abord, puis par fichier, puis par ligne.
- [ ] T057 [P] [US2] Implémenter `internal/compose/locate.go` : `func Locate(dir string, files []string) ([]string, error)`, avec recherche par défaut dans l'ordre `compose.yaml`, `compose.yml`, `docker-compose.yaml`, `docker-compose.yml`. Si aucun fichier n'est trouvé, renvoyer une erreur typée qui produit une `ComposeIssue` `file-not-found`. Si le fichier existe mais n'est pas lisible (droits) ou n'est pas en UTF-8 valide, produire une `ComposeIssue` `file-unreadable` (bloquante) qui explique la cause. Ouverture **en lecture seule** uniquement.
- [ ] T058 [US2] Implémenter `internal/compose/syntax.go`. Première passe avec `go.yaml.in/yaml/v3` (API `Node`) :
  - détection des tabulations d'indentation, avec la ligne exacte ;
  - erreurs de syntaxe YAML, ligne et colonne extraites du message ;
  - clés dupliquées ;
  - construction d'un index `KeyPath` → (ligne, colonne), réutilisé par les passes suivantes.

  Dépend de T056.
- [ ] T059 [US2] Implémenter `internal/compose/keys.go` : extraire la liste des clés autorisées à chaque niveau (racine, `services.*`, `networks.*`, `volumes.*`, `secrets.*`, `configs.*`) du schéma JSON embarqué par compose-go v2.16.1. Détecter les clés inconnues avec une suggestion via `textdist.Closest` (distance ≤ 2), en utilisant l'index de T058.
- [ ] T060 [US2] Implémenter `internal/compose/semantic.go`. Seconde passe avec le chargeur `compose-go/v2` :
  - aucun accès réseau, aucune résolution de chemins distants ;
  - interpolation par une fonction de recherche qui ne fait que noter si chaque variable est **définie** (environnement ou `.env`), sans jamais en conserver la valeur ;
  - conversion des erreurs (types incorrects, références non déclarées, cycles `depends_on`, service sans `image` ni `build`, variables non résolues) en `ComposeIssue`, positionnées grâce à l'index de T058.
- [ ] T061 [US2] Implémenter `internal/compose/secrets.go` : signaler les `plaintext-secret` (nom de variable sensible avec une valeur littérale, selon les règles de R7) avec `Suggestion` = `.env` non versionné ou `secrets:`. Faire passer tous les `Excerpt` et `Fixed` par `redact`.
- [ ] T062 [US2] Implémenter `internal/compose/excerpt.go` (3 lignes de contexte autour de l'erreur et version corrigée par type de problème : remplacer les tabulations par des espaces, renommer la clé suggérée, ajouter le service manquant en commentaire…) et `internal/compose/analyze.go` (`func Analyze(files []string) []ComposeIssue`, qui enchaîne locate → syntax → keys → semantic → secrets → Sort ; si un problème bloquant est trouvé, les passes suivantes ne sont pas lancées)
- [ ] T063 [P] [US2] Rédiger `internal/catalog/data/compose.yaml`, une entrée pédagogique par `Kind` : `compose.yaml-syntax`, `compose.tab-indent`, `compose.unknown-key`, `compose.wrong-type`, `compose.undefined-reference`, `compose.dependency-cycle` (rôle de `depends_on`), `compose.plaintext-secret` (avec `AntiPattern`), `compose.file-not-found`, `compose.file-unreadable` (droits et encodage UTF-8), `compose.unresolved-variable`, `compose.no-project` (`ps`, `logs`, `down` hors d'un projet : rôle du nom de projet et du répertoire courant). Chaque entrée a un concept, une cause, des corrections et un `Deepen`, et ses `Tests` pointent vers des sorties `docker compose` réelles ajoutées à `testdata/errors/compose-*.stderr.txt`.
- [ ] T064 [US2] Implémenter `internal/render/compose.go`. Pour chaque `ComposeIssue` : en-tête `fichier:ligne:colonne`, service et clé ; extrait YAML coloré par `HighlightLines` (ligne fautive surlignée, marqueur de colonne) ; bloc pédagogique tiré de `compose.<kind>` ; version corrigée colorée. Afficher `✔ Aucun problème détecté dans <fichier>` s'il n'y a aucun problème. Ajouter des golden files dans `testdata/golden/us2/` (sans couleur et avec couleur).
- [ ] T065 [US2] Brancher la famille `compose` dans `cmd/docker-telemach/explain.go` : si `cmd.Family == compose`, appeler `compose.Analyze(cmd.ComposeFiles)` et afficher les problèmes trouvés. Si aucun n'est trouvé, revenir au diagnostic par catalogue.
- [ ] T066 [US2] Implémenter `cmd/docker-telemach/check.go` : `docker telemach check [-f <fichier>]... [<répertoire>]`. Codes de sortie `0` (aucun problème), `1` (au moins un problème), `2` (usage invalide ou fichier introuvable). Lecture seule stricte. Sortie colorée selon `render.ResolveColor` (FR-007a). Faire passer T052 à T054.

**Checkpoint**: les scénarios V6 à V8 du quickstart passent ; SC-002 ≥ 90 % ; les User Stories 1 et 2 fonctionnent indépendamment.

---

## Phase 5: User Story 3 - Approfondir un concept à la demande (Priority: P3)

**Goal**: après une explication, offrir un niveau approfondi (concept, exemple exécutable coloré, exercice sans risque) via `docker telemach deepen` (FR-009).

**Independent Test**: après le scénario V1, `docker telemach deepen` puis `docker telemach deepen --exercise` affichent un contenu exact, en français, relié à l'erreur d'origine (quickstart V9).

### Tests for User Story 3 ⚠️

- [ ] T067 [P] [US3] Écrire `cmd/docker-telemach/deepen_test.go` (avec un `XDG_STATE_HOME` temporaire). Cas couverts :
  - sans diagnostic récent : message expliquant comment en obtenir un, code `0` ;
  - avec `LastDiagnosis` pointant sur `run.port-already-allocated` : explication et exemple ;
  - avec `--exercise` : l'exercice s'ajoute ;
  - `EntryID` inconnu du catalogue : traité comme « pas de diagnostic ».
- [ ] T068 [P] [US3] Écrire `internal/catalog/safety_test.go` (FR-010, principe III de la constitution v1.1.0). Il vérifie, pour chaque entrée, `Deepen.Example`, `Deepen.Exercise` et toute `Fix` marquée `Safe: true`. Les valeurs issues de la commande de l'apprenant (gabarits `{{.image}}`…) sont exclues : elles reprennent sa commande et ne sont pas des exemples produits.
  - **Échec bloquant** :
    - présence de `--privileged`, `chmod 666`, `--user root`, `-u 0` ou `--network host` ;
    - secret littéral (détecté par `redact`) ;
    - image absente de la liste revue `internal/catalog/data/example-images.txt`. Ce fichier est à créer : une image par ligne, en commentaire la justification « non-root » ou « minimale » ; une image qui tourne en root par défaut n'y figure que si elle est toujours utilisée avec `--user`.
  - **Avertissement non bloquant** (`t.Logf`, le test reste vert) : image sans tag explicite ou avec le tag `latest`.

  Les éléments bloquants ne sont autorisés que dans une `Fix` marquée `Safe: false` accompagnée d'un `AntiPattern`.

### Implementation for User Story 3

- [ ] T069 [US3] Implémenter `internal/render/deepen.go` : `func RenderDeepen(w io.Writer, e *catalog.CatalogEntry, captures map[string]string, exercise bool, opts Options)`. Afficher le titre du concept, `Explanation`, `Example` coloré par chroma (lexer `bash`, `yaml` ou `docker` selon le contenu), `Exercise` si demandé, puis `DocURL`. Ajouter des golden files dans `testdata/golden/us3/`.
- [ ] T070 [US3] Implémenter `cmd/docker-telemach/deepen.go` : `docker telemach deepen [--exercise]`, qui lit `state.Load()` et retrouve l'entrée par `EntryID` dans le catalogue. Sortie colorée selon `render.ResolveColor` (FR-007a). Faire passer T067.
- [ ] T071 [US3] Relire et compléter le champ `Deepen` de toutes les entrées de `internal/catalog/data/*.yaml` : chaque `Example` est exécutable tel quel et chaque `Exercise` est court, sans risque et vérifiable. Faire passer T068.

**Checkpoint**: le scénario V9 passe ; les trois histoires fonctionnent.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: performance, sécurité transverse, documentation, validation finale

- [ ] T072 [P] Écrire `internal/diagnose/bench_test.go` : banc d'essai du pipeline complet d'`explain` sur le corpus. L'objectif interne est < 200 ms par erreur (SC-003 : < 2 s pour 95 % des cas) ; un test échoue au-delà de 2 s.
- [ ] T073 [P] Écrire `tests/integration/security_test.go` (`//go:build integration`). Il reproduit V12 (`-e DB_PASSWORD=SuperSecret123`) et vérifie que la valeur n'apparaît ni dans la sortie d'`explain`, ni sous `$XDG_STATE_HOME/telemach/` (SC-006). Il vérifie aussi qu'aucune sous-commande ne lance de processus `docker` (SC-007), en plaçant dans `PATH` un faux `docker` qui journalise ses appels : seul `docker --version` est autorisé, une seule fois, par le script `shell-init` (FR-001b, FR-015) ; tout autre appel fait échouer le test.
- [ ] T074 Ajouter à `.github/workflows/ci.yml` :
  - un job `fuzz` (`go test -fuzz=FuzzSanitize -fuzztime=30s ./internal/sanitize`, `FuzzRedact` dans `./internal/redact`, et une cible `FuzzDiagnose` à créer dans `internal/diagnose/fuzz_test.go`) ;
  - un job `integration` (`go test -tags integration ./tests/integration/...` sur `ubuntu-latest`, qui fournit Docker) ;
  - l'exécution de `tests/shell` sous bash et zsh (installer zsh).

  Actions épinglées par SHA.
- [ ] T075 [P] Rédiger `README.md` (français) : présentation pédagogique, installation (`go build -o ~/.docker/cli-plugins/docker-telemach ./cmd/docker-telemach`), activation (`eval "$(docker telemach shell-init zsh)"`), commandes `check` et `deepen`, désactivation (`TELEMACH_DISABLE=1`), couleurs (`--color`, `NO_COLOR`), confidentialité (100 % local, secrets masqués)
- [ ] T076 [P] Rédiger `docs/catalogue.md` (français) : comment ajouter une entrée au catalogue (champs, règles de validation de data-model.md, ajout obligatoire d'un cas de test dans `testdata/errors/`, règles anti-pattern de T068)
- [ ] T077 Exécuter localement `golangci-lint run`, `govulncheck ./...`, `gitleaks detect` et `go test -race ./...`, et corriger tous les signalements. Consigner dans `LESSON-LEARNED.md` tout incident survenu pendant l'implémentation (principe V).
- [ ] T078 Dérouler manuellement les scénarios V1 à V15 de `specs/001-docker-error-guidance/quickstart.md` avec Docker 29.x et zsh puis bash, et noter les écarts éventuels pour correction

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance. **Les gates CI (T005) et la protection de `main` (T010) doivent être en place avant la première fusion de code.**
- **Foundational (Phase 2)** : dépend de la Phase 1 et **bloque toutes les histoires**.
- **US1 (Phase 3)** : dépend de la Phase 2 uniquement.
- **US2 (Phase 4)** : dépend de la Phase 2 uniquement, et non de US1. Il réutilise `render`, `catalog`, `textdist` et `explain`, qui sont fondamentaux.
- **US3 (Phase 5)** : dépend de la Phase 2. Pour être démontrable, il lui faut au moins des entrées de catalogue (US1 ou US2) ; T067 utilise `run.port-already-allocated`, issu de US1.
- **Polish (Phase 6)** : dépend des histoires livrées.

### Dépendances fines notables

- T026 → T027, T028 ; T022 + T026 → T028 ; T024 + T025 → T030
- T028, T029, T030, T032, T033 → T034
- T035 → T036, T039 à T045 (les `Tests` du catalogue référencent le corpus)
- T056 → T058 → T059, T060 → T061, T062 → T064, T065, T066
- T051 → T052, T053, T054
- T069 → T070 ; T068 → T071

### Within Each User Story

- Tests écrits d'abord, et qui échouent
- Catalogue et types → logique d'analyse → rendu → branchement CLI
- Histoire terminée et validée par son checkpoint avant de passer à la priorité suivante

### Parallel Opportunities

- Phase 1 : T004 à T009 en parallèle après T001 à T003 ; T010 en dernier (après T005)
- Phase 2 : tous les tests T011 à T018 en parallèle ; puis T019 à T025, T029, T031 et T032 en parallèle
- US1 : T035 à T038 en parallèle ; puis les sept fichiers de catalogue T039 à T045 en parallèle
- US2 : T051 à T055 en parallèle ; T056, T057 et T063 en parallèle
- Une fois la Phase 2 terminée, US1 et US2 peuvent avancer en parallèle

---

## Parallel Example: User Story 1

```bash
# Tests de US1 en parallèle :
Task: "Constituer le corpus testdata/errors/ (≥ 30 erreurs réelles)"
Task: "Écrire internal/diagnose/corpus_test.go (SC-001 ≥ 90 %)"
Task: "Écrire tests/integration/docker_test.go (V1, V2, V4)"
Task: "Écrire internal/render/explanation_golden_test.go"

# Catalogue de US1 en parallèle (un fichier par catégorie) :
Task: "Rédiger internal/catalog/data/network.yaml"
Task: "Rédiger internal/catalog/data/image.yaml"
Task: "Rédiger internal/catalog/data/container.yaml"
Task: "Rédiger internal/catalog/data/daemon.yaml et permission.yaml"
Task: "Rédiger internal/catalog/data/volume.yaml"
Task: "Rédiger internal/catalog/data/build.yaml"
Task: "Rédiger internal/catalog/data/runtime.yaml"
```

## Parallel Example: User Story 2

```bash
Task: "Constituer testdata/compose/ (≥ 20 cas)"
Task: "Implémenter internal/compose/issue.go"
Task: "Implémenter internal/compose/locate.go"
Task: "Rédiger internal/catalog/data/compose.yaml"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 : Setup, avec les gates CI actives et `main` protégée (T010)
2. Phase 2 : Foundational (capture shell, pipeline et repli « non reconnue »)
3. Phase 3 : US1
4. **STOP and VALIDATE** : quickstart V1 à V5, V10 et V11, SC-001 ≥ 90 %
5. Démontrer à des apprenants (mesure de SC-004)

### Incremental Delivery

1. Setup + Foundational → l'intégration shell fonctionne et toute erreur est au moins reformulée honnêtement
2. + US1 → MVP pour les commandes simples
3. + US2 → Compose, après une erreur et avec `check`
4. + US3 → approfondissement
5. Chaque incrément passe toutes les gates CI avant fusion

---

## Notes

- [P] = fichiers différents, pas de dépendance sur une tâche inachevée
- Le libellé [USx] relie chaque tâche à son histoire (traçabilité avec spec.md)
- Commiter après chaque tâche ou groupe logique, via une pull request soumise aux gates
- Tout incident (CI rouge inattendue, faille, régression) → entrée dans `LESSON-LEARNED.md`
- Hors périmètre : Windows natif/PowerShell, `docker-compose` v1, Kubernetes et Swarm, analyse statique complète des Dockerfile, distribution goreleaser (feature ultérieure)
