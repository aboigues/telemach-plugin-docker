# Modèle de données — 002 Distribution

Pas de base de données : les « entités » sont des artefacts publiés et leurs règles de
cohérence.

## Version

| Champ | Description | Règle |
|---|---|---|
| `tag` | tag git, ex. `v0.1.0`, `v0.2.0-rc.1` | `^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$` ; pointe sur un commit de `main` |
| `version` | tag sans le `v`, ex. `0.1.0` | texte exact de `docker telemach --version` et de `main.version` |
| `prerelease` | vrai si le tag a un suffixe `-…` | ne met à jour ni `latest` ni `X.Y` (image), Release marquée pré-version |
| `commit` | SHA du commit tagué | identique dans la provenance des binaires et de l'image |

**Cycle de vie** : `tag poussé` → `vérifié` (format, ancêtre de `main`, pas déjà publié) →
`gates vertes` → `Release brouillon + image poussée` → `attestée et signée` → `Release
publiée (immuable)`. Tout échec avant « publiée » laisse au plus un brouillon, supprimable par
le mainteneur ; la relance repart de « vérifié ».

## Release

| Élément | Contenu |
|---|---|
| binaires | 4 fichiers `docker-telemach_<version>_<os>_<arch>` (voir [contrat](contracts/release-assets.md)) |
| `checksums.txt` | SHA-256 de chaque binaire et de chaque SBOM |
| `checksums.txt.sigstore.json` | bundle cosign (signature keyless + certificat + preuve Rekor) |
| SBOM | un SPDX JSON par binaire |
| attestations | provenance SLSA par binaire (stockée par GitHub, vérifiable par `gh attestation verify`) ; attestation de release créée par GitHub à la publication immuable |
| notes | français : en-tête (installation et vérification), changements groupés, pied de page |

**Invariants** : une Release par tag ; immuable une fois publiée ; aucun asset ajouté à la
main.

## Binaire de plateforme

| Champ | Valeurs |
|---|---|
| `os` | `linux`, `darwin` |
| `arch` | `amd64`, `arm64` |
| `sha256` | identique dans `checksums.txt` et dans l'image (FR-009) |
| infos de build Go | module `github.com/aboigues/telemach-plugin-docker`, réglage `-ldflags` contenant `-X main.version=<version>` |

## Image

| Champ | Description |
|---|---|
| dépôt | `docker.io/telemachlearning/telemach` |
| tags | voir [contrat des tags](contracts/image-tags.md) |
| plateformes | `linux/amd64`, `linux/arm64` (index multi-architecture) |
| contenu | `/usr/local/bin/telemach-image` (point d'entrée), `/opt/telemach/linux/docker-telemach`, `/opt/telemach/darwin/docker-telemach` (architecture de l'image) |
| utilisateur | `65532:65532` |
| attachements | SBOM (buildx), provenance (`actions/attest`, poussée dans le registre), signature cosign par digest |
| labels OCI | `org.opencontainers.image.source`, `.version`, `.revision`, `.licenses=Apache-2.0`, `.title`, `.description`, `.url=https://www.telemach-learning.fr` |

## Installation (opération `install`)

Entrées : dossier monté `/sortie`, variable `HOST_OS` (défaut `linux`), identité du processus
(fixée par `--user`), architecture de l'image.

États de sortie :

| Situation | Résultat | Code |
|---|---|---|
| copie réussie (nouvelle installation) | binaire `0755` dans `/sortie/docker-telemach`, message de succès | 0 |
| copie réussie (mise à jour) | idem, message « ancienne → nouvelle version » | 0 |
| même version déjà installée | binaire remplacé (sommes identiques), message « déjà à jour » | 0 |
| `/sortie` non monté | rien n'est écrit, explication du bind mount | 1 |
| `/sortie` appartient à root (processus non-root) | rien n'est écrit, explication + `sudo chown` / `mkdir -p` | 1 |
| `/sortie` non accessible en écriture | rien n'est écrit, explication de `--user` | 1 |
| fichier existant non TELEMACH ou lien symbolique | rien n'est écrit, explication | 1 |
| `HOST_OS` non pris en charge | rien n'est écrit, systèmes acceptés, renvoi WSL2 pour Windows | 2 |
| argument inconnu pour `install` | aide de `install` | 2 |

## Environnement « release » (GitHub)

| Réglage | Valeur |
|---|---|
| règle de déploiement | tags `v*` uniquement |
| secret | `DOCKERHUB_TOKEN` (jeton Docker Hub « Read & Write », dépôt `telemach` seulement), créé par le mainteneur |
| variable | `DOCKERHUB_USERNAME=telemachlearning` (non secrète) |
| approbation | aucune (le push du tag vaut décision) |
