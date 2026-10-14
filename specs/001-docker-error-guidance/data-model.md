# Modèle de données (Phase 1) : Guidage pédagogique à partir des erreurs Docker

**Feature** : `001-docker-error-guidance` | **Date** : 2026-10-14

Les noms d'entités et de champs sont en anglais (code, principe II) ; les descriptions sont en
français. Aucune base de données : les entités vivent en mémoire, sauf le catalogue (embarqué
dans le binaire) et le dernier diagnostic (fichier d'état local).

---

## CapturedError — erreur Docker capturée

Ce que l'intégration shell transmet à `docker telemach explain`.

| Champ | Type | Description | Validation |
|-------|------|-------------|------------|
| `Args` | `[]string` | Arguments de la commande, sans le `docker` initial | Non vide ; **masqués** (R7) avant tout usage hors mémoire |
| `ExitCode` | `int` | Code de sortie de la commande | 1 à 255 (0 = pas d'explication) |
| `Stderr` | `string` | Texte d'erreur capturé | Taille max. 1 Mio (au-delà : tronqué, en gardant la fin) ; séquences ANSI retirées ; peut être vide (commande interactive) |
| `CapturedAt` | `time.Time` | Horodatage | — |
| `DockerVersion` | `string` | Version du client Docker (FR-001b) | Optionnel ; fournie par l'option `--docker-version`, lue une fois par l'intégration shell via `docker --version` ; format `^[0-9]+\.[0-9]+\.[0-9]+([-+.][0-9A-Za-z.-]+)?$`, ignorée si invalide |

## Command — commande analysée

Dérivée de `CapturedError.Args`.

| Champ | Type | Description |
|-------|------|-------------|
| `Family` | `enum` | `docker` ou `compose` |
| `Subcommand` | `string` | `run`, `build`, `pull`, `up`, `config`… (FR-003) |
| `Flags` | `map[string]string` | Options reconnues (`-p`, `--name`, `-f`…), valeurs masquées si sensibles |
| `Positionals` | `[]string` | Arguments hors options (image, conteneur, commande du conteneur) ; `Image()` en déduit l'image visée |
| `ComposeFiles` | `[]string` | Fichiers Compose concernés (`-f`, sinon recherche par défaut) |
| `Interactive` | `bool` | `-it`, `exec -it`, `attach`… (R2) |

## CatalogEntry — entrée du catalogue d'erreurs

Fichiers YAML embarqués, un par catégorie (`internal/catalog/data/*.yaml`).

| Champ | Type | Description | Validation |
|-------|------|-------------|------------|
| `ID` | `string` | Identifiant stable, ex. `run.port-already-allocated` | Unique ; `^[a-z]+(\.[a-z0-9-]+)+$` |
| `Category` | `enum` | `network`, `image`, `container`, `volume`, `daemon`, `permission`, `build`, `compose`, `runtime` | Obligatoire |
| `Applies` | `[]string` | Sous-commandes concernées (vide = toutes) | — |
| `ExitCodes` | `[]int` | Codes de sortie concernés (optionnel) | — |
| `Fallback` | `string` | `application` : repli quand l'erreur vient du programme du conteneur ; `compose-issue` : pédagogie d'un type de problème Compose | Optionnel ; une entrée a des motifs, des codes de sortie ou un `fallback` |
| `Patterns` | `[]string` | Expressions régulières RE2 avec groupes nommés | ≥ 1 (sauf entrée fondée sur le seul code de sortie) ; compilées au chargement |
| `Concept` | `ConceptRef` | Concept Docker enseigné | Obligatoire |
| `Summary` | `string` (fr) | Résumé en une phrase, peut utiliser les groupes capturés | Obligatoire, ≤ 160 caractères |
| `Cause` | `string` (fr) | Cause probable | Obligatoire |
| `Fixes` | `[]Fix` | Actions correctives | ≥ 1 (FR-005) |
| `AntiPattern` | `*AntiPatternNote` | Mise en garde si la solution rapide est dangereuse (FR-010) | Optionnel |
| `Deepen` | `Deepening` | Contenu approfondi (US3) | Obligatoire |
| `DocURL` | `string` | Lien vers la documentation officielle | `https://docs.docker.com/…` |
| `Tests` | `[]string` | Cas du corpus `testdata/errors/` qui doivent correspondre (ou dossiers de `testdata/compose/` pour `compose-issue`) | ≥ 1 |

**Fix** : `Description` (fr), `Command` (gabarit facultatif, ex. `docker run -p {{.alt_port}}:80 …`),
`Safe` (`bool`, faux = signalé comme anti-pattern).

**ConceptRef** : `Name` (terme Docker d'origine, ex. « port publishing »), `Definition` (fr,
une ou deux phrases, FR-006).

**Deepening** : `Explanation` (fr), `Example` (commande ou fichier exécutable),
`Exercise` (fr, manipulation courte et sans risque).

## ComposeIssue — problème détecté dans un fichier Compose

| Champ | Type | Description |
|-------|------|-------------|
| `Kind` | `enum` | `yaml-syntax`, `tab-indent`, `unknown-key`, `wrong-type`, `undefined-reference`, `dependency-cycle`, `unresolved-variable`, `plaintext-secret`, `file-not-found`, `file-unreadable` (droits, encodage non UTF-8) |
| `File` | `string` | Chemin du fichier |
| `Line`, `Column` | `int` | Position (1-based) ; 0 si inconnue |
| `Service` | `string` | Service concerné, si applicable |
| `KeyPath` | `string` | Chemin de la clé, ex. `services.web.port` |
| `Suggestion` | `string` | Clé, valeur ou variable proposée, ex. `ports`, `.env`, `backend` |
| `Detail` | `string` | Précision en français (référence manquante, type attendu…) |
| `Excerpt` | `[]Line` | Lignes autour de l'erreur, valeurs sensibles masquées |
| `Fixed` | `[]Line` | Extrait corrigé proposé |
| `Blocking` | `bool` | Vrai si l'erreur empêche l'analyse du reste |

Règles : les problèmes sont triés par fichier puis par ligne ; une erreur bloquante est
affichée en premier (edge case « plusieurs erreurs »).

## Diagnosis — résultat de l'analyse

| Champ | Type | Description |
|-------|------|-------------|
| `Confidence` | `enum` | `recognized`, `partial`, `unknown` (FR-008) |
| `Origin` | `enum` | `docker`, `application`, `unknown` (FR-004, R3) |
| `Entry` | `*CatalogEntry` | Entrée reconnue, si présente |
| `Captures` | `map[string]string` | Valeurs extraites des groupes nommés, masquées |
| `ComposeIssues` | `[]ComposeIssue` | Pour la famille `compose` |
| `Fingerprint` | `string` | SHA-256 de la commande et de l'erreur normalisées (R8) |

**Transitions** : `CapturedError` → (analyse de la commande) `Command` → (catalogue + codes de
sortie + analyse Compose) `Diagnosis` → (rendu) `Explanation`.

## Explanation — explication rendue

| Champ | Description |
|-------|-------------|
| `Summary`, `Concept`, `Cause`, `Fixes`, `AntiPattern` | Issus du `Diagnosis`, gabarits remplis |
| `OriginalError` | Message d'origine, masqué et neutralisé (FR-007) |
| `Repeated` | Vrai si même empreinte depuis moins de 10 min : rendu court |
| `DeepenAvailable` | Vrai si `Deepen` existe : indique `docker telemach deepen` |
| `DockerVersion` | Version du client issue de `CapturedError.DockerVersion` (FR-001b), affichée dans le bloc « Erreur non reconnue » ; vide si inconnue |

## LastDiagnosis — état local persistant

Fichier `$XDG_STATE_HOME/telemach/last.json` (répertoire `0700`, fichier `0600`), écrasé à
chaque diagnostic.

| Champ | Type | Description |
|-------|------|-------------|
| `SchemaVersion` | `int` | `2` (un fichier d'un autre schéma est ignoré puis réécrit) |
| `Fingerprint` | `string` | Empreinte du dernier diagnostic |
| `EntryID` | `string` | Identifiant du catalogue, ou vide |
| `Captures` | `map[string]string` | **Déjà masquées** |
| `RecordedAt` | `time.Time` | Pour la fenêtre de répétition de 10 minutes |
| `Origin`, `Confidence` | `string` | Origine et niveau de reconnaissance du diagnostic |
| `ExitCode` | `int` | Code de sortie de la commande |
| `Interactive` | `bool` | Commande interactive (stderr non capturé) |
| `DockerVersion` | `string` | Version du client Docker, si connue |
| `Stderr` | `string` | Message d'origine **déjà nettoyé et masqué**, 64 Kio au plus (on garde la fin) |
| `Family`, `Subcommand` | `string` | Famille (`docker`, `compose`) et sous-commande |
| `ComposeDir`, `ComposeFiles` | `string`, `[]string` | Projet Compose à réanalyser par `last` |

Invariant : aucune donnée non masquée n'est écrite sur disque. Un fichier illisible ou d'un
schéma inconnu est ignoré et réécrit, sans erreur visible pour l'apprenant.
