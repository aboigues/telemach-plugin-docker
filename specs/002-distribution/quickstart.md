# Quickstart — Validation de la distribution (002)

Scénarios de bout en bout qui prouvent que la feature fonctionne. Ils suivent **à la lettre**
les commandes du README, sans raccourci d'environnement (LL-006). Les détails des formats sont
dans les [contrats](contracts/).

## Prérequis

- Une machine (ou VM) **sans Go** par plateforme cible : Linux amd64, Linux arm64 (ou WSL2),
  macOS arm64 ; macOS amd64 si disponible.
- Docker Engine ou Docker Desktop, bash ou zsh, `curl`, `sha256sum` (Linux) ou
  `shasum -a 256` (macOS).
- Facultatif pour la vérification avancée : cosign v3.1.3, `gh` connecté.
- Côté mainteneur, une seule fois : dépôt Docker Hub `telemachlearning/telemach` créé,
  environnement GitHub « release » (tags `v*`) avec le secret `DOCKERHUB_TOKEN` et la
  variable `DOCKERHUB_USERNAME`, releases immuables activées (voir `CONTRIBUTING.md`).

## V0 — Gates sur la pull request (avant tout code de publication, LL-005)

1. Ouvrir la PR de la feature.
2. **Attendu** : les jobs `goreleaser-check` (configuration valide, construction snapshot des
   4 binaires et de l'image) et `image-scan` (aucune vulnérabilité haute ou critique) sont
   verts, en plus des gates existantes ; `dockerfile-lint` analyse le `Dockerfile`.

## V1 — Publication (US4)

1. Après la fusion, sur `main` à jour : `git tag -a v0.2.0 -m "…" && git push origin v0.2.0`
   (première Release ; `v0.1.0`, antérieur à la chaîne de publication, reste sans Release).
2. **Attendu** en moins de 30 min : jobs `verify`, `gates`, `release` verts ; Release
   `v0.2.0` publiée avec les fichiers du [contrat](contracts/release-assets.md) et des notes
   en français ; image `telemachlearning/telemach:0.2.0`, `0.2` et `latest` sur Docker Hub,
   pour `linux/amd64` et `linux/arm64`.
3. Relancer `gh workflow run release.yml --ref v0.2.0`. **Attendu** : échec dans `verify`
   (« Release déjà publiée »), aucun artefact modifié.
4. Pousser `v0.0.0-test.1` sur un commit qui n'est pas dans `main` (par exemple le dernier
   commit d'une branche de travail). **Attendu** : échec dans `verify`, rien n'est publié ;
   supprimer ensuite le tag (`git push --delete origin v0.0.0-test.1`), possible car aucune
   Release ne l'a verrouillé.

## V2 — Installation depuis la Release (US1, SC-001, SC-003)

Sur chaque machine sans Go, chronomètre en main, en copiant les commandes du README :

1. Télécharger le binaire et `checksums.txt` ; `sha256sum --check --ignore-missing
   checksums.txt` → `OK`.
2. Vérifier la signature (`cosign verify-blob …` du contrat) → `Verified OK`.
3. Installer dans `~/.docker/cli-plugins/docker-telemach`, puis `docker telemach --version`
   → `TELEMACH PLUGIN DOCKER 0.2.0`.
4. Activer `shell-init`, puis `docker run --rm ngnix` → explication en français.
5. Altérer une copie du binaire (`printf x >> copie`) et relancer la vérification de somme →
   échec visible.

**Attendu** : étapes 1 à 4 en moins de 5 minutes (SC-001).

## V3 — Installation par l'image (US2, SC-002)

Linux / WSL2 (supprimer d'abord tout plugin existant) :

```bash
mkdir -p ~/.docker/cli-plugins
docker run --rm --user "$(id -u):$(id -g)" -v ~/.docker/cli-plugins:/sortie telemachlearning/telemach:0.2.0 install
ls -l ~/.docker/cli-plugins/docker-telemach    # appartient à l'utilisateur, exécutable
docker telemach --version                      # TELEMACH PLUGIN DOCKER 0.2.0
```

Chronométrer la commande `docker run … install` avec l'image déjà téléchargée : **moins de
1 minute** (SC-002).

macOS : même chose avec `-e HOST_OS=darwin`. **Attendu** : binaire darwin de la bonne
architecture, `docker telemach --version` fonctionne, propriétaire = l'utilisateur (valide le
point ouvert R7).

Refus attendus (rien n'est écrit, message en français avec concept et correction) :

| Commande | Message attendu |
|---|---|
| sans `-v …:/sortie` | dossier non monté (bind mount) |
| sans `--user` (Linux) | dossier non accessible en écriture, explication de `--user` |
| `rmdir ~/.docker/cli-plugins` puis commande sans `mkdir -p` (Linux) | dossier appartenant à root, correction `sudo chown` puis `mkdir -p` |
| `-e HOST_OS=windows` | système non pris en charge, renvoi vers WSL2 |
| un fichier `docker-telemach` qui n'est pas TELEMACH dans le dossier | refus de l'écraser |

Mise à jour : relancer avec une image plus récente → message « ancienne → nouvelle version ».

## V4 — Essai sans installation (US3)

Dans `testdata/compose/…` (un fichier Compose fautif du corpus) :

```bash
docker run --rm -t -v "$PWD":/projet:ro telemachlearning/telemach:0.2.0 check /projet
```

**Attendu** : même résultat que `docker telemach check .` installé ; aucun fichier créé ou
modifié (`git status` propre) ; dans un dossier sans fichier Compose, message expliquant
comment monter le bon dossier.

## V5 — Vérifications de sécurité de l'image (SC-005, SC-006)

```bash
docker buildx imagetools inspect telemachlearning/telemach:0.2.0      # 2 plateformes
docker image inspect telemachlearning/telemach:0.2.0 --format '{{.Config.User}}'   # 65532:65532
docker run --rm --entrypoint sh telemachlearning/telemach:0.2.0       # échoue : aucun shell
cosign verify telemachlearning/telemach:0.2.0 --certificate-identity-regexp … --certificate-oidc-issuer …
gh attestation verify oci://docker.io/telemachlearning/telemach:0.2.0 --repo aboigues/telemach-plugin-docker
```

**Attendu** : tout réussit (sauf `sh`, qui doit échouer) ; taille compressée affichée par
Docker Hub < 15 Mo par architecture.

Mêmes binaires que la Release (FR-009), pour chaque architecture :

```bash
cid=$(docker create --platform linux/amd64 telemachlearning/telemach:0.2.0)
docker cp "$cid":/opt/telemach/linux/docker-telemach ./img_linux_amd64
docker cp "$cid":/opt/telemach/darwin/docker-telemach ./img_darwin_amd64
docker rm "$cid"
sha256sum img_linux_amd64 img_darwin_amd64    # = sommes de docker-telemach_0.2.0_linux_amd64 et _darwin_amd64 dans checksums.txt
```

## V6 — Reproductibilité (FR-005)

Depuis un clone au tag, avec Go 1.27.2 :

```bash
git checkout v0.2.0
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
  -ldflags "-s -w -buildid= -X main.version=0.2.0" \
  -o docker-telemach_0.2.0_linux_amd64 ./cmd/docker-telemach
sha256sum docker-telemach_0.2.0_linux_amd64    # même somme que dans checksums.txt
```

(Les options exactes seront recopiées de `.goreleaser.yaml` à l'implémentation.)
