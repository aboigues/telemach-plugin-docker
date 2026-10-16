# Quickstart — Validation de l'analyse de la ligne de commande (003)

Scénarios de bout en bout ; formats détaillés dans le [contrat](contracts/cli-lint.md).

## Prérequis

- Plugin compilé depuis la branche et installé dans `~/.docker/cli-plugins/` (procédure du
  README), intégration shell rechargée (nouveau terminal) ; Docker en marche.
- bash **et** zsh pour V3.

## V1 — Issue #9 (US1)

```bash
docker run -d nginx:1.30-alpine -p 8090:80
```

**Attendu** : identifiant du conteneur sur stdout (comme sans TELEMACH), puis l'avertissement
« Option Docker placée après l'image » avec `docker run -d -p 8090:80 nginx:1.30-alpine` ;
`echo $?` → 0. Exécuter la commande corrigée (après `docker stop` puis `docker rm` du premier) :
`curl -sI http://localhost:8090` répond. Relancer la commande fautive dans les 10 minutes →
rappel d'une ligne. `docker telemach last` → l'avertissement complet, avec date et commande.

## V2 — Pas de faux positif (FR-003, SC-002)

```bash
docker run --rm alpine:3.24 ls -l /
docker run --rm postgres:18-alpine postgres --version
docker run --rm -p 8080:80 -d nginx:1.30-alpine
```

**Attendu** : aucun avertissement (le troisième est correctement écrit ; arrêter le conteneur
ensuite).

## V3 — Garanties shell (SC-004), bash et zsh

`go test ./tests/shell/...` : pour chaque cas du corpus, stdout, stderr de Docker et code de
sortie identiques avec et sans TELEMACH ; `lint` appelé seulement pour `run|create|exec`
réussies (faux binaire journalisant ses appels) ; `TELEMACH_LINT=0` → aucun appel.

## V4 — Pièges silencieux (US2)

| Commande | Attendu |
|---|---|
| `mkdir -p data && docker run --rm -v data:/d alpine:3.24 ls /d` | ⚠ volume nommé, correction `-v ./data:/d` ; `docker volume rm data` ensuite |
| `docker run -d --rm -p 80 nginx:1.30-alpine` | ℹ port aléatoire, `docker port <conteneur>` |
| `unset DB_HOST; docker run --rm -e DB_HOST alpine:3.24 env` | ⚠ variable non définie |
| `docker run --rm alpine true` (deux fois) | ℹ image sans tag la première fois seulement |
| `docker run -d --rm --network host -p 8080:80 nginx:1.30-alpine` | ⚠ ports ignorés en réseau host |
| `docker run --rm -e DB_PASSWORD=Secret123 alpine:3.24 true` | ⚠ secret en clair, valeur jamais affichée (`grep -r Secret123 ~/.local/state/telemach` vide) |
| `docker run --rm --privileged alpine:3.24 true` | ⚠ anti-pattern privilèges |

## V5 — Échec avec piège (FR-010)

```bash
docker exec mon-conteneur -it sh     # conteneur existant
```

**Attendu** : un seul affichage : l'explication présente `-it` mal placé comme cause probable
et propose `docker exec -it mon-conteneur sh`.

## V6 — Contrôle (US3)

`TELEMACH_LINT=0 docker run -d nginx:1.30-alpine -p 8090:80` → aucun avertissement ; une
erreur Docker reste expliquée. `docker telemach lint -- run -d nginx -p 8090:80` → même
avertissement au conditionnel, sans rien exécuter, code 1.

## V7 — Délai (SC-003)

Test d'intégration : 40 exécutions de `docker-telemach lint --after -- run …` → p95 < 50 ms.
