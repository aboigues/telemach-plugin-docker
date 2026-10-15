# Contrat — Artefacts d'une Release GitHub

URL de base : `https://github.com/aboigues/telemach-plugin-docker/releases/download/v<version>/`

## Fichiers

| Fichier | Description |
|---|---|
| `docker-telemach_<version>_linux_amd64` | binaire statique Linux x86-64 |
| `docker-telemach_<version>_linux_arm64` | binaire statique Linux ARM64 |
| `docker-telemach_<version>_darwin_amd64` | binaire macOS Intel |
| `docker-telemach_<version>_darwin_arm64` | binaire macOS Apple Silicon |
| `docker-telemach_<version>_<os>_<arch>.sbom.json` | SBOM SPDX JSON de chaque binaire |
| `checksums.txt` | une ligne `<sha256>  <fichier>` par binaire et par SBOM (format `sha256sum`) |
| `checksums.txt.sigstore.json` | bundle cosign de `checksums.txt` |

Ces noms sont stables : le README, les notes de Release et les scripts des utilisateurs en
dépendent. Les changer est un changement incompatible (version MAJOR).

## Garanties

1. `./docker-telemach_<version>_<os>_<arch> --version` affiche
   `TELEMACH PLUGIN DOCKER <version>` en première ligne.
2. `sha256sum --check --ignore-missing checksums.txt` réussit pour tout binaire téléchargé.
3. Signature :

   ```bash
   cosign verify-blob \
     --bundle checksums.txt.sigstore.json \
     --certificate-identity-regexp '^https://github\.com/aboigues/telemach-plugin-docker/\.github/workflows/release\.yml@refs/tags/v' \
     --certificate-oidc-issuer https://token.actions.githubusercontent.com \
     checksums.txt
   ```

4. Provenance : `gh attestation verify docker-telemach_<version>_<os>_<arch> --repo aboigues/telemach-plugin-docker`
   réussit et désigne le workflow `release.yml` et le commit du tag.
5. Reconstruire le binaire depuis le tag (procédure du [quickstart](../quickstart.md)) donne
   la même somme SHA-256.
6. La Release est immuable : aucun fichier n'est ajouté, modifié ou supprimé après
   publication.

## Notes de Release (structure)

```text
## Installer TELEMACH <version>
<commandes Linux / macOS : téléchargement, vérification, installation, --version>
<commande d'installation par l'image : telemachlearning/telemach:<version>>

## Nouveautés
<commits feat>
## Corrections
<commits fix>
## Autres changements
<autres commits, hors docs/ci/test>

## Vérifier l'authenticité
<commandes cosign et gh attestation>
```
