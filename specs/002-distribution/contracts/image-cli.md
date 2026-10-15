# Contrat — Interface de l'image `telemachlearning/telemach`

Point d'entrée : `/usr/local/bin/telemach-image`. Utilisateur par défaut : `65532:65532`.

## Commandes

| Arguments | Effet |
|---|---|
| *(aucun)*, `--help`, `-h`, `help` | aide en français : les deux usages, avec les commandes complètes à copier (FR-015) ; code 0 |
| `install` | installe le plugin dans `/sortie` (ci-dessous) |
| `install --help` | aide détaillée de l'installation ; code 0 |
| toute autre commande (`check …`, `--version`, `explain …`) | exécution directe de `/opt/telemach/linux/docker-telemach` avec les mêmes arguments ; sortie et code de sortie identiques au plugin installé (FR-012) |

## Usages documentés (forme canonique, à reprendre telle quelle dans le README)

Essai sans installation, depuis le dossier du projet :

```bash
docker run --rm -t -v "$PWD":/projet:ro telemachlearning/telemach:<version> check /projet
```

Installation, Linux et WSL2 :

```bash
mkdir -p ~/.docker/cli-plugins
docker run --rm --user "$(id -u):$(id -g)" \
  -v ~/.docker/cli-plugins:/sortie \
  telemachlearning/telemach:<version> install
```

Installation, macOS (Docker Desktop) :

```bash
mkdir -p ~/.docker/cli-plugins
docker run --rm --user "$(id -u):$(id -g)" -e HOST_OS=darwin \
  -v ~/.docker/cli-plugins:/sortie \
  telemachlearning/telemach:<version> install
```

## `install` — entrées

| Entrée | Valeurs | Défaut |
|---|---|---|
| montage `/sortie` | dossier des plugins Docker de l'hôte | obligatoire |
| `HOST_OS` | `linux`, `darwin` (insensible à la casse) | `linux` |
| identité (`--user`) | celle de l'utilisateur hôte | `65532:65532` (échoue en général sous Linux, avec explication) |

## `install` — sorties

Sur stdout en cas de succès, sur stderr en cas de refus ; texte en français ; couleurs selon
les mêmes règles que le plugin (`NO_COLOR`, terminal).

Succès (exemple) :

```text
✔ TELEMACH 0.2.0 installé dans ~/.docker/cli-plugins (macOS, Apple Silicon — darwin/arm64).
  Version remplacée : 0.1.0.

📘 Ce qui vient de se passer
   Le conteneur a écrit dans un dossier de ta machine grâce au montage
   « -v ~/.docker/cli-plugins:/sortie » (bind mount), avec ton identité (--user).

➜ Étape suivante : active l'explication automatique des erreurs
   echo 'eval "$(docker telemach shell-init zsh)"' >> ~/.zshrc
   puis vérifie : docker telemach --version
```

Le chemin affiché côté hôte est générique (`~/.docker/cli-plugins`) : le conteneur ne connaît
que `/sortie`.

Refus (exemple, dossier non monté) :

```text
✖ Aucun dossier de destination : /sortie n'est pas monté.

📘 Concept — montage (bind mount)
   Un conteneur ne voit pas les fichiers de ta machine. L'option -v DOSSIER_MACHINE:/sortie
   lui donne accès à un dossier précis, ici celui des plugins Docker.

💡 Correction
   mkdir -p ~/.docker/cli-plugins
   docker run --rm --user "$(id -u):$(id -g)" -v ~/.docker/cli-plugins:/sortie telemachlearning/telemach:0.2.0 install
```

Cas de refus obligatoires (FR-014), chacun avec concept + correction : dossier non monté ;
dossier appartenant à root ; dossier non accessible en écriture ; fichier existant non
TELEMACH ou lien symbolique ; `HOST_OS` non pris en charge.

## Codes de sortie de `install`

| Code | Signification |
|---|---|
| 0 | installé ou déjà à jour |
| 1 | refus lié à l'environnement (montage, droits, fichier existant) ; rien n'a été écrit |
| 2 | erreur d'utilisation (`HOST_OS` invalide, argument inconnu) |

## Garanties

- `install` n'écrit que `/sortie/docker-telemach` (et un fichier temporaire dans `/sortie`,
  renommé ou supprimé avant la fin) ; jamais ailleurs.
- `check` lit le dossier monté en lecture seule et n'écrit rien.
- Aucun accès réseau, aucune élévation de privilèges, aucun shell dans l'image.
