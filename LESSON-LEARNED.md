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
