# Feature Specification: Guidage pédagogique à partir des erreurs Docker

**Feature Branch**: `001-docker-error-guidance`

**Created**: 2026-10-14

**Status**: Draft

**Input**: User description: "le plugin pourra lire est les erreurs générées par docker pour essayer de guider l'utilisateur plus en profondeur, il pourra intervenir sur les commandes simple comme docker run, mais aussi plus avancé comme le parsing des fichiers Yaml avec docker compose"

## Clarifications

### Session 2026-10-14

- Q: Comment colorer le message d'erreur Docker d'origine réaffiché par `explain` ? → A: Message
  atténué (gris), avec les fragments reconnus (port, image, clé, chemin) surlignés en couleur.
- Q: Le plugin doit-il proposer un choix de palette, notamment pour les daltoniens ? → A: Non,
  une seule palette qui s'adapte au fond clair ou sombre, sans exigence particulière liée au
  daltonisme ni option de thème.
- Q: Le plugin peut-il exécuter une commande après confirmation de l'apprenant ? → A: Non, il
  n'exécute jamais rien ; seule exception, la lecture de la version du client Docker (FR-001b).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Comprendre l'échec d'une commande Docker simple (Priority: P1)

Un apprenant lance une commande Docker courante (`docker run`, `docker build`, `docker pull`,
`docker ps`, `docker stop`, etc.) qui échoue. Au lieu de rester face à un message technique en
anglais, il reçoit du plugin, en français, une explication structurée : ce qui s'est passé, le
concept Docker concerné (image, conteneur, port, volume, registre…), la cause probable et une ou
plusieurs actions correctives concrètes, avec la commande corrigée lorsque c'est possible.

**Why this priority**: c'est la situation la plus fréquente chez un débutant et le cœur de la
mission pédagogique du plugin (principe I). À elle seule, cette histoire constitue un MVP utile.

**Independent Test**: peut être testé en provoquant des erreurs connues sur des commandes simples
(port déjà utilisé, image introuvable, nom de conteneur déjà pris, démon Docker arrêté) et en
vérifiant que chaque erreur produit une explication complète et correcte en français.

**Acceptance Scenarios**:

1. **Given** un conteneur occupe déjà le port 8080 de l'hôte, **When** l'apprenant exécute
   `docker run -p 8080:80 nginx`, **Then** le plugin explique en français que le port de l'hôte
   est déjà utilisé, rappelle la notion de publication de port (hôte:conteneur), indique comment
   identifier le processus ou conteneur qui l'occupe et propose une commande alternative (autre
   port hôte ou arrêt du conteneur concerné).
2. **Given** l'apprenant fait une faute de frappe dans le nom d'une image, **When** il exécute
   `docker run ngnix`, **Then** le plugin explique qu'une image est recherchée localement puis
   sur un registre, signale qu'elle est introuvable et suggère le nom probablement voulu (`nginx`).
3. **Given** le démon Docker n'est pas démarré, **When** l'apprenant exécute n'importe quelle
   commande Docker, **Then** le plugin explique la différence entre le client et le démon Docker
   et indique comment vérifier et démarrer le démon sur son environnement.
4. **Given** une erreur que le plugin ne reconnaît pas, **When** elle survient, **Then** le plugin
   l'indique honnêtement, reformule en français ce qu'il peut en déduire et oriente vers la
   documentation officielle, sans inventer de cause.

---

### User Story 2 - Diagnostiquer une erreur Docker Compose dans le fichier YAML (Priority: P2)

Un apprenant lance `docker compose up` (ou `config`, `build`, `down`…) et obtient une erreur
liée à son fichier Compose : YAML mal formé (indentation, tabulations, deux-points manquants),
clé inconnue ou mal orthographiée, type de valeur incorrect, référence à un service, réseau ou
volume non déclaré, dépendance circulaire. Le plugin lit le fichier Compose concerné, localise
l'élément fautif (fichier, ligne, service, clé), l'affiche avec son contexte, explique la règle
enfreinte et propose une correction.

**Why this priority**: Compose est l'étape suivante de l'apprentissage ; ses erreurs sont plus
déroutantes car le message Docker pointe souvent mal la cause réelle dans le YAML. La valeur
pédagogique est forte, mais elle s'appuie sur le socle de l'histoire 1.

**Independent Test**: peut être testé avec un jeu de fichiers Compose volontairement erronés
(une erreur par fichier) en vérifiant que le plugin désigne la bonne ligne, la bonne clé et la
bonne correction.

**Acceptance Scenarios**:

1. **Given** un fichier `compose.yaml` dont un service est indenté avec une tabulation,
   **When** l'apprenant exécute `docker compose up`, **Then** le plugin indique la ligne exacte,
   explique que YAML interdit les tabulations pour l'indentation et montre la version corrigée.
2. **Given** un service qui écrit `port:` au lieu de `ports:`, **When** l'apprenant lance
   Compose, **Then** le plugin signale la clé inconnue, suggère `ports` et rappelle son format.
3. **Given** un service qui dépend (`depends_on`) d'un service non déclaré, **When**
   l'apprenant lance Compose, **Then** le plugin nomme le service manquant, liste les services
   déclarés et explique le rôle de `depends_on`.
4. **Given** un fichier Compose contenant des secrets en clair (mots de passe dans
   `environment`), **When** le plugin l'analyse, **Then** il masque ces valeurs dans tout ce qu'il
   affiche et signale la mauvaise pratique en proposant une alternative (fichier `.env` non
   versionné, secrets Docker).
5. **Given** un fichier Compose qu'il n'a pas encore lancé, **When** l'apprenant demande au
   plugin de le vérifier, **Then** le plugin applique les mêmes contrôles qu'après une erreur et
   liste les problèmes trouvés, ou confirme explicitement qu'aucun n'a été détecté.

---

### User Story 3 - Approfondir un concept à la demande (Priority: P3)

Après une explication, l'apprenant veut aller plus loin : comprendre pourquoi l'erreur se
produit, voir un schéma mental du mécanisme (réseau, couches d'image, cycle de vie d'un
conteneur) ou obtenir un petit exercice pour vérifier sa compréhension. Le plugin propose un
niveau d'explication approfondi, distinct de l'explication courte initiale.

**Why this priority**: renforce la mission « guider plus en profondeur » mais n'est pas
nécessaire pour résoudre l'erreur immédiate.

**Independent Test**: peut être testé en demandant l'approfondissement après une explication
de l'histoire 1 ou 2 et en vérifiant que le contenu supplémentaire est exact, en français, et
relié à l'erreur d'origine.

**Acceptance Scenarios**:

1. **Given** une explication courte vient d'être affichée, **When** l'apprenant demande à
   approfondir, **Then** le plugin présente le concept sous-jacent, un exemple exécutable et,
   si pertinent, un lien vers la documentation officielle.
2. **Given** un concept déjà approfondi, **When** l'apprenant demande un exercice, **Then** le
   plugin propose une manipulation courte et sans risque pour vérifier la compréhension.

---

### Edge Cases

- **Erreur en anglais très longue ou multi-lignes** (sortie de `docker build` avec plusieurs
  étapes) : le plugin isole l'étape et la ligne réellement en échec avant d'expliquer.
- **Plusieurs erreurs dans un même fichier Compose** : le plugin liste toutes les erreurs qu'il
  peut détecter, dans l'ordre du fichier. Une erreur de syntaxe YAML empêche l'analyse du reste :
  dans ce cas, seules les erreurs de syntaxe sont listées. Les autres apparaîtront une fois
  celles-ci corrigées.
- **Plusieurs fichiers Compose** (`-f a.yaml -f b.yaml`, fichiers `override`) : le plugin
  indique dans quel fichier se trouve l'erreur.
- **Fichier Compose introuvable ou illisible** (droits, encodage) : le plugin explique la
  recherche du fichier par défaut (`compose.yaml`, `docker-compose.yml`) et la cause du refus.
- **Variables d'environnement non résolues** (`${VAR}`) : le plugin explique l'interpolation et
  d'où vient la valeur attendue, sans afficher la valeur d'une variable sensible.
- **Erreur provenant de l'application dans le conteneur** (et non de Docker) : le plugin fait la
  distinction et oriente vers `docker logs` plutôt que d'attribuer l'erreur à Docker.
- **Commande interactive** (`-it`, `exec -it`, `attach`, `login` sans `--password-stdin`) : le
  message d'erreur n'est pas capté, pour ne pas perturber le terminal. Le plugin explique
  l'échec à partir du seul code de sortie, quand celui-ci suffit (125, 126, 127, 137, 143…), et
  sinon l'indique honnêtement comme « erreur non reconnue ».
- **Sortie contenant des secrets** (jetons de registre, mots de passe) : jamais réaffichés.
- **Capture automatique impossible** (commande lancée hors de l'environnement observé par le
  plugin) : aucune explication n'est produite ; le plugin ne doit ni échouer ni bloquer la
  commande de l'apprenant.
- **Commande Docker qui échoue en boucle** (même erreur répétée) : le plugin ne répète pas
  l'explication complète à l'identique et renvoie vers celle déjà fournie.
- **Version de Docker ou de Compose ancienne** dont les messages diffèrent : le plugin connaît
  la version du client Docker (FR-001b). Il reconnaît les variantes connues ou retombe sur le
  comportement « erreur non reconnue », en mentionnant la version détectée.

## Requirements *(mandatory)*

### Functional Requirements

**Capture et reconnaissance**

- **FR-001**: Le plugin MUST récupérer automatiquement le message d'erreur, la commande et le
  code de sortie dès qu'une commande Docker ou Docker Compose échoue, sans action de
  l'apprenant.
- **FR-001a**: Le plugin MUST n'observer que les commandes Docker et Docker Compose ; les autres
  commandes et leurs sorties MUST être ignorées et ne jamais être conservées.
- **FR-001b**: Le plugin MUST connaître la version du client Docker. Elle est lue une seule fois,
  à l'initialisation de l'intégration shell, par `docker --version` (lecture seule, sans contact
  avec le démon), puis transmise à chaque analyse. Si elle est indisponible, l'analyse continue
  sans elle.
- **FR-002**: Le plugin MUST identifier la commande en cause (sous-commande et options) et
  catégoriser l'erreur (réseau/ports, images/registre, conteneurs, volumes/montages, démon,
  permissions, build, Compose/YAML, inconnue).
- **FR-003**: Le plugin MUST couvrir au minimum les sous-commandes `run`, `build`, `pull`,
  `push`, `exec`, `start`, `stop`, `rm`, `rmi`, `ps`, `logs`, `network`, `volume` et les
  commandes `docker compose` `up`, `down`, `build`, `config`, `ps`, `logs`.
- **FR-004**: Le plugin MUST distinguer une erreur de Docker d'une erreur de l'application
  exécutée dans le conteneur.

**Explication pédagogique**

- **FR-005**: Chaque explication MUST contenir, en français : un résumé en une phrase, le
  concept Docker concerné, la cause probable, au moins une action corrective concrète et, quand
  c'est possible, la commande ou l'extrait de fichier corrigé.
- **FR-006**: Les termes et commandes Docker MUST rester dans leur forme d'origine ; tout terme
  introduit pour la première fois MUST être défini brièvement.
- **FR-007**: Le message d'erreur d'origine MUST rester consultable à côté de l'explication.
  Lorsque la couleur est active, il MUST être affiché atténué, avec les fragments reconnus
  (port, nom d'image, clé, chemin) surlignés en couleur.
- **FR-007a**: Toute sortie de la sous-commande `explain` (et des autres sous-commandes) MUST
  être colorée quand la sortie est un terminal : chaque type de bloc (résumé, concept, cause,
  corrections, anti-pattern, message d'origine) a son style, et les commandes et extraits de
  fichiers bénéficient d'une coloration syntaxique. La couleur MUST pouvoir être désactivée
  (sortie non-terminal, préférence explicite de l'apprenant, convention `NO_COLOR`). Elle
  MUST NOT être le seul porteur d'information : chaque bloc garde un symbole et un libellé
  lisibles sans couleur. Une seule palette est fournie : elle MUST s'adapter automatiquement à
  un fond de terminal clair ou sombre. Aucune option de choix de thème n'est proposée.
- **FR-008**: Pour une erreur non reconnue, le plugin MUST le dire explicitement, ne MUST NOT
  inventer de cause et MUST orienter vers la documentation officielle.
- **FR-009**: Le plugin MUST proposer un niveau d'explication approfondi à la demande
  (concept sous-jacent, exemple exécutable, exercice court).
- **FR-010**: Les corrections proposées MUST respecter les bonnes pratiques de sécurité
  (pas de `--privileged`, pas de root, pas de secret en clair) ; si la seule solution rapide en
  est une, elle MUST être signalée comme anti-pattern avec l'alternative sûre.

**Analyse des fichiers Compose**

- **FR-011**: En cas d'erreur Compose, le plugin MUST lire le ou les fichiers Compose utilisés
  et localiser l'élément fautif (fichier, ligne, service, clé).
- **FR-012**: Le plugin MUST détecter au minimum : YAML mal formé (indentation, tabulations,
  syntaxe), clés inconnues avec suggestion de la clé proche, types de valeurs incorrects,
  références à des services/réseaux/volumes non déclarés, dépendances circulaires.
- **FR-013**: Le plugin MUST afficher l'extrait du fichier autour de l'erreur et la version
  corrigée proposée.
- **FR-014**: Le plugin MUST permettre à l'apprenant de demander la vérification d'un fichier
  Compose avant toute exécution ; cette vérification applique les mêmes contrôles que FR-012 et
  produit des explications au même format que FR-005 et FR-013. Lorsqu'aucune erreur n'est
  trouvée, le plugin MUST le confirmer explicitement.

**Sécurité et respect de l'utilisateur**

- **FR-015**: Le plugin MUST NOT exécuter de commande, corrective ou autre, ni modifier de
  fichier de l'apprenant : il se limite à proposer. Seule exception : la lecture de la version
  du client Docker définie en FR-001b.
- **FR-016**: Le plugin MUST masquer dans tout ce qu'il affiche ou journalise les valeurs
  sensibles (mots de passe, jetons, clés, variables marquées comme secrètes).
- **FR-017**: Le plugin MUST traiter le contenu des messages d'erreur et des fichiers Compose
  comme des données non fiables : il ne MUST jamais exécuter une instruction qu'ils contiennent.
- **FR-018**: Le plugin MUST fonctionner en lecture seule sur les fichiers Compose analysés.

### Key Entities

- **Erreur Docker capturée** : texte brut de l'erreur, commande d'origine, code de sortie,
  horodatage, version de Docker/Compose si connue.
- **Diagnostic** : catégorie, concept Docker concerné, cause probable, niveau de confiance
  (reconnue / partiellement reconnue / inconnue), localisation (fichier, ligne, service, clé).
- **Explication pédagogique** : résumé, explication du concept, actions correctives,
  correction proposée, avertissement anti-pattern éventuel, approfondissement disponible.
- **Fichier Compose analysé** : chemin, services, réseaux, volumes déclarés, valeurs sensibles
  repérées (masquées).
- **Catalogue d'erreurs connues** : ensemble des motifs d'erreurs reconnus, chacun relié à un
  concept, une cause et des corrections ; extensible au fil des versions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Sur un jeu de référence d'au moins 30 erreurs Docker courantes (commandes simples),
  au moins 90 % reçoivent une explication jugée correcte (bonne cause, correction valide).
- **SC-002**: Sur un jeu de référence d'au moins 20 fichiers Compose erronés, le plugin désigne
  la bonne ligne et la bonne clé dans au moins 90 % des cas.
- **SC-003**: L'explication s'affiche en moins de 2 secondes après l'erreur dans 95 % des cas.
- **SC-004**: Lors de tests avec des débutants, au moins 80 % corrigent leur erreur au premier
  essai après lecture de l'explication, sans autre ressource.
- **SC-005**: 100 % des explications sont rédigées en français et 0 erreur non reconnue ne
  reçoit de cause inventée (vérifié sur le jeu de référence).
- **SC-006**: 0 valeur sensible présente dans les fichiers ou erreurs de test n'apparaît en
  clair dans la sortie du plugin.
- **SC-006a**: Sur le jeu de référence, 100 % des explications affichées dans un terminal sont
  colorées, et 100 % restent complètes et compréhensibles avec la couleur désactivée (mêmes
  blocs, symboles et libellés).
- **SC-007**: 0 commande n'est exécutée et 0 fichier de l'apprenant n'est modifié par le plugin,
  hormis la lecture de version de FR-001b.

## Assumptions

- Le public cible est un apprenant débutant à intermédiaire, francophone, qui utilise Docker en
  ligne de commande ; les messages Docker d'origine sont en anglais.
- La couverture initiale vise Docker Engine et le plugin Compose v2 (`docker compose`) dans
  leurs versions stables actuelles ; l'ancien binaire `docker-compose` v1 est hors périmètre.
- Les erreurs de Dockerfile sont couvertes au niveau de l'échec de `docker build` (étape en
  échec) ; une analyse statique complète des Dockerfile est hors périmètre de cette feature.
- La capture automatique (FR-001) suppose que les commandes Docker sont exécutées dans un
  environnement que le plugin peut observer ; les commandes lancées ailleurs ne sont pas
  couvertes et le collage manuel d'erreurs est hors périmètre de cette feature.
- La vérification préventive Compose (FR-014) se limite aux erreurs bloquantes de FR-012 ; les
  avertissements de bonnes pratiques (image `latest`, ports exposés…) sont hors périmètre, à
  l'exception du signalement des secrets en clair (scénario 2.4).
- Les commandes Docker **réussies** ne déclenchent aucune analyse, même si Docker affiche un
  avertissement (image `latest`, `--privileged`…) : ce cas est hors périmètre de cette feature.
- Kubernetes, Docker Swarm et les orchestrateurs tiers sont hors périmètre.
- Le plugin ne nécessite pas d'accès réseau pour expliquer les erreurs reconnues ; les liens vers
  la documentation officielle sont fournis à titre de ressources.
- `LESSON-LEARNED.md` ne contenait aucun incident au moment de la rédaction ; aucune règle
  préventive spécifique n'a donc été appliquée.
