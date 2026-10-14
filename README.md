# TELEMACH PLUGIN DOCKER

**TELEMACH** est un plugin pédagogique pour Docker : quand une commande `docker` ou
`docker compose` échoue, il t'explique **en français** ce qui s'est passé, le concept Docker
en jeu, la cause probable et comment corriger, avec une sortie largement colorée.

```text
$ docker run -p 8080:80 nginx:1.30-alpine
docker: Error response from daemon: … Bind for 0.0.0.0:8080 failed: port is already allocated

✖ Le port 8080 de ta machine est déjà utilisé.

📘 Concept — port publishing
   L'option « -p PORT_MACHINE:PORT_CONTENEUR » relie un port de ta machine à un
   port du conteneur. Un port de la machine ne peut être relié qu'une seule fois.

🔎 Cause probable
   Un autre conteneur (ou un autre programme de ta machine) écoute déjà sur le port 8080.

💡 Corrections
   1. Voir quel conteneur publie déjà ce port.
      docker ps --filter publish=8080
   2. Publier le conteneur sur un autre port de ta machine.
      docker run -p 8081:80 nginx:1.30-alpine

➜ Pour aller plus loin : docker telemach deepen
```

## Ce que fait TELEMACH

- **Explique automatiquement** les erreurs des commandes `docker` (run, build, pull, push,
  exec, start, stop, rm, rmi, logs, network, volume…) et `docker compose`.
- **Distingue** une erreur de Docker d'une erreur du programme qui tourne dans le conteneur.
- **Analyse les fichiers Compose** ligne par ligne : syntaxe YAML, tabulations, clés
  inconnues (avec la clé probable), types de valeurs, références non déclarées, dépendances
  en boucle, variables non définies, secrets écrits en clair.
- **Approfondit** un concept à la demande, avec un exemple et un exercice sans risque.
- Reconnaît honnêtement une erreur qu'il ne connaît pas, **sans jamais inventer de cause**.

## Confidentialité et sécurité

- Fonctionne **100 % en local**, sans aucun accès réseau.
- **N'exécute aucune commande** et ne modifie aucun de tes fichiers : il propose, tu décides.
  Seule exception : `docker --version`, lu une fois à l'activation.
- **Masque les secrets** (mots de passe, jetons, clés) partout où il affiche ou enregistre
  quelque chose.
- Le dernier diagnostic, déjà masqué, est conservé dans `~/.local/state/telemach/`
  (droits `0600`) pour la commande `deepen`.

## Installation

Prérequis : Go 1.27 (la chaîne d'outils exacte est téléchargée automatiquement), Docker
Engine avec le plugin Compose v2, et bash ou zsh.

```bash
git clone https://github.com/aboigues/telemach-plugin-docker.git
cd telemach-plugin-docker
mkdir -p ~/.docker/cli-plugins
go build -o ~/.docker/cli-plugins/docker-telemach ./cmd/docker-telemach
docker telemach --help
```

Active ensuite l'explication automatique dans ton shell, par exemple dans `~/.zshrc` :

```bash
eval "$(docker telemach shell-init zsh)"     # ou : shell-init bash dans ~/.bashrc
```

La fonction `docker` ainsi définie se comporte exactement comme `docker` : même sortie,
même code de sortie. Elle ajoute seulement une explication quand une commande échoue.

## Commandes

| Commande | Rôle |
|----------|------|
| `docker telemach check [-f fichier]... [dossier]` | Vérifie un projet Compose **avant** de le lancer (code 0 : aucun problème, 1 : problème trouvé, 2 : fichier introuvable) |
| `docker telemach deepen [--exercise]` | Approfondit le concept de la dernière erreur expliquée |
| `docker telemach shell-init bash\|zsh` | Affiche le code d'intégration à évaluer dans ton shell |
| `docker telemach explain …` | Utilisée par l'intégration shell (tu n'as pas à l'appeler) |

## Couleurs

La sortie est colorée dans un terminal ; chaque bloc garde aussi un symbole et un libellé
(✖, 📘, 🔎, 💡, ⚠), pour rester lisible sans couleur.

- `--color=auto|always|never` (par défaut `auto`) ;
- `NO_COLOR=1` désactive les couleurs, `CLICOLOR_FORCE=1` les force ;
- la palette s'adapte aux fonds clairs et sombres (variable `COLORFGBG` ou interrogation du
  terminal).

## Désactivation

- ponctuelle : `TELEMACH_DISABLE=1 docker …` ;
- définitive : retire la ligne `eval "$(docker telemach shell-init …)"` de ton fichier de
  configuration du shell.

## Contribuer

Le projet suit une [constitution](.specify/memory/constitution.md) : pédagogie, français
pour l'humain et anglais pour le code, sécurité élevée, gates GitHub Actions obligatoires et
mémoire des incidents ([`LESSON-LEARNED.md`](LESSON-LEARNED.md)). Voir
[`CONTRIBUTING.md`](CONTRIBUTING.md) et, pour enrichir le catalogue d'erreurs,
[`docs/catalogue.md`](docs/catalogue.md).

Signalement de vulnérabilité : voir [`SECURITY.md`](SECURITY.md).
