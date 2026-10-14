# Contribuer à TELEMACH PLUGIN DOCKER

Merci de ton intérêt ! Ce projet est un outil pédagogique : la qualité des explications compte
autant que celle du code. Les règles ci-dessous découlent de la
[constitution](.specify/memory/constitution.md).

## Avant de commencer

- Relis [`LESSON-LEARNED.md`](LESSON-LEARNED.md) : chaque incident passé y laisse une règle
  préventive à appliquer.
- Toute nouvelle fonctionnalité passe par Spec Kit (`/speckit-specify` → `/speckit-plan` →
  `/speckit-tasks` → `/speckit-implement`).

## Langue

- **En français** : spécifications, documentation, commentaires du code, messages affichés par
  le plugin, catalogue d'erreurs, `LESSON-LEARNED.md`. Accents obligatoires.
- **En anglais** : le code (identifiants, noms de fichiers de code), les noms de branches, les
  clés de configuration.
- Les termes et commandes Docker restent dans leur forme d'origine.

## Sécurité

- Ne versionne jamais de secret. Les seuls faux secrets autorisés sont dans `testdata/secrets/`.
- Toute nouvelle dépendance est justifiée dans le plan de la feature et épinglée sur la dernière
  version stable vérifiée.
- Le plugin n'exécute aucune commande et ne modifie aucun fichier de l'apprenant.

## Gates et fusion

La branche `main` est protégée :

- pas de push direct, pas de force-push, pas de suppression ;
- fusion uniquement par pull request, historique linéaire, conversations résolues ;
- aucune exception pour les administrateurs (`enforce_admins`) ;
- checks obligatoires : `lint`, `test`, `vuln`, `secrets`, `dependency-review`,
  `dockerfile-lint`.

**Workflow de fusion** : Claude Code commit, pousse et ouvre la pull request ; le mainteneur
relit la PR puis la fusionne lui-même. Ce clic sur « Merge » vaut validation humaine : aucune
approbation formelle n'est exigée, GitHub interdisant d'approuver sa propre PR.

## Incidents

Tout incident (CI rouge inattendue, faille, régression, gate contournée) est consigné dans
`LESSON-LEARNED.md`, dans la pull request qui le corrige.
