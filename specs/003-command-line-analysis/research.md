# Recherche — 003 Analyse de la ligne de commande

LESSON-LEARNED relu avant ce plan (LL-001 à LL-009). Aucune nouvelle dépendance ni action
GitHub : rien à vérifier sur un registre pour cette feature.

## Mesures de départ (2026-10-15, WSL2, binaire `-s -w`)

| Commande | Médiane | p95 |
|---|---|---|
| `docker-telemach --version` (démarrage seul) | 24,6 ms | 27,9 ms |
| `docker-telemach explain …` (catalogue chargé, rendu complet) | 31,8 ms | 36,7 ms |

→ SC-003 (< 50 ms au p95) est tenable pour une analyse avec rendu ; une ligne sans piège peut
sortir avant le chargement du catalogue.

---

## R1 — Où détecter : nouveau paquet `internal/lint`

- **Décision** : un paquet `internal/lint`, fonction pure
  `Analyze(cmd command.Command, args []string, env Env) []Finding`. `Env` fournit deux lectures
  locales injectables : `HasVar(name) bool` (présence d'une variable, jamais sa valeur) et
  `DirExists(name) bool` (dossier dans le dossier courant). Une règle par piège, chacune une
  petite fonction Go avec un identifiant stable.
- **Justification** : la détection est procédurale (ordre des arguments, formats de valeurs,
  combinaisons d'options) et ne se prête pas aux expressions régulières sur stderr du
  catalogue ; une fonction pure se teste par tables, sans shell ni démon.
- **Alternatives** : règles déclaratives en YAML (moteur de règles à écrire, moins lisible pour
  des conditions combinées) ; analyse dans le script shell (non testable finement, double code
  bash/zsh, contraire à LL-002).

## R2 — Textes pédagogiques : le catalogue existant

- **Décision** : les textes (concept, résumé, cause, corrections, anti-pattern, approfondissement)
  vivent dans le catalogue embarqué, nouveau fichier `internal/catalog/data/cmdline.yaml`,
  entrées `cmdline.*` avec `fallback: cmdline` (sans motif, comme `compose-issue`) et un nouveau
  champ `severity: warning|info`. Les règles de `internal/lint` renvoient l'identifiant
  d'entrée et des captures (`{{.option}}`, `{{.fixed_command}}`, `{{.image}}`…).
- **Justification** : un seul endroit pour la pédagogie ; `deepen` et les tests de sécurité du
  catalogue (images d'exemple taguées, non-root) s'appliquent sans code nouveau ; un piège =
  une entrée + une règle + ses cas de test (FR-004, extensibilité).
- **Alternatives** : textes dans le code Go (contraire à l'organisation du projet).

## R3 — Déclenchement depuis l'intégration shell (FR-001, FR-009, FR-013)

- **Décision** :
  - Le script `shell-init` repère, **en shell et sans appel au plugin**, si la commande est
    `run`, `create` ou `exec` (ou `container run|create|exec`), en sautant les options globales
    de Docker (`-c/--context`, `-H/--host`, `--config`, `-l/--log-level`, `--tls*`, `-D`…). Les
    autres commandes ne subissent aucun appel supplémentaire (SC-003).
  - **Code de sortie nul** : après la commande (et après la session pour une commande
    interactive), appel de `docker-telemach lint --after -- "$@"` ; sa sortie va sur stderr.
  - **Code de sortie non nul** : appel inchangé de `explain`, qui exécute aussi l'analyse et
    intègre les pièges trouvés à l'explication (R5) : un seul affichage (FR-010).
  - `TELEMACH_LINT=0` : le script n'appelle pas `lint` et `explain` n'ajoute pas les pièges
    (FR-016) ; `TELEMACH_DISABLE=1` reste prioritaire.
  - Le code de sortie de Docker est renvoyé tel quel ; un échec de `lint` est ignoré
    (`|| true`).
- **Justification** : réutilise les garanties de la 001 (stdin/stdout intacts, code de sortie
  préservé, aucun fichier temporaire pour `lint` puisque rien n'est capturé en cas de succès).
- **Alternatives** : appeler le plugin pour toute commande (délai sur `docker ps` etc., contraire
  à SC-003) ; afficher avant l'exécution (écarté en clarification).

## R4 — Détecter sans faux positif les options mal placées (FR-002, FR-003)

- **Décision** : signaux **forts** et **faibles**, évalués sur les arguments qui suivent
  l'image (`run`, `create`) ou le conteneur (`exec`), avant un éventuel `--` :
  - **forts** (suffisent seuls) — options propres à Docker dont la **valeur a le format Docker** :
    `-p/--publish HÔTE:CONTENEUR` (`[IP:]port[-port]:port[-port][/tcp|udp|sctp]`),
    `-v/--volume SOURCE:/CHEMIN_ABSOLU[:options]`, `-e/--env NOM=VALEUR` (NOM au format
    d'une variable), `--env-file`, `--network/--net VALEUR`, `--restart no|always|on-failure[:n]|unless-stopped`,
    `--mount type=…`, et les options sans équivalent courant dans les programmes : `--rm`,
    `-it`, `-ti`, `--detach`, `--interactive`, `--tty`, `--privileged` ;
  - **faibles** (signalés seulement si au moins un signal fort est présent dans la même ligne) :
    `-d`, `-i`, `-t`, `--name VALEUR`, `-u/--user`, `-w/--workdir`, `-m/--memory` ;
  - Pour `exec`, mêmes principes avec les options d'`exec` (`-it`, `-e`, `-u`, `-w`, `-d`).
- **Justification** : un programme de conteneur accepte souvent `-p`, `-v`, `-d`, `--name`…
  (`postgres -p 5433`, `ls -d`, `curl -v`) ; le format de la valeur (`8090:80`, `./x:/y`,
  `NOM=VALEUR`) ou une option sans équivalent courant lève l'ambiguïté. En cas de doute :
  silence (FR-003, SC-002).
- **Corpus** : au moins 10 lignes fautives et 20 lignes légitimes « piégeuses »
  (arguments d'entrypoint réels de postgres, redis, nginx, curl, ls, node, python, java).

## R5 — Commande en échec avec piège (FR-010)

- **Décision** : `explain` appelle `lint.Analyze` sur les mêmes arguments. Si au moins un piège
  de niveau ⚠ est trouvé : l'explication de l'erreur est affichée normalement, suivie d'une
  section « ⚠ Dans ta commande » qui présente le piège comme **cause probable** (résumé,
  concept, correction) ; si l'erreur n'est pas reconnue (bloc « non reconnue »), le piège
  devient le contenu principal. Les informations (ℹ) ne sont pas ajoutées à une explication
  d'erreur.
- **Justification** : un seul affichage, pas de contradiction ; la cause la plus utile en
  premier quand l'erreur Docker elle-même est opaque (par exemple « exec: "-it": executable
  file not found »).

## R6 — Commande corrigée (FR-006, FR-008)

- **Décision** : reconstruction à partir des arguments d'origine : options globales et
  sous-commande inchangées, options déjà bien placées, **puis** options mal placées (dans leur
  ordre d'origine, avec leur valeur), puis l'image ou le conteneur, puis le reste des
  arguments. Les arguments sont nettoyés (`sanitize`) puis masqués (`redact.RedactArgs`) ;
  affichage avec guillemets simples pour tout argument contenant un espace ou un caractère
  spécial du shell. Si un secret est masqué, la correction précise « remplace •••••• par ta
  valeur (ou utilise --env-file) ».
- Pour les pièges 1 à 7, la correction remplace seulement l'élément en cause
  (`data:` → `./data:`, `-p 80` → `-p 8080:80`, `-e NOM` → `-e NOM=<valeur>`, etc.) ; pour le
  piège 7 (anti-pattern), pas de « correction » automatique mais l'explication des
  alternatives (capacités ciblées `--cap-add`, API distante, etc.).

## R7 — Répétition et information 24 h (FR-015, piège 4)

- **Décision** : nouveau fichier d'état `hints.json` (même dossier privé, `0600`, écriture
  atomique comme `last.json`) : `map[empreinte]date du dernier affichage complet`, purgé des
  entrées de plus de 24 h, plafonné à 200 entrées. Empreinte = identifiant du piège + éléments
  significatifs (options en cause, image pour le piège 4), **jamais** de valeur masquée.
  - affichage complet déjà fait < 10 min → rappel d'une ligne avec la commande corrigée ;
  - piège 4 (image sans tag) déjà affiché < 24 h pour la même image → rien.
- **Justification** : `last.json` ne garde que le dernier diagnostic ; plusieurs pièges et une
  fenêtre de 24 h demandent une petite mémoire séparée.

## R8 — `docker telemach last` (FR-017)

- **Décision** : un avertissement complet est aussi enregistré comme dernier diagnostic
  (`last.json`), avec un champ `Findings` (identifiant + captures de chaque piège) et la
  commande masquée (champ `Args` ajouté par la PR #10) ; `last` les réaffiche avec l'en-tête
  de date et de commande. Un rappel court n'écrase pas le dernier diagnostic complet.

## R9 — Commande `lint` utilisable à la main

- **Décision** : `docker telemach lint [--after] -- <args docker…>` est une sous-commande
  visible : l'apprenant peut faire vérifier une commande **sans l'exécuter**
  (`docker telemach lint -- run -d nginx -p 8090:80`). `--after` (utilisé par l'intégration
  shell) indique que la commande a été exécutée, ce qui permet les formulations « le conteneur
  a démarré, mais… » (FR-007). Codes de sortie : 0 aucun piège ou informations seulement,
  1 au moins un avertissement, 2 usage invalide.
- **Justification** : valeur pédagogique (vérifier avant d'exécuter), tests simples, même
  principe que `check` pour Compose.

## R10 — Performance (SC-003)

- **Décision** : `lint` sort immédiatement (sans charger le catalogue ni le thème) si
  l'analyse ne trouve rien ; mesure dans les tests d'intégration (p95 < 50 ms, tolérance CI
  documentée) et contrôle qu'aucune commande hors `run|create|exec` n'appelle le plugin
  (test du script shell avec un faux binaire qui journalise ses appels).
