# Recherche : analyse des conteneurs arrêtés en erreur

**Feature** : `004-exited-containers-analysis` | **Date** : 2026-10-16

Mesures et essais réalisés le 2026-10-16 sur WSL2, Docker 29.8.0 (Docker Desktop, contexte
`default`, socket `unix:///var/run/docker.sock`).

## R1 — Comment interroger le démon

- **Décision** : appeler le binaire `docker` de l'apprenant par `os/exec`, avec une liste
  d'arguments (jamais de shell, jamais de concaténation), au travers d'un seul paquet
  `internal/dockerread` qui n'expose que des lectures (R2).
- **Justification** : la CLI applique d'elle-même le contexte courant, `DOCKER_HOST`,
  `DOCKER_CONTEXT`, `DOCKER_CONFIG` et les certificats TLS : TELEMACH parle exactement au même
  démon que l'apprenant (FR-010) sans réimplémenter cette logique. Aucune nouvelle dépendance
  (principe III, chaîne d'approvisionnement). Coût mesuré d'un aller-retour : ~50 ms
  (`docker version`).
- **Alternatives écartées** : client Go de l'API Engine (`github.com/moby/moby/client`) :
  dépendance lourde, gestion des contextes et du TLS à refaire, risque de parler à un autre
  démon que la CLI ; socket HTTP à la main : même problème, et plus de surface.
- **Binaire** : `docker` trouvé dans le PATH (la CLI exporte
  `DOCKER_CLI_PLUGIN_ORIGINAL_CLI_COMMAND=docker`, un nom, pas un chemin). Un programme lancé
  par le shell ne voit pas la fonction `docker` de l'intégration (les fonctions ne sont pas
  exportées) : c'est bien le vrai binaire qui est appelé.

## R2 — Lectures autorisées (FR-010)

- **Décision** : liste fermée de sous-commandes, vérifiée par un test (faux `docker` qui
  journalise ses appels) :

  | Lecture | Usage |
  |---|---|
  | `ps -a --no-trunc --filter status=exited --filter status=restarting --filter status=created --format '{{json .}}'` | candidats de `ps` |
  | `inspect <id>…` (sortie JSON) | état, code, `FinishedAt`, `OOMKilled`, `Error`, redémarrages, image, commande, noms des variables, étiquettes Compose |
  | `logs --tail 200 <id>` | fin du journal (stdout et stderr) |
  | `events --since <t0> --until <t1> --filter type=container --filter event=create --filter event=die --format '{{json .}}'` | vérification après `run -d` (R5) |

  Chaque appel a un délai maximal (5 s) ; une erreur de la CLI (démon injoignable, permission
  refusée) est passée au diagnostic existant (`explain`) au lieu d'être réinterprétée.
- **Justification** : une liste blanche testée rend FR-010 vérifiable (SC-002) ; `inspect`
  et `logs` ne modifient rien ; `events` avec `--until` lit l'historique et rend la main.
- **Variables d'environnement** : la sortie d'`inspect` contient leurs valeurs. Le décodage
  JSON les réduit **immédiatement** à leurs noms (type dédié qui ne conserve que la partie
  avant `=`) ; aucune valeur n'est gardée en mémoire au-delà du décodage, ni affichée, ni
  écrite (FR-009, SC-003).

## R3 — Options globales devant le nom du plugin (bug existant)

- **Constat** : la CLI transmet au plugin les options globales placées avant son nom :
  `docker --context default telemach ps` donne `--context default telemach ps`. Or `main.go`
  ne retire `telemach` que s'il est en première position : aujourd'hui, **toutes** les
  sous-commandes échouent (« unknown command "telemach" ») dès qu'une option globale est
  utilisée. Reproduit le 2026-10-16 avec un `DOCKER_CONFIG` temporaire.
- **Décision** : `main` sépare les options globales de Docker (même liste que
  `internal/command`) jusqu'au mot `telemach`, et les transmet en tête de chaque lecture de R2
  (`docker --context x ps …`). Corrigé dans cette feature (nécessaire à FR-010 : « même
  démon »), avec une entrée LESSON-LEARNED.

## R4 — Sélection des conteneurs pour `ps` (FR-001, FR-002, SC-004)

- **Mesures** : `ps -a` : 150 à 290 ms ; `inspect` groupé : ~12 ms par conteneur (33
  conteneurs : 360 à 510 ms) ; `logs --tail 200` sur un journal de 150 000 lignes : 95 à
  140 ms (pilote `json-file`). Inspecter 200 conteneurs coûterait ~2,5 s : trop pour SC-004.
- **Décision** :
  1. `ps` filtré sur `exited`, `restarting` et `created` ; le champ `Status` (« Exited (1) 3
     minutes ago ») donne le code sans inspection : les codes 0 sont écartés à ce stade. Un
     conteneur `created` (statut « Created ») est inspecté : c'est le cas d'un démarrage qui a
     échoué (vérifié : `docker run -d … commande-inexistante` renvoie 127, et le conteneur reste
     `created` avec `State.ExitCode` 127 et l'erreur dans `State.Error`) ; un `created` sans
     erreur (simple `docker create`) est écarté ;
  2. `inspect` groupé des **25** candidats les plus récemment créés (un seul appel) ; tri par
     `FinishedAt` décroissant ; un conteneur `restarting` est rangé par son dernier arrêt ;
  3. `logs --tail 200` du seul conteneur expliqué.
  Budget : ~1 s. Limite documentée : un conteneur créé il y a longtemps, relancé puis arrêté
  récemment, peut sortir des 25 ; `docker telemach ps <nom>` le trouve toujours.
- **Désignation** (`ps <nom>`) : `inspect` direct du nom ou du préfixe d'identifiant ;
  « introuvable » → noms proches (distance d'édition, paquet `textdist` existant) parmi les
  noms de `ps -a`.

## R5 — Retrouver le conteneur lancé par `docker run -d` (FR-013)

- **Contrainte** : l'intégration shell ne redirige jamais la sortie standard (garantie 2 de
  la 001) : l'identifiant affiché par `docker run -d` n'est pas lu. Modifier la commande
  (`--cidfile`) est interdit (FR-012 de la 003).
- **Essai** : quatre lancements (`postgres:18-alpine` sans mot de passe, `alpine` `--rm` avec
  `exit 3`, `alpine` `--rm` avec `true`, `nginx-unprivileged` qui reste démarré), puis
  `docker events --since t0 --until now --filter event=die` : **95 ms**, trois événements
  `die` avec nom, image **telle que tapée** (`postgres:18-alpine`) et `exitCode` (1, 3, 0),
  y compris pour les conteneurs déjà supprimés par `--rm` ; aucun pour celui qui tourne.
- **Décision** : le conteneur lancé est le premier événement `create` postérieur à `t0`
  (début de la commande) dont l'image est celle de la ligne (et le nom celui de `--name`, s'il
  est donné) ; il est en erreur si un événement `die` de même identifiant a un `exitCode`
  différent de 0. Le cas `--rm` est ainsi couvert, avec son code (l'option « code inconnu »
  envisagée dans la spec devient inutile) ; un `--rm` sorti avec 0 n'est pas signalé.
- **Limite** : deux lancements simultanés de la même image dans deux terminaux peuvent être
  confondus ; sans conséquence grave (le signal renvoie à `ps`), documentée.
- **t0** : relevé par le shell avant la commande, sans processus supplémentaire :
  `$EPOCHREALTIME` (bash ≥ 5, zsh avec `zmodload zsh/datetime` ; vérifiés : bash 5.2.21,
  zsh 5.9) ; à défaut, `$SECONDS` ne suffit pas (pas d'horloge absolue) et la vérification est
  désactivée. Bash écrit `$EPOCHREALTIME` avec le séparateur décimal de la locale
  (`1792151265,045881` en `LC_NUMERIC=fr_FR`) : `watch` accepte la virgule comme le point
  (non vérifiable ici, locale française absente ; couvert par un test unitaire).

## R6 — Tâche de fond et affichage à l'invite (FR-014, FR-015)

- **Essais** (vrais shells interactifs dans un pseudo-terminal, bash 5 et zsh 5.9) : une tâche
  lancée avec `&` **depuis la fonction `docker`**, qui s'exécute dans un sous-shell, ne produit
  aucun message de tâche (`[1] …`, « done ») ; avec stdin, stdout et stderr redirigés vers
  `/dev/null`, une substitution `x=$(docker run -d …)` revient en 2 ms au lieu d'attendre la
  tâche.
- **Décision** :
  - après un `run|container run` **réussi**, dans un shell **interactif**, si
    `TELEMACH_WATCH` ≠ `0` : `command "$__telemach_bin" watch --since "$t0" --shell-pid $$ --
    "$@" </dev/null >/dev/null 2>&1 &` ; c'est `watch` (Go) qui décide s'il y a `-d` ou
    `--detach` (même découpage que `internal/lint`) et sort aussitôt sinon ;
  - `watch` attend 3 s, lit les événements (R5), et s'il y a un arrêt en erreur écrit **une
    ligne** dans `$XDG_RUNTIME_DIR/telemach/pending-<pid du shell>` (dossier `0700`, fichier
    `0600`, ajout atomique) — seulement si le shell est toujours vivant (`kill(pid, 0)`) ;
  - un crochet d'invite (`precmd` via `add-zsh-hook` ; `PROMPT_COMMAND` en bash, forme chaîne
    ou tableau, ajout idempotent) affiche puis supprime ce fichier ; s'il n'existe pas, le
    crochet ne lance aucun processus (simple test `[[ -s … ]]`) ;
  - fichiers de shells disparus : supprimés au chargement suivant de `shell-init`.
- **Justification** : l'affichage à l'invite garantit qu'on n'écrit jamais au milieu d'une
  saisie ou d'une sortie ; le test de vie du shell et le nettoyage au chargement couvrent la
  fermeture du terminal (LL-003).
- **Alternative écartée** : écrire directement sur le terminal depuis la tâche de fond
  (mélange avec la saisie en cours, `zle` et `readline` ne le gèrent pas proprement).

## R7 — Diagnostic d'un conteneur arrêté (FR-004, FR-005, FR-007)

- **Décision** : nouveau paquet `internal/exited`, pur (sans accès au démon) : entrée = état
  décodé + lignes de journal masquées ; sortie = entrée de catalogue + captures. Ordre :
  1. motifs de journal (entrées `fallback: exited` avec `patterns`, appliqués aux 200 dernières
     lignes, la ligne la plus récente qui correspond l'emporte) ;
  2. état : `OOMKilled` → manque de mémoire ; `State.Error` non vide → erreur de démarrage
     enregistrée par Docker ; redémarrages > 0 → mention de la politique de redémarrage ;
  3. code de sortie : entrées `exited.code-<n>` (1, 2, 125, 126, 127, 137, 139, 143) ;
  4. sinon `exited.unknown` : extrait du journal, signification générique, `docker logs`.
  Le code 137 sans `OOMKilled` et le code 143 sont présentés comme un arrêt de l'extérieur
  (`docker stop`, `docker kill`), souvent normal. Vérifié le 2026-10-16 : `--memory 8m` avec
  une boucle qui consomme → 137, `OOMKilled` vrai ; `alpine:3.24 sleep 300` puis `docker stop`
  → arrêt forcé (le programme, PID 1, ignore SIGTERM), 137, `OOMKilled` faux.
- **Catalogue** : nouveau fichier `internal/catalog/data/exited.yaml` ; les entrées
  `runtime.exit-*` de la 001 sont rédigées pour une commande qui échoue, d'où des entrées
  propres (même concepts, autre formulation : « le conteneur s'est arrêté… »).
- **Corpus** : `testdata/exited/capture.sh` crée de vrais conteneurs en erreur (≥ 12 cas de
  SC-001) et enregistre `inspect` (variables réduites aux noms) et la fin du journal ; les
  tags d'images sont vérifiés sur Docker Hub au moment de l'écriture (LL-004). Lignes
  légitimes voisines ajoutées pour les motifs de journal (LL-010 : une ligne « permission
  denied » dans un journal par ailleurs normal ne suffit pas si le code est 0).

## R8 — État conservé (FR-012)

- **Décision** : `LastDiagnosis.Kind = "exited"`, avec les captures, l'extrait de journal
  masqué (20 lignes au plus) et le nom du conteneur ; `last` le réaffiche sans interroger le
  démon ; `deepen` utilise l'entrée retenue. Aucune valeur de variable n'est jamais écrite.

## R9 — Performance de la vérification (SC-005)

- **Décision** : le coût ajouté à l'invite est celui du lancement en arrière-plan (un `fork`)
  et du test `[[ -s … ]]` à chaque invite ; mesuré par un test shell (p95 < 50 ms ajoutés,
  LL-012). Le coût de `watch` lui-même (≈ 3 s d'attente + ~100 ms de lecture) n'est jamais
  attendu par le shell.

## R10 — Mise à jour de la feature 001 (FR-010, hypothèses)

- `specs/001-docker-error-guidance/spec.md` FR-015 et SC-007 renvoient vers l'exception de la
  004 ; `TestSC007NoCommandExecuted` reste inchangé pour `explain`, `check`, `lint` ; un
  nouveau test vérifie que `ps` et `watch` n'appellent que les lectures de R2.
- Une intégration shell chargée par une version antérieure ne lance pas `watch` :
  l'avertissement de version de `last` et `deepen` (issue #7, LL-013) le signale.
