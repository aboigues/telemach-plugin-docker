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

_Aucun incident consigné pour le moment._
