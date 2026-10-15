---

description: "Liste des tâches pour la feature 002-distribution"
---

# Tasks: Distribution du plugin sans Go

**Input**: Design documents from `/specs/002-distribution/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (release-assets.md,
image-cli.md, image-tags.md), quickstart.md

**Tests**: inclus. Le principe IV exige des tests bloquants en CI, FR-021 exige que les
procédures documentées soient testées de bout en bout (LL-006), et les refus de `install`
(FR-014) sont des comportements de sécurité. Dans chaque histoire, les tests sont écrits
**avant** l'implémentation et doivent d'abord échouer.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Conventions du projet (à respecter dans chaque tâche)

- Projet Go unique : `cmd/docker-telemach/` (plugin, inchangé), `cmd/telemach-image/`
  (nouveau point d'entrée de l'image), `internal/…`, `tests/` (voir plan.md).
- **Code et identifiants en anglais ; commentaires, messages, notes de Release et
  documentation en français**, accents compris (principe II).
- Relire `LESSON-LEARNED.md` avant de commencer chaque phase (principe V).
- Versions et SHA des actions : **uniquement** ceux du tableau de research.md (vérifiés le
  2026-10-15). Toute action ou outil supplémentaire : vérifier la dernière version stable réelle
  et l'épingler par SHA complet (LL-004).
- Tout outil Go lancé par `go install …@version` ou `go run …@version` reçoit
  `GOTOOLCHAIN=go1.27.2` (LL-001).
- Workflows : `permissions` minimales déclarées par job ; jamais d'expression `${{ … }}` venant
  d'un tag, d'une entrée ou d'un nom de branche directement dans un bloc `run:` (passer par
  `env:`).
- Le jeton Docker Hub n'est jamais lu, affiché ni manipulé hors du job `release` ; Claude Code
  ne le manipule jamais.
- Actions visibles de l'extérieur (push, PR, réglages du dépôt, lancement d'une publication) :
  **demander confirmation à l'utilisateur** avant de les exécuter.
- Infrastructure déjà en place (2026-10-15) : dépôt Docker Hub public
  `telemachlearning/telemach` ; environnement GitHub `release` restreint aux tags `v*`, avec la
  variable `DOCKERHUB_USERNAME=telemachlearning` et le secret `DOCKERHUB_TOKEN` ; releases
  immuables activées.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Outils locaux et squelette du nouveau point d'entrée

- [ ] T001 Installer GoReleaser en local pour les essais, compilé avec la chaîne d'outils du projet (LL-001) : `GOBIN=<scratchpad>/bin GOTOOLCHAIN=go1.27.2 go install github.com/goreleaser/goreleaser/v2@v2.18.3`, puis `goreleaser --version`. Vérifier que `docker` répond dans WSL (`docker version`) ; sinon demander à l'utilisateur d'activer l'intégration WSL de Docker Desktop, et noter que les essais d'image se feront dans la CI (T009).
- [ ] T002 [P] Créer `cmd/telemach-image/doc.go` : commentaire de paquet en français (« point d'entrée de l'image telemachlearning/telemach : installe le plugin sur la machine hôte ou exécute le plugin linux embarqué »).
- [ ] T003 [P] Créer `.dockerignore` à la racine : tout exclure (`*`) sauf ce que le `Dockerfile` copie (le contexte réel est préparé par GoReleaser ; le fichier protège les constructions locales et satisfait le principe III).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Configuration GoReleaser, image minimale et gates des PR. Ces éléments servent à toutes les histoires.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Tests fondamentaux (écrire d'abord)

- [ ] T004 [P] Écrire `cmd/telemach-image/main_test.go` :
  - aucun argument, `--help`, `-h` et `help` affichent l'aide en français, code 0 ; l'aide contient les trois commandes canoniques de `contracts/image-cli.md` § « Usages documentés » (essai, installation Linux, installation macOS), avec le tag de version `<version>` remplacé par la version du binaire ;
  - toute autre commande (`check /projet`, `--version`) est transmise **telle quelle** au binaire cible : la fonction d'exécution est injectable (`exec func(path string, argv []string, env []string) error`), le test vérifie le chemin `/opt/telemach/linux/docker-telemach`, `argv[0]` et les arguments.

### Implémentation fondamentale

- [ ] T005 Implémenter `cmd/telemach-image/main.go` et `cmd/telemach-image/messages.go` : variable `version` injectée par `-X main.version=…` (défaut `dev`) ; aiguillage `help`/`install`/autre ; exécution par `syscall.Exec` du binaire linux avec les mêmes arguments et l'environnement courant ; aide en français rendue avec `internal/render` (mêmes couleurs, symboles et règles `NO_COLOR`/terminal que le plugin). `install` renvoie provisoirement « non implémenté » (code 2), complété en US2. Faire passer T004.
- [ ] T006 Créer `Dockerfile` à la racine (contexte GoReleaser `dockers_v2`, research R5/R6) :
  ```dockerfile
  # syntax n'est pas nécessaire : instructions standard uniquement
  FROM scratch
  ARG TARGETPLATFORM
  ARG TARGETARCH
  COPY $TARGETPLATFORM/telemach-image /usr/local/bin/telemach-image
  COPY $TARGETPLATFORM/docker-telemach /opt/telemach/linux/docker-telemach
  COPY darwin/$TARGETARCH/docker-telemach /opt/telemach/darwin/docker-telemach
  USER 65532:65532
  ENTRYPOINT ["/usr/local/bin/telemach-image"]
  ```
  Les labels OCI (`org.opencontainers.image.source`, `.version`, `.revision`, `.licenses=Apache-2.0`, `.title`, `.description`, `.url=https://www.telemach-learning.fr`) sont posés par GoReleaser (T007). Pas de `VOLUME`, pas de shell, aucun `RUN`. Vérifier avec hadolint v2.15.1 en local.
- [ ] T007 Créer `.goreleaser.yaml` (GoReleaser OSS v2.18.3, `version: 2`), conforme à `contracts/release-assets.md` et `contracts/image-tags.md` :
  - `builds` : id `docker-telemach` (`./cmd/docker-telemach`, `goos: [linux, darwin]`, `goarch: [amd64, arm64]`) et id `telemach-image` (`./cmd/telemach-image`, `goos: [linux]`, `goarch: [amd64, arm64]`) ; pour les deux : `env: [CGO_ENABLED=0]`, `flags: [-trimpath]`, `ldflags: ["-s -w -buildid= -X main.version={{ .Version }}"]`, `mod_timestamp: "{{ .CommitTimestamp }}"` ;
  - `archives` : id `docker-telemach`, `ids: [docker-telemach]`, `formats: [binary]`, `name_template: "docker-telemach_{{ .Version }}_{{ .Os }}_{{ .Arch }}"` (le binaire `telemach-image` n'est **pas** publié dans la Release) ;
  - `checksum` : `name_template: checksums.txt`, algorithme sha256 ;
  - `sboms` : `artifacts: binary`, `ids: [docker-telemach]`, documents `{{ .ArtifactName }}.sbom.json` (Syft) ;
  - `signs` : `cmd: cosign`, `artifacts: checksum`, `signature: "${artifact}.sigstore.json"`, `args: ["sign-blob", "--bundle=${signature}", "${artifact}", "--yes"]` ;
  - `dockers_v2` : `images: [docker.io/telemachlearning/telemach]`, `ids: [docker-telemach, telemach-image]`, `platforms: [linux/amd64, linux/arm64]`, `tags: ["{{ .Version }}", "{{ if not .Prerelease }}{{ .Major }}.{{ .Minor }}{{ end }}", "{{ if not .Prerelease }}latest{{ end }}"]`, `sbom: true`, labels OCI de T006 ;
  - `release` : `draft: true`, `prerelease: auto`, `replace_existing_draft: false`, `header`/`footer` en français selon la structure de `contracts/release-assets.md` (commandes d'installation et de vérification avec `{{ .Version }}`, commande d'installation par l'image `telemachlearning/telemach:{{ .Version }}`) ;
  - `changelog` : `use: git`, groupes « Nouveautés » (`^feat`), « Corrections » (`^fix`), « Autres changements » ; exclure `^docs`, `^ci`, `^test`.
  Valider avec `goreleaser check`.
- [ ] T008 Vérifier les **risques R6** avec `goreleaser release --snapshot --clean` (en local si Docker répond, sinon dans la CI via T009) et consigner le résultat dans `specs/002-distribution/research.md` § R6 :
  (a) le contexte `dockers_v2` contient bien `darwin/<arch>/docker-telemach` et l'image contient les trois fichiers attendus (`docker create` + `docker export | tar -t`) ;
  (b) un modèle de tag vide (pré-version) est ignoré : snapshot avec un tag de pré-version factice dans un **clone jetable** (jamais dans le dépôt).
  **Si (a) ou (b) échoue** : appliquer le repli R6 (image construite par `docker/build-push-action` v7.4.0 `c3c9e263c25d99ce0380d002d59b67737d91b0dc` sur un contexte préparé à partir de `dist/`, mêmes binaires) et adapter T006, T007, T009 et T023 en conséquence.
- [ ] T009 Modifier `.github/workflows/ci.yml` :
  - déclencheurs : retirer `tags: ['v*']` de `push` (la CI d'un tag est exécutée par `release.yml`) ; ajouter `workflow_call:` ;
  - nouveau job `goreleaser-check` (`contents: read`) : checkout (`fetch-depth: 0`), setup-go, installation de Syft via `anchore/sbom-action/download-syft` (SHA de research), `goreleaser-action` (SHA de research, `version: v2.18.3`) avec `args: check`, puis `args: release --snapshot --clean --skip=sign,docker` (les tests du contrat des artefacts sont branchés sur ce job en T013) ;
  - nouveau job `image-scan` (`contents: read`) : construction de l'image `linux/amd64` à partir d'un snapshot (setup-qemu, setup-buildx, `goreleaser release --snapshot --clean --skip=sign,sbom`, ou le repli de T008), puis `anchore/scan-action` (SHA de research) avec `image: <image snapshot>`, `fail-build: true`, `severity-cutoff: high`, `only-fixed: false` ;
  - garder le job `build` existant.
- [ ] T010 Pousser la branche `002-distribution` et ouvrir une PR **en brouillon** pour valider les nouvelles gates avant d'écrire le reste du code (LL-005, quickstart V0). **Demander confirmation avant le push.** Attendu : `goreleaser-check`, `image-scan`, `dockerfile-lint` (qui analyse désormais le `Dockerfile`) et les gates existantes au vert. Toute gate rouge sans lien avec le code → entrée dans `LESSON-LEARNED.md`.

**Checkpoint**: snapshot complet (4 binaires, image 2 architectures) construit et scanné dans la CI.

---

## Phase 3: User Story 1 - Installer le plugin depuis une Release, sans Go (Priority: P1) 🎯 MVP

**Goal**: chaque tag `vX.Y.Z` publie une Release signée et attestée contenant les quatre binaires.

**Independent Test**: sur une machine sans Go, suivre uniquement le README (téléchargement, `sha256sum --check`, `cosign verify-blob`, installation) puis `docker telemach --version` affiche la version de la Release (quickstart V2).

### Tests for User Story 1 ⚠️

- [ ] T011 [P] [US1] Écrire `tests/release/dist_test.go` (étiquette de build `dist`, dossier lu dans `DIST_DIR`, `t.Skip` si absent), conforme à `contracts/release-assets.md` :
  - exactement quatre binaires nommés `docker-telemach_<version>_<os>_<arch>` pour `linux|darwin` × `amd64|arm64`, et aucun binaire `telemach-image` parmi les artefacts de la Release ;
  - `checksums.txt` couvre chaque binaire et chaque SBOM, sommes recalculées identiques ;
  - un SBOM `….sbom.json` par binaire, JSON SPDX valide (champ `spdxVersion`) ;
  - le binaire de la plateforme courante affiche `TELEMACH PLUGIN DOCKER <version>` en première ligne de `--version` ;
  - `debug/buildinfo.ReadFile` sur chaque binaire (y compris darwin) : chemin de module `github.com/aboigues/telemach-plugin-docker`, réglage `-ldflags` contenant `-X main.version=<version>`.
- [ ] T012 [P] [US1] Écrire `tests/release/reproducible_test.go` (étiquette `dist`) : reconstruit `docker-telemach` linux/amd64 avec exactement les options de `.goreleaser.yaml` (T007) et compare sa somme à celle de `checksums.txt` (FR-005, quickstart V6).

### Implementation for User Story 1

- [ ] T013 [US1] Brancher T011 et T012 dans le job `goreleaser-check` de `.github/workflows/ci.yml` (étape `go test -tags dist ./tests/release/...` avec `DIST_DIR=dist`) et les faire passer.
- [ ] T014 [US1] Créer `.github/workflows/release.yml` (partie binaires) :
  - `name: release` ; déclencheurs `push: tags: ['v*']` et `workflow_dispatch:` **sans entrée** (lancé avec `--ref vX.Y.Z`, research R9) ; `permissions: {}` au niveau du workflow ; `concurrency: { group: release-${{ github.ref }}, cancel-in-progress: false }` ;
  - job `verify` (`contents: read`) : contrôles minimaux pour cette histoire (`GITHUB_REF` commence par `refs/tags/v` et respecte `^refs/tags/v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`, lu via `env:`) ; les autres contrôles sont ajoutés en US4 ;
  - job `gates` : `uses: ./.github/workflows/ci.yml` (permissions `contents: read`) ;
  - job `release` (`needs: [verify, gates]`, `environment: release`, permissions `contents: write`, `id-token: write`, `attestations: write`) : checkout (`fetch-depth: 0`, `persist-credentials: false`), setup-go, `cosign-installer` (SHA de research), installation de Syft, `goreleaser-action` (`args: release --clean`, `GITHUB_TOKEN` fourni par `secrets.GITHUB_TOKEN`), puis `actions/attest` (SHA de research) avec `subject-path: dist/docker-telemach_*` (binaires seulement), puis publication du brouillon : `gh release edit "$TAG" --draft=false` (`TAG` lu depuis `GITHUB_REF_NAME` via `env:`).
  Pour cette histoire, désactiver la partie image (`--skip=docker`) ; elle est activée en US2 (T026).
- [ ] T015 [US1] Finaliser l'en-tête et le pied de page des notes de Release dans `.goreleaser.yaml` (`release.header`, `release.footer`) : commandes Linux et macOS **identiques** à celles du README (T016), vérification `sha256sum`/`shasum -a 256`, commande `cosign verify-blob` et `gh attestation verify` exactement comme dans `contracts/release-assets.md` § Garanties.
- [ ] T016 [US1] Réécrire la section « Installation » de `README.md` dans l'ordre exigé par FR-020 ; dans cette histoire, rédiger « Installer depuis une Release (recommandé) » : variables `VERSION=0.1.0`, `OS` et `ARCH` détectés (`uname -s | tr A-Z a-z`, `uname -m` converti `x86_64→amd64`, `aarch64|arm64→arm64`), téléchargement en ligne de commande (`curl -fsSLO`) du binaire et de `checksums.txt`, vérification de somme (Linux `sha256sum --check --ignore-missing`, macOS `shasum -a 256 --check --ignore-missing`), vérification facultative de la signature (cosign v3.1.3) et de la provenance (`gh`), copie dans `~/.docker/cli-plugins/docker-telemach`, `chmod +x`, `docker telemach --version`. Ajouter l'avertissement sur le téléchargement par navigateur sous macOS (Edge Cases). Déplacer la procédure actuelle sous « Compiler depuis les sources » en dernier.
- [ ] T017 [US1] Mettre à jour `specs/002-distribution/quickstart.md` § V6 avec les options exactes de `.goreleaser.yaml`.

**Checkpoint**: un tag de pré-version poussé sur un dépôt de répétition (ou la première Release en Phase 7) produit une Release conforme au contrat, vérifiable avec les commandes du README.

---

## Phase 4: User Story 2 - Installer le plugin avec Docker seulement (Priority: P2)

**Goal**: `docker run … telemachlearning/telemach:X.Y.Z install` installe le bon binaire, avec des refus pédagogiques sûrs.

**Independent Test**: quickstart V3 sur Linux (et macOS) : commande documentée, fichier appartenant à l'utilisateur, `docker telemach --version` correct ; tous les refus du tableau V3.

### Tests for User Story 2 ⚠️

- [ ] T018 [P] [US2] Écrire `cmd/telemach-image/install_test.go`, une table de cas couvrant **toutes** les lignes du tableau « Installation » de `data-model.md`. Les accès système sont injectables (racine des sources `/opt/telemach`, dossier de destination, lecteur de `/proc/self/mountinfo`, `os.Getuid`, `HOST_OS`) ; les binaires sources et existants sont de vrais petits binaires Go compilés dans `TestMain` avec `-ldflags "-X main.version=…"` et le chemin de module du projet (pour `debug/buildinfo`). Cas :
  - nouvelle installation : fichier `docker-telemach` en `0755`, contenu identique à la source, message de succès, code 0 ;
  - mise à jour : message « Version remplacée : 0.1.0 », code 0 ; même version : message « déjà à jour », code 0 ;
  - `HOST_OS` absent → linux ; `darwin`/`DARWIN` → binaire darwin ; `windows` → code 2, renvoi WSL2 ; valeur inconnue → code 2 ;
  - `/sortie` non monté → code 1, rien d'écrit, concept « bind mount » ;
  - `/sortie` appartenant à root, processus non-root → code 1, correction `sudo chown "$(id -u):$(id -g)" ~/.docker/cli-plugins` ;
  - `/sortie` non accessible en écriture → code 1, explication de `--user "$(id -u):$(id -g)"` ;
  - fichier existant non TELEMACH, lien symbolique, dossier nommé `docker-telemach` → code 1, rien de modifié ;
  - écriture atomique : en cas d'échec de copie simulé, aucun fichier partiel ni temporaire ne reste dans `/sortie` ;
  - argument inconnu après `install` → aide de `install`, code 2 ; `install --help` → code 0.
  Les messages sont vérifiés par fichiers golden (sans couleur) dans `cmd/telemach-image/testdata/golden/`.

### Implementation for User Story 2

- [ ] T019 [US2] Implémenter `cmd/telemach-image/install.go` selon research R7 et `contracts/image-cli.md` : détection du montage via `/proc/self/mountinfo` ; contrôles propriétaire root / écriture / fichier existant (`os.Lstat`, refus des liens symboliques et des fichiers non TELEMACH via `debug/buildinfo.ReadFile`) ; lecture des versions (source et existante) dans le réglage `-ldflags` (`-X main.version=…`) ; copie dans un fichier temporaire de `/sortie` (`os.CreateTemp`), `Chmod 0755`, `Sync`, `Rename`, suppression du temporaire en cas d'échec ; architecture affichée à partir de `runtime.GOARCH`. Faire passer T018.
- [ ] T020 [US2] Rédiger les messages de `install` dans `cmd/telemach-image/messages.go` : succès et chaque refus exactement selon la structure de `contracts/image-cli.md` (✔/✖, `📘 Concept`, `💡 Correction` avec la commande complète à copier, tag `X.Y.Z` explicite, `➜ Étape suivante` avec `shell-init` pour zsh et bash) ; libellés d'architecture lisibles (« Apple Silicon — darwin/arm64 », « Intel — darwin/amd64 », « x86-64 — linux/amd64 », « ARM64 — linux/arm64 »). Mettre à jour les fichiers golden.
- [ ] T021 [P] [US2] Écrire `tests/integration/image_test.go` (étiquette `integration`, `t.Skip` sans démon Docker) qui reproduit **à la lettre** les commandes du README (LL-006) : préparer un contexte au format `dockers_v2` (binaires compilés par `go build` avec les options de T007 pour la plateforme courante et darwin de même architecture, sous `linux/<arch>/` et `darwin/<arch>/`), `docker buildx build --load -t telemach-test:<version>` ; dans un `HOME` temporaire : `mkdir -p ~/.docker/cli-plugins` puis `docker run --rm --user "$(id -u):$(id -g)" -v ~/.docker/cli-plugins:/sortie telemach-test:<version> install` ; vérifier propriétaire = utilisateur, droits `0755`, `--version` du fichier installé ; puis les refus du tableau quickstart V3 (sans montage, sans `--user`, dossier créé par Docker, `HOST_OS=windows`, fichier étranger). Nettoyer l'image à la fin.
- [ ] T022 [US2] Ajouter dans `release.yml`, job `verify`, le contrôle « tag d'image absent » : `curl -fsS https://hub.docker.com/v2/repositories/telemachlearning/telemach/tags/$VERSION/` doit répondre 404 (API publique, sans secret), sinon échec « image X.Y.Z déjà publiée ».
- [ ] T023 [US2] Activer la partie image dans `release.yml`, job `release` : setup-qemu et setup-buildx (SHA de research) ; `docker/login-action` (SHA de research) avec `username: ${{ vars.DOCKERHUB_USERNAME }}` et `password: ${{ secrets.DOCKERHUB_TOKEN }}` ; retirer `--skip=docker` de GoReleaser.
- [ ] T024 [US2] Dans `release.yml`, après GoReleaser : récupérer le digest de l'index publié (`docker buildx imagetools inspect docker.io/telemachlearning/telemach:$VERSION --format '{{json .Manifest}}'` → `digest`), signer par digest (`cosign sign --yes docker.io/telemachlearning/telemach@$DIGEST`) et attester (`actions/attest` avec `subject-name: index.docker.io/telemachlearning/telemach`, `subject-digest: $DIGEST`, `push-to-registry: true`). Ajouter `packages: write` **uniquement si** `actions/attest` l'exige pour un registre externe (vérifier sa documentation à ce moment-là ; sinon ne rien ajouter).
- [ ] T025 [US2] Rédiger dans `README.md` la section « Installer avec Docker seulement » : commandes Linux/WSL2 et macOS **identiques** à `contracts/image-cli.md` (`mkdir -p` d'abord, `--user`, tag `0.1.0` explicite), explication courte de chaque option (pédagogie : bind mount, identité, `HOST_OS`), mise à jour, rappel sur `latest` (principe III).
- [ ] T026 [P] [US2] Créer `docs/dockerhub.md` : texte en français de la page Docker Hub (présentation, deux usages, vérification de signature `cosign verify` de `contracts/image-tags.md`, lien vers le dépôt et https://www.telemach-learning.fr). L'utilisateur le copie lui-même dans Docker Hub.

**Checkpoint**: `tests/integration/image_test.go` vert dans la CI ; l'image snapshot installe correctement le plugin.

---

## Phase 5: User Story 3 - Essayer TELEMACH sans rien installer (Priority: P3)

**Goal**: `docker run --rm -t -v "$PWD":/projet:ro telemachlearning/telemach:X.Y.Z check /projet` donne la même vérification qu'un plugin installé.

**Independent Test**: quickstart V4 sur un fichier Compose fautif du corpus ; aucun fichier modifié.

### Tests for User Story 3 ⚠️

- [ ] T027 [P] [US3] Ajouter à `tests/integration/image_test.go` : pour deux cas de `testdata/compose/` (un valide, un fautif), comparer la sortie de `docker run --rm -v <dossier>:/projet:ro telemach-test:<version> check --color=never /projet` avec celle de `docker-telemach check --color=never <dossier>` exécuté sur l'hôte (stdout, stderr et code de sortie identiques, chemins mis à part) ; dans un dossier sans fichier Compose, vérifier le message propre à l'image (T029) et le code de sortie 2.
- [ ] T028 [P] [US3] Ajouter à `cmd/telemach-image/main_test.go` : `check` sur un dossier inexistant ou sans `compose.yaml`, `compose.yml`, `docker-compose.yaml` ni `docker-compose.yml` → message de l'image (pas d'exécution du plugin), code 2.

### Implementation for User Story 3

- [ ] T029 [US3] Dans `cmd/telemach-image/main.go` et `messages.go` : avant de transmettre `check`, si le dossier visé (dernier argument positionnel, `/projet` par défaut s'il n'y en a pas, en ignorant `-f`) n'existe pas ou ne contient aucun des quatre noms de fichiers Compose, afficher en français « aucun fichier Compose trouvé dans /projet », le concept de montage et la commande d'essai complète à copier (tag explicite), code 2 (même code que le plugin pour un fichier absent, `cmd/docker-telemach/check.go`). Faire passer T028.
- [ ] T030 [US3] Vérifier que `check` fonctionne dans l'image `scratch` sans `HOME` ni `/tmp` (aucune écriture, aucun accès à un dossier d'état) et que les couleurs s'activent avec `-t` ; si le plugin tente d'accéder à un dossier d'état pour `check`, corriger dans `cmd/docker-telemach/check.go` pour que `check` reste en lecture seule (garantie de `contracts/image-cli.md`). Faire passer T027.
- [ ] T031 [US3] Rédiger dans `README.md` la section « Essayer sans installer » : commande canonique de `contracts/image-cli.md`, rôle de `:ro`, de `-t`, et la limite sur les variables d'environnement (Edge Cases) avec l'exemple `-e NOM_VARIABLE`.

**Checkpoint**: l'essai donne le même résultat que le plugin installé et n'écrit rien.

---

## Phase 6: User Story 4 - Publier une version en toute confiance (Priority: P2)

**Goal**: un push de tag suffit ; rien n'est publié hors de `main`, rien n'est jamais écrasé.

**Independent Test**: quickstart V1 (publication, relance refusée, tag hors `main` refusé).

### Implementation for User Story 4

- [ ] T032 [US4] Compléter le job `verify` de `.github/workflows/release.yml` (variables via `env:` uniquement) : checkout `fetch-depth: 0` ; `git merge-base --is-ancestor "$GITHUB_SHA" origin/main` sinon échec « le tag doit pointer sur un commit de main » ; `gh release view "$TAG"` doit échouer (sinon « Release déjà publiée », y compris un brouillon : demander de le supprimer) ; garder le contrôle d'image de T022. Messages d'échec en français dans le journal (`::error::`).
- [ ] T033 [US4] Vérifier les permissions de chaque job de `release.yml` et `ci.yml` (lecture seule partout sauf `release`) ; vérifier qu'aucun `${{ github.ref_name }}`, `${{ inputs.* }}` ou nom de branche n'apparaît dans un bloc `run:` (`grep -n '\${{' .github/workflows/*.yml` relu ligne à ligne) ; vérifier que toutes les actions sont épinglées par SHA complet suivi du commentaire de version.
- [ ] T034 [US4] Ajouter à `CONTRIBUTING.md` une section « Publier une version » : prérequis (dépôt Docker Hub, jeton Read & Write limité au dépôt avec sa date d'expiration à renouveler, environnement `release` restreint aux tags `v*`, variable `DOCKERHUB_USERNAME`, secret `DOCKERHUB_TOKEN`, releases immuables activées — LL-005) ; procédure (`git tag -a vX.Y.Z -m … && git push origin vX.Y.Z`, ou `gh workflow run release.yml --ref vX.Y.Z` pour un tag existant) ; pré-versions ; que faire en cas d'échec (supprimer le brouillon, relancer ; une Release publiée ne se corrige que par une nouvelle version) ; rotation du jeton.
- [ ] T035 [US4] Mettre à jour la protection de `main` : ajouter `goreleaser-check`, `image-scan` et `build` aux checks obligatoires (en conservant les six existants et `enforce_admins`). **Action visible : demander confirmation à l'utilisateur.** Mettre à jour la liste des checks dans `CONTRIBUTING.md` § « Gates et fusion ».

**Checkpoint**: la chaîne refuse un tag hors `main` et une relance sur une version déjà publiée.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [ ] T036 [P] `go test -race ./...`, `golangci-lint run` (binaire compilé avec `GOTOOLCHAIN=go1.27.2`, LL-001), `goreleaser check`, hadolint sur `Dockerfile` ; tout au vert.
- [ ] T037 [P] Relire `README.md`, `CONTRIBUTING.md`, `docs/dockerhub.md` et les notes de Release : français correct et accentué, chaque commande exécutable telle quelle avec un tag explicite, ordre FR-020 respecté, liens valides.
- [ ] T038 Passer la PR de brouillon à prête, attendre toutes les gates vertes ; l'utilisateur relit et fusionne (workflow PR habituel).
- [ ] T039 Première Release : après fusion et **confirmation de l'utilisateur**, `gh workflow run release.yml --ref v0.1.0` ; suivre l'exécution ; vérifier quickstart V1 (Release, image `0.1.0`, `0.1`, `latest` sur deux architectures, durée < 30 min, SC-004) et que l'étape de connexion Docker Hub valide le jeton.
- [ ] T040 Validation SC-007 sur les plateformes cibles (quickstart V2 à V5) : Linux amd64 (WSL2 de l'utilisateur, Docker Desktop), Linux arm64 si disponible, macOS arm64 (**par l'utilisateur** ; confirmer le point ouvert R7 sur la propriété des fichiers) ; reporter les résultats et toute anomalie dans `specs/002-distribution/research.md`.
- [ ] T041 Vérifications de sécurité post-publication (quickstart V5) : `cosign verify` de l'image, `gh attestation verify` des binaires et de l'image, absence de shell, utilisateur `65532:65532`, taille compressée < 15 Mo par architecture (SC-006).
- [ ] T042 Consigner dans `LESSON-LEARNED.md` tout incident survenu pendant la feature (gate rouge, publication ratée, mauvaise manipulation), avec règle préventive et suivi (principe V).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance.
- **Foundational (Phase 2)** : dépend de la Phase 1 et **bloque toutes les histoires**. T010 (PR brouillon, gates vertes) clôt la phase.
- **US1 (Phase 3)** : dépend de la Phase 2.
- **US2 (Phase 4)** : dépend de la Phase 2 ; T022 à T024 modifient `release.yml`, donc **après T014** (US1).
- **US3 (Phase 5)** : dépend de la Phase 2 (aiguillage de `main.go`) et de l'image de test de T021 pour T027.
- **US4 (Phase 6)** : dépend de T014 (US1) et de T022 (US2) pour compléter `verify`.
- **Polish (Phase 7)** : dépend de toutes les histoires ; T039 après fusion de la PR.

### Dépendances fines notables

- T004 → T005 ; T005 + T006 + T007 → T008 → T009 → T010
- T011, T012 → T013 ; T007 → T015 ; T016 ↔ T015 (commandes identiques)
- T018 → T019 → T020 ; T019 + T006 → T021 ; T014 → T022 → T023 → T024
- T021 → T027 ; T028 → T029 → T030
- T014 + T022 → T032 ; T009 → T035 ; T038 → T039 → T040, T041

### Within Each User Story

- Tests écrits d'abord, et qui échouent
- Code Go → workflow → documentation
- Histoire validée par son checkpoint avant la suivante

### Parallel Opportunities

- Phase 1 : T002 et T003 en parallèle (T001 indépendant).
- US1 : T011 et T012 en parallèle.
- US2 : T018, T021 (préparation du contexte) et T026 en parallèle ; T025 dès que T020 est stable.
- US3 : T027 et T028 en parallèle.
- Polish : T036 et T037 en parallèle.

---

## Parallel Example: User Story 1

```bash
Task: "Écrire tests/release/dist_test.go (contrat des artefacts)"
Task: "Écrire tests/release/reproducible_test.go (FR-005)"
```

## Parallel Example: User Story 2

```bash
Task: "Écrire cmd/telemach-image/install_test.go (toutes les lignes du tableau Installation)"
Task: "Écrire tests/integration/image_test.go (commandes du README à la lettre)"
Task: "Rédiger docs/dockerhub.md"
```

---

## Implementation Strategy

### MVP d'abord (US1)

1. Phases 1 et 2 : snapshot complet construit et scanné dans une PR brouillon (gates validées
   avant le code, LL-005).
2. Phase 3 (US1) : Release de binaires signés et attestés + README.
3. **S'arrêter et valider** : à ce stade, la fusion permet déjà de publier `v0.1.0` en binaires
   seuls (`--skip=docker`) si l'on veut livrer tôt.

### Livraison incrémentale

1. US1 → binaires sans Go (MVP).
2. US2 → installation par l'image (publication de l'image activée).
3. US3 → essai sans installation.
4. US4 → durcissement de la publication, documentation mainteneur, checks obligatoires.
5. Phase 7 → première Release `v0.1.0`, validation multi-plateforme.

Une seule PR pour la feature (comme la 001), avec des commits par phase ; la première Release
n'est lancée qu'après fusion.

---

## Notes

- [P] = fichiers différents, pas de dépendance sur une tâche inachevée.
- Les essais d'image en local exigent l'intégration WSL de Docker Desktop ; à défaut, la CI
  (PR brouillon) sert de banc d'essai.
- Ne jamais pousser de tag de test sur le dépôt réel : les releases y sont immuables. Utiliser
  un clone jetable (snapshot) ou un dépôt de répétition.
- Commit après chaque tâche ou groupe logique ; messages en français, terminés par la ligne
  d'attribution.
