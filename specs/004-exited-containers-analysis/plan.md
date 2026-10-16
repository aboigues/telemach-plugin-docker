# Implementation Plan: Analyse des conteneurs arrêtés en erreur

**Branch**: `004-exited-containers-analysis` | **Date**: 2026-10-16 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/004-exited-containers-analysis/spec.md`

## Summary

`docker telemach ps` explique en français le dernier conteneur arrêté avec un code différent
de 0 (issue #8) : fin du journal, concept, cause probable, correction. Après un `docker run -d`
réussi, l'intégration shell lance en arrière-plan `docker telemach watch`, qui vérifie une fois
l'état du conteneur environ 3 s plus tard et laisse, en cas d'arrêt en erreur, une ligne affichée
à l'invite suivante.

Choix techniques (voir [research.md](research.md)) : lectures du démon par la CLI `docker` de
l'apprenant, limitées à une liste fermée (`ps`, `inspect`, `logs --tail`, `events --until`) dans
un paquet `internal/dockerread` ; repérage du conteneur lancé par le journal d'événements du démon
(couvre `--rm`) ; diagnostic pur dans `internal/exited` avec un nouveau fichier de catalogue
`exited.yaml` ; affichage par un crochet d'invite (`precmd`, `PROMPT_COMMAND`). Correction au
passage d'un bug existant : les options globales placées avant `telemach` cassaient toutes les
sous-commandes.

## Technical Context

**Language/Version**: Go 1.27.2 (inchangé) ; scripts bash et zsh de `shell-init`.

**Primary Dependencies**: aucune nouvelle dépendance. Réutilise `internal/catalog`,
`internal/render`, `internal/redact`, `internal/sanitize`, `internal/state`, `internal/command`,
`internal/textdist`, cobra. Le binaire `docker` de l'apprenant est appelé par `os/exec`
(liste d'arguments, sans shell).

**Storage**: `last.json` (nouveau `Kind: exited`, extrait de journal masqué) ; fichiers
`pending-<pid>` dans `$XDG_RUNTIME_DIR/telemach/` (dossier `0700`, fichiers `0600`), supprimés
dès leur affichage ou au chargement suivant de `shell-init`.

**Testing**:
- tests par tables de `internal/exited` sur un corpus capturé sur un vrai démon
  (`testdata/exited/`, ≥ 12 cas de SC-001, plus des cas légitimes voisins pour les motifs de
  journal) ;
- `internal/dockerread` avec un faux `docker` qui journalise ses appels : seules les lectures
  de la liste fermée sont faites (SC-002), délais et erreurs de la CLI ;
- golden files du rendu (analyse complète, liste des autres conteneurs, signal d'une ligne) ;
- tests shell sous bash **et** zsh (LL-002) : lancement de `watch` seulement après un `run`
  réussi en session interactive, redirections, `TELEMACH_WATCH=0`, crochet d'invite (affichage
  puis suppression, aucun processus quand rien n'est en attente), idempotence du chargement,
  nettoyage des fichiers de shells disparus ;
- tests d'intégration avec un vrai démon : scénarios de SC-001, état des conteneurs identique
  avant et après (SC-002), secrets dans le journal et les variables (SC-003), 200 conteneurs et
  journal de 100 000 lignes (SC-004), vérification après `run -d` (SC-005).

**Target Platform**: Linux, macOS, WSL2 (bash, zsh) — inchangé.

**Project Type**: CLI (plugin Docker).

**Performance Goals**: `ps` < 2 s au p95 avec 200 conteneurs (budget mesuré ~1 s : `ps` ~250 ms,
`inspect` de 25 candidats ~300 ms, `logs --tail 200` ~130 ms) ; invite retardée de moins de
50 ms au p95 par la vérification (lancement en arrière-plan, test de fichier à l'invite).

**Constraints**: lectures seules auprès du même démon que la CLI de l'apprenant ; aucune valeur
de variable d'environnement conservée après décodage ; aucune écriture sur le terminal hors de
l'invite ; aucun processus ni fichier laissé derrière ; garanties de l'intégration shell (001,
003) inchangées.

**Scale/Scope**: 2 sous-commandes (`ps`, `watch`), ~15 entrées de catalogue, ~12 scénarios de
corpus réels, 2 scripts shell modifiés.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principe | Vérification | Statut |
|----------|--------------|--------|
| **I. Vocation pédagogique** | Chaque cause a une entrée de catalogue complète (concept, cause, correction exécutable, approfondissement) ; distinction pédagogique des codes 137/143 (arrêt de l'extérieur, souvent normal) ; cause inconnue dite honnêtement, avec le chemin vers `docker logs` ; le signal après `run -d` relie une erreur invisible à son explication. | ✅ |
| **II. Langue** | Identifiants en anglais (`ps`, `watch`, `internal/exited`, `TELEMACH_WATCH`) ; textes, messages, commentaires et documentation en français. | ✅ |
| **III. Sécurité** | Sorties Docker validées (JSON décodé, nettoyage, masquage) ; exécution par liste d'arguments, sans shell (pas d'injection) ; moindre privilège (droits de l'apprenant, pas de `sudo`, pas d'autre socket) ; liste fermée de lectures testée ; valeurs des variables réduites aux noms dès le décodage ; aucune nouvelle dépendance ; exemples avec tags vérifiés et utilisateur non-root. | ✅ |
| **IV. Gates** | Aucune nouvelle gate : tests unitaires, shell (bash/zsh) et d'intégration dans les jobs existants `test` et `integration` ; le job `integration` dispose d'un démon Docker. | ✅ |
| **V. LESSON-LEARNED** | Relu (LL-001 à LL-013). **LL-002** : tests sous bash et zsh, stdout et stderr. **LL-003** : nettoyage des fichiers `pending-*` avant de compter sur leur suppression à l'affichage. **LL-004** : tags des images du corpus vérifiés sur Docker Hub. **LL-006** : tests shell sans raccourci de PATH. **LL-008** : validations et commit par `&&`. **LL-009** : vérifications ad hoc en `bash -c` (rappel : une mesure de cette recherche a d'abord été faussée sous zsh). **LL-010** : lignes légitimes adversariales pour chaque motif de journal. **LL-012** : objectifs SC-004 et SC-005 mesurés par des tests. **LL-013** : une intégration d'une version antérieure ne lance pas `watch`, l'avertissement de version le signale. | ✅ |

**Dérogation assumée** : la règle « le plugin n'exécute aucune commande » (001, FR-015) reçoit
une exception, décidée par l'utilisateur lors de la clarification du 2026-10-16 et bornée par
FR-010 : elle n'est pas contraire à la constitution (qui encadre l'exécution sans l'interdire),
mais elle est reportée dans la spec et les tests de la 001.

**Après la phase 1** : aucune autre dérogation.

## Project Structure

### Documentation (this feature)

```text
specs/004-exited-containers-analysis/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli-ps.md        # sous-commandes ps et watch, intégration shell, last et deepen
├── checklists/requirements.md
└── tasks.md             # Phase 2 (/speckit-tasks)
```

### Source Code (repository root)

```text
internal/dockerread/           # NOUVEAU : lectures du démon par la CLI (liste fermée)
├── doc.go
├── read.go                    # Runner (exec, délai, options globales), ps, inspect, logs, events
├── types.go                   # Container, State, EnvNames (valeurs écartées au décodage)
└── *_test.go                  # faux docker qui journalise ses appels

internal/exited/               # NOUVEAU : diagnostic pur d'un conteneur arrêté
├── doc.go
├── diagnose.go                # motifs de journal → état → code → inconnu
├── select.go                  # candidats de ps, tri par FinishedAt, désignation par nom
├── watch.go                   # correspondance create/die après run -d
└── *_test.go                  # corpus testdata/exited/

internal/catalog/data/exited.yaml  # NOUVEAU : causes et codes de sortie des conteneurs
internal/catalog/                  # + fallback « exited »
internal/render/                   # + rendu de l'analyse, de la liste et du signal
internal/state/                    # + Kind exited, extrait de journal ; fichiers pending
internal/shellinit/scripts/        # bash.sh, zsh.sh : t0, lancement de watch, crochet d'invite

cmd/docker-telemach/
├── main.go                    # options globales avant « telemach » (R3)
├── ps.go                      # NOUVEAU : sous-commande ps
├── watch.go                   # NOUVEAU : sous-commande cachée watch
├── last.go, deepen.go         # + Kind exited

testdata/exited/               # NOUVEAU : capture.sh, inspect et journaux capturés, attendus
tests/shell/                   # + watch et crochet d'invite (bash et zsh)
tests/integration/             # + scénarios réels, état inchangé, secrets, volume, run -d
specs/001-docker-error-guidance/spec.md, contracts/cli.md  # exception FR-015 / SC-007
README.md, docs/catalogue.md   # documentation (ps, signal après run -d, TELEMACH_WATCH)
```

**Structure Decision**: projet Go unique existant ; deux nouveaux paquets isolent l'accès au
démon (`internal/dockerread`, seul à exécuter `docker`) et le diagnostic pur
(`internal/exited`, testé sur des données capturées, sans démon).

## Complexity Tracking

| Écart | Pourquoi nécessaire | Alternative plus simple écartée parce que |
|-------|---------------------|-------------------------------------------|
| Exception à « le plugin n'exécute aucune commande » | Lire l'état et le journal d'un conteneur demande d'interroger le démon (US1) ; le signal après `run -d` a été choisi par l'utilisateur (option C) | Option B (ligne d'information sans lecture) écartée par l'utilisateur ; aucun moyen de connaître l'état d'un conteneur sans le démon |
| Tâche de fond lancée par l'intégration shell | Vérifier l'état ~3 s après la commande sans retarder l'invite (FR-014) | Vérifier de façon synchrone retarderait l'invite de 3 s ; écrire directement sur le terminal mélangerait le signal à la saisie |
