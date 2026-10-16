# Feature Specification: Analyse des conteneurs arrêtés en erreur

**Feature Branch**: `004-exited-containers-analysis`

**Created**: 2026-10-16

**Status**: Draft

**Input**: User description: "004 — Analyse des conteneurs arrêtés en erreur (issue GitHub #8). Ajouter une sous-commande « docker telemach ps » qui repère le ou les derniers conteneurs terminés avec un code de sortie différent de 0 (exited ≠ 0) et tente d'expliquer la cause en français, de façon pédagogique comme les explications existantes (concept, cause probable, corrections, approfondissement). Cas typiques : un conteneur lancé avec -d qui s'arrête aussitôt (le docker run a réussi, donc aucune explication n'a été affichée), code 1 de l'application, 125/126/127, 137 (OOM ou kill), 139, 143. Particularité : pour lire l'état des conteneurs et la fin de leurs journaux, le plugin devra interroger le démon Docker, ce qui remet en cause la règle actuelle « le plugin n'exécute aucune commande » (features 001 et 003) ; la spec doit cadrer précisément ces lectures (lecture seule, sans modifier ni relancer aucun conteneur, sans réseau externe), masquer les secrets des journaux et de la configuration, et ne jamais afficher de variable d'environnement en clair."

## Clarifications

### Session 2026-10-16

- Q: Faut-il orienter l'apprenant vers `docker telemach ps` quand un conteneur lancé en
  arrière-plan s'arrête sans qu'il le voie ? → A: Oui, par une vérification automatique : après
  un `docker run -d` réussi, l'intégration shell vérifie quelques secondes plus tard l'état du
  conteneur lancé et signale un arrêt en erreur (option C). L'exception de FR-010 est élargie à
  cette seule lecture, encadrée par FR-013 à FR-016.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Comprendre pourquoi le dernier conteneur s'est arrêté en erreur (Priority: P1)

Un apprenant lance `docker run -d postgres:18-alpine`. Docker répond par l'identifiant du
conteneur (code 0) : aucune erreur, donc aucune explication de TELEMACH. Trois secondes plus
tard, le conteneur s'est arrêté avec le code 1 ; `docker ps` ne le montre plus, et l'apprenant
ne sait pas où chercher. Il tape `docker telemach ps` : TELEMACH retrouve le dernier conteneur
arrêté en erreur, montre la fin de ses journaux, et explique en français que l'image exige la
variable `POSTGRES_PASSWORD`, avec la correction à copier.

**Why this priority**: c'est le cas de l'issue #8. Un conteneur lancé en arrière-plan qui
s'arrête aussitôt est une situation de débutant très fréquente, et la seule que les features
précédentes ne couvrent pas : la commande a réussi, l'erreur n'apparaît que plus tard, dans
les journaux du conteneur.

**Independent Test**: créer de vrais conteneurs arrêtés en erreur (corpus de scénarios
reproductibles), lancer `docker telemach ps` et vérifier le conteneur choisi, la cause
proposée et l'absence de toute modification des conteneurs.

**Acceptance Scenarios**:

1. **Given** un conteneur `postgres:18-alpine` lancé avec `-d` sans `POSTGRES_PASSWORD` et
   arrêté avec le code 1, **When** l'apprenant tape `docker telemach ps`, **Then** TELEMACH
   affiche le nom du conteneur, son image, son code de sortie et depuis quand il est arrêté,
   les dernières lignes de ses journaux, puis une explication : `📘 Concept` (code de sortie,
   cycle de vie d'un conteneur), `🔎 Cause probable` (variable obligatoire absente, d'après le
   journal), `💡 Correction` (par exemple `--env-file` avec la variable, ou la variable
   `POSTGRES_PASSWORD_FILE`), et `➜ Pour aller plus loin`.
2. **Given** un conteneur arrêté avec le code 137 parce qu'il a dépassé sa limite de mémoire,
   **When** l'apprenant tape `docker telemach ps`, **Then** TELEMACH indique que le noyau a
   arrêté le programme faute de mémoire (et non un `docker stop`), et propose d'augmenter la
   limite ou de réduire la consommation.
3. **Given** un conteneur arrêté avec le code 137 ou 143 à la suite d'un `docker stop` ou d'un
   `docker kill`, **When** l'apprenant tape `docker telemach ps`, **Then** TELEMACH explique que
   le conteneur a été arrêté de l'extérieur (signal), ce qui est souvent normal, et distingue
   l'arrêt propre (143) de l'arrêt forcé après délai (137).
4. **Given** un conteneur arrêté en erreur dont ni le code ni les journaux ne correspondent à
   une cause connue, **When** l'apprenant tape `docker telemach ps`, **Then** TELEMACH affiche
   les dernières lignes du journal et la signification générale du code, dit honnêtement que la
   cause n'est pas reconnue, et indique comment lire le journal complet (`docker logs <nom>`).
5. **Given** aucun conteneur arrêté avec un code différent de 0, **When** l'apprenant tape
   `docker telemach ps`, **Then** TELEMACH le dit en une phrase et rappelle qu'un conteneur
   lancé avec `--rm` est supprimé à l'arrêt, donc introuvable ensuite.
6. **Given** n'importe lequel des scénarios précédents, **When** `docker telemach ps` s'est
   exécuté, **Then** l'état de tous les conteneurs (statut, code, journaux, configuration) est
   identique à ce qu'il était avant : TELEMACH n'a rien démarré, arrêté, supprimé ni modifié.

---

### User Story 2 - Être prévenu qu'un conteneur lancé en arrière-plan s'est arrêté (Priority: P2)

Le même apprenant ne connaît pas encore `docker telemach ps`. Après son
`docker run -d postgres:18-alpine`, il continue à travailler. TELEMACH vérifie seul, quelques
secondes plus tard, que le conteneur tourne toujours ; il s'est arrêté avec le code 1. À
l'affichage suivant de son invite de commande, l'apprenant lit une ligne qui le lui signale et
lui indique `docker telemach ps` pour comprendre pourquoi.

**Why this priority**: sans ce signal, l'apprenant de l'issue #8 ne sait pas qu'il y a un
problème, ni où chercher ; c'est ce qui relie l'erreur, invisible, à l'explication de US1.

**Independent Test**: avec l'intégration shell active (bash et zsh), lancer en arrière-plan un
conteneur qui s'arrête aussitôt en erreur, puis un conteneur qui continue de tourner, et
vérifier le signal, son moment d'affichage, et l'absence d'effet sur le terminal et sur la
commande.

**Acceptance Scenarios**:

1. **Given** l'intégration shell active, **When** l'apprenant exécute
   `docker run -d postgres:18-alpine` (code 0) et que le conteneur s'arrête avec le code 1 dans
   les secondes qui suivent, **Then** à l'affichage suivant de l'invite, après la vérification,
   TELEMACH affiche une seule ligne : « ⚠ Le conteneur <nom> s'est arrêté avec le code 1, <n> s
   après son lancement : docker telemach ps pour comprendre pourquoi ».
2. **Given** la même commande pour un conteneur qui continue de tourner (`nginx`), **When** la
   vérification a lieu, **Then** rien n'est affiché.
3. **Given** la vérification en attente, **When** l'apprenant tape d'autres commandes ou que
   leur sortie s'affiche, **Then** le signal n'apparaît jamais au milieu d'une ligne en cours de
   saisie ni au milieu de la sortie d'une commande, et aucun message de tâche de fond du shell
   (numéro de tâche, « terminé ») n'apparaît.
4. **Given** `docker run -d` affiche l'identifiant du conteneur, **When** TELEMACH prévoit sa
   vérification, **Then** la sortie standard, la sortie d'erreur et le code de sortie de la
   commande sont exactement ceux qu'elle aurait produits sans TELEMACH, et l'invite revient
   aussi vite.
5. **Given** la vérification désactivée (variable dédiée) ou `TELEMACH_DISABLE=1`, **When**
   l'apprenant lance un conteneur en arrière-plan, **Then** TELEMACH n'interroge pas le démon.

---

### User Story 3 - Voir et choisir parmi les conteneurs en erreur (Priority: P3)

Un apprenant a plusieurs conteneurs arrêtés en erreur, ou un conteneur qui redémarre en boucle
(`--restart always`). Il veut voir lesquels posent problème, puis faire expliquer celui qui
l'intéresse.

**Why this priority**: utile dès qu'on manipule plusieurs conteneurs (Compose, exercices
enchaînés), mais le cas le plus fréquent, un seul conteneur, est couvert par US1 et US2.

**Independent Test**: créer plusieurs conteneurs en erreur à des moments différents, dont un en
boucle de redémarrage et un d'un projet Compose ; vérifier la liste, l'ordre et la sélection.

**Acceptance Scenarios**:

1. **Given** plusieurs conteneurs arrêtés en erreur, **When** l'apprenant tape
   `docker telemach ps`, **Then** TELEMACH explique le plus récent et liste, sous l'explication,
   les autres conteneurs en erreur (au plus 10, du plus récent au plus ancien), avec leur nom,
   leur code, leur ancienneté, une cause probable en une ligne, et la commande pour en faire
   expliquer un.
2. **Given** un conteneur nommé `api` arrêté en erreur, **When** l'apprenant tape
   `docker telemach ps api` (ou un préfixe de son identifiant), **Then** TELEMACH explique ce
   conteneur, même s'il n'est pas le plus récent ; si le nom ne désigne aucun conteneur, il le
   dit et propose les noms proches.
3. **Given** un conteneur qui redémarre en boucle après des arrêts en erreur, **When**
   l'apprenant tape `docker telemach ps`, **Then** ce conteneur est inclus, avec son dernier
   code de sortie et le nombre de redémarrages, et l'explication mentionne la politique de
   redémarrage.
4. **Given** un conteneur arrêté en erreur créé par Docker Compose, **When** il apparaît,
   **Then** TELEMACH indique aussi le projet et le service Compose correspondants.

---

### User Story 4 - Retrouver l'analyse et approfondir (Priority: P4)

Après une analyse, l'apprenant veut la relire plus tard ou approfondir le concept, comme pour
les erreurs et les avertissements.

**Why this priority**: cohérence avec le reste de TELEMACH ; la valeur principale est livrée
par US1.

**Independent Test**: après `docker telemach ps`, lancer `docker telemach last` et
`docker telemach deepen`.

**Acceptance Scenarios**:

1. **Given** une analyse affichée par `docker telemach ps`, **When** l'apprenant tape
   `docker telemach last`, **Then** la même analyse est réaffichée, précédée de sa date et du
   conteneur concerné, sans interroger à nouveau le démon.
2. **Given** la même analyse, **When** l'apprenant tape `docker telemach deepen`, **Then**
   TELEMACH approfondit le concept de la cause reconnue (par exemple codes de sortie et
   signaux, limites de mémoire).

---

### Edge Cases

- **Démon injoignable ou droits insuffisants** sur le socket Docker : l'explication existante
  correspondante s'affiche (démon arrêté, permission refusée) ; aucun contournement (pas de
  `sudo`, pas d'autre socket).
- **Conteneur supprimé** (`--rm`, `docker rm`) : il n'est plus visible ; TELEMACH ne peut pas
  l'analyser et le dit si l'apprenant le désigne par son nom.
- **Conteneur arrêté avec le code 0** : il n'est pas en erreur et n'est pas analysé, même s'il
  s'est arrêté plus tôt qu'attendu (programme qui se termine normalement) ; le message « aucun
  conteneur en erreur » le rappelle quand des conteneurs arrêtés avec 0 existent.
- **Conteneur en cours d'exécution** : jamais analysé, sauf s'il est en boucle de redémarrage
  (US3-3).
- **Journaux absents ou illisibles** (pilote de journalisation qui ne permet pas leur lecture,
  journal vide) : l'analyse repose alors sur le code de sortie et l'état, et le dit.
- **Journaux très volumineux** : seule la fin est lue (FR-006) ; la durée de la commande ne
  dépend pas de la taille du journal.
- **Secrets dans les journaux ou la configuration** : mots de passe affichés par le programme,
  variables d'environnement définies avec `-e` : toujours masqués (FR-008, FR-009).
- **Caractères spéciaux et séquences de contrôle** dans les journaux, noms ou commandes :
  nettoyés avant tout affichage, comme pour les messages d'erreur.
- **Contexte Docker distant** (`docker context`, `DOCKER_HOST`) : TELEMACH interroge le même
  démon que la CLI Docker de l'apprenant, et aucun autre.
- **Nombreux conteneurs** (plusieurs centaines) : la commande reste rapide (SC-004).
- **Conteneur qui s'arrête après la vérification** (au-delà du délai de FR-014) : il n'est pas
  signalé automatiquement ; `docker telemach ps` le trouve.
- **Conteneur lancé avec `-d --rm`** qui s'arrête aussitôt : il est supprimé avant la
  vérification ; le signal le dit (« s'est arrêté et a été supprimé (--rm) : son journal est
  perdu, relance-le sans --rm pour le lire »), sans pouvoir donner de code.
- **Plusieurs `docker run -d` à la suite** : chaque conteneur est vérifié une fois ; les
  signaux sont regroupés à l'invite suivante.
- **Terminal fermé avant la vérification** : la vérification est abandonnée, sans processus
  ni fichier laissé derrière.
- **Session non interactive** (script shell qui charge l'intégration) : aucune vérification,
  car il n'y a pas d'invite où afficher le signal.

## Requirements *(mandatory)*

### Functional Requirements

**Sélection**

- **FR-001**: TELEMACH MUST fournir la sous-commande `docker telemach ps`, qui repère les
  conteneurs arrêtés avec un code de sortie différent de 0, ainsi que les conteneurs en boucle
  de redémarrage dont le dernier arrêt a un code différent de 0.
- **FR-002**: Sans argument, la commande MUST analyser le conteneur en erreur arrêté le plus
  récemment et lister les autres (au plus 10, du plus récent au plus ancien) ; avec un nom ou un
  préfixe d'identifiant, elle MUST analyser ce conteneur, et le dire clairement s'il n'existe
  pas, s'il est en cours d'exécution ou s'il s'est arrêté avec le code 0.

**Explication**

- **FR-003**: L'analyse MUST suivre la structure des explications existantes, en français :
  résumé, `📘 Concept`, `🔎 Cause probable`, `💡 Correction` avec des commandes prêtes à copier,
  `➜ Pour aller plus loin`, et indiquer le nom du conteneur, son image, sa commande, son code
  de sortie et depuis quand il est arrêté.
- **FR-004**: La cause probable MUST combiner trois sources, de la plus précise à la plus
  générale : un motif reconnu dans la fin du journal (par exemple variable obligatoire absente,
  permission refusée, adresse déjà utilisée, fichier introuvable, format d'exécutable
  incompatible), l'état du conteneur (arrêt pour manque de mémoire, erreur de démarrage
  enregistrée par Docker, redémarrages), puis la signification du code de sortie (1, 2, 125,
  126, 127, 137, 139, 143 et code générique).
- **FR-005**: Quand aucune source ne donne de cause précise, TELEMACH MUST le dire et ne jamais
  inventer de cause (même règle d'honnêteté que pour les erreurs inconnues).
- **FR-006**: TELEMACH MUST afficher au plus les 20 dernières lignes du journal du conteneur, et
  n'en lire que la fin (au plus 200 lignes), quelle que soit sa taille.
- **FR-007**: Les explications des causes et des codes de sortie MUST provenir du catalogue
  pédagogique, avec la même extensibilité que les erreurs : ajouter une cause ne demande qu'une
  entrée et ses cas de test.

**Sécurité et lectures autorisées**

- **FR-008**: Les journaux, la commande, les noms et toute valeur affichée ou conservée MUST être
  nettoyés et masqués comme les messages d'erreur (secrets, séquences de contrôle).
- **FR-009**: Les valeurs des variables d'environnement d'un conteneur MUST NOT être affichées,
  conservées ni transmises ; seuls leurs noms peuvent être utilisés, par exemple pour constater
  qu'une variable attendue est absente.
- **FR-010**: Cette commande et la vérification de FR-013 sont les **seules exceptions** à la
  règle « le plugin n'exécute aucune commande » (FR-015 de la feature 001), et elles MUST rester
  dans ce cadre :
  - interrogation du démon Docker **uniquement** sur demande explicite de l'apprenant
    (`docker telemach ps`), à une exception près : la vérification automatique de FR-013, qui
    ne lit que l'**état** du seul conteneur qui vient d'être lancé, une seule fois, sans lire
    son journal ;
  - **lectures seules** : liste des conteneurs, état et configuration d'un conteneur, fin de
    son journal ; aucune action qui démarre, arrête, supprime, modifie, inspecte l'intérieur ou
    télécharge quoi que ce soit ;
  - auprès du **même démon** que la CLI Docker de l'apprenant (contexte et variables de
    connexion respectés), sans aucun autre accès réseau ;
  - aucune autre commande exécutée.
- **FR-011**: Les droits demandés MUST être ceux de l'apprenant : aucune élévation (pas de
  `sudo`), aucun autre socket que celui de sa CLI Docker.

**Intégration au reste de TELEMACH**

- **FR-012**: La dernière analyse MUST être conservée (masquée) comme dernier diagnostic, pour
  que `docker telemach last` la réaffiche sans interroger à nouveau le démon et que
  `docker telemach deepen` approfondisse son concept.
**Vérification après un lancement en arrière-plan**

- **FR-013**: Après un `docker run` (ou `docker container run`) **réussi** lancé en arrière-plan
  (`-d`, `--detach`) dans une session interactive où l'intégration shell est active, TELEMACH
  MUST vérifier une seule fois l'état du conteneur lancé, et, s'il s'est arrêté avec un code
  différent de 0 (ou a été supprimé après un arrêt), le signaler en une ligne qui indique son
  nom, son code, le délai depuis son lancement et la commande `docker telemach ps`.
- **FR-014**: La vérification MUST avoir lieu environ 3 secondes après la fin de la commande,
  en arrière-plan, sans jamais retarder le retour de l'invite ; son signal MUST s'afficher à
  l'affichage suivant de l'invite (jamais au milieu d'une saisie ni de la sortie d'une autre
  commande), sur la sortie d'erreur, sans message de tâche de fond du shell.
- **FR-015**: La vérification MUST préserver toutes les garanties de l'intégration shell
  (FR-011 à FR-014 de la feature 003 : entrées, sorties et code de la commande inchangés,
  aucune commande relancée, bash et zsh, aucun fichier ni processus laissé derrière, y compris
  à la fermeture du terminal) ; si le démon est injoignable, elle se tait.
- **FR-016**: La vérification MUST pouvoir être désactivée seule, sans désactiver le reste de
  TELEMACH, et `TELEMACH_DISABLE=1` MUST continuer de tout désactiver.

**Commande**

- **FR-017**: La commande MUST fonctionner à l'identique sous bash et zsh, avec ou sans
  l'intégration shell (c'est une sous-commande du plugin, appelée directement), avec des codes
  de sortie documentés : 0 (analyse affichée, ou aucun conteneur en erreur), 1 (démon
  injoignable ou conteneur désigné introuvable, explication affichée), 2 (usage invalide).

### Key Entities

- **Conteneur en erreur** : nom, identifiant court, image, commande, code de sortie, date
  d'arrêt, arrêt pour manque de mémoire, erreur de démarrage enregistrée, nombre de
  redémarrages, politique de redémarrage, projet et service Compose éventuels, noms (jamais les
  valeurs) des variables d'environnement.
- **Extrait de journal** : dernières lignes du journal, nettoyées et masquées.
- **Vérification en attente** : conteneur lancé, moment du lancement ; supprimée une fois la
  vérification faite, son signal affiché ou la session fermée.
- **Analyse** : conteneur concerné, cause retenue (motif de journal, état ou code), entrée du
  catalogue, date ; conservée comme dernier diagnostic.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Sur un corpus d'au moins 12 scénarios réels reproductibles (variable obligatoire
  absente, permission refusée, adresse déjà utilisée, fichier ou commande introuvable,
  exécutable non exécutable, format d'exécutable incompatible, manque de mémoire, arrêt par
  `docker stop`, arrêt forcé, erreur de segmentation, erreur applicative générique, boucle de
  redémarrage), la cause proposée est correcte dans au moins 90 % des cas, et aucun cas ne
  reçoit une cause fausse présentée comme certaine.
- **SC-002**: Dans 100 % des scénarios du corpus, l'état de tous les conteneurs est identique
  avant et après `docker telemach ps` (statut, code de sortie, nombre de redémarrages, journal),
  et aucune autre commande que les lectures autorisées par FR-010 n'est effectuée.
- **SC-003**: Dans 100 % des cas de test contenant des secrets (dans le journal, dans les
  variables d'environnement définies au lancement), aucune valeur secrète n'apparaît à l'écran ni
  dans l'état conservé.
- **SC-004**: L'analyse s'affiche en moins de 2 secondes dans 95 % des cas, avec au moins
  200 conteneurs sur la machine et un journal de plus de 100 000 lignes.
- **SC-005**: Pour 100 % des conteneurs du corpus lancés en arrière-plan qui s'arrêtent en
  erreur dans la seconde qui suit leur lancement, le signal de FR-013 est affiché à l'invite
  suivante ; pour 100 % des conteneurs qui continuent de tourner, rien n'est affiché ; le retour
  de l'invite n'est pas retardé de manière perceptible (moins de 50 ms ajoutées au p95).
- **SC-006**: Sur l'exemple de l'issue #8 (conteneur lancé en arrière-plan qui s'arrête
  aussitôt), un débutant identifie la cause et corrige sa commande du premier coup (validé par
  l'auteur de l'issue).

## Assumptions

- La feature s'appuie sur le catalogue, le rendu, le masquage et l'état des features 001 à 003 ;
  les explications des codes 126, 127, 137 et 143 existent déjà pour les commandes en échec et
  sont réutilisées ou complétées.
- L'exception à la règle « le plugin n'exécute aucune commande » est volontaire et limitée à
  `docker telemach ps` et à la vérification de FR-013 (lecture de l'état d'un seul conteneur,
  choisie par l'apprenant lors de la clarification du 2026-10-16) (FR-010) ; elle sera reportée dans la spécification et les tests de la
  feature 001 (FR-015, SC-007), qui continuent de s'appliquer à toutes les autres commandes du
  plugin. La constitution n'interdit pas d'interroger le démon ; elle exige la validation des
  sorties Docker, l'absence d'injection de commande et le moindre privilège, qui s'appliquent ici.
- Seuls les conteneurs encore présents sur la machine sont analysables ; un conteneur supprimé
  (`--rm`) ne laisse aucune trace exploitable.
- Les conteneurs arrêtés avec le code 0 et les conteneurs en cours d'exécution (hors boucle de
  redémarrage) sont hors périmètre.
- Docker Compose : les conteneurs créés par Compose sont analysés comme les autres, avec leur
  projet et leur service ; une commande dédiée à un projet Compose entier est hors périmètre.
- L'analyse de l'intérieur d'un conteneur (fichiers, processus) est hors périmètre : elle
  demanderait de l'exécuter ou d'y entrer, ce que FR-010 interdit.
