# Quickstart : valider l'analyse des conteneurs arrêtés en erreur

**Feature** : `004-exited-containers-analysis`

Prérequis : plugin compilé depuis la branche et installé dans `~/.docker/cli-plugins/`, démon
Docker démarré, **nouveau terminal** ouvert après l'installation (LL-013), intégration chargée
(`eval "$(docker telemach shell-init zsh)"`). Ressources de test préfixées `essai-`, supprimées
à la fin.

## V1 — Exemple de l'issue #8 (US1, US2)

```bash
docker run -d --name essai-pg postgres:18-alpine
```

Attendu : l'identifiant s'affiche, code 0. Appuyer sur Entrée après ~4 s : une ligne
« ⚠ Le conteneur « essai-pg » s'est arrêté avec le code 1 … docker telemach ps … ».
Puis `docker telemach ps` : journal « You must specify POSTGRES_PASSWORD… », cause « variable
obligatoire absente », correction avec `--env-file`. `echo $?` → 0.

## V2 — Rien à signaler

```bash
docker run -d --name essai-web -p 18080:8080 nginxinc/nginx-unprivileged:1.30-alpine
```

Attendu : aucune ligne à l'invite après 4 s. `docker telemach ps` n'analyse pas `essai-web`.

## V3 — Codes de sortie (US1)

| Commande | Attendu de `docker telemach ps` |
|---|---|
| `docker run -d --name essai-127 --user 1000:1000 alpine:3.24 commande-inexistante` | `docker run` échoue lui-même (127, déjà expliqué) ; le conteneur reste « Created » : `ps` explique l'erreur de démarrage (commande introuvable dans l'image) |
| `docker run -d --name essai-oom --memory 8m --user 1000:1000 alpine:3.24 sh -c 'x=a; while :; do x=$x$x; done'` | manque de mémoire (137, `OOMKilled`), limite 8 Mo |
| `docker run -d --name essai-stop --user 1000:1000 alpine:3.24 sleep 300` puis `docker stop essai-stop` | arrêt forcé de l'extérieur (137, `OOMKilled` faux : `sleep`, PID 1, ignore SIGTERM), souvent normal |
| `docker run -d --name essai-1 --user 1000:1000 alpine:3.24 sh -c 'echo démarrage; exit 1'` | erreur générique (1), journal affiché, cause dite non précise |

## V4 — Liste et désignation (US3)

Après V1 et V3 : `docker telemach ps` liste les autres conteneurs en erreur (code, ancienneté,
cause en une ligne) ; `docker telemach ps essai-127` explique celui-ci ;
`docker telemach ps essai-inconnu` → introuvable, noms proches proposés, code 1.

## V5 — `--rm` et désactivation (US2)

```bash
docker run -d --rm --user 1000:1000 alpine:3.24 sh -c 'exit 3'
```

Attendu : à l'invite suivante, « … s'est arrêté avec le code 3 … et a été supprimé (--rm) … ».
`TELEMACH_WATCH=0 docker run -d --rm --user 1000:1000 alpine:3.24 sh -c 'exit 3'` : aucune ligne.

## V6 — Sécurité (SC-002, SC-003)

```bash
docker run -d --name essai-secret -e API_TOKEN=Secret123 --user 1000:1000 alpine:3.24 sh -c 'echo "token=Secret123"; exit 1'
docker telemach ps essai-secret
grep -r Secret123 ~/.local/state/telemach "${XDG_RUNTIME_DIR:-/tmp}/telemach" ; echo "grep rc=$?"
```

Attendu : ni à l'écran ni dans l'état, `grep rc=1`. État des conteneurs identique avant et après
(`docker ps -a --format '{{.Names}} {{.Status}}'` comparé).

## V7 — `last`, `deepen`, contexte

`docker telemach last` réaffiche l'analyse (démon non interrogé) ; `docker telemach deepen`
approfondit ; `docker --context default telemach ps` fonctionne (R3).

## Nettoyage

```bash
docker rm -f essai-pg essai-web essai-127 essai-oom essai-stop essai-1 essai-secret
```
