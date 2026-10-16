# Modèle de données : analyse des conteneurs arrêtés en erreur

**Feature** : `004-exited-containers-analysis` | **Date** : 2026-10-16

## Container (`internal/dockerread`)

Décodé de la sortie JSON de `docker inspect`. Toutes les chaînes affichées passent par
`sanitize.Clean` puis `redact.Redact` (`redact.RedactArgs` pour la commande).

| Champ | Source `inspect` | Règle |
|---|---|---|
| `ID` | `Id` | affiché sur 12 caractères |
| `Name` | `Name` | sans le `/` initial |
| `Image` | `Config.Image` | telle que tapée au lancement |
| `Command` | `Path` + `Args` | masquée (`RedactArgs`) |
| `ExitCode` | `State.ExitCode` | |
| `FinishedAt` | `State.FinishedAt` | heure UTC ; ancienneté affichée en français |
| `StartedAt` | `State.StartedAt` | délai de vie = `FinishedAt − StartedAt` |
| `OOMKilled` | `State.OOMKilled` | |
| `Error` | `State.Error` | erreur de démarrage enregistrée par Docker |
| `Restarting` | `State.Restarting` | |
| `RestartCount` | `RestartCount` | |
| `RestartPolicy` | `HostConfig.RestartPolicy.Name` | `no`, `always`, `unless-stopped`, `on-failure` |
| `MemoryLimit` | `HostConfig.Memory` | octets, 0 = sans limite |
| `LogDriver` | `HostConfig.LogConfig.Type` | pour expliquer un journal illisible |
| `EnvNames` | `Config.Env` | **noms seulement** : le type `EnvNames` implémente `UnmarshalJSON` et ne garde que la partie avant `=` ; aucune valeur conservée |
| `ComposeProject`, `ComposeService` | étiquettes `com.docker.compose.project` et `.service` | facultatifs |

## Candidate (`docker ps`)

Ligne de `docker ps -a --filter status=exited --filter status=restarting --filter status=created --format '{{json .}}'` :
`ID`, `Names`, `Image`, `Status` (« Exited (1) 3 minutes ago », « Restarting (1) 5 seconds
ago »), `CreatedAt`. Le code est extrait de `Status` (`^(Exited|Restarting) \((\d+)\)`) ; un
statut qui ne correspond pas à ce motif (dont « Created ») est inspecté (pas de devinette) ; un
`created` n'est retenu que si `State.ExitCode` ≠ 0 et `State.Error` non vide (démarrage échoué).

## LogExcerpt

| Champ | Règle |
|---|---|
| `Lines` | ≤ 200 lignes lues (`logs --tail 200`), stdout et stderr confondus, chacune nettoyée et masquée ; ≤ 20 affichées et conservées |
| `Unavailable` | vrai si la CLI répond que le pilote ne permet pas la lecture ; message d'origine masqué conservé |

## Diagnosis (`internal/exited`)

| Champ | Description |
|---|---|
| `Container` | conteneur analysé (ci-dessus) |
| `EntryID` | entrée retenue (`exited.*`) |
| `Source` | `log` (motif du journal), `state` (OOM, erreur de démarrage), `code` (code de sortie) ou `unknown` |
| `Captures` | valeurs pour les gabarits (nom, image, code, délai, variable attendue, ligne du journal en cause…), nettoyées et masquées |
| `Others` | jusqu'à 10 autres conteneurs en erreur : nom, code, ancienneté, résumé d'une ligne |

Ordre de décision (research R7) : motif de journal (ligne la plus récente qui correspond) →
état (`OOMKilled`, `Error`) → code (`exited.code-<n>`) → `exited.unknown`. Une entrée de motif
peut exiger un code (par exemple « permission denied » seulement si `ExitCode` ≠ 0) et une
image (par exemple les messages de Postgres ou de MariaDB).

## Entrées de catalogue `exited.*` (fichier `exited.yaml`)

| Identifiant | Source | Condition (résumé) |
|---|---|---|
| `exited.missing-required-env` | log | message d'une image officielle qui exige une variable (Postgres `POSTGRES_PASSWORD`, MariaDB/MySQL mot de passe root) ; capture `variable` ; correction par `--env-file` ou variable `*_FILE` |
| `exited.permission-denied` | log | « permission denied » / « Permission denied » et code ≠ 0 |
| `exited.address-in-use` | log | « address already in use » / « EADDRINUSE » |
| `exited.file-not-found` | log | « no such file or directory » sur un fichier de l'application, code ≠ 0 |
| `exited.exec-format` | log ou état | « exec format error » (architecture de l'image) |
| `exited.oom` | état | `OOMKilled` vrai (code 137) ; capture `memory_limit` |
| `exited.start-error` | état | `State.Error` non vide (erreur enregistrée au démarrage) |
| `exited.code-1` | code | erreur générale de l'application |
| `exited.code-2` | code | usage incorrect du programme (arguments) |
| `exited.code-125` | code | erreur de Docker à la création ou au démarrage |
| `exited.code-126` | code | commande trouvée mais non exécutable |
| `exited.code-127` | code | commande introuvable dans l'image |
| `exited.code-137` | code | arrêt forcé (SIGKILL) : `docker kill`, ou `docker stop` après délai |
| `exited.code-139` | code | erreur de segmentation (SIGSEGV) |
| `exited.code-143` | code | arrêt demandé (SIGTERM) : `docker stop`, souvent normal |
| `exited.unknown` | — | aucune source précise : signification générique du code, `docker logs <nom>` |

Toute entrée `exited.*` a `fallback: exited`, un concept, une cause, au moins une correction ou
un anti-pattern, un approfondissement exécutable avec images taguées et non-root, et au moins un
cas du corpus `testdata/exited/`.

## PendingSignal (fichiers `pending-<pid>`)

| Champ | Règle |
|---|---|
| emplacement | `$XDG_RUNTIME_DIR/telemach/pending-<pid du shell>` (repli : dossier d'état) ; dossier `0700`, fichier `0600` |
| contenu | une ligne par conteneur signalé, déjà rendue (texte nettoyé et masqué, sans couleur si `NO_COLOR`) |
| écriture | par `watch`, en ajout atomique, seulement si le shell `pid` est vivant |
| lecture | par le crochet d'invite du shell `pid`, qui affiche puis supprime |
| nettoyage | au chargement de `shell-init` : fichiers des `pid` qui ne sont plus vivants |

## LastDiagnosis (extension de `internal/state`)

`Kind` gagne la valeur `exited` ; champs ajoutés : `Container` (nom, identifiant court),
`LogExcerpt` (≤ 20 lignes masquées). `EntryID` et `Captures` comme pour les erreurs. Jamais de
valeur de variable d'environnement.
