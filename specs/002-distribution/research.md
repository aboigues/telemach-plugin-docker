# Recherche — 002 Distribution du plugin sans Go

Toutes les versions ci-dessous ont été vérifiées le **2026-10-15** via l'API GitHub
(`gh api repos/<owner>/<repo>/releases/latest`), conformément à la règle globale et à LL-004.
Les SHA de commit des actions ont été résolus à partir des tags (déréférencement des tags
annotés).

## Versions retenues

| Outil / action | Version | Épinglage |
|---|---|---|
| GoReleaser (OSS) | v2.18.3 (2026-10-09) | `version: v2.18.3` dans l'action |
| `goreleaser/goreleaser-action` | v7.3.0 | `a4f02ab5355a589de60d917a659ce617ed19b3de` |
| cosign | v3.1.3 (2026-08-06) | installé par l'action ci-dessous |
| `sigstore/cosign-installer` | v4.1.2 | `6f9f17788090df1f26f669e9d70d6ae9567deba6` |
| `actions/attest` | v4.2.2 | `1e69f48acb82d1966a394da916b4c1698aa569d6` |
| `docker/setup-qemu-action` | v4.4.0 | `99012661954931238ded8c8b007157a8430204e1` |
| `docker/setup-buildx-action` | v4.4.1 | `f87e5991a6d7451dcb8d9637bfbc97413f497069` |
| `docker/login-action` | v4.6.0 | `dbcb813823bdd20940b903addbd779551569679f` |
| `anchore/scan-action` (Grype) | v7.4.2 | `27805bf3b4e84b4a5c980df22ed233c00390a439` |
| Syft (SBOM, appelé par GoReleaser) | v1.54.1 (2026-10-06) | installé via `anchore/sbom-action/download-syft` v0.24.3 `66cbf4bc1f1c0d2edc94016e65bc221b6bb0ad6c` |

Versions des outils fixées explicitement dans les actions (sinon elles prennent leur version
par défaut, contraire au principe III) : `cosign-installer` → `cosign-release: v3.1.3` ;
`download-syft` → `syft-version: v1.54.1` ; `anchore/scan-action` → `grype-version: v0.120.1`
(dernière version vérifiée le 2026-10-15, publiée le 2026-10-06) ; `goreleaser-action` →
`version: v2.18.3`. Ces versions ne sont pas suivies par Dependabot : elles sont revues chaque
trimestre avec `LESSON-LEARNED.md` (voir CONTRIBUTING).

### Licence et maintenance (constitution, « Chaîne d'approvisionnement »)

Vérifié le 2026-10-15 via l'API GitHub (dépôt non archivé, dernier push). Tous ces outils ne
servent qu'en CI : aucun n'est embarqué dans le plugin ni dans l'image. Toutes les licences
sont compatibles avec Apache-2.0.

| Projet | Licence | Maintenance |
|---|---|---|
| goreleaser/goreleaser | MIT | actif (push 2026-10-14), ~16 k étoiles |
| goreleaser/goreleaser-action | MIT | actif (2026-10-12) |
| sigstore/cosign | Apache-2.0 | actif (2026-10-14), projet OpenSSF |
| sigstore/cosign-installer | Apache-2.0 | actif (2026-10-13) |
| anchore/syft | Apache-2.0 | actif (2026-10-15), ~9,7 k étoiles |
| anchore/grype | Apache-2.0 | actif (2026-10-15), ~13 k étoiles |
| anchore/scan-action | MIT | actif (2026-10-09) |
| anchore/sbom-action | Apache-2.0 | actif (2026-10-12) |
| actions/attest | MIT | actif (2026-10-14), maintenu par GitHub |
| docker/setup-qemu-action, setup-buildx-action, login-action | Apache-2.0 | actifs (2026-10-03 à 2026-10-14), maintenus par Docker |

Actions non retenues mais vérifiées : `docker/metadata-action` v6.2.0, `docker/build-push-action`
v7.4.0 (GoReleaser pilote buildx lui-même), `actions/attest-build-provenance` v4.2.2 (n'est plus
qu'une surcouche de `actions/attest`, qui est recommandée pour les nouveaux usages).

Remarque : `goreleaser-action` v7.3.0 a été publiée le 2026-10-12. L'épinglage par SHA protège
contre une réécriture du tag ; Dependabot proposera les mises à jour suivantes.

---

## R1 — Outil de construction et de publication des binaires

- **Décision** : GoReleaser OSS v2.18.3, piloté par `goreleaser-action`, configuration
  `.goreleaser.yaml` à la racine.
- **Justification** : un seul outil produit les quatre binaires (`CGO_ENABLED=0`,
  `-trimpath`, `mod_timestamp` = date du commit → constructions reproductibles, FR-005), le
  fichier de sommes, le SBOM (via Syft), la signature cosign du fichier de sommes, les notes de
  Release et l'image multi-architecture (`dockers_v2`). Il réutilise exactement l'injection de
  version du job `build` (`-X main.version={{.Version}}`).
- **Alternatives** : script maison `go build` + `gh release` (plus de code à maintenir, et à
  sécuriser soi-même) ; ko (images seulement, pas de binaires darwin dans la Release).

## R2 — Format des binaires de la Release

- **Décision** : binaires bruts, sans archive, nommés
  `docker-telemach_<version>_<os>_<arch>` (ex. `docker-telemach_0.1.0_linux_amd64`), plus
  `checksums.txt`, `checksums.txt.sigstore.json` (signature cosign) et un SBOM SPDX par binaire.
- **Justification** : pour un débutant, `curl -fsSLo … && sha256sum --check` puis `chmod +x`
  est plus simple qu'une extraction d'archive ; le binaire se place directement dans
  `~/.docker/cli-plugins/docker-telemach`.
- **Alternatives** : archives `.tar.gz` (une étape de plus, sans bénéfice : un seul fichier).

## R3 — Signature et provenance (keyless)

- **Décision** :
  1. **Binaires** : GoReleaser signe `checksums.txt` avec `cosign sign-blob` en mode keyless
     (identité OIDC du workflow, journal de transparence Rekor) → bundle
     `checksums.txt.sigstore.json`. Vérification : `cosign verify-blob --bundle …
     --certificate-identity-regexp '^https://github.com/aboigues/telemach-plugin-docker/'
     --certificate-oidc-issuer https://token.actions.githubusercontent.com checksums.txt`.
  2. **Provenance SLSA** : `actions/attest` (build provenance) sur chaque binaire ;
     vérification alternative par `gh attestation verify <fichier> --repo
     aboigues/telemach-plugin-docker`.
  3. **Image** : `cosign sign` par digest (keyless) et `actions/attest` avec
     `push-to-registry: true` ; SBOM de l'image attaché par buildx (`sbom: true` dans
     `dockers_v2`). Le nom du sujet de l'attestation (`index.docker.io/…` ou `docker.io/…`) et
     la nécessité éventuelle de `packages: write` sont tranchés en T024 d'après la
     documentation de `actions/attest`, puis reportés à l'identique dans
     `contracts/image-tags.md` et `quickstart.md` (un seul nom partout).
- **Justification** : aucune clé à garder (FR-004, FR-011) ; deux voies de vérification, dont
  une (cosign) qui ne demande pas de compte GitHub. La somme SHA-256 reste la vérification
  minimale documentée en premier.
- **Alternatives** : clé cosign gérée (secret à protéger et à faire tourner) ; GPG (peu
  pédagogique, gestion de clés).

## R4 — Releases non modifiables (FR-006)

- **Décision** : activer les **releases immuables** du dépôt (GA depuis le 2025-10-28 ; une
  release publiée ne peut plus recevoir, modifier ni perdre d'asset, et son tag ne peut plus
  être déplacé ni supprimé). GoReleaser crée la Release **en brouillon** (`release.draft:
  true`), les attestations sont produites, puis le workflow la publie (`gh release edit
  --draft=false`) — ordre recommandé par GitHub. Avant toute construction, le workflow vérifie
  qu'aucune Release ni aucun tag d'image `X.Y.Z` n'existe déjà, et échoue sinon.
- **Justification** : la garantie est apportée par la plateforme, pas seulement par le
  workflow. C'est aussi ce qui a protégé une version de `trivy-action` lors de la
  compromission de mars 2026.
- **Limite connue** : la publication immuable crée une attestation de release ; un ajout
  manuel d'asset après le brouillon pourrait la casser (signalé par la communauté, non
  confirmé) : aucun ajout manuel n'est prévu.
- **Action mainteneur** : activer le réglage (Settings › General › Releases) — documenté dans
  `CONTRIBUTING.md` (LL-005).

## R5 — Image : base, contenu et point d'entrée

- **Décision** :
  - `FROM scratch` : aucun shell, aucun gestionnaire de paquets, aucune bibliothèque (le
    plugin est statique et hors ligne : pas besoin de certificats CA ni de fuseaux horaires).
    `scratch` n'est pas une image téléchargée : il n'y a pas de digest à épingler, l'exigence
    FR-010 est satisfaite par construction.
  - `USER 65532:65532` (non-root), aucun `VOLUME`, `LABEL` OCI (source, licence, version).
  - Contenu : `/opt/telemach/linux/docker-telemach` et `/opt/telemach/darwin/docker-telemach`
    (même architecture que l'image), plus un petit point d'entrée
    `/usr/local/bin/telemach-image`.
  - Point d'entrée : nouveau programme Go `cmd/telemach-image` (bibliothèque standard et
    `internal/render` du projet, pour les couleurs et symboles ; aucune nouvelle dépendance).
    `install` → copie (R7) ; `--help` ou aucun argument → aide en français (FR-015) ; toute
    autre commande (`check`, `--version`…) → remplacement du processus par le binaire linux
    (`syscall.Exec`), pour un comportement identique au plugin installé (FR-012).
- **Justification** : séparer le point d'entrée évite d'ajouter au plugin une commande
  `install` qui n'a de sens que dans l'image. Taille estimée : 2 × ~13 Mo non compressés,
  ~10 Mo compressés par architecture (mesuré le 2026-10-15) → SC-006 tenu.
- **Alternatives** : `gcr.io/distroless/static:nonroot` épinglée par digest (apporte des
  fichiers inutiles et une dépendance externe à surveiller) ; sous-commande `install` cachée
  dans le plugin (pollue l'interface du plugin, mélange deux responsabilités).

## R6 — Construction de l'image multi-architecture

- **Décision** : `dockers_v2` de GoReleaser (disponible depuis v2.12), plateformes
  `linux/amd64` et `linux/arm64`, Dockerfile qui ne fait que copier des binaires déjà
  construits. Le contexte contient un sous-dossier par plateforme
  (`linux/amd64/docker-telemach`, `darwin/arm64/docker-telemach`…) :
  `COPY $TARGETPLATFORM/docker-telemach /opt/telemach/linux/` et
  `COPY darwin/$TARGETARCH/docker-telemach /opt/telemach/darwin/`.
  Tags : `{{.Version}}`, et pour une version stable seulement `{{.Major}}.{{.Minor}}` et
  `latest`.
- **À vérifier dès la première tâche d'implémentation** (risques) : (a) les binaires darwin
  sont bien présents dans le contexte de `dockers_v2` (filtre `ids`) ; (b) un modèle de tag
  vide pour une pré-version est bien ignoré. **Repli** si (a) ou (b) échoue : construire
  l'image avec `docker/build-push-action` v7.4.0 (`c3c9e263c25d99ce0380d002d59b67737d91b0dc`)
  sur un contexte préparé à partir des binaires de la Release (mêmes sommes, FR-009).
- **Note** : `dockers_v2` sera renommé `dockers` en GoReleaser v3 (sans date) ; à surveiller.

## R7 — Comportement de `install` (droits, propriétaire, système)

- **Décision** :
  - Destination fixe `/sortie` ; refus si ce n'est pas un point de montage (heuristique :
    dossier absent ou vide de toute écriture possible, et absent de `/proc/self/mountinfo`).
  - Système cible : `HOST_OS` = `linux` (défaut) ou `darwin` ; toute autre valeur est refusée
    (`windows` → message renvoyant vers WSL2).
  - Refus si `/sortie` appartient à root alors que le processus n'est pas root : c'est le
    signe que Docker a créé le dossier lui-même (cas le plus fréquent sous Linux quand
    `mkdir -p` a été oublié).
  - Refus si `/sortie` n'est pas accessible en écriture (cas typique : pas de
    `--user "$(id -u):$(id -g)"` sous Linux) ; le message explique l'identité du processus
    dans le conteneur et donne la commande corrigée.
  - Fichier existant : remplacé seulement s'il s'agit d'un fichier ordinaire reconnu comme
    binaire TELEMACH (lecture des informations de build Go avec `debug/buildinfo.ReadFile`,
    qui fonctionne aussi sur un binaire darwin : chemin de module
    `github.com/aboigues/telemach-plugin-docker`). Un lien symbolique ou un autre fichier est
    refusé. L'ancienne version est lue dans les réglages de build (`-X main.version=…`).
  - Écriture atomique : fichier temporaire dans `/sortie`, droits `0755`, puis renommage.
- **Point ouvert à valider sur Mac (SC-007)** : la propriété apparente des fichiers d'un
  montage Docker Desktop (virtiofs) avec `--user`. La règle « refus si propriétaire root » est
  choisie parce qu'elle reste correcte quel que soit le mappage ; elle sera confirmée lors du
  test sur macOS.
- **Alternatives** : laisser Docker créer le dossier (fichier appartenant à root, plugin
  impossible à mettre à jour sans `sudo`) ; exécuter l'image en root (contraire au
  principe III).

## R8 — Gate de scan de l'image (FR-016)

- **Décision** : **Grype** via `anchore/scan-action` v7.4.2 (épinglé par SHA), sévérité
  `high` bloquante, sur l'image construite pour `linux/amd64` dans chaque pull request (job
  `image-scan`, ajouté aux checks obligatoires de `main`). Grype analyse les modules Go
  inscrits dans les binaires. Le lint du Dockerfile est assuré par le job existant
  `dockerfile-lint` (hadolint), qui trouvera désormais le `Dockerfile`.
- **Justification** : `aquasecurity/trivy-action` a été compromise le 2026-03-19 (réécriture
  de 75 à 76 tags pour voler des secrets de runners) ; même épinglée par SHA, elle n'est pas
  retenue pour un projet qui enseigne la sécurité. Grype est maintenu par Anchore, déjà
  fournisseur de Syft retenu pour le SBOM.
- **Alternatives** : Trivy (écarté, ci-dessus) ; Docker Scout (nécessite un compte Docker
  dans les PR, donc un secret exposé aux PR) ; `govulncheck` seul (déjà présent, ne couvre
  pas l'image en tant que telle).

## R9 — Enchaînement de la publication (FR-001, FR-002, FR-017, FR-018)

- **Décision** : nouveau workflow `.github/workflows/release.yml` :
  - déclencheurs : `push` de tags `v*` et `workflow_dispatch` **sans entrée**, lancé sur le
    tag lui-même (`gh workflow run release.yml --ref vX.Y.Z`), pour relancer un tag dont la
    publication automatique a échoué. Un lancement manuel exécute le workflow **tel qu'il est
    au tag** : un tag antérieur à `release.yml` (cas de `v0.1.0`) ne peut pas être publié, d'où
    une première Release `v0.2.0` posée après la fusion.
    Ainsi `GITHUB_REF` vaut toujours `refs/tags/vX.Y.Z` : l'identité des signatures keyless
    (`release.yml@refs/tags/v…`) et la règle de déploiement de l'environnement restent les
    mêmes dans les deux cas. Une entrée `tag` lancée depuis `main` signerait avec l'identité
    `refs/heads/main`, que les commandes de vérification documentées refuseraient ;
  - job `verify` (lecture seule) : `GITHUB_REF` commence par `refs/tags/v` et le nom suit
    `^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`, vérifié en shell sur une variable
    d'environnement (jamais d'interpolation `${{ }}` dans un script : pas d'injection), tag
    ancêtre de `origin/main`
    (`git merge-base --is-ancestor`), absence de Release et de tag d'image existants ;
  - job `gates` : appel de `ci.yml` rendu réutilisable (`on: workflow_call`) — mêmes gates
    que les PR, sur le commit tagué ;
  - job `release` (`needs: [verify, gates]`, `environment: release`) : seul job avec
    `contents: write`, `id-token: write`, `attestations: write`, `packages: write` non
    nécessaire ; connexion Docker Hub avec `DOCKERHUB_TOKEN` (utilisateur
    `telemachlearning`), GoReleaser, attestations, signature de l'image, publication du
    brouillon.
  - Le déclencheur `tags: ['v*']` ajouté à `ci.yml` par la PR #4 est retiré : la CI du tag
    est désormais exécutée par `release.yml` (évite une double exécution).
- **Environnement « release »** : restreint aux tags `v*` (règle de déploiement), secret
  `DOCKERHUB_TOKEN` créé par le mainteneur, aucune approbation requise (voir spec,
  Assumptions).
- **Alternatives** : publication déclenchée par la fusion sur `main` (perd la décision
  explicite de version) ; jeton disponible au niveau du dépôt (exposé à tous les workflows).

## R10 — Notes de Release en français (FR-007)

- **Décision** : `changelog` GoReleaser (commits depuis le tag précédent, déjà rédigés en
  français par convention du dépôt, groupés feat / fix / autres) encadré d'un `header` et
  d'un `footer` en français contenant les commandes d'installation et de vérification
  (modèles avec `{{ .Version }}`).

## R11 — Vérification de la reproductibilité (FR-005)

- **Décision** : `CGO_ENABLED=0`, `-trimpath`, `-buildid=`, `mod_timestamp:
  {{ .CommitTimestamp }}`, chaîne d'outils épinglée par `go.mod` (`toolchain go1.27.2`). Le
  quickstart contient la procédure de reconstruction locale d'un binaire et la comparaison
  de sa somme avec `checksums.txt`.
