<!-- Description de la page Docker Hub telemachlearning/telemach.
     À copier à la main dans Docker Hub (Repository › General) à chaque changement :
     - Short description (100 caractères au plus) :
       Plugin pédagogique : explique en français les erreurs Docker et Compose, et comment les corriger.
     - Category : Developer tools
     - Overview : tout ce qui suit ce commentaire.
     Le nom du sujet de provenance est celui de research.md R3 : docker.io/telemachlearning/telemach. -->

# TELEMACH — explique en français les erreurs Docker

**TELEMACH** est un plugin pédagogique pour Docker : quand une commande `docker` ou
`docker compose` échoue, il t'explique **en français** ce qui s'est passé, le concept Docker
en jeu, la cause probable et comment corriger.

## Ce que fait le plugin

- **Explique les erreurs** de `docker` et `docker compose`, avec la commande corrigée à copier.
- **Vérifie un fichier Compose** avant de le lancer (`docker telemach check`) : clé inconnue,
  tabulation, service non déclaré, secret en clair…, avec la ligne exacte.
- **Repère les pièges** d'une commande qui réussit sans faire ce que tu voulais : option placée
  après l'image (`docker run -d nginx -p 8080:80`), volume nommé au lieu du dossier local, port
  ignoré en réseau `host`, secret dans la commande, `--privileged`…
- **Explique un conteneur arrêté** (`docker telemach ps`) : après un `docker run -d` qui
  s'arrête aussitôt, il te prévient à l'invite suivante, montre la fin du journal et donne la
  cause (variable obligatoire absente, manque de mémoire, commande introuvable…).
- **Ne fait que lire** : il ne modifie aucun fichier et ne démarre, n'arrête ni ne supprime
  aucun conteneur ; les secrets sont masqués partout.

Cette image sert à deux choses.

## Essayer sans rien installer

Depuis le dossier de ton projet, vérifie ton fichier Compose :

```bash
docker run --rm -t -v "$PWD":/projet:ro telemachlearning/telemach:0.3.0 check /projet
```

`:ro` monte ton dossier en lecture seule : l'image ne peut rien y modifier.

## Installer le plugin sur ta machine

Linux et WSL2 :

```bash
mkdir -p ~/.docker/cli-plugins
docker run --rm --user "$(id -u):$(id -g)" -v ~/.docker/cli-plugins:/sortie telemachlearning/telemach:0.3.0 install
```

macOS (Docker Desktop) : ajoute `-e HOST_OS=darwin` avant `-v`.

Puis active l'explication automatique des erreurs :
`eval "$(docker telemach shell-init zsh)"` (ou `bash`) dans ton fichier de configuration du shell.

## Tags

- `X.Y.Z` (ex. `0.3.0`) : une version précise, jamais réécrite — **recommandé** ;
- `X.Y` et `latest` : suivent la dernière version stable.

## Sécurité

- `FROM scratch` : ni shell, ni gestionnaire de paquets ; utilisateur non-root (`65532`).
- Image multi-architecture (`linux/amd64`, `linux/arm64`), analysée à chaque modification.
- Signée sans clé (Sigstore) et accompagnée d'un SBOM et d'une attestation de provenance :

```bash
cosign verify telemachlearning/telemach:0.3.0 \
  --certificate-identity-regexp '^https://github\.com/aboigues/telemach-plugin-docker/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
gh attestation verify oci://docker.io/telemachlearning/telemach:0.3.0 --repo aboigues/telemach-plugin-docker
```

## Liens

- Code source, documentation complète et binaires : https://github.com/aboigues/telemach-plugin-docker
- Site : https://www.telemach-learning.fr
- Licence : Apache 2.0
