---

description: "Liste des tâches pour la feature 003-command-line-analysis"
---

# Tasks: Analyse de la ligne de commande, même en cas de succès

**Input**: Design documents from `/specs/003-command-line-analysis/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-lint.md, quickstart.md

**Tests**: inclus. SC-001 et SC-002 exigent des corpus mesurés (lignes fautives et légitimes),
SC-004 et LL-002 exigent des tests du script shell sous bash **et** zsh, et le principe IV des
tests bloquants en CI. Dans chaque histoire, les tests sont écrits **avant** l'implémentation et
doivent d'abord échouer.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Conventions du projet (à respecter dans chaque tâche)

- Code et identifiants en anglais ; commentaires, messages, catalogue et documentation en
  français, accents compris (principe II).
- Relire `LESSON-LEARNED.md` avant chaque phase (principe V). En particulier : **LL-002** (bash
  et zsh, assertions sur stdout et stderr), **LL-006** (tests shell sans raccourci de PATH),
  **LL-007** (toute nouvelle étiquette de build ajoutée à `.golangci.yml`), **LL-008**
  (validation et commit enchaînés par `&&` uniquement), **LL-009** (vérifications ad hoc en
  `bash -c`, échec si une valeur comparée est vide).
- Toute chaîne issue de la ligne de commande passe par `sanitize.Clean` puis
  `redact.RedactArgs`/`redact.Redact` avant affichage ou écriture sur disque.
- Aucune exécution de commande, aucun accès réseau ; seules lectures locales : présence d'une
  variable d'environnement (`os.LookupEnv`, jamais la valeur affichée ni conservée) et
  existence d'un dossier relatif au dossier courant.
- Images d'exemple du catalogue et des corpus : tag explicite vérifié sur Docker Hub (LL-004 ;
  tags déjà vérifiés : `nginx:1.30-alpine`, `alpine:3.24`, `postgres:18-alpine`,
  `redis:8-alpine`), images non-root quand l'exemple est exécutable.
- Workflow : commits par phase sur la branche `003-command-line-analysis`, une PR, fusion par
  l'utilisateur.

---

## État de reprise (2026-10-15, session interrompue à la demande de l'utilisateur)

- Fait : phases 1 et 2 (T001–T014, commit `ab8c7a1`) ; US1 complète (T015–T021) ; T017 écrit mais non exécuté en local (pas de démon Docker dans WSL), à valider par le job `integration` de la CI.
  Corpus misplaced : 15/15 détectés, 36/36 légitimes silencieux. Règle « image sans tag »
  (piège 4, prévue en T023) déjà écrite dans `internal/lint/pitfalls.go` avec son entrée de
  catalogue (nécessaire à la ligne de l'issue #9).
- T020 terminé (2026-10-16) : `--width 200` dans `TestLintIssue9Phrasing`, golden files
  `cmd/docker-telemach/testdata/golden/issue9-{after,manual}.golden`.
- Décision prise en T021 : la section « ⚠ Dans ta commande — cause probable » est affichée
  **avant** l'explication générique (contrat mis à jour) : l'explication du code 126 seule
  était trompeuse pour `docker exec web -it sh`.
- Corrections de l'analyse intégrées au code : U1 (liste complète des options à valeur de
  Docker 29.8, `command.Locate`), C2/C3 (corpus), I3 (`lint` ne reçoit pas
  `--docker-version`) ; adresses web écartées de la règle des volumes
  (`curl -v https://…`) ; les pièges 1 à 7 ne portent que sur les options lues par Docker
  (avant l'image). Non appliqués (textes de spec, accord requis) : I1, I2, I4.
- Reste : US2 (T022–T026, sauf piège 4 déjà fait), US3 (T027–T030),
  phase 6 (T031–T036).

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 Créer `internal/lint/doc.go` (commentaire de paquet en français : « analyse de la ligne de commande docker, sans exécution, pour repérer les écritures que Docker accepte mais qui ne font pas ce qui est voulu ») et le dossier `testdata/cmdline/` avec un `README.md` décrivant le format des corpus : un cas par ligne, `ATTENDU<TAB>VARIABLES<TAB>DOSSIERS<TAB>ARGUMENTS`, où `ATTENDU` est la liste des identifiants `cmdline.*` attendus séparés par des virgules (ou `-` pour aucun), `VARIABLES` les variables d'environnement présentes (ou `-`), `DOSSIERS` les dossiers existants dans le dossier courant (ou `-`), `ARGUMENTS` la ligne après `docker` (découpée comme le ferait le shell, guillemets simples et doubles pris en charge).

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Tests fondamentaux (écrire d'abord)

- [X] T002 [P] Étendre `internal/catalog/catalog_test.go` : une entrée `fallback: cmdline` sans `patterns` est valide si elle a `severity: warning` ou `severity: info`, un concept, une cause, au moins une correction ou un `anti_pattern`, et un `deepen` ; une entrée `cmdline` sans `severity`, ou avec une autre valeur, est refusée ; `severity` est refusée sur une entrée qui n'est pas `cmdline`.
- [X] T003 [P] Étendre `internal/command/command_test.go` : `create` accepte les mêmes options à valeur que `run` (`docker create -p 8080:80 --name web nginx` → image `nginx`, aucun argument de conteneur) ; `container create` idem.
- [X] T004 [P] Étendre `internal/state/state_test.go` : `LastDiagnosis.Kind` vaut `error` par défaut (fichier existant sans le champ) ; `Findings` (`[]{EntryID, Captures}`) est conservé ; nouveau `HintStore` (fichier `hints.json`) : « `SchemaVersion` 1 », droits `0600`, écriture atomique, **purge à chaque écriture des entrées de plus de 24 h**, **au-delà de 200 entrées, les plus anciennes sont supprimées**, fichier illisible ou d'un autre schéma → ignoré sans erreur ; méthode `Decide(fingerprint string, imageRule bool, now time.Time) Display` qui renvoie `Full`, `Reminder` (complet il y a < 10 min) ou `Silent` (règle `cmdline.image-without-tag`, même empreinte, < 24 h), selon le tableau de `data-model.md` § HintStore.
- [X] T005 [P] Écrire `internal/lint/fix_test.go` : reconstruction d'une commande corrigée (options globales et sous-commande inchangées, options bien placées, **puis** options déplacées dans leur ordre d'origine, puis image/conteneur, puis le reste ; `--` et ce qui suit conservés tels quels) ; citation shell (guillemets simples pour un argument contenant espace, `$`, `` ` ``, `"`, `'`, `;`, `&`, `|`, `<`, `>`, `*`, `?`, `(`, `)` ; échappement de `'` en `'\''`) ; arguments masqués (`-e API_TOKEN=••••••`) et indicateur `HasMaskedSecret`.
- [X] T006 [P] Écrire `internal/render/lint_golden_test.go` (golden files dans `testdata/golden/lint/`) : avertissement complet (⚠ résumé, effet, `📘 Concept`, `💡 Correction`, `➜ Pour aller plus loin`), information (ℹ en deux lignes), rappel d'une ligne (`↺ …`), plusieurs pièges dans un seul affichage (avertissements puis informations), section « ⚠ Dans ta commande — cause probable » ajoutée à une explication d'erreur ; rendu sans couleur et avec couleur ; largeur 60 et 100 (repli propre de la commande corrigée).
- [X] T007 [P] Étendre `tests/shell/shellinit_test.go` (bash et zsh ; faux `docker` et faux `docker-telemach` qui journalise ses appels dans un fichier ; binaire **hors PATH**, LL-006) : garanties 8 à 10 de `contracts/cli-lint.md` — `run`, `create`, `exec`, `container run|create|exec` réussies → **un** appel `lint --after -- <args>` après la commande ; mêmes commandes précédées d'options globales (`--context x`, `-H tcp://h:2375`, `--log-level debug`) → appel aussi ; `ps`, `images`, `pull`, `build`, `compose up` réussies → **aucun** appel ; commande interactive `run -it` réussie → appel après la fin ; stdout, stderr de Docker et code de sortie identiques à une exécution sans TELEMACH ; `TELEMACH_LINT=0` → aucun appel ; `TELEMACH_DISABLE=1` → aucun appel ; échec de `lint` (code 1 ou 2) → code de sortie de Docker inchangé. Adapter `TestSuccessIsNotExplained` (une commande réussie hors `run|create|exec` n'appelle toujours rien).

### Implémentation fondamentale

- [X] T008 [P] Étendre `internal/catalog/schema.go` et `internal/catalog/loader.go` : champ `Severity string \`yaml:"severity"\`` ; valeur `cmdline` acceptée pour `Fallback` (message d'erreur listant `application, compose-issue, cmdline`) ; règles de validation de T002 ; `diagnose.Diagnose` ne doit **jamais** retenir une entrée `cmdline` pour un message d'erreur. Faire passer T002.
- [X] T009 [P] Dans `internal/command/command.go`, donner à `create` les options à valeur de `run` (`valueFlags["create"] = valueFlags["run"]`). Faire passer T003.
- [X] T010 [P] Étendre `internal/state/state.go` (champs `Kind`, `Findings`) et créer `internal/state/hints.go` (`HintStore`, `Display`, `Decide`, `Record`) selon `data-model.md`. Faire passer T004.
- [X] T011 Créer `internal/lint/lint.go` : types `Finding{EntryID, Severity, Captures, Fingerprint}`, `Env{HasVar func(string) bool; DirExists func(string) bool}`, `Analyze(cmd command.Command, args []string, env Env) []Finding` (aucun résultat hors `run|create|exec`, tri : avertissements puis informations, puis ordre d'apparition ; une règle = au plus un Finding par ligne) et le registre des règles (vide pour l'instant) ; créer `internal/lint/fix.go` (reconstruction, citation, masquage). Faire passer T005.
- [X] T012 Ajouter à `internal/render/` (nouveau fichier `lint.go`) : `RenderFindings(w, findings, cat, opts)` (complet), `RenderReminder`, `RenderInline` (section « Dans ta commande » pour `explain`), avec les symboles `⚠`, `ℹ`, `↺` définis dans `theme.go` ; la couleur n'est jamais le seul porteur d'information. Créer des entrées de catalogue factices de test dans `internal/render/testdata/` (pas dans le catalogue réel). Faire passer T006.
- [X] T013 Créer `cmd/docker-telemach/lint.go` : sous-commande visible `lint [--after] -- <args>` selon `contracts/cli-lint.md` ; `TELEMACH_LINT=0` → rien, code 0 ; sous-commande hors `run|create|exec` → rien, code 0 ; aucun Finding → sortie **avant** le chargement du catalogue et du thème (R10) ; sinon décision `HintStore` par Finding (`Full`/`Reminder`/`Silent`), rendu sur stderr, enregistrement de `LastDiagnosis{Kind: cmdline, Findings, Args masqués, RecordedAt}` si au moins un affichage complet ; codes 0/1/2. Brancher la commande dans `rootCommand` (`main.go`) et l'aide française. Tests dans `cmd/docker-telemach/lint_test.go` (TELEMACH_LINT, sous-commande ignorée, codes de sortie, rien sur stdout).
- [X] T014 Modifier `internal/shellinit/scripts/bash.sh` et `zsh.sh` : fonction interne `__telemach_lintable` qui trouve la sous-commande en sautant les options globales de Docker (`-c|--context|-H|--host|--config|-l|--log-level|--tlscacert|--tlscert|--tlskey` avec valeur, formes `--opt=val`, `-D|--debug|--tls|--tlsverify` sans valeur) et reconnaît `run|create|exec` et `container run|create|exec` ; après une commande **réussie** (chemin interactif et non interactif), si `TELEMACH_LINT` ≠ `0` et `__telemach_lintable "$@"`, appeler `command "$__telemach_bin" lint --after --docker-version … -- "$@" || true` ; aucune autre variable globale que `__telemach_*`. Faire passer T007 sous bash **et** zsh. Mettre à jour `specs/001-docker-error-guidance/contracts/cli.md` § shell-init (renvoi vers les garanties 8 à 10).

**Checkpoint**: `docker telemach lint` existe, est appelé au bon moment, et n'a encore aucune règle (aucun affichage).

---

## Phase 3: User Story 1 - Être averti d'une option Docker placée après l'image (Priority: P1) 🎯 MVP

**Goal**: l'exemple de l'issue #9 est signalé avec la commande corrigée, sans aucun faux positif sur les commandes légitimes.

**Independent Test**: corpus `testdata/cmdline/misplaced-*.tsv` (fautif et légitime) ; quickstart V1, V2 et V5.

### Tests for User Story 1 ⚠️

- [X] T015 [P] [US1] Constituer `testdata/cmdline/misplaced-faulty.tsv` (**au moins 10 lignes** : `run -d nginx -p 8090:80`, `run nginx:1.30-alpine -p 8080:80 --name web`, `run alpine:3.24 --rm echo hi`, `create redis:8-alpine -v ./data:/data`, `run postgres:18-alpine -e POSTGRES_PASSWORD=x`, `container run alpine:3.24 -it sh`, `exec web -it sh`, `exec web -e A=1 env`, `run --context x nginx -p 1:2`, une ligne avec `--` après l'image…) et `testdata/cmdline/misplaced-legit.tsv` (**au moins 25 lignes**, dont **au moins 20** passant des options légitimes au programme du conteneur : `run postgres:18-alpine postgres -p 5433`, `run redis:8-alpine redis-server --port 7000`, `run alpine:3.24 ls -d /tmp`, `run alpine:3.24 sh -c 'echo -e x'`, `run curlimages/curl:<tag vérifié> -v https://example.org`, `run nginx:1.30-alpine nginx -g 'daemon off;'`, `run node:<tag vérifié> node -e "1"`, `run python:<tag vérifié> python -m http.server 8000`, `exec web ls -l`, `container run -it alpine:3.24 sh`, `run alpine:3.24 -- -p 8080:80`…). Vérifier chaque tag d'image ajouté sur Docker Hub (LL-004).
- [X] T016 [P] [US1] Écrire `internal/lint/corpus_test.go` : charge tous les `testdata/cmdline/*.tsv` (format du README de T001), construit `Env` à partir des colonnes, compare l'ensemble des identifiants obtenus à `ATTENDU` ; rapport final « N fautifs détectés / M légitimes silencieux » ; échec dès un faux positif (SC-002) ou un oubli (SC-001).
- [X] T017 [P] [US1] Ajouter à `tests/integration/docker_test.go` (étiquette `integration`, vrai démon, intégration shell activée comme dans le README) : `docker run -d nginx:1.30-alpine -p 18090:80` → stdout = identifiant du conteneur, code 0, stderr contient « Option Docker placée après l'image » et la commande corrigée ; exécuter la commande corrigée (après suppression du premier conteneur) et vérifier que `docker port` publie `18090` ; nettoyage des conteneurs.

### Implementation for User Story 1

- [X] T018 [US1] Implémenter `internal/lint/misplaced.go` (research R4) : arguments après l'image (`run`, `create`) ou le conteneur (`exec`), arrêt à `--` ; **signaux forts** — `-p/--publish` au format `[IP:]port[-port]:port[-port][/tcp|udp|sctp]`, `-v/--volume SOURCE:/CHEMIN_ABSOLU[:options]`, `-e/--env NOM=VALEUR` (NOM `^[A-Za-z_][A-Za-z0-9_]*$`), `--env-file`, `--network/--net VALEUR`, `--restart no|always|on-failure[:n]|unless-stopped`, `--mount type=…`, `--rm`, `-it`, `-ti`, `--detach`, `--interactive`, `--tty`, `--privileged` (formes `--opt=val` comprises) ; **signaux faibles** retenus seulement si un signal fort est présent : `-d`, `-i`, `-t`, `--name V`, `-u/--user V`, `-w/--workdir V`, `-m/--memory V` ; pour `exec` : options d'`exec` (`-it`, `-ti`, `-i`, `-t`, `-e`, `-u`, `-w`, `-d`, `--detach`). Captures : `options` (masquées), `image` ou `container`, `fixed_command` (via `fix.go`), `effect` (par exemple « le port 8090 de ta machine n'est pas publié » pour `-p`, « le conteneur ne sera pas supprimé à l'arrêt » pour `--rm`). Faire passer T016 pour les fichiers `misplaced-*`.
- [X] T019 [P] [US1] Ajouter à `internal/catalog/data/cmdline.yaml` les entrées `cmdline.option-after-image` et `cmdline.option-after-container` (`fallback: cmdline`, `severity: warning`, concept « ordre des arguments de docker run » / « de docker exec » avec la forme `docker run [OPTIONS] IMAGE [COMMANDE] [ARG...]`, cause, correction `{{.fixed_command}}`, conseil `docker rm -f <conteneur>` quand `--after`, deepen avec exemple exécutable et exercice) ; mettre à jour `docs/catalogue.md` (section « Pièges de la ligne de commande »).
- [X] T020 [US1] Formulations selon `--after` : au passé dans `lint --after` (« a été transmis au conteneur », « le conteneur a démarré, mais… »), au conditionnel sans `--after` ; capture `when` gérée par `cmd/docker-telemach/lint.go` et utilisée dans les gabarits de `cmdline.yaml`. Golden files de rendu réel dans `cmd/docker-telemach/testdata/golden/` (issue #9 avec et sans `--after`).
- [X] T021 [US1] Intégrer l'analyse à `explain` (FR-010, `contracts/cli-lint.md` § explain) dans `cmd/docker-telemach/explain.go` : si `TELEMACH_LINT` ≠ `0` et au moins un Finding de niveau warning, rendu `RenderInline` après l'explication reconnue, ou à la place du bloc « non reconnue » ; les Findings sont ajoutés à `LastDiagnosis.Findings` ; informations ignorées. Tests dans `cmd/docker-telemach/explain_test.go` : `exec web -it sh` avec le message « exec: "-it": executable file not found in $PATH » → un seul affichage, `-it` présenté comme cause probable, correction `docker exec -it web sh`.

**Checkpoint**: V1, V2, V5 du quickstart passent ; corpus misplaced : 100 % détectés, 0 faux positif.

---

## Phase 4: User Story 2 - Être averti d'autres pièges silencieux (Priority: P2)

**Goal**: les 7 pièges de FR-004 sont signalés, chacun sans ambiguïté.

**Independent Test**: un corpus par piège (fautif et légitimes voisins) ; quickstart V4.

### Tests for User Story 2 ⚠️

- [ ] T022 [P] [US2] Constituer `testdata/cmdline/pitfalls-faulty.tsv` (**au moins 3 lignes par piège**, soit ≥ 21) et `testdata/cmdline/pitfalls-legit.tsv` (**au moins 25 lignes** voisines, sans avertissement) couvrant, avec les colonnes VARIABLES et DOSSIERS : (1) `-v data:/d` avec dossier `data` présent → `cmdline.named-volume-not-dir` ; sans dossier `data` → rien ; `-v ./data:/d`, `-v /abs:/d`, `-v mon-volume:/d` sans dossier → rien ; (2) `-p 80`, `-p 80/udp`, `--publish 443` → `cmdline.random-host-port` ; `-p 8080:80`, `-P` → rien ; (3) `-e DB_HOST` sans variable → `cmdline.env-unset` ; avec variable présente → rien ; `-e DB_HOST=x` → rien ; (4) `run nginx`, `run nginx:latest`, `create library/redis` → `cmdline.image-without-tag` ; `nginx:1.30-alpine`, `nginx@sha256:…` , `registry:5000/app:1.0` → rien ; (5) `--network host -p 8080:80`, `--net=host -P` → `cmdline.ports-ignored-host-network` ; `--network bridge -p 8080:80` → rien ; (6) `-e DB_PASSWORD=Secret123`, `--env API_TOKEN=abc`, `-e AWS_SECRET_ACCESS_KEY=…` → `cmdline.secret-in-command` ; `--env-file .env`, `-e DB_PASSWORD` (variable présente) → rien ; (7) `--privileged`, `-v /var/run/docker.sock:/var/run/docker.sock`, `--mount type=bind,source=/run/docker.sock,target=/s` → `cmdline.privileged`. Plusieurs pièges dans une même ligne : ensemble exact attendu. Le corpus de T016 les charge automatiquement.

### Implementation for User Story 2

- [ ] T023 [US2] Implémenter dans `internal/lint/pitfalls.go` les règles 1 à 7 selon `data-model.md` (conditions exactes du tableau « Entrées de catalogue ») : source de volume `^[A-Za-z0-9][A-Za-z0-9_.-]*$` **et** `env.DirExists(source)` ; `-p` sans `:` ; `-e NOM` sans `=` **et** `!env.HasVar(NOM)` ; image sans tag ni digest (le dernier segment après `/` ne contient pas `:`, pas de `@`) ou `:latest` ; `--network|--net host` (formes `=` comprises) avec `-p|--publish|-P|--publish-all` ; valeur de `-e/--env` modifiée par `redact.Redact` ; `--privileged` ou source de montage `/var/run/docker.sock`/`/run/docker.sock` (`-v` et `--mount source=`/`src=`). Empreintes : identifiant + noms d'options (+ image pour la règle 4), **jamais** de valeur. Faire passer T016 pour les fichiers `pitfalls-*`.
- [ ] T024 [P] [US2] Ajouter les 7 entrées correspondantes à `internal/catalog/data/cmdline.yaml` (niveaux de `data-model.md` ; `cmdline.privileged` avec `anti_pattern` et alternatives `--cap-add` ciblé, API distante protégée ; `cmdline.secret-in-command` : `--env-file` avec droits `0600`, secrets Docker, jamais la valeur ; `cmdline.image-without-tag` : rappel du principe « tag explicite » ; chaque entrée avec deepen exécutable et images taguées non-root) ; mettre à jour `docs/catalogue.md`. Les tests de sécurité du catalogue (`internal/catalog/safety_test.go`) passent.
- [ ] T025 [US2] Brancher la lecture locale dans `cmd/docker-telemach/lint.go` et `explain.go` : `Env.HasVar` = `os.LookupEnv` (présence seule), `Env.DirExists` = `os.Stat` d'un nom **relatif simple** (sans `/` ni `..`) dans le dossier courant, dossier uniquement ; tests avec `t.Setenv` et `t.Chdir` dans `cmd/docker-telemach/lint_test.go`.
- [ ] T026 [US2] Ajouter à `tests/integration/docker_test.go` les cas de quickstart V4 exécutables sur le runner (volume nommé avec dossier `data` dans un dossier temporaire, `-p 80`, variable non définie, image sans tag répétée deux fois → information une seule fois, `--network host -p`, secret jamais présent dans `~/.local/state/telemach` (`grep` vide), `--privileged`) ; nettoyer conteneurs et volume `data`.

**Checkpoint**: corpus pitfalls : 100 % détectés, 0 faux positif ; V4 passe.

---

## Phase 5: User Story 3 - Garder le contrôle sur les avertissements (Priority: P3)

**Goal**: rappel court, désactivation séparée, `last` et `deepen` pour les avertissements.

**Independent Test**: quickstart V1 (répétition, `last`) et V6.

### Tests for User Story 3 ⚠️

- [ ] T027 [P] [US3] Étendre `cmd/docker-telemach/lint_test.go` : même ligne fautive deux fois en < 10 min → rappel d'une ligne avec la commande corrigée et renvoi à `docker telemach last` ; après 10 min → complet ; information « image sans tag » : une fois par image et par 24 h ; un rappel ne réécrit pas `last.json`.
- [ ] T028 [P] [US3] Étendre `cmd/docker-telemach/last_test.go` et `deepen_test.go` : après un avertissement complet, `last` affiche l'en-tête (date, ancienneté, commande masquée) puis le même rendu complet ; `deepen` approfondit le premier piège (deepen de son entrée) ; une erreur plus récente remplace l'avertissement dans `last`.

### Implementation for User Story 3

- [ ] T029 [US3] Compléter `cmd/docker-telemach/last.go` et `deepen.go` pour `LastDiagnosis.Kind == "cmdline"` (rendu `RenderFindings` à partir de `Findings`, sans `--after` dans le libellé si l'ancienneté dépasse 10 min) ; faire passer T027 et T028.
- [ ] T030 [US3] Documenter dans `README.md` : section « Avertissements sur la ligne de commande » (exemple de l'issue #9, les 8 pièges en une ligne chacun, rappel court, `TELEMACH_LINT=0`, `docker telemach lint -- …` pour vérifier une commande sans l'exécuter) ; tableau « Commandes » et § « Désactivation ponctuelle » mis à jour ; `specs/001-docker-error-guidance/contracts/cli.md` renvoie vers `contracts/cli-lint.md`.

**Checkpoint**: US1 à US3 fonctionnent indépendamment.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T031 [P] Ajouter à `tests/integration/docker_test.go` la mesure SC-003 : 40 exécutions de `docker-telemach lint --after -- run -d nginx -p 8090:80` et 40 sans piège → p95 < 50 ms (marge documentée si le runner CI est plus lent : seuil CI 100 ms, seuil local 50 ms) ; rapporter les valeurs dans `specs/003-command-line-analysis/research.md`.
- [ ] T032 [P] `go test -race ./...`, `go test -tags integration ./tests/integration/...` (Docker local), `golangci-lint run` (binaire compilé avec `GOTOOLCHAIN=go1.27.2`, LL-001), tests shell sous bash **et** zsh ; tout vert, validations enchaînées par `&&` (LL-008).
- [ ] T033 Relire la documentation et le catalogue `cmdline.yaml` : français correct et accentué, commandes exécutables telles quelles avec tags explicites, ton pédagogique (« probablement »), aucun secret d'exemple réaliste.
- [ ] T034 Pousser la branche et ouvrir la PR (corps : résumé, corpus SC-001/SC-002 chiffrés, mesures SC-003, « Closes #9 ») ; attendre les 11 checks verts ; l'utilisateur relit et fusionne.
- [ ] T035 Après fusion, proposer à l'utilisateur la publication `v0.3.0` (**action irréversible : confirmation obligatoire**) ; si accord, tag annoté sur `main`, suivi du workflow `release`, vérifications post-publication de la 002 (quickstart 002 V1, V2, V5), puis validation manuelle du quickstart 003 V1 dans le terminal de l'utilisateur.
- [ ] T036 Consigner dans `LESSON-LEARNED.md` tout incident survenu pendant la feature (principe V), dans la PR qui le corrige.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance.
- **Foundational (Phase 2)** : dépend de la Phase 1 et **bloque toutes les histoires**.
- **US1 (Phase 3)** : dépend de la Phase 2.
- **US2 (Phase 4)** : dépend de la Phase 2 ; T023 et T024 partagent `pitfalls.go`/`cmdline.yaml` avec US1 seulement par ajout (pas de conflit fonctionnel) ; T025 dépend de T013.
- **US3 (Phase 5)** : dépend de la Phase 2 ; démontrable dès qu'un piège existe (US1).
- **Polish (Phase 6)** : dépend des histoires livrées ; T035 après fusion (T034).

### Dépendances fines notables

- T002 → T008 ; T003 → T009 ; T004 → T010 ; T005 → T011 ; T006 → T012 ; T007 → T014
- T008 + T010 + T011 + T012 → T013 → T014
- T015 + T016 → T018 ; T018 + T019 → T020 → T021 ; T017 après T014 et T020
- T022 → T023 ; T023 + T024 + T025 → T026
- T027 + T028 → T029 ; T034 → T035

### Within Each User Story

- Tests écrits d'abord, et qui échouent
- Règles (`internal/lint`) → catalogue → branchement CLI → intégration
- Histoire validée par son checkpoint avant la suivante

### Parallel Opportunities

- Phase 2 : tests T002 à T007 en parallèle ; puis T008, T009, T010 en parallèle.
- US1 : T015, T016, T017 en parallèle ; T019 en parallèle de T018.
- US2 : T022 seul d'abord ; T024 en parallèle de T023.
- US3 : T027 et T028 en parallèle.
- Polish : T031 et T032 en parallèle.

---

## Parallel Example: User Story 1

```bash
Task: "Constituer testdata/cmdline/misplaced-faulty.tsv et misplaced-legit.tsv"
Task: "Écrire internal/lint/corpus_test.go"
Task: "Ajouter le scénario issue #9 à tests/integration/docker_test.go"
```

## Parallel Example: User Story 2

```bash
Task: "Implémenter internal/lint/pitfalls.go (règles 1 à 7)"
Task: "Ajouter les 7 entrées à internal/catalog/data/cmdline.yaml"
```

---

## Implementation Strategy

### MVP d'abord (US1)

1. Phases 1 et 2 : `lint` branché, appelé au bon moment, sans règle.
2. Phase 3 (US1) : l'issue #9 est résolue (options après l'image ou le conteneur), y compris en
   cas d'échec (`exec web -it sh`).
3. **S'arrêter et valider** : corpus misplaced sans faux positif, quickstart V1/V2/V5.

### Livraison incrémentale

1. US1 → issue #9 (MVP).
2. US2 → les 7 pièges silencieux.
3. US3 → confort (rappel, `last`, `deepen`, désactivation, documentation).
4. Phase 6 → mesures, PR « Closes #9 », publication `v0.3.0` sur confirmation.

---

## Notes

- [P] = fichiers différents, pas de dépendance sur une tâche inachevée.
- Les tests shell et d'intégration doivent reproduire exactement l'installation documentée
  (LL-006) ; les vérifications ad hoc s'exécutent en `bash -c` (LL-009).
- Commit après chaque tâche ou groupe logique ; messages en français, terminés par la ligne
  d'attribution.
