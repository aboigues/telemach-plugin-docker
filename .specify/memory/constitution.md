<!--
Sync Impact Report
==================
Version change: 1.0.0 → 1.1.0
Motif du bump : MINOR. Le principe III est précisé : une image sans tag explicite dans un exemple
devient un avertissement non bloquant. Aucun principe n'est supprimé ni redéfini de façon
incompatible ; la règle est assouplie et clarifiée, ce qui ne rend non conforme aucun artefact
existant.

Principes modifiés :
- III. Sécurité élevée (NON NÉGOCIABLE) : ajout de la règle « tag d'image dans les exemples »
  (avertissement non bloquant). Les autres bonnes pratiques des exemples (non-root, images
  minimales, pas de secret, `.dockerignore`) restent MUST.

Sections ajoutées : aucune.
Sections supprimées : aucune.

Templates dépendants : non modifiés (ils lisent la constitution à l'exécution).

TODO différés :
- Mettre à jour T067 (safety_test) : une image sans tag ou en `latest` produit un avertissement
  (t.Log), pas un échec ; les règles non-root et images minimales restent bloquantes.
- Ajouter dans tasks.md une tâche de protection de la branche `main` (constat C2 de l'analyse).
-->

# TELEMACH PLUGIN DOCKER Constitution

## Core Principles

### I. Vocation pédagogique

TELEMACH PLUGIN DOCKER est un plugin pédagogique dont la finalité est de faciliter
l'apprentissage et la compréhension de Docker. En conséquence :

- Chaque fonctionnalité MUST apporter une valeur d'apprentissage explicite : elle explique
  le *pourquoi* (concept Docker sous-jacent), pas seulement le *comment*.
- Les messages du plugin (succès, avertissements, erreurs) MUST être compréhensibles par
  un débutant : ils nomment le concept Docker concerné, décrivent la cause probable et
  proposent une action corrective concrète.
- Le vocabulaire Docker (image, conteneur, volume, réseau, couche, registre, etc.) MUST
  être employé de façon rigoureuse et cohérente ; tout terme introduit pour la première
  fois MUST être défini ou relié à une ressource de référence.
- Les exemples fournis MUST être exécutables tels quels et refléter les bonnes pratiques
  (notamment de sécurité, cf. principe III) : on n'enseigne jamais un anti-pattern sans
  le signaler explicitement comme tel.

**Justification** : un outil pédagogique qui enseigne de mauvaises pratiques ou produit des
messages opaques échoue à sa mission, même s'il fonctionne techniquement.

### II. Français pour l'humain, anglais pour le code

La langue est fixée selon le destinataire du contenu :

- MUST être rédigés en **français** : les spécifications (`spec.md`, `plan.md`,
  `tasks.md`, etc.), la documentation, le glossaire et tout le vocable pédagogique, les
  messages affichés par le plugin, les commentaires dans le code, le fichier
  `LESSON-LEARNED.md` et la présente constitution.
- MUST être rédigés en **anglais** : le code source (identifiants de variables, fonctions,
  classes, modules, fichiers de code), les noms de branches et les clés de configuration.
- Les termes techniques Docker et les commandes (`docker run`, `Dockerfile`, `COPY`,
  etc.) restent dans leur forme d'origine, y compris au sein d'un texte français.
- Le français MUST respecter l'orthographe complète, accents et diacritiques compris.

**Justification** : le public cible apprend en français ; le code en anglais reste
interopérable avec l'écosystème Docker et les outils de développement.

### III. Sécurité élevée (NON NÉGOCIABLE)

Un niveau de sécurité élevé est exigé pour le plugin comme pour ce qu'il enseigne :

- Les secrets (jeton, mot de passe, clé) MUST NOT être versionnés, journalisés ni affichés ;
  les secrets transitent uniquement par des mécanismes dédiés (variables d'environnement
  protégées, GitHub Secrets, Docker secrets).
- Toute entrée externe (arguments, fichiers, sorties de commandes Docker) MUST être validée ;
  l'exécution de commandes shell MUST NOT reposer sur une concaténation de chaînes non
  échappées (pas d'injection de commande).
- Le plugin applique le principe du moindre privilège : il MUST NOT exiger de droits
  root, de `--privileged`, ni de montage du socket Docker sans justification documentée
  et avertissement explicite à l'utilisateur.
- Les dépendances, outils et images de base MUST être épinglés sur une version stable
  vérifiée au moment de l'ajout (recherche de la dernière version réelle, pas de
  supposition) ; les images Docker SHOULD être épinglées par digest.
- Les exemples Docker produits MUST suivre les bonnes pratiques : utilisateur non-root,
  images minimales, pas de secret dans les couches, `.dockerignore` présent.
- Les exemples SHOULD référencer les images avec un tag explicite. Une image sans tag (donc
  `latest` implicite) ou avec le tag `latest` MUST déclencher un **avertissement non
  bloquant** dans les contrôles automatisés (tests, CI), et non un échec. Le contenu
  pédagogique SHOULD alors rappeler pourquoi un tag explicite est préférable.
  **Justification** : pour un débutant, `docker run nginx` est l'exemple le plus lisible ;
  le bloquer nuirait à la pédagogie, mais l'avertissement garde la bonne pratique visible.

**Justification** : un plugin qui pilote Docker manipule une surface d'attaque proche de
l'hôte ; et un outil pédagogique qui banalise l'insécurité propage de mauvaises habitudes.

### IV. Gates GitHub Actions obligatoires (NON NÉGOCIABLE)

Aucune modification n'atteint la branche principale sans passer des gates automatisées
GitHub Actions :

- La branche principale MUST être protégée : fusion uniquement via pull request, gates
  au statut « required », aucun push direct ni contournement administrateur.
- Les gates minimales MUST inclure : lint, tests automatisés, analyse statique de sécurité
  (SAST), détection de secrets, audit des dépendances (vulnérabilités connues) et, dès
  qu'un Dockerfile existe, lint Dockerfile et scan de l'image.
- Les workflows eux-mêmes MUST être durcis : actions tierces épinglées par SHA de commit
  complet, `permissions` minimales déclarées explicitement, pas de `pull_request_target`
  exécutant du code non fiable.
- Une gate en échec MUST bloquer la fusion ; la désactiver ou l'affaiblir constitue un
  amendement à cette constitution.

**Justification** : la sécurité exigée par le principe III n'est crédible que si elle est
vérifiée mécaniquement à chaque changement.

### V. Mémoire des incidents (LESSON-LEARNED.md)

Toute erreur commise une fois MUST NOT se reproduire :

- Tout incident (bug en production ou en CI, faille, régression, gate contournée, erreur
  de processus, mauvaise manipulation) MUST être consigné dans `LESSON-LEARNED.md` à la
  racine du dépôt.
- Chaque entrée MUST contenir : date (AAAA-MM-JJ), contexte, cause racine, impact,
  correctif appliqué et règle préventive à retenir.
- `LESSON-LEARNED.md` MUST être relu au démarrage de chaque session de travail et avant
  chaque étape Spec Kit (`/speckit-specify`, `/speckit-plan`, `/speckit-tasks`,
  `/speckit-implement`) ; les règles préventives pertinentes MUST être prises en compte
  dans les artefacts produits.
- Quand une leçon est automatisable, elle SHOULD être transformée en gate (principe IV)
  ou en test de non-régression.

**Justification** : la relecture systématique transforme chaque incident en garde-fou
durable plutôt qu'en connaissance perdue.

## Exigences de sécurité

- **Chaîne d'approvisionnement** : chaque nouvelle dépendance est justifiée dans le plan
  de la feature (besoin, alternative envisagée, maintenance du projet, licence) ; les
  fichiers de verrouillage sont versionnés.
- **Mises à jour** : les mises à jour de sécurité des dépendances et des actions sont
  automatisées (ex. Dependabot ou équivalent) et traitées en priorité.
- **Vulnérabilités** : une vulnérabilité de sévérité critique ou haute détectée par les
  gates bloque la fusion jusqu'à correction ou dérogation documentée (justification,
  date d'expiration) dans la pull request.
- **Signalement** : le dépôt fournit un fichier `SECURITY.md` décrivant la procédure de
  signalement responsable des vulnérabilités.
- **Journalisation** : les journaux du plugin MUST NOT contenir de secrets ni de données
  personnelles.

## Workflow de développement et quality gates

1. **Démarrage** : relire `LESSON-LEARNED.md` (principe V) et la présente constitution.
2. **Spécification** : toute feature passe par Spec Kit (`/speckit-specify` →
   `/speckit-plan` → `/speckit-tasks` → `/speckit-implement`), artefacts rédigés en
   français.
3. **Constitution Check** : chaque `plan.md` vérifie explicitement la conformité aux
   cinq principes ; toute dérogation est justifiée dans la section « Complexity Tracking ».
4. **Revue** : chaque pull request est relue et vérifie, en plus du code, la qualité
   pédagogique des messages, le respect de la règle linguistique (principe II) et
   l'absence de régression de sécurité.
5. **Fusion** : uniquement lorsque toutes les gates GitHub Actions requises sont vertes
   (principe IV).
6. **Incident** : tout incident déclenche l'ajout d'une entrée dans `LESSON-LEARNED.md`
   dans la pull request de correction.

## Governance

- Cette constitution prévaut sur toute autre pratique ou convention du projet ; en cas de
  conflit, elle s'applique.
- **Amendement** : toute modification passe par une pull request dédiée, décrivant le
  changement, sa justification et son impact sur les artefacts existants (plan de
  migration si nécessaire), et soumise aux mêmes gates que le code.
- **Versionnement sémantique** :
  - MAJOR : suppression ou redéfinition incompatible d'un principe ou d'une règle de
    gouvernance ;
  - MINOR : ajout d'un principe ou d'une section, ou extension substantielle d'une règle ;
  - PATCH : clarification, reformulation, correction typographique sans effet normatif.
- **Conformité** : chaque revue de pull request et chaque `plan.md` vérifient la
  conformité à la constitution ; toute complexité ou dérogation MUST être justifiée par
  écrit.
- **Revue périodique** : la constitution et `LESSON-LEARNED.md` sont revus au moins une
  fois par trimestre pour intégrer les leçons récurrentes sous forme de règles.

**Version**: 1.1.0 | **Ratified**: 2026-10-14 | **Last Amended**: 2026-10-14
