---

description: "Liste des tâches pour la feature 004-exited-containers-analysis"
---

# Tasks: Analyse des conteneurs arrêtés en erreur

**Input**: Design documents from `/specs/004-exited-containers-analysis/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-ps.md, quickstart.md

**Tests**: inclus. SC-001 à SC-005 exigent un corpus réel, des contrôles d'état avant/après, des
cas avec secrets et des mesures (LL-012) ; LL-002 exige les tests du script shell sous bash
**et** zsh ; le principe IV, des tests bloquants en CI. Dans chaque histoire, les tests sont
écrits **avant** l'implémentation et doivent d'abord échouer.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Conventions du projet (à respecter dans chaque tâche)

- Code et identifiants en anglais ; commentaires, messages, catalogue et documentation en
  français, accents compris (principe II).
- Relire `LESSON-LEARNED.md` avant chaque phase (principe V). En particulier : **LL-002** (bash
  et zsh, assertions sur stdout et stderr), **LL-003** (nettoyage installé avant la création de
  la ressource), **LL-004** (tags d'images vérifiés sur Docker Hub), **LL-006** (tests shell sans
  raccourci de PATH), **LL-008** (validation et commit enchaînés par `&&` uniquement), **LL-009**
  (vérifications ad hoc en `bash -c`, échec si une valeur comparée est vide), **LL-010** (lignes
  légitimes adversariales pour chaque motif), **LL-011** (toute proposition non tranchée recopiée
  en entier dans l'état de reprise), **LL-012** (objectif chiffré → test qui le mesure),
  **LL-013** (état chargé dans le shell versionné).
- **Seul `internal/dockerread` exécute `docker`**, par liste d'arguments (jamais de shell), et
  seulement les lectures de research R2 : `ps`, `inspect`, `logs --tail 200`,
  `events --until`. Toute autre sous-commande est refusée par le code et par un test.
- Toute chaîne venant du démon (journal, nom, commande, image) passe par `sanitize.Clean` puis
  `redact.Redact` (`redact.RedactArgs` pour une commande) avant affichage ou écriture sur disque.
  Les valeurs des variables d'environnement sont écartées **au décodage** (`EnvNames`).
- Images d'exemple du catalogue et du corpus : tag explicite vérifié sur Docker Hub (déjà
  vérifiés : `nginxinc/nginx-unprivileged:1.30-alpine`, `alpine:3.24`, `postgres:18-alpine`,
  `redis:8-alpine`), images non-root quand l'exemple est exécutable (`--user 1000:1000` pour
  `alpine`).
- Workflow : commits par phase sur la branche `004-exited-containers-analysis`, une PR
  (« Closes #8 »), fusion par l'utilisateur ; v0.3.0 seulement après la fusion, sur
  confirmation.

---

## État de reprise

- 2026-10-16 : phases 1 et 2 faites (T001–T011). `docker --context x telemach …` corrigé
  (LL-014) ; `internal/dockerread` (liste fermée, `EnvNames`, délai, erreurs masquées) ;
  catégorie et `fallback: exited` (écartés de `diagnose`) ; `LastDiagnosis` `Kind: exited` ;
  rendu `RenderExited`, `RenderNoExited`, `SignalLine` avec golden files.
- US1 faite (T012–T020) : corpus réel de 20 cas, 20/20 causes correctes ; `docker telemach ps`
  testé sur le vrai démon (état inchangé, aucun secret). Écarts au plan, documentés dans
  research R7 : `OOMKilled` d'abord ; motifs sur les 10 dernières lignes ; pas d'entrée
  `exited.code-125` ; nouvelle entrée `exited.option-after-image` (issue #9 vue après coup,
  deux messages réels selon l'entrypoint). Mesure réelle de `ps` : 1,7 s avec 34 conteneurs
  arrêtés (à étudier en T031).
- Aucune proposition en attente de décision.
- US2 faite (T021–T026) : `watch` (lecture `events` seulement), un fichier de signal par
  conteneur (atomique), crochet d'invite bash et zsh qui préserve `$?` ; vérifié dans de vrais
  terminaux (pseudo-terminal) : signal à l'invite suivante, aucun message de tâche. Ajout au
  lancement : 4 à 5 ms au p95 (SC-005). Écarts : `$EPOCHREALTIME` de zsh a 10 décimales
  (tronquées à 9) ; `TELEMACH_LINT=0` ne désactive plus la détection de `run`.
- US3 et US4 faites (T027–T030) : liste des autres conteneurs (résumé français), désignation
  par nom ou préfixe d'identifiant (vérifié sur le démon), redémarrages et Compose ; `last`
  réaffiche l'analyse sans le démon, `deepen` l'approfondit.
- Reste : phase 7 (T031–T036).

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 Créer `internal/dockerread/doc.go` (« lectures du démon Docker par la CLI de l'apprenant, en lecture seule, liste fermée de sous-commandes »), `internal/exited/doc.go` (« diagnostic d'un conteneur arrêté en erreur, sans accès au démon ») et `testdata/exited/README.md` décrivant le format d'un cas : dossier `testdata/exited/<cas>/` avec `inspect.json` (sortie de `docker inspect`, variables d'environnement réduites aux noms **avant** écriture), `logs.txt` (fin du journal), `expected.yaml` (`entry_id`, `source`: `log|state|code|unknown`, captures attendues) et `case.sh` (commande qui reproduit le cas).

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Tests fondamentaux (écrire d'abord)

- [X] T002 [P] Écrire `cmd/docker-telemach/main_test.go` : `splitPluginArgs` sur `["telemach","ps"]`, `["--context","x","telemach","ps"]`, `["-H","tcp://h:2375","--log-level=debug","telemach","last"]`, `["--tlsverify","telemach","--version"]` → options globales `[--context x]`… et arguments du plugin `[ps]`… ; sans `telemach` (appel direct du binaire) → tout est argument du plugin ; une option globale inconnue avant `telemach` est conservée telle quelle (pas d'échec).
- [X] T003 [P] Écrire `internal/dockerread/read_test.go` avec un faux `docker` (script qui journalise ses arguments dans un fichier, sorties JSON tirées de `testdata/exited/`) : `ListCandidates`, `Inspect`, `Logs`, `Events` produisent **exactement** les arguments de research R2 (options globales en tête) ; toute autre sous-commande demandée au `Runner` est refusée (erreur, aucun appel) ; délai dépassé (faux `docker` qui dort 10 s, délai 5 s) → erreur `ErrTimeout`, processus tué ; code non nul de la CLI → erreur portant stderr **masqué** ; `EnvNames` : `["DB_PASSWORD=Secret123","PATH=/usr/bin","EMPTY="]` → `["DB_PASSWORD","PATH","EMPTY"]` et aucune trace de `Secret123` dans la valeur décodée (`fmt.Sprintf("%#v")` compris).
- [X] T004 [P] Étendre `internal/catalog/catalog_test.go` : une entrée `fallback: exited` est valide avec ou sans `patterns` (motifs de journal) ou `exit_codes`, un concept, une cause, une correction ou un `anti_pattern`, un `deepen` ; `tests` désigne un dossier de `testdata/exited/` qui contient `expected.yaml` ; `diagnose.Diagnose` ne retient **jamais** une entrée `exited` pour un message d'erreur de commande.
- [X] T005 [P] Étendre `internal/state/state_test.go` : `LastDiagnosis.Kind` accepte `exited` ; champs `Container` (nom, identifiant court) et `LogExcerpt` (≤ 20 lignes) conservés ; un fichier sans ces champs reste lisible.
- [X] T006 [P] Écrire `internal/render/exited_golden_test.go` (golden files dans `testdata/golden/exited/`, entrées factices de test) : analyse complète (en-tête « Le conteneur « x » (image) s'est arrêté avec le code N, D après son démarrage (il y a A). », bloc « Fin du journal (k dernières lignes sur 20) », `📘 Concept`, `🔎 Cause probable`, `💡 Corrections`, `➜ Pour aller plus loin`), analyse sans journal (pilote illisible), liste des autres conteneurs, message « aucun conteneur en erreur », signal d'une ligne (avec et sans `--rm`) ; sans et avec couleur ; largeur 60 et 100 ; les lignes du journal ne sont jamais repliées (troncature à la largeur avec `…`).

### Implémentation fondamentale

- [X] T007 Corriger `cmd/docker-telemach/main.go` (research R3) : `splitPluginArgs(args) (global, plugin []string)` qui sépare les options globales de Docker placées avant `telemach` (liste partagée avec `internal/command`, formes `--opt val` et `--opt=val`, booléennes `-D|--debug|--tls|--tlsverify`) ; `app.dockerGlobal` les conserve pour `internal/dockerread`. Faire passer T002. Ajouter l'entrée **LL-014** à `LESSON-LEARNED.md` (bug : options globales devant `telemach` → « unknown command » sur toutes les sous-commandes ; cause : `main` ne testait que `args[0]` ; règle préventive : tester chaque point d'entrée avec les options globales de Docker ; suivi : `main_test.go`).
- [X] T008 Implémenter `internal/dockerread/read.go` et `types.go` (data-model.md § Container, Candidate, LogExcerpt) : `Runner{Bin string; Global []string; Timeout time.Duration}` avec `exec.CommandContext`, liste blanche de sous-commandes vérifiée avant tout appel, délai de 5 s par lecture, stdin vide ; `ListCandidates` (`ps -a --no-trunc --filter status=exited --filter status=restarting --filter status=created --format '{{json .}}'`), `Inspect(ids...)`, `Logs(id, 200)` (stdout et stderr confondus, chaque ligne nettoyée puis masquée), `Events(since, until)` ; type `EnvNames` dont `UnmarshalJSON` ne garde que la partie avant `=` ; extraction du code depuis `Status` par `^(Exited|Restarting) \((\d+)\)`. Faire passer T003.
- [X] T009 [P] Étendre `internal/catalog/schema.go` et `loader.go` : valeur `exited` pour `Fallback` (message d'erreur listant `application, compose-issue, cmdline, exited`) ; validation de `tests` vers `testdata/exited/<cas>/expected.yaml` ; exclusion des entrées `exited` dans `internal/diagnose`. Faire passer T004.
- [X] T010 [P] Étendre `internal/state/state.go` : `KindExited`, `IsExited()`, champs `Container` et `LogExcerpt` (data-model.md § LastDiagnosis, « Jamais de valeur de variable d'environnement »). Faire passer T005.
- [X] T011 Ajouter `internal/render/exited.go` : `RenderExited(w, analysis, opts)`, `RenderNoExited`, `RenderOthers`, `RenderSignal` (une ligne `⚠ …`), réutilisant les libellés et symboles de `theme.go` ; la couleur n'est jamais le seul porteur d'information. Faire passer T006.

**Checkpoint**: `docker --context default telemach --version` fonctionne ; les lectures du démon sont possibles, limitées et testées ; rien n'est encore branché.

---

## Phase 3: User Story 1 - Comprendre pourquoi le dernier conteneur s'est arrêté en erreur (Priority: P1) 🎯 MVP

**Goal**: `docker telemach ps` explique le dernier conteneur arrêté en erreur, sans rien modifier.

**Independent Test**: corpus `testdata/exited/` (≥ 12 cas réels) ; quickstart V1 (partie `ps`), V3, V6.

### Tests for User Story 1 ⚠️

- [X] T012 [P] [US1] Écrire `testdata/exited/capture.sh` (bash, `set -uo pipefail`, piège de nettoyage installé **avant** de créer les conteneurs, LL-003 ; ressources préfixées `telemach-exited-`) et capturer au moins 12 cas (SC-001) : `missing-env-postgres` (`postgres:18-alpine` sans mot de passe), `missing-env-mariadb` (image et tag vérifiés sur Docker Hub, LL-004), `permission-denied`, `address-in-use` (deux processus sur le même port dans un même réseau de conteneur), `file-not-found`, `not-executable` (126), `command-not-found` (127, conteneur `created` avec `State.Error`), `exec-format` (si l'architecture le permet, sinon cas retranscrit et signalé dans le script), `oom` (`--memory 8m`), `stopped` (`docker stop` sur `sleep`, 137 sans OOM), `sigterm-handled` (143), `segfault` (139), `generic-1`, `restart-loop` (`--restart on-failure:3`) ; plus des cas **légitimes voisins** (LL-010) : journal contenant « permission denied » mais code 0 (non retenu par `ps`), journal d'un programme qui affiche « address already in use » puis réessaie et s'arrête avec 1 pour une autre raison. Variables réduites aux noms avant écriture d'`inspect.json`.
- [X] T013 [P] [US1] Écrire `internal/exited/diagnose_test.go` : charge chaque `testdata/exited/<cas>/`, appelle `Diagnose`, compare `entry_id`, `source` et captures à `expected.yaml` ; rapport « N/M causes correctes » ; échec si moins de 90 % (SC-001) **ou** si un cas attendu `unknown` reçoit une cause précise ; ordre de décision de research R7 (journal > état > code > inconnu) vérifié par des cas croisés (journal « permission denied » et `OOMKilled` → journal).
- [X] T014 [P] [US1] Ajouter à `tests/integration/exited_test.go` (étiquette `integration`, vrai démon) : V1 (partie `ps`), V3 et V6 du quickstart ; pour chaque scénario, `docker ps -a --format '{{.ID}} {{.Status}} {{.RunningFor}}'`, `RestartCount` et `docker logs | sha256sum` relevés avant et après `docker telemach ps` → identiques (SC-002) ; `grep -r Secret123` dans l'écran, `~/.local/state/telemach` et le dossier `pending` → vide (SC-003) ; codes de sortie 0/1/2 du contrat ; nettoyage des conteneurs.
- [X] T015 [P] [US1] Ajouter à `cmd/docker-telemach/ps_test.go` (faux `docker` hors PATH du test, LL-006) : seules les lectures de research R2 apparaissent dans le journal des appels (aucun `start`, `stop`, `rm`, `exec`, `pull`, `run`, `kill`, `restart`) ; démon injoignable (faux `docker` qui renvoie « Cannot connect to the Docker daemon ») → explication existante du démon arrêté, code 1 ; aucun conteneur en erreur → message du contrat, code 0.

### Implementation for User Story 1

- [X] T016 [US1] Implémenter `internal/exited/diagnose.go` (research R7, data-model.md § Diagnosis) : motifs des entrées `fallback: exited` appliqués aux lignes du journal, de la plus récente à la plus ancienne (condition facultative sur le code ou l'image) ; puis état (`OOMKilled` → `exited.oom` avec `memory_limit` lisible ; `State.Error` non vide → `exited.start-error`) ; puis `exited.code-<n>` ; sinon `exited.unknown`. Captures : nom, image, code, délai de vie (`FinishedAt − StartedAt`), ancienneté, variable attendue, ligne du journal en cause (masquée). Faire passer T013.
- [X] T017 [P] [US1] Créer `internal/catalog/data/exited.yaml` avec les 16 entrées de data-model.md (§ Entrées de catalogue), chacune avec concept, cause, corrections exécutables (`--env-file` avec droits `0600` et variable `*_FILE` pour `exited.missing-required-env` — jamais de mot de passe en clair ; `--memory` pour `exited.oom` ; `docker logs <nom>` pour `exited.unknown`), `deepen` avec exemple exécutable non-root et tags vérifiés, et `tests` vers les cas de T012 ; mettre à jour `docs/catalogue.md` (valeur `exited`, ajout d'une cause en 4 étapes). Les tests de sécurité du catalogue (`internal/catalog/safety_test.go`) passent.
- [X] T018 [US1] Implémenter `internal/exited/select.go` (research R4) : candidats de `ListCandidates` (codes 0 écartés depuis `Status`, « Created » inspecté et retenu seulement si `ExitCode` ≠ 0 et `Error` non vide), `Inspect` groupé des **25** candidats les plus récemment créés, tri par `FinishedAt` décroissant (dernier arrêt pour `restarting`) ; tests par tables dans `internal/exited/select_test.go`.
- [X] T019 [US1] Créer `cmd/docker-telemach/ps.go` : sous-commande visible `ps [CONTENEUR]` selon `contracts/cli-ps.md` (aide française, codes 0/1/2) ; sans argument, analyse le plus récent ; `Logs(id, 200)`, 20 lignes affichées ; erreur de la CLI → passée à `explain` (diagnostic existant : démon arrêté, permission refusée) ; enregistrement de `LastDiagnosis{Kind: exited, …}` masqué. Brancher dans `rootCommand` (`main.go`). Faire passer T014 et T015.
- [X] T020 [US1] Reporter l'exception dans la feature 001 : `specs/001-docker-error-guidance/spec.md` (FR-015 et SC-007 : « sauf `docker telemach ps` et la vérification après `docker run -d`, voir feature 004 FR-010 »), `specs/001-docker-error-guidance/contracts/cli.md` (renvoi vers `contracts/cli-ps.md`) ; `TestSC007NoCommandExecuted` inchangé pour `explain`, `check`, `lint`.

**Checkpoint**: quickstart V1 (partie `ps`), V3 et V6 passent ; corpus ≥ 90 % ; état des conteneurs inchangé.

---

## Phase 4: User Story 2 - Être prévenu qu'un conteneur lancé en arrière-plan s'est arrêté (Priority: P2)

**Goal**: après un `docker run -d` réussi, une ligne à l'invite suivante signale un arrêt en erreur.

**Independent Test**: tests shell (bash et zsh) et quickstart V1 (signal), V2, V5.

### Tests for User Story 2 ⚠️

- [X] T021 [P] [US2] Écrire `internal/exited/watch_test.go` (research R5) : événements tirés d'une capture réelle (`testdata/exited/events/*.jsonl`, à capturer par `capture.sh`) ; le conteneur lancé est le premier `create` postérieur à `t0` dont l'image est celle de la ligne (et le nom celui de `--name`) ; `die` avec code ≠ 0 → signal ; `die` 0 → rien ; pas de `die` → rien ; `--rm` → signal « … et a été supprimé (--rm) … » avec le code ; deux `create` de la même image → le premier ; `t0` avec virgule (`1792151265,045881`) ou point ; ligne sans `-d`/`--detach` (formes groupées `-dit`, `--detach=true`) → aucune lecture.
- [X] T022 [P] [US2] Étendre `tests/shell/shellinit_test.go` (bash et zsh ; `bash -i -c` et `zsh -i -c` pour une session interactive ; faux `docker-telemach` qui journalise ses appels et la liste de ses descripteurs ouverts ; binaire **hors PATH**, LL-006) : `run -d` réussi → **un** appel `watch --since <t0> --shell-pid <pid> -- <args>` avec stdin, stdout et stderr reliés à `/dev/null`, sans attente (la commande revient en moins de 200 ms même si le faux `watch` dort 5 s) ; `x=$(docker run -d …)` ne bloque pas ; `run` en échec, `ps`, session non interactive, `TELEMACH_WATCH=0`, `TELEMACH_DISABLE=1` → aucun appel ; stdout, stderr et code de sortie de `docker` inchangés ; crochet d'invite : fichier `pending-<pid>` présent → affiché sur stderr puis supprimé, absent → rien et aucun processus lancé (journal des appels vide) ; double `eval` de `shell-init` → un seul crochet (`add-zsh-hook -L precmd`, contenu de `PROMPT_COMMAND` sous ses formes chaîne et tableau) ; au chargement, `pending-<pid mort>` supprimé, `pending-<pid vivant>` conservé ; p95 du coût ajouté à l'invite et au lancement < 50 ms sur 40 mesures (SC-005, LL-012).
- [X] T023 [P] [US2] Ajouter à `tests/integration/exited_test.go` : `watch` appelé directement (comme par le script) après de vrais lancements V1, V2 et V5 → contenu attendu du fichier `pending-<pid>` (signal pour `essai-pg` et pour le `--rm` sorti avec 3, rien pour `nginx` ni pour un `--rm` sorti avec 0) ; avec un `pid` de shell mort → aucun fichier écrit ; seules les lectures `events` apparaissent (aucun `inspect`, aucun `logs`).

### Implementation for User Story 2

- [X] T024 [US2] Implémenter `internal/exited/watch.go` (correspondance `create`/`die`, décision « détaché » à partir du découpage de `internal/command` et de `internal/lint`) et `internal/state/pending.go` (dossier `$XDG_RUNTIME_DIR/telemach/` en `0700`, repli dossier d'état ; fichier `pending-<pid>` en `0600`, ajout atomique ; écriture seulement si `kill(pid, 0)` réussit ; `CleanStale()` des `pid` morts). Faire passer T021.
- [X] T025 [US2] Créer `cmd/docker-telemach/watch.go` : sous-commande **cachée** `watch --since T0 --shell-pid PID -- <args>` selon `contracts/cli-ps.md` (sortie immédiate sans `-d`, `TELEMACH_WATCH=0` ou `TELEMACH_DISABLE=1` ; attente de 3 s ; `Events` ; signal rendu par `RenderSignal`, sans couleur si `NO_COLOR` ; code 0 toujours). Faire passer T023.
- [X] T026 [US2] Modifier `internal/shellinit/scripts/bash.sh` et `zsh.sh` (garanties 11 à 14 du contrat) : relevé de `t0` (`$EPOCHREALTIME`, `zmodload zsh/datetime` en zsh ; vérification désactivée si indisponible) ; après un `run`/`container run` **réussi** en session interactive (`[[ -o interactive ]]`, `[[ $- == *i* ]]`) et si `TELEMACH_WATCH` ≠ `0`, `command "$__telemach_bin" watch … </dev/null >/dev/null 2>&1 &` ; crochet `__telemach_prompt` (zsh : `autoload -Uz add-zsh-hook; add-zsh-hook precmd __telemach_prompt` ; bash : ajout idempotent à `PROMPT_COMMAND` chaîne ou tableau) qui n'appelle aucun programme si `pending-$$` est absent ; nettoyage des `pending-*` de shells morts fait par `shell-init` lui-même au moment où il génère le script (`state.CleanStale`, dans `cmd/docker-telemach/shellinit.go`) ; seules nouvelles variables : `__telemach_*`. Faire passer T022 sous bash **et** zsh. Mettre à jour `specs/001-docker-error-guidance/contracts/cli.md` § shell-init (renvoi aux garanties 11 à 14).

**Checkpoint**: V1 (signal), V2 et V5 passent dans un vrai terminal ; aucune régression des tests shell existants.

---

## Phase 5: User Story 3 - Voir et choisir parmi les conteneurs en erreur (Priority: P3)

**Goal**: liste des autres conteneurs en erreur, désignation par nom, boucles de redémarrage, Compose.

**Independent Test**: quickstart V4 ; cas `restart-loop` et conteneur Compose du corpus.

### Tests for User Story 3 ⚠️

- [X] T027 [P] [US3] Étendre `cmd/docker-telemach/ps_test.go` et `internal/exited/select_test.go` : liste « Autres conteneurs arrêtés en erreur » (au plus 10, du plus récent au plus ancien, nom, code, ancienneté, résumé d'une ligne) ; `ps api` et `ps 3f1c` (préfixe) ; `ps inconnu` → introuvable, noms proches (`textdist`), code 1 ; conteneur en cours d'exécution ou arrêté avec 0 désigné → message clair, code 1 ; `restarting` inclus avec son nombre de redémarrages et la politique ; étiquettes Compose → « projet x, service y ».

### Implementation for User Story 3

- [X] T028 [US3] Compléter `internal/exited/select.go` (désignation par nom ou préfixe, noms proches), `internal/exited/diagnose.go` (captures `restart_count`, `restart_policy`, `compose_project`, `compose_service`) et `internal/render/exited.go` (`RenderOthers`, mentions redémarrage et Compose) ; brancher dans `cmd/docker-telemach/ps.go`. Faire passer T027.

**Checkpoint**: V4 passe.

---

## Phase 6: User Story 4 - Retrouver l'analyse et approfondir (Priority: P4)

**Goal**: `last` et `deepen` après une analyse de `ps`.

**Independent Test**: quickstart V7.

### Tests for User Story 4 ⚠️

- [X] T029 [P] [US4] Étendre `cmd/docker-telemach/last_test.go` et `deepen_test.go` : après `ps`, `last` affiche l'en-tête « analyse enregistrée le … : conteneur <nom> » puis le même rendu, **sans** appel au faux `docker` ; `deepen` approfondit l'entrée retenue ; un signal de `watch` ne remplace pas `last.json` ; l'avertissement d'intégration périmée (issue #7) reste affiché en tête.

### Implementation for User Story 4

- [X] T030 [US4] Compléter `cmd/docker-telemach/last.go` et `deepen.go` pour `Kind == exited`. Faire passer T029.

**Checkpoint**: US1 à US4 fonctionnent indépendamment.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T031 [P] Ajouter à `tests/integration/exited_test.go` la mesure SC-004 : 200 conteneurs arrêtés (dont 20 en erreur) et un journal de 120 000 lignes ; 20 exécutions de `docker telemach ps` → p95 < 2 s (seuil CI documenté si le runner est plus lent) ; rapporter les valeurs dans `specs/004-exited-containers-analysis/research.md` ; nettoyage garanti par un piège installé avant la création (LL-003).
- [ ] T032 [P] Documenter dans `README.md` : section « Comprendre un conteneur qui s'est arrêté » (exemple de l'issue #8, signal après `run -d`, `docker telemach ps [nom]`, `TELEMACH_WATCH=0`, ce qui est lu et ce qui ne l'est jamais) ; tableau « Commandes » ; § « Confidentialité et sécurité » (exception à « n'exécute aucune commande », lectures seules, valeurs des variables jamais lues au-delà du décodage) ; § « Mettre à jour » (relancer ses terminaux pour activer le signal).
- [ ] T033 `go test -race ./...`, `go test -tags integration ./tests/integration/...` (Docker local), `golangci-lint run` (binaire compilé avec `GOTOOLCHAIN=go1.27.2`, LL-001), tests shell sous bash **et** zsh ; tout vert, validations enchaînées par `&&` (LL-008).
- [ ] T034 Relire `internal/catalog/data/exited.yaml`, la documentation et les messages : français correct et accentué, commandes exécutables telles quelles avec tags explicites, ton pédagogique (« probablement »), aucun secret d'exemple réaliste ; validation manuelle du quickstart V1 à V7 dans un vrai terminal (bash et zsh).
- [ ] T035 Passer `spec.md` en `Status: Accepted`, pousser la branche et ouvrir la PR (corps : résumé, corpus SC-001 chiffré, mesures SC-004 et SC-005, exception de sécurité, « Closes #8 ») ; attendre les 11 checks verts ; l'utilisateur relit et fusionne.
- [ ] T036 Après fusion, proposer à l'utilisateur la publication `v0.3.0` (**action irréversible : confirmation obligatoire**) ; si accord, tag annoté sur `main`, suivi du workflow `release`, vérifications post-publication de la 002 (quickstart 002 V1, V2, V5), puis validation de l'exemple de l'issue #8 par l'utilisateur (SC-006). Consigner dans `LESSON-LEARNED.md` tout incident survenu pendant la feature (principe V).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance.
- **Foundational (Phase 2)** : dépend de la Phase 1 et **bloque toutes les histoires**.
- **US1 (Phase 3)** : dépend de la Phase 2.
- **US2 (Phase 4)** : dépend de la Phase 2 ; indépendante de US1 pour le code (`watch` n'utilise
  ni `inspect` ni `logs`), mais son signal renvoie à `ps` : livrer US1 d'abord.
- **US3 (Phase 5)** : dépend de US1 (`ps.go`, `select.go`).
- **US4 (Phase 6)** : dépend de US1 (`LastDiagnosis` exited).
- **Polish (Phase 7)** : dépend des histoires livrées ; T036 après fusion (T035).

### Dépendances fines notables

- T002 → T007 ; T003 → T008 ; T004 → T009 ; T005 → T010 ; T006 → T011
- T012 → T013, T017 (cas et `tests` du catalogue) ; T012 → T021 (événements capturés)
- T013 → T016 ; T016 + T017 + T018 → T019 ; T014 + T015 → T019
- T021 → T024 → T025 ; T022 → T026 ; T023 après T025
- T027 → T028 ; T029 → T030 ; T035 → T036

### Within Each User Story

- Tests écrits d'abord, et qui échouent
- Données capturées (`testdata/exited/`) → diagnostic pur (`internal/exited`) → catalogue →
  branchement CLI → intégration
- Histoire validée par son checkpoint avant la suivante

### Parallel Opportunities

- Phase 2 : tests T002 à T006 en parallèle ; puis T009 et T010 en parallèle de T007/T008.
- US1 : T012 à T015 en parallèle ; T017 en parallèle de T016 et T018.
- US2 : T021, T022, T023 en parallèle.
- Polish : T031 et T032 en parallèle.

---

## Parallel Example: User Story 1

```bash
Task: "Écrire testdata/exited/capture.sh et capturer les 12 cas"
Task: "Écrire internal/exited/diagnose_test.go"
Task: "Ajouter les scénarios ps à tests/integration/exited_test.go"
Task: "Ajouter cmd/docker-telemach/ps_test.go (lectures autorisées seulement)"
```

## Parallel Example: User Story 2

```bash
Task: "Écrire internal/exited/watch_test.go"
Task: "Étendre tests/shell/shellinit_test.go (watch et crochet d'invite)"
Task: "Ajouter les scénarios watch à tests/integration/exited_test.go"
```

---

## Implementation Strategy

### MVP d'abord (US1)

1. Phases 1 et 2 : options globales corrigées, lectures du démon limitées et testées.
2. Phase 3 (US1) : `docker telemach ps` répond à l'issue #8 pour qui connaît la commande.
3. **S'arrêter et valider** : corpus ≥ 90 %, état inchangé, aucun secret.

### Livraison incrémentale

1. US1 → `ps` (MVP).
2. US2 → signal après `run -d` (choix C de la clarification).
3. US3 → liste, désignation, redémarrages, Compose.
4. US4 → `last` et `deepen`.
5. Phase 7 → mesures, documentation, PR « Closes #8 », puis `v0.3.0` sur confirmation.

---

## Notes

- [P] = fichiers différents, pas de dépendance sur une tâche inachevée.
- Les tests shell et d'intégration reproduisent exactement l'installation documentée (LL-006) ;
  les vérifications ad hoc s'exécutent en `bash -c` (LL-009).
- Commit après chaque tâche ou groupe logique ; messages en français, terminés par la ligne
  d'attribution.
