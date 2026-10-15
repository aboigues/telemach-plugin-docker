# Modèle de données — 003 Analyse de la ligne de commande

## Finding (résultat d'une règle de `internal/lint`)

| Champ | Type | Règle |
|---|---|---|
| `EntryID` | texte | identifiant d'une entrée `cmdline.*` du catalogue (tableau ci-dessous) |
| `Severity` | `warning` \| `info` | repris de l'entrée de catalogue |
| `Captures` | map texte → texte | valeurs insérées dans les gabarits ; **nettoyées et masquées** |
| `Fingerprint` | texte | identifiant + éléments significatifs (noms d'options, image) ; **jamais de valeur** |

Ordre de présentation : avertissements d'abord, puis informations ; à niveau égal, ordre
d'apparition dans la ligne. Une même règle ne produit qu'un Finding par ligne (elle regroupe
toutes les options concernées).

## Entrées de catalogue `cmdline.*`

| Identifiant | Niveau | Condition (résumé, détail en research R4/R5) | Captures principales |
|---|---|---|---|
| `cmdline.option-after-image` | warning | `run`/`create` : signal fort après l'image, ou faible + fort | `options`, `image`, `fixed_command`, `effect` |
| `cmdline.option-after-container` | warning | `exec` : idem après le conteneur | `options`, `container`, `fixed_command` |
| `cmdline.named-volume-not-dir` | warning | source `-v` sans `/` ni `.` initial **et** dossier de ce nom dans le dossier courant | `name`, `target`, `fixed_command` |
| `cmdline.random-host-port` | info | `-p` sans port de la machine | `cport`, `fixed_command` |
| `cmdline.env-unset` | warning | `-e NOM` sans `=` **et** `NOM` absente de l'environnement | `name`, `fixed_command` |
| `cmdline.image-without-tag` | info | image sans tag ni digest, ou `:latest` ; 24 h par image | `image` |
| `cmdline.ports-ignored-host-network` | warning | `--network host`/`--net host` **et** `-p`/`-P`/`--publish` | `options`, `fixed_command` |
| `cmdline.secret-in-command` | warning | valeur de `-e`/`--env` masquée par `redact` | `name` (jamais la valeur) |
| `cmdline.privileged` | warning (anti-pattern) | `--privileged`, ou source de montage `/var/run/docker.sock` (ou `/run/docker.sock`) | `option` |

Validation du catalogue (tests) : toute entrée `cmdline.*` a `fallback: cmdline`, une
`severity`, un concept, une cause, au moins une correction ou un anti-pattern, un
approfondissement, des images d'exemple taguées et non-root.

## LastDiagnosis (extension de `internal/state`)

| Champ ajouté | Description |
|---|---|
| `Kind` | `error` (défaut, fichiers existants) ou `cmdline` (avertissement seul) |
| `Findings` | liste `{EntryID, Captures}` des pièges affichés en complet |

Un rappel court ne réécrit pas `last.json`. `Args` (ajouté par la PR #10) contient la commande
masquée.

## HintStore (`hints.json`)

| Champ | Description |
|---|---|
| `SchemaVersion` | 1 |
| `Shown` | map `Fingerprint` → date du dernier affichage complet |

Règles : purge à chaque écriture des entrées de plus de 24 h ; au-delà de 200 entrées, les plus
anciennes sont supprimées ; fichier `0600`, écriture atomique ; fichier illisible ou d'un autre
schéma → ignoré (aucune erreur visible).

| Situation | Affichage |
|---|---|
| empreinte inconnue | complet, empreinte enregistrée |
| affichée en complet il y a < 10 min | rappel d'une ligne (avec commande corrigée) |
| `cmdline.image-without-tag`, même image, < 24 h | rien |
| sinon | complet, date mise à jour |
