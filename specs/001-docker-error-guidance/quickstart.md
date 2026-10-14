# Guide de validation (Quickstart) : Guidage pédagogique à partir des erreurs Docker

**Feature** : `001-docker-error-guidance` | **Date** : 2026-10-14

Ce guide prouve, de bout en bout, que la feature fonctionne. Il ne décrit pas
l'implémentation : voir [contracts/cli.md](contracts/cli.md) et [data-model.md](data-model.md).

## Prérequis

- Go **1.27.2** (le `toolchain` du `go.mod` le télécharge automatiquement si besoin)
- Docker Engine 29.x avec le plugin Compose v2 (`docker compose version`)
- bash ou zsh

## Installation locale

```bash
go build -o ~/.docker/cli-plugins/docker-telemach ./cmd/docker-telemach
docker telemach --help                        # le plugin est découvert par Docker
eval "$(docker telemach shell-init zsh)"      # ou bash
```

## Scénarios de validation

### V1 — Port déjà utilisé (US1, scénario 1)

```bash
docker run -d --name telemach-a -p 8080:80 nginx
docker run --rm -p 8080:80 nginx; echo "code: $?"
```

Attendu : le message Docker d'origine s'affiche, suivi de l'explication en français (concept
*port publishing*, cause, deux corrections dont `-p 8081:80`). `code: 125` : le code de sortie
est préservé.

Nettoyage : `docker rm -f telemach-a`

### V2 — Image mal orthographiée (US1, scénario 2)

```bash
docker run --rm ngnix
```

Attendu : explication du parcours « image locale → registre » et suggestion de `nginx`.

### V3 — Démon arrêté (US1, scénario 3)

Avec le démon arrêté (ou `DOCKER_HOST=unix:///nonexistent docker ps`) : explication
client/démon et vérification du démarrage.

### V4 — Erreur applicative (FR-004)

```bash
docker run --rm alpine sh -c 'exit 3'
```

Attendu : le plugin indique que l'erreur vient du programme dans le conteneur (code 3), et non
de Docker, puis oriente vers `docker logs`.

### V5 — Erreur non reconnue (US1, scénario 4, FR-008)

```bash
docker-telemach explain --exit-code 1 --stderr-file <(echo "erreur totalement inventée") -- run x
```

Attendu : bloc « Erreur non reconnue », aucune cause inventée, lien vers la documentation.

### V6 — Compose : tabulation et clé inconnue (US2, scénarios 1 et 2)

```bash
cd testdata/compose/tab-indent && docker compose up
cd ../unknown-key-port && docker compose up
```

Attendu : ligne exacte, extrait YAML coloré avec la ligne fautive surlignée, version corrigée
(`ports:`).

### V7 — Compose : dépendance non déclarée et secret en clair (US2, scénarios 3 et 4)

```bash
docker telemach check testdata/compose/undefined-depends-on
docker telemach check testdata/compose/plaintext-password
```

Attendu : service manquant nommé et services déclarés listés ; mot de passe affiché `••••••`,
avec l'alternative `.env` ou secrets Docker.

### V8 — Vérification préventive sans erreur (US2, scénario 5, FR-014)

```bash
docker telemach check testdata/compose/valid; echo "code: $?"
```

Attendu : `✔ Aucun problème détecté`, `code: 0`.

### V9 — Approfondissement (US3)

Après V1 : `docker telemach deepen`, puis `docker telemach deepen --exercise`.
Attendu : concept détaillé, exemple coloré, exercice sans risque.

### V10 — Répétition (edge case)

Relancer deux fois la commande en échec de V1 : la seconde explication est courte et renvoie
à `deepen`.

### V11 — Coloration et accessibilité (R6)

```bash
docker run --rm ngnix 2>&1 | cat          # pas de couleur (sortie non terminal)
NO_COLOR=1 docker run --rm ngnix          # pas de couleur
CLICOLOR_FORCE=1 docker run --rm ngnix 2>&1 | cat   # couleur forcée
```

Attendu : dans tous les cas, les symboles et libellés (`✖`, `📘`, `💡`, `⚠`) restent présents.

### V12 — Sécurité

```bash
docker run --rm -e DB_PASSWORD=SuperSecret123 alpine sh -c 'exit 1'
grep -r SuperSecret123 ~/.local/state/telemach/ && echo "FUITE" || echo "OK"
```

Attendu : `SuperSecret123` n'apparaît ni à l'écran dans l'explication, ni sur disque (`OK`).

### V13 — Version du client Docker (FR-001b)

```bash
eval "$(docker telemach shell-init zsh)"
docker-telemach explain --exit-code 1 --docker-version 29.8.0 \
  --stderr-file <(echo "erreur totalement inventée") -- run x
```

Attendu : le bloc « Erreur non reconnue » mentionne `Docker 29.8.0`. Avec
`--docker-version n/importe`, la version est ignorée sans erreur.

### V14 — Commande interactive (cas limite)

```bash
docker run -it --rm alpine sh -c 'exit 127'
```

Attendu : le terminal interactif fonctionne normalement (stderr non redirigé), et l'explication
s'appuie sur le seul code de sortie : commande introuvable dans le conteneur (127).

### V15 — Fichier Compose illisible (cas limite)

```bash
mkdir -p /tmp/telemach-v15 && printf 'services:\n  web:\n    image: nginx\n' > /tmp/telemach-v15/compose.yaml
chmod 000 /tmp/telemach-v15/compose.yaml
docker telemach check /tmp/telemach-v15; echo "code: $?"
chmod 600 /tmp/telemach-v15/compose.yaml && rm -rf /tmp/telemach-v15
```

Attendu : problème `file-unreadable`, avec l'explication des droits du fichier ; `code: 1`.

## Validation automatisée

```bash
go test ./...                                   # unitaires, golden files, corpus SC-001/002/006
go test -tags integration ./...                 # avec un vrai démon Docker
go test -fuzz=FuzzMatch -fuzztime=30s ./internal/diagnose
```

Le rapport du corpus affiche le taux de reconnaissance (≥ 90 % attendu pour SC-001 et SC-002)
et le nombre de fuites de secrets (0 attendu pour SC-006).
