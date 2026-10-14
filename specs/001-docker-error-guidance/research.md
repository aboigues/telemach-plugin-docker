# Recherche (Phase 0) : Guidage pédagogique à partir des erreurs Docker

**Feature** : `001-docker-error-guidance` | **Date** : 2026-10-14

Toutes les versions citées ont été vérifiées le 2026-10-14 sur les registres officiels
(`go.dev/dl`, `proxy.golang.org`, GitHub Releases), conformément au principe III.

---

## R1. Forme du livrable et langage

- **Decision** : plugin Docker CLI écrit en **Go 1.27.2**, livré sous forme d'un binaire
  statique unique `docker-telemach`, installé dans `~/.docker/cli-plugins/` et appelé via
  `docker telemach …`.
- **Rationale** : choix validé par l'utilisateur. Go est le langage de l'écosystème Docker ;
  binaire autonome sans interpréteur ni dépendance à l'exécution ; démarrage en quelques
  millisecondes (SC-003) ; réutilisation du parseur officiel Compose ; fonctionnement 100 %
  local, sans réseau.
- **Alternatives considered** : CLI Python (dépendances et interpréteur requis chez
  l'apprenant) ; plugin Claude Code (ne couvre pas les commandes tapées hors de Claude Code,
  contraire à « autonome »).
- **Note d'environnement** : la machine de développement dispose de Go 1.22.2, obsolète ; le
  `go.mod` déclarera `go 1.27` et `toolchain go1.27.2`, ce qui déclenche le téléchargement
  automatique de la bonne chaîne d'outils.

## R2. Capture automatique des erreurs (FR-001, FR-001a)

- **Decision** : **intégration shell** générée par `docker telemach shell-init bash|zsh`, à
  activer par `eval "$(docker telemach shell-init zsh)"` dans le fichier rc. Elle définit une
  fonction `docker` qui :
  1. exécute `command docker "$@"` sans toucher à stdin/stdout ;
  2. duplique stderr (`2> >(tee "$tmp" >&2)`) vers un fichier temporaire créé avec
     `mktemp` dans `$XDG_RUNTIME_DIR` (ou `$TMPDIR`), permissions `0600` ;
  3. si le code de sortie est non nul, appelle
     `command docker-telemach explain --exit-code "$rc" --stderr-file "$tmp" -- "$@"` ;
  4. supprime le fichier temporaire (`trap` de nettoyage) et **renvoie le code de sortie
     d'origine**, inchangé.
- **Rationale** : seule méthode qui donne accès au **texte** de l'erreur pour toute commande
  `docker` tapée dans le terminal, sans modifier Docker. Seules les commandes `docker` passent
  par la fonction : FR-001a est garanti par construction. L'apprenant garde la sortie Docker
  originale à l'écran (FR-007).
- **Alternatives considered** :
  - *Hooks des plugins Docker CLI* (`docker-cli-plugin-hooks`) : le contrat officiel
    (`cli-plugins/hooks`) ne transmet que la commande racine et les **noms** des options, ni
    leurs valeurs ni la sortie d'erreur. Impossible d'expliquer l'erreur. Rejeté.
  - *Remplacer le binaire `docker` par un wrapper dans le PATH* : intrusif, fragile lors des
    mises à jour de Docker, risque de casser d'autres outils. Rejeté.
  - *`preexec`/`precmd` du shell* : ne donne pas accès à stderr. Rejeté.
- **Version du client Docker (FR-001b)** : lue une fois à l'évaluation du script par
  `command docker --version`, une commande purement locale qui n'interroge pas le démon (contrairement à
  `docker version`). Le résultat est mis en cache dans une variable du shell et transmis par
  `--docker-version`. Aucune autre exécution n'est autorisée (FR-015).
- **Garde-fous** :
  - commandes interactives (`-it`, `exec -it`, `attach`, `login` sans `--password-stdin`) :
    stderr n'est pas redirigé, pour ne pas casser le TTY. Seul le code de sortie est
    transmis, et l'explication se limite aux cas reconnaissables sans texte d'erreur ;
  - si `docker-telemach` est absent ou échoue, la fonction se comporte exactement comme
    `docker` (edge case « capture impossible ») ;
  - variable `TELEMACH_DISABLE=1` pour désactiver ponctuellement.

## R3. Distinguer l'erreur Docker de l'erreur applicative (FR-004)

- **Decision** : s'appuyer sur la convention des codes de sortie de `docker run` :
  `125` = erreur du client ou du démon Docker, `126` = commande du conteneur non exécutable,
  `127` = commande du conteneur introuvable, autre code = code de sortie du processus du
  conteneur (erreur applicative → orienter vers `docker logs`). Les codes `137` (SIGKILL,
  souvent OOM) et `143` (SIGTERM) ont une explication dédiée.
- **Rationale** : convention documentée et stable, valeur pédagogique forte (comprendre le
  cycle de vie d'un conteneur).
- **Alternatives considered** : analyse du texte seule (ambiguë quand l'application écrit sur
  stderr).

## R4. Reconnaissance des erreurs : catalogue déclaratif

- **Decision** : un **catalogue d'erreurs** en fichiers YAML embarqués dans le binaire
  (`go:embed`), un fichier par catégorie. Chaque entrée associe des motifs (expressions
  régulières RE2 de la bibliothèque standard `regexp`) à un concept, une cause, des
  corrections et un approfondissement, tous rédigés en français, avec des variables
  capturées (ex. `{{.port}}`). Le catalogue est validé par des tests (schéma, unicité des
  identifiants, présence d'au moins un cas de test par entrée).
- **Rationale** : sépare le contenu pédagogique du code (principes I et II) ; extensible sans
  toucher au moteur ; le moteur RE2 de Go garantit un temps linéaire (pas de ReDoS, même sur
  une sortie d'erreur hostile — principe III).
- **Alternatives considered** : règles codées en dur en Go (contenu pédagogique mêlé au code,
  difficile à relire) ; modèle de langage local (lourd, non déterministe, risque d'inventer
  des causes, contraire à FR-008).

## R5. Analyse des fichiers Compose (FR-011 à FR-014)

- **Decision** : analyse en deux passes.
  1. **Passe syntaxique et positionnelle** avec `go.yaml.in/yaml/v3` **v3.0.5** (API `Node`,
     qui conserve ligne et colonne) : détection des tabulations, du YAML mal formé, des clés
     inconnues (comparées aux clés du schéma Compose) et suggestion de la clé la plus proche
     par distance de Levenshtein (implémentation interne, quelques lignes, pour éviter une
     dépendance).
  2. **Passe sémantique** avec `github.com/compose-spec/compose-go/v2` **v2.16.1**, le
     parseur de référence utilisé par Docker Compose : validation par le schéma officiel,
     références de services, réseaux et volumes, cycles `depends_on`. Les erreurs renvoyées
     sont rattachées aux positions de la passe 1.
- **Rationale** : compose-go garantit la fidélité au comportement réel de `docker compose` ;
  yaml.v3 fournit les numéros de ligne que compose-go ne donne pas toujours.
  `go.yaml.in/yaml/v3` est la continuation maintenue de `gopkg.in/yaml.v3`, archivé.
- **Point de vigilance** : compose-go v2.16.1 dépend transitivement de `go.yaml.in/yaml/v4`
  **v4.0.0-rc.4**, une pré-version. Notre code n'utilise directement que la v3 stable. La
  dépendance transitive est acceptée, car c'est celle du parseur de référence de Docker
  Compose, et elle est surveillée par `govulncheck`. Réévaluer le passage à la v4 dès sa
  version stable.
- **Sécurité** : chargement en **lecture seule** (FR-018) ; l'interpolation est faite sans
  jamais afficher de valeur ; les fichiers `.env` ne sont lus que pour savoir si une variable
  est définie ; tout ce qui est affiché passe par le masquage (R7).
- **Alternatives considered** : `github.com/goccy/go-yaml` v1.19.2 (bons messages d'erreur,
  mais ajoute un troisième parseur YAML à l'arbre de dépendances) ; validation par le
  schéma JSON seul (pas de résolution des références ni de l'interpolation).

## R6. Coloration syntaxique et mise en forme de la sortie

- **Decision** :
  - `charm.land/lipgloss/v2` **v2.0.6** pour la mise en forme des explications : encadrés,
    titres, styles par rôle (résumé, concept, cause, correction, anti-pattern) ;
  - `github.com/alecthomas/chroma/v2` **v2.27.0** pour la **coloration syntaxique** des
    commandes shell, des extraits YAML et des Dockerfile, avec mise en évidence de la ligne
    fautive et un marqueur de colonne ;
  - `golang.org/x/term` **v0.47.0** pour détecter un terminal ;
  - coloration de la sortie Docker d'origine (FR-007) : le message brut est réaffiché
    atténué, avec les fragments reconnus (port, image, clé) surlignés.
- **Règles d'accessibilité et de robustesse** :
  - option `--color=auto|always|never` (`auto` par défaut) ; respect de `NO_COLOR` et de
    `CLICOLOR_FORCE` ; pas de couleur si la sortie n'est pas un terminal ;
  - la couleur n'est **jamais** le seul porteur d'information : chaque bloc a aussi un
    symbole et un libellé (`✖ Erreur`, `💡 Correction`, `⚠ Anti-pattern`, `📘 Concept`) ;
  - palette lisible sur fond clair comme sombre (couleurs adaptatives de lipgloss) ;
  - les séquences d'échappement ANSI présentes dans le texte de l'erreur ou du fichier sont
    **neutralisées avant affichage**, pour empêcher l'injection de séquences de terminal
    (principe III).
- **Alternatives considered** : `github.com/fatih/color` v1.19.0 (couleurs seules, sans mise
  en page ni coloration syntaxique) ; codes ANSI écrits à la main (fragile, non accessible).

## R7. Masquage des secrets (FR-016)

- **Decision** : un module `redact` appliqué à **toute** chaîne avant affichage ou
  persistance : valeurs des variables dont le nom contient `PASSWORD`, `PASSWD`, `SECRET`,
  `TOKEN`, `KEY`, `CREDENTIAL`, `AUTH` (insensible à la casse) dans `-e`, `--env`,
  `environment:` et les URL de connexion ; motifs de jetons connus (ex. `ghp_`, `glpat-`,
  `dckr_pat_`, en-têtes `Authorization`) ; identifiants dans les URL (`user:pass@`). Valeur
  remplacée par `••••••`.
- **Rationale** : défense en profondeur ; testé par un corpus dédié (SC-006).

## R8. État local : approfondissement et répétitions (US3, edge cases)

- **Decision** : le dernier diagnostic (déjà masqué) est enregistré dans
  `$XDG_STATE_HOME/telemach/last.json` (défaut `~/.local/state/telemach/`), répertoire `0700`,
  fichier `0600`, et écrasé à chaque fois (pas d'historique). Il permet
  `docker telemach deepen` et la détection des répétitions : si l'empreinte (SHA-256 de la
  commande et de l'erreur normalisées) est identique à la précédente et date de moins de
  10 minutes, une version courte est affichée, avec renvoi à l'explication complète.
- **Rationale** : pas de base de données ; données minimales, locales et masquées.

## R9. Interface en ligne de commande et protocole des plugins Docker

- **Decision** : `github.com/spf13/cobra` **v1.10.2** pour les sous-commandes (c'est aussi la
  bibliothèque de la CLI Docker, donc une ergonomie cohérente). Le protocole des plugins
  Docker CLI (sous-commande `docker-cli-plugin-metadata` renvoyant
  `{"SchemaVersion":"0.1.0","Vendor":…,"Version":…,"ShortDescription":…}`) est implémenté
  directement, **sans** importer `github.com/docker/cli` (module `+incompatible` très lourd ;
  surface d'attaque réduite).
- **Alternatives considered** : bibliothèque standard `flag` seule (aide et sous-commandes
  moins ergonomiques).

## R10. Tests et quality gates

- **Decision** :
  - tests unitaires et de golden files avec la bibliothèque standard `testing` et
    `github.com/google/go-cmp` v0.7.0 (diff lisible) ; sorties de référence en `--color=never`
    et `--color=always` ;
  - **corpus de référence** : `testdata/errors/` (≥ 30 erreurs réelles de Docker 29.x,
    SC-001) et `testdata/compose/` (≥ 20 fichiers erronés, avec la ligne et la clé attendues,
    SC-002), plus `testdata/secrets/` (SC-006) ;
  - tests d'intégration réels (vrai démon Docker) derrière l'étiquette de build
    `integration`, exécutés en CI sur `ubuntu-latest` ;
  - tests du script shell (bash et zsh) : code de sortie préservé, stdout intact, nettoyage
    du fichier temporaire ;
  - fuzzing (`go test -fuzz`) du matcher et du masquage.
- **Protection de `main`** (principe IV) : dépôt public `aboigues/telemach-plugin-docker`,
  checks obligatoires, `enforce_admins`, PR obligatoire sans approbation formelle : Claude Code
  ouvre la PR et l'utilisateur la relit puis la fusionne, ce qui vaut validation humaine. Un
  compte machine permettant une approbation formelle a été écarté, l'utilisateur préférant
  garder son workflow.
- **Lint des Dockerfile** : hadolint v2.15.1 sur les Dockerfile livrés, `testdata/` exclu.
- **Gates GitHub Actions** (principe IV) : `golangci-lint` v2.14.0 (incluant `gosec`),
  `go test -race`, `govulncheck` (`golang.org/x/vuln` v1.9.0), `gitleaks` v8.30.1, revue des
  dépendances, actions épinglées par SHA, `permissions: contents: read`.
- **Distribution** : `goreleaser` v2.18.3 (binaires Linux, macOS et Windows, sommes de
  contrôle, SBOM), à planifier dans une feature ultérieure.
