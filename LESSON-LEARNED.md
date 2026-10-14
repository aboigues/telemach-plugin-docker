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
