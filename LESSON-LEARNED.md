# LESSON-LEARNED — TELEMACH PLUGIN DOCKER

Registre des incidents du projet et des leçons qui en découlent (principe V de la
[constitution](.specify/memory/constitution.md)).

## Règles d'usage

- **Relecture obligatoire** : ce fichier MUST être relu au démarrage de chaque session de
  travail et avant chaque étape Spec Kit (`/speckit-specify`, `/speckit-plan`,
  `/speckit-tasks`, `/speckit-implement`).
- **Quoi consigner** : tout incident — bug en production ou en CI, faille de sécurité,
  régression, gate contournée, erreur de processus, mauvaise manipulation.
- **Quand** : l'entrée est ajoutée dans la pull request qui corrige l'incident.
- **Ordre** : les entrées les plus récentes en haut de la section « Incidents ».
- **Automatisation** : quand une leçon peut être vérifiée mécaniquement, elle SHOULD devenir
  une gate GitHub Actions ou un test de non-régression ; le noter dans le champ « Suivi ».
- **Pas de secret** : aucune entrée ne contient de jeton, mot de passe ou donnée personnelle.

## Modèle d'entrée

```markdown
### LL-XXX — Titre court de l'incident

- **Date** : AAAA-MM-JJ
- **Catégorie** : sécurité | CI/gates | bug | régression | processus | pédagogie | langue
- **Contexte** : ce qui était en cours au moment de l'incident.
- **Cause racine** : pourquoi c'est arrivé (pas seulement le symptôme).
- **Impact** : conséquences observées ou évitées de justesse.
- **Correctif** : ce qui a été fait (lien vers la PR / le commit).
- **Règle préventive** : la règle à appliquer désormais, formulée de façon actionnable.
- **Suivi** : gate ou test ajouté (ou « non automatisable » avec justification).
```

## Incidents

### LL-012 — Objectif de performance annoncé sans test qui le mesure

- **Date** : 2026-10-16
- **Catégorie** : régression
- **Contexte** : feature 003, mesure SC-003 (T031) : délai ajouté par `lint` < 50 ms au p95.
- **Cause racine** : le plan citait un démarrage de 25 ms mesuré à la main, une seule fois ;
  aucun test ne le vérifiait. Entre-temps, la coloration syntaxique (chroma) ajoutait ~17 ms
  d'initialisation à **chaque** appel du plugin, y compris pour les explications d'erreur.
- **Impact** : p95 de 146 ms au lieu de < 50 ms ; détecté par le premier test de mesure, avant
  toute PR.
- **Correctif** : définitions chroma utilisées embarquées et chargées à la demande
  (`internal/render/syntax/`) ; p95 ramené à 26 ms.
- **Règle préventive** : tout objectif chiffré (délai, taille, mémoire) reçoit un test qui le
  mesure dans la tâche même où il est annoncé ; une nouvelle dépendance est vérifiée avec
  `GODEBUG=inittrace=1` (coût d'initialisation) avant d'être adoptée.
- **Suivi** : automatisé — `TestSC003LintLatency` (job `integration`, seuil 100 ms en CI).

### LL-011 — Propositions d'analyse perdues entre deux sessions

- **Date** : 2026-10-16
- **Catégorie** : processus
- **Contexte** : feature 003 ; `/speckit-analyze` avait proposé 10 corrections le 2026-10-15,
  et l'implémentation avait été lancée sans les valider toutes.
- **Cause racine** : le rapport d'analyse n'existe que dans la conversation (commande en lecture
  seule) ; l'état de reprise notait seulement « I1, I2, I4 non appliqués », sans leur contenu.
- **Impact** : trois corrections de texte perdues à la reprise ; il a fallu relancer l'analyse.
- **Correctif** : analyse relancée, toutes ses recommandations appliquées (F1 à F13).
- **Règle préventive** : toute proposition non tranchée (analyse, revue) est recopiée **en
  entier** dans l'état de reprise de `tasks.md` avant la fin de la session, jamais par son seul
  identifiant.
- **Suivi** : non automatisable (geste de fin de session).

### LL-010 — Faux positifs d'un piège fondé sur une heuristique large

- **Date** : 2026-10-16
- **Catégorie** : bug
- **Contexte** : feature 003, piège « secret écrit en clair » (FR-004.6), qui réutilisait les
  règles de masquage de `internal/redact`.
- **Cause racine** : les règles de masquage reconnaissent `KEY`, `AUTH` ou `SECRET` n'importe où
  dans un nom ; c'est voulu pour masquer (mieux vaut masquer à tort), mais pas pour avertir. Le
  corpus légitime ne contenait que des exemples de la spec, aucun nom piégeux.
- **Impact** : `-e MONKEY=1`, `-e AUTHOR=alex`, `-e KEYCLOAK_URL=…` déclenchaient un
  avertissement (SC-002 violé) ; détecté par `/speckit-analyze` avant toute PR.
- **Correctif** : avertissement limité aux noms sans ambiguïté (`secretName`) ou aux valeurs
  reconnaissables seules ; 7 lignes légitimes piégeuses et 4 lignes fautives ajoutées au corpus.
- **Règle préventive** : une règle qui réutilise une heuristique prévue pour un autre usage
  reçoit, dans le corpus légitime, des cas **adversariaux** de cette heuristique (noms et valeurs
  qui la déclenchent sans être fautifs) avant d'être considérée comme terminée.
- **Suivi** : automatisé — `testdata/cmdline/pitfalls-legit.tsv` et `TestSecretName`.

### LL-009 — Vérification affichée comme réussie sans avoir rien comparé (zsh)

- **Date** : 2026-10-15
- **Catégorie** : processus
- **Contexte** : vérification post-publication de v0.2.0 (FR-009 : binaires de l'image
  identiques à ceux de la Release), script lancé dans le shell de l'outil (zsh).
- **Cause racine** : `set -- $p` ne découpe pas une variable en mots sous zsh (contrairement
  à bash) ; les deux sommes comparées étaient vides, donc égales, et le script a affiché
  « identique à la Release ».
- **Impact** : faux positif repéré immédiatement grâce aux erreurs `No such file` ; contrôle
  refait en bash (les 4 binaires sont bien identiques). Aucune conséquence, mais une
  publication aurait pu être validée à tort.
- **Correctif** : vérification réécrite en `bash -c`, avec contrôle explicite que la valeur
  attendue n'est pas vide (`[ -n "$want" ] && [ "$got" = "$want" ]`).
- **Règle préventive** : tout script de vérification ad hoc s'exécute dans `bash -c` (jamais
  dans le shell interactif zsh) et échoue si une valeur comparée est vide ; un contrôle qui
  n'a rien comparé est un échec, pas un succès.
- **Suivi** : non automatisable (vérification manuelle post-publication) ; la comparaison
  FR-009 pourra devenir une étape du workflow `release` lors d'une prochaine évolution.

### LL-008 — Commit local malgré un lint en échec (enchaînement non bloquant)

- **Date** : 2026-10-15
- **Catégorie** : processus
- **Contexte** : feature 002, validation locale avant commit (`golangci-lint run … ; git commit …`).
- **Cause racine** : les commandes de validation et de commit étaient enchaînées par `;` ; puis
  `set -e` a été utilisé, mais il n'interrompt pas l'enchaînement dans le shell de l'outil
  (zsh). Le commit a été créé alors que le lint venait d'échouer (deux fois).
- **Impact** : deux commits locaux non conformes (errcheck, gosec G703), corrigés avant tout
  push ; aucune gate contournée côté dépôt.
- **Correctif** : commit de correction puis `--amend` du commit non poussé ; validations
  rejouées.
- **Règle préventive** : enchaîner validation et commit **uniquement** par `&&` (jamais `;`,
  ne pas compter sur `set -e`) ; un commit n'est créé que si chaque validation a réussi.
- **Suivi** : non automatisable dans la CI (geste local) ; les gates obligatoires de `main`
  restent le filet de sécurité.

### LL-007 — Le lint ignorait les tests à étiquette de build

- **Date** : 2026-10-15
- **Catégorie** : CI/gates
- **Contexte** : feature 002, ajout de `tests/release` (étiquette `dist`).
- **Cause racine** : golangci-lint n'analyse que les fichiers compilés sans étiquette ; les
  tests `integration` (001) et `dist` (002) n'étaient donc jamais lintés, ni en local ni en CI.
- **Impact** : deux `defer f.Close()` non vérifiés détectés seulement en lançant le lint à la
  main avec `--build-tags` ; aucune gate ne les aurait vus.
- **Correctif** : `run.build-tags: [integration, dist]` dans `.golangci.yml` (PR #5).
- **Règle préventive** : toute nouvelle étiquette de build est ajoutée à `run.build-tags` de
  `.golangci.yml` dans la même PR.
- **Suivi** : automatisé — la gate `lint` couvre désormais ces fichiers.

### LL-006 — Fonction shell inactive après une installation réelle

- **Date** : 2026-10-14
- **Catégorie** : bug
- **Contexte** : première installation locale, en suivant le README (binaire dans
  `~/.docker/cli-plugins/`, `eval "$(docker telemach shell-init zsh)"` dans `~/.zshrc`).
- **Cause racine** : le script généré cherchait `docker-telemach` dans le PATH ; or le dossier
  des plugins Docker n'y est pas. La fonction se désactivait silencieusement. Les tests
  plaçaient le binaire dans le PATH et ne reproduisaient donc pas l'installation réelle.
- **Impact** : aucune explication affichée après une erreur ; détecté au premier test
  d'installation, avant la fusion.
- **Correctif** : `shell-init` inscrit dans le script le chemin absolu de son propre binaire
  (protégé pour le shell) ; les tests d'intégration n'ajoutent plus le binaire au PATH ; un
  test couvre un chemin contenant espace et apostrophe.
- **Règle préventive** : les tests de bout en bout doivent reproduire **exactement** la
  procédure d'installation documentée, sans raccourci d'environnement (PATH, variables).
- **Suivi** : automatisé — `TestBinaryOutsidePath` (bash et zsh) et tests d'intégration.

### LL-005 — Gate dependency-review rouge : Dependency graph désactivé sur le dépôt

- **Date** : 2026-10-14
- **Catégorie** : CI/gates
- **Contexte** : première pull request (#1) après la mise en place des gates (T005, T010).
- **Cause racine** : `actions/dependency-review-action` exige que le Dependency graph soit
  activé dans les réglages du dépôt ; ce prérequis n'était ni vérifié ni documenté.
- **Impact** : la gate obligatoire `dependency-review` échouait sans lien avec le code,
  bloquant la fusion.
- **Correctif** : activation des alertes Dependabot (et donc du Dependency graph) par
  `gh api -X PUT repos/aboigues/telemach-plugin-docker/vulnerability-alerts`, puis relance.
- **Règle préventive** : lors de la création d'une gate CI, vérifier ses prérequis côté
  dépôt (réglages de sécurité, permissions) et les documenter dans `CONTRIBUTING.md` ; ouvrir
  une PR de test dès que les gates sont en place, avant d'écrire du code.
- **Suivi** : non automatisable dans la CI elle-même (réglage du dépôt) ; prérequis ajouté
  à `CONTRIBUTING.md`.

### LL-004 — Tags d'images d'exemple choisis de mémoire, sans vérification

- **Date** : 2026-10-14
- **Catégorie** : processus
- **Contexte** : rédaction du catalogue (T039–T045) et du corpus.
- **Cause racine** : les tags `alpine:3.22` et `nginx…:1.29-alpine` ont été écrits de
  mémoire, alors que la règle impose de vérifier la dernière version stable réelle.
- **Impact** : exemples pédagogiques en retard de deux versions d'alpine ; détecté avant
  fusion, en vérifiant les tags sur Docker Hub pour le test de sécurité des exemples.
- **Correctif** : passage à `alpine:3.24`, nginx branche stable `1.30-alpine`
  (`1.31` = mainline), `postgres:18-alpine`, `redis:8-alpine` ; corpus recapturé.
- **Règle préventive** : avant d'écrire un tag d'image, consulter le registre (Docker Hub)
  pour connaître la dernière version stable ; pour nginx, la branche stable a un numéro
  mineur pair.
- **Suivi** : partiellement automatisé — `internal/catalog/safety_test.go` signale les tags
  absents ou `latest` ; la fraîcheur des tags reste à vérifier à chaque revue trimestrielle.

### LL-003 — Fichier temporaire non supprimé après Ctrl-C (course entre mktemp et trap)

- **Date** : 2026-10-14
- **Catégorie** : bug
- **Contexte** : intégration shell (T032), test `TestInterruptCleansUp` instable sous bash.
- **Cause racine** : le `trap` de nettoyage était installé *après* `mktemp` ; un SIGINT reçu
  entre les deux laissait le fichier (contenant potentiellement un message d'erreur).
- **Impact** : fichier `telemach.*` résiduel dans `$XDG_RUNTIME_DIR`. Détecté en test.
- **Correctif** : pièges installés avant la création du fichier ; le test attend que la
  commande tourne réellement avant d'envoyer le signal.
- **Règle préventive** : toujours installer le `trap` de nettoyage **avant** de créer la
  ressource à nettoyer, et tester l'interruption à plusieurs moments.
- **Suivi** : automatisé — `tests/shell` (`TestInterruptCleansUp`, bash et zsh).

### LL-002 — zsh duplique stdout avec MULTIOS

- **Date** : 2026-10-14
- **Catégorie** : bug
- **Contexte** : intégration shell zsh (T032), redirection `{ cmd 2>&1 1>&3 3>&- | tee … >&2; } 3>&1`.
- **Cause racine** : l'option zsh `MULTIOS` (active par défaut) transforme une seconde
  redirection de stdout en duplication : la sortie standard de docker partait à la fois
  vers le terminal, le fichier temporaire et stderr.
- **Impact** : sortie standard affichée deux fois et mélangée au message d'erreur analysé.
  Détecté par `TestFailureIsExplained/zsh` avant toute fusion.
- **Correctif** : `setopt NO_MULTIOS` dans la fonction (exécutée en sous-shell, sans effet
  sur les options de l'utilisateur) ; le test vérifie que stdout n'apparaît jamais sur stderr.
- **Règle préventive** : tout code shell partagé entre bash et zsh doit être testé sous les
  **deux** shells, avec des assertions sur stdout **et** stderr, pas seulement sur stdout.
- **Suivi** : automatisé — `tests/shell` exécuté sous bash et zsh en CI.

### LL-001 — golangci-lint refuse un module dont la version de Go est plus récente

- **Date** : 2026-10-14
- **Catégorie** : CI/gates
- **Contexte** : mise en place des gates (T004/T005) avec golangci-lint v2.14.0 sur un module
  déclarant `go 1.27`.
- **Cause racine** : un binaire golangci-lint ne peut analyser que du code ciblant une version
  de Go inférieure ou égale à celle qui l'a compilé. Le binaire publié v2.14.0 est compilé avec
  Go 1.26 ; `go install` sans `GOTOOLCHAIN` a aussi produit un binaire Go 1.26.
- **Impact** : `can't load config: the Go language version (go1.26) used to build golangci-lint
  is lower than the targeted Go version (1.27.2)` — la gate `lint` aurait été rouge dès la
  première PR. Détecté en local avant tout push.
- **Correctif** : CI en `install-mode: goinstall` après `setup-go` (Go du projet) ; en local,
  `GOTOOLCHAIN=go1.27.2 go install …@v2.14.0`.
- **Règle préventive** : à chaque montée de version de Go du projet, vérifier que chaque outil
  d'analyse (golangci-lint, govulncheck…) est compilé avec une version de Go au moins égale ;
  sinon le compiler avec la chaîne d'outils du projet.
- **Suivi** : automatisé — la gate `lint` échoue immédiatement si la règle est violée.
- **Récidive (2026-10-14, T077)** : `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0` a été
  compilé avec Go 1.26 (version minimale de son module) et a refusé le projet. Corrigé en
  fixant `GOTOOLCHAIN: go1.27.2` dans le job `vuln`. Règle complétée : tout outil lancé par
  `go run …@version` ou `go install …@version` reçoit `GOTOOLCHAIN` égal à la chaîne d'outils
  du projet.
