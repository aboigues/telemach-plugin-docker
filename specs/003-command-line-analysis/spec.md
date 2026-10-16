# Feature Specification: Analyse de la ligne de commande, même en cas de succès

**Feature Branch**: `003-command-line-analysis`

**Created**: 2026-10-15

**Status**: Accepted

**Input**: User description: "003-command-line-analysis — Analyser la ligne de commande docker tapée par l'apprenant, même quand elle réussit, pour repérer les erreurs d'écriture que Docker accepte sans broncher (issue GitHub #9). Exemple : `docker run -d nginx -p 8090:80` : le conteneur démarre (code 0), mais le mapping de port n'est pas pris en compte, car tout ce qui suit le nom de l'image est passé au conteneur comme commande. Attendu : TELEMACH avertit en français, de façon pédagogique (concept, cause, commande corrigée à copier), sans jamais bloquer, retarder sensiblement ni modifier la commande, ni changer son code de sortie ou ses sorties. Le plugin n'exécute aucune commande supplémentaire et reste hors ligne. Les secrets de la ligne de commande sont masqués."

## Clarifications

### Session 2026-10-15

- Q: Quand afficher l'avertissement sur la ligne de commande ? → A: Après la commande, comme les
  explications d'erreur (sortie de Docker d'abord, puis l'avertissement, qui peut dire ce qui
  s'est réellement passé).
- Q: Quels pièges silencieux couvrir au-delà des options placées après l'image ? → A: Un
  catalogue large (voir US2 et FR-004), chaque piège restant détectable sans ambiguïté.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Être averti d'une option Docker placée après l'image (Priority: P1)

Un apprenant tape `docker run -d nginx -p 8090:80`. Docker démarre le conteneur sans erreur, mais
le port n'est pas publié : `-p 8090:80` a été transmis au conteneur, pas à Docker. Sans aide,
l'apprenant ne comprend pas pourquoi `http://localhost:8090` ne répond pas. TELEMACH l'avertit :
l'ordre des éléments compte (`docker run [OPTIONS] IMAGE [COMMANDE] [ARG...]`), ce qui suit
l'image est donné au conteneur, et voici la commande corrigée
(`docker run -d -p 8090:80 nginx`).

**Why this priority**: c'est le cas de l'issue #9 et une erreur de débutant très fréquente, que
rien ne signale aujourd'hui puisque Docker répond « succès ».

**Independent Test**: lancer chaque commande d'un corpus de lignes « options après l'image » et
vérifier l'avertissement et la commande corrigée ; lancer un corpus de lignes correctes
(y compris des commandes de conteneur légitimes comme `docker run alpine ls -l`) et vérifier
qu'aucun avertissement n'apparaît.

**Acceptance Scenarios**:

1. **Given** l'intégration shell active, **When** l'apprenant exécute
   `docker run -d nginx -p 8090:80` et que la commande réussit, **Then** TELEMACH affiche un
   avertissement en français qui nomme l'option mal placée (`-p 8090:80`), explique la forme
   `docker run [OPTIONS] IMAGE [COMMANDE] [ARG...]` et propose `docker run -d -p 8090:80 nginx`.
2. **Given** la même commande, **When** l'avertissement est affiché, **Then** la sortie standard
   de Docker (identifiant du conteneur), sa sortie d'erreur et son code de sortie sont
   exactement ceux qu'il aurait produits sans TELEMACH.
3. **Given** `docker run --rm alpine ls -l` ou `docker run postgres:18-alpine -c max_connections=200`
   (arguments légitimes destinés au programme du conteneur), **When** la commande s'exécute,
   **Then** aucun avertissement n'est affiché.
4. **Given** `docker exec mon-conteneur -it sh`, **When** la commande s'exécute, **Then**
   TELEMACH signale que `-it` a été placé après le nom du conteneur et propose
   `docker exec -it mon-conteneur sh`.
5. **Given** une option mal placée contenant un secret (`docker run nginx -e API_TOKEN=abc123`),
   **When** l'avertissement est affiché, **Then** la valeur du secret est masquée dans
   l'avertissement et dans la commande corrigée proposée.

---

### User Story 2 - Être averti d'autres pièges silencieux de la ligne de commande (Priority: P2)

Au-delà de l'ordre des options, certaines écritures sont acceptées par Docker mais ne font pas
ce que le débutant croit, ou enseignent une mauvaise pratique. TELEMACH les signale avec le même
format, à l'un de deux niveaux : **avertissement** (⚠, la commande ne fait probablement pas ce
qui est voulu) ou **information** (ℹ, la commande fonctionne mais une bonne pratique est
ignorée). Le catalogue de cette feature (FR-004) couvre : volume nommé au lieu du dossier local,
port de la machine choisi au hasard, variable d'environnement non définie, image sans tag
explicite, ports publiés ignorés en réseau `host`, secret écrit en clair dans la commande,
privilèges excessifs.

**Why this priority**: même valeur pédagogique que US1, mais chaque piège supplémentaire
élargit le corpus à maintenir et le risque de faux positifs.

**Independent Test**: un corpus par piège retenu, avec des lignes fautives (avertissement
attendu) et des lignes correctes voisines (aucun avertissement).

**Acceptance Scenarios**:

1. **Given** un dossier `./data` dans le dossier courant, **When** l'apprenant exécute
   `docker run -v data:/app/data alpine:3.24 ls /app/data`, **Then** TELEMACH avertit que `data`
   désigne un volume nommé géré par Docker, pas le dossier `./data`, et propose
   `-v ./data:/app/data`.
2. **Given** `docker run -d -p 80 nginx:1.30-alpine`, **When** la commande réussit, **Then**
   TELEMACH informe que Docker a choisi au hasard le port de la machine et indique comment le
   connaître (`docker port <conteneur>`) ou le fixer (`-p 8080:80`).
3. **Given** la variable `DB_HOST` non définie dans le shell, **When** l'apprenant exécute
   `docker run -e DB_HOST mon-app:1.0`, **Then** TELEMACH avertit que la variable sera absente
   du conteneur et propose `-e DB_HOST=<valeur>` ; si la variable est définie, aucun
   avertissement.
4. **Given** `docker run nginx` ou `docker run nginx:latest`, **When** la commande réussit,
   **Then** TELEMACH informe qu'un tag explicite (`nginx:<version>`, choisi sur la page de
   l'image ; aucun tag précis n'est suggéré, il vieillirait, LL-004) rend la commande
   reproductible ; cette information n'est pas répétée pour la même image avant 24 heures.
5. **Given** `docker run -d --network host -p 8080:80 nginx:1.30-alpine`, **When** la commande
   réussit, **Then** TELEMACH avertit que `-p` est ignoré en réseau `host` (le conteneur utilise
   directement les ports de la machine).
6. **Given** `docker run -e DB_PASSWORD=SuperSecret123 postgres:18-alpine`, **When** la commande
   s'exécute, **Then** TELEMACH avertit qu'un secret écrit dans la commande est visible dans
   l'historique du shell et dans `docker inspect`, propose `--env-file` ou un secret Docker, et
   n'affiche jamais la valeur.
7. **Given** `docker run --privileged …` ou un montage du socket Docker
   (`-v /var/run/docker.sock:/var/run/docker.sock`), **When** la commande s'exécute, **Then**
   TELEMACH avertit que le conteneur obtient un contrôle quasi total de la machine, et présente
   l'option comme un anti-pattern pour un usage courant (principe III).
8. **Given** des lignes correctes voisines de chaque piège (volume nommé alors qu'aucun dossier
   `./data` n'existe, `-p 8080:80`, variable définie, tag explicite, réseau par défaut, secret
   passé par `--env-file`), **When** elles s'exécutent, **Then** aucun avertissement n'est
   affiché.

---

### User Story 3 - Garder le contrôle sur les avertissements (Priority: P3)

Un apprenant qui a compris la règle ne veut pas relire le même avertissement à chaque commande.
Un formateur veut pouvoir désactiver ces avertissements pendant une démonstration.

**Why this priority**: confort d'usage ; sans lui, l'outil devient vite agaçant, mais la valeur
pédagogique principale est apportée par US1.

**Independent Test**: répéter la même commande fautive plusieurs fois de suite et vérifier la
forme courte ; désactiver les avertissements et vérifier leur absence, sans effet sur les
explications d'erreur.

**Acceptance Scenarios**:

1. **Given** un avertissement complet déjà affiché pour le même piège dans les 10 dernières
   minutes, **When** l'apprenant refait la même erreur, **Then** TELEMACH n'affiche qu'un rappel
   d'une ligne, avec la commande corrigée.
2. **Given** les avertissements de ligne de commande désactivés, **When** l'apprenant tape une
   commande fautive, **Then** aucun avertissement n'apparaît, et les explications d'erreur
   (codes de sortie non nuls) continuent de fonctionner.
3. **Given** `TELEMACH_DISABLE=1`, **When** une commande s'exécute, **Then** ni avertissement ni
   explication n'apparaît (comportement actuel inchangé).

---

### Edge Cases

- **Commande en échec** : si la ligne contient un piège **et** que Docker échoue (par exemple le
  conteneur reçoit `-p` et s'arrête aussitôt en erreur), l'explication de l'erreur et
  l'avertissement ne doivent pas se contredire ni se répéter ; le piège de la ligne de commande,
  cause probable de l'échec, est mentionné une seule fois.
- **Commande interactive** (`docker run -it alpine sh`, `docker exec -it …`) : l'avertissement
  ne doit jamais perturber le terminal de la session interactive.
- **Faux positifs** : un argument qui ressemble à une option Docker mais qui est légitime pour le
  programme du conteneur (`postgres -p 5433`, `redis-server --port 7000`, `curl -v …`,
  `nginx -g 'daemon off;'`) ne déclenche aucun avertissement, sauf si sa forme ne laisse aucun
  doute (par exemple `-p 8090:80`, publication de port au format `HÔTE:CONTENEUR`).
- **Séparateur `--`** : ce qui suit `--` dans la ligne n'est jamais analysé comme une option
  Docker mal placée.
- **Commandes de gestion** : `docker container run …`, `docker container create …`,
  `docker container exec …` sont traitées comme `docker run`, `docker create`, `docker exec`.
- **Commande sans image ni conteneur** (`docker run -d`) : Docker échoue déjà avec un message
  explicite, l'explication d'erreur existante s'applique ; pas d'avertissement supplémentaire.
- **Caractères spéciaux et séquences de contrôle** dans les arguments : nettoyés avant tout
  affichage, comme pour les messages d'erreur.
- **Lignes très longues** : la commande corrigée reste affichée en entier, sur une ligne à
  part, et n'est **jamais** repliée par TELEMACH (elle doit rester copiable telle quelle) ; seul
  le texte explicatif est replié à la largeur du terminal.

## Requirements *(mandatory)*

### Functional Requirements

**Détection**

- **FR-001**: TELEMACH MUST analyser la ligne de commande de `docker run`, `docker create` et
  `docker exec` (et de leurs équivalents `docker container …`) **quel que soit le code de
  sortie** de Docker.
- **FR-002**: TELEMACH MUST détecter une option Docker placée après le nom de l'image (`run`,
  `create`) ou du conteneur (`exec`) lorsque sa forme est sans ambiguïté : option propre à Docker
  pour cette sous-commande, avec une valeur au format attendu par Docker (par exemple publication
  de port `HÔTE:CONTENEUR`, montage `SOURCE:CIBLE`, variable `NOM=VALEUR`, `--name VALEUR`,
  `-d`, `--rm`, `-it`).
- **FR-003**: TELEMACH MUST NOT avertir lorsque les arguments placés après l'image peuvent
  raisonnablement être destinés au programme du conteneur ; en cas de doute, il s'abstient
  (aucun faux positif sur le corpus de commandes légitimes, SC-002).
- **FR-004**: Au-delà de FR-002, TELEMACH MUST détecter, pour `run` et `create`, les pièges
  suivants, chacun uniquement lorsque sa condition est remplie sans ambiguïté :
  1. **volume nommé au lieu du dossier local** (⚠) : source de `-v`/`--volume` sans `/` ni `.`
     initial, alors qu'un dossier de ce nom existe dans le dossier courant ;
  2. **port de la machine aléatoire** (ℹ) : publication de port sans port de la machine
     (`-p 80`, `-p 80/udp`) ; les intervalles (`-p 8000-8010`) et la forme `-p IP::PORT` ne
     sont pas signalés (doute : silence) ;
  3. **variable d'environnement non définie** (⚠) : `-e NOM`/`--env NOM` sans valeur, alors
     que `NOM` n'est pas définie dans l'environnement de l'apprenant ;
  4. **image sans tag explicite** (ℹ) : image sans tag ni digest, ou avec le tag `latest` ;
     non répétée pour la même image avant 24 heures ;
  5. **ports ignorés en réseau `host`** (⚠) : `--network host` (ou `--net host`) combiné à une
     publication de port ; le piège 2 n'est alors pas signalé en plus (la publication est
     ignorée de toute façon) ;
  6. **secret en clair dans la commande** (⚠) : valeur de `-e`/`--env` masquée par les règles
     de masquage existantes **et**, sans ambiguïté, secrète : nom contenant un mot entier qui
     désigne un secret (`PASSWORD`, `SECRET`, `TOKEN`, `KEY`, `AUTH`…, `DB_PASSWORD`, `apiKey`),
     sauf s'il se termine par un mot de référence (`_FILE`, `_PATH`, `_URL`, `_ID`…), ou valeur
     reconnaissable par elle-même (jeton connu, identifiants dans une URL). `MONKEY`, `AUTHOR` ou
     `KEYCLOAK_URL` ne sont pas signalés, même si leur valeur reste masquée à l'affichage ;
  7. **privilèges excessifs** (⚠, anti-pattern) : `--privileged`, ou montage du socket Docker
     (`/var/run/docker.sock`).
  Le catalogue MUST être extensible : ajouter un piège ne demande qu'une entrée et ses cas de
  test (fautifs et légitimes voisins).

**Avertissement**

- **FR-005**: L'avertissement MUST être en français et suivre la structure des explications
  existantes : symbole d'avertissement et résumé, `📘 Concept` (forme
  `docker run [OPTIONS] IMAGE [COMMANDE] [ARG...]`, ou son équivalent pour `exec`),
  `🔎 Cause probable`, `💡 Correction` avec la commande corrigée complète, prête à copier.
- **FR-006**: La commande corrigée MUST conserver toutes les options et tous les arguments de
  l'apprenant, en déplaçant seulement les options mal placées avant l'image ou le conteneur,
  dans leur ordre d'origine.
- **FR-007**: L'avertissement MUST préciser ce que la commande a réellement fait (par exemple :
  « le conteneur a démarré, mais le port 8090 n'est pas publié ») et, si utile, comment corriger
  le conteneur déjà lancé (l'arrêter et le relancer avec la commande corrigée).
- **FR-008**: Les secrets présents dans la ligne de commande MUST être masqués dans
  l'avertissement, la commande corrigée et tout ce qui est conservé sur disque (principe III).
- **FR-009**: L'avertissement MUST être affiché **après** la fin de la commande, à la suite de
  la sortie de Docker (comme les explications d'erreur) ; pour une session interactive, il
  apparaît à la fin de la session. Plusieurs pièges dans une même ligne sont regroupés dans un
  seul affichage, avertissements d'abord, informations ensuite.
- **FR-010**: Lorsque la commande échoue et qu'un piège de la ligne de commande est détecté,
  l'explication MUST le mentionner une seule fois, comme cause probable, sans répéter un
  avertissement séparé.

**Garanties de l'intégration shell (inchangées)**

- **FR-011**: L'entrée standard, la sortie standard, le code de sortie et le texte de la sortie
  d'erreur de Docker MUST rester identiques à une exécution sans TELEMACH ; l'avertissement est
  écrit sur la sortie d'erreur, distinctement de celle de Docker.
- **FR-012**: TELEMACH MUST NOT bloquer, modifier ni relancer la commande ; il n'exécute aucune
  commande supplémentaire et n'accède pas au réseau.
- **FR-013**: Les commandes interactives MUST NOT être perturbées : l'avertissement n'est jamais
  écrit pendant une session interactive.
- **FR-014**: L'analyse MUST fonctionner à l'identique sous bash et sous zsh (LL-002) et ne
  laisser aucun fichier temporaire, y compris en cas d'interruption (LL-003).

**Contrôle par l'apprenant**

- **FR-015**: Une même erreur de ligne de commande répétée dans les 10 minutes MUST être signalée
  par un rappel d'une ligne (même règle que les erreurs répétées).
- **FR-016**: Les avertissements de ligne de commande MUST pouvoir être désactivés seuls, sans
  désactiver l'explication des erreurs ; `TELEMACH_DISABLE=1` continue de tout désactiver.
- **FR-017**: `docker telemach last` MUST pouvoir réafficher le dernier avertissement complet,
  avec la même indication de date et de commande que pour les erreurs.

### Key Entities

- **Ligne de commande analysée** : sous-commande, options placées avant l'image ou le
  conteneur, image ou conteneur, arguments qui suivent ; arguments nettoyés et masqués.
- **Piège de ligne de commande** : identifiant, condition de détection sans ambiguïté, concept
  Docker, cause, règle de correction, exemples fautifs et exemples légitimes voisins.
- **Avertissement** : piège reconnu, éléments de la ligne en cause, commande corrigée,
  date ; conservé comme dernier diagnostic pour `last` et pour la règle de répétition.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Sur un corpus d'au moins 40 lignes fautives (au moins 10 pour les options après
  l'image ou le conteneur, au moins 3 par piège de FR-004), 100 % déclenchent un avertissement avec une commande
  corrigée qui, exécutée, produit l'effet voulu (vérifié sur un vrai démon Docker pour un
  échantillon).
- **SC-002**: Sur un corpus d'au moins 50 lignes correctes, dont au moins 20 qui passent des
  options légitimes au programme du conteneur, **aucun** avertissement n'est affiché.
- **SC-003**: Le délai ajouté à une commande analysée est imperceptible pour l'apprenant : moins
  de 50 ms dans 95 % des cas, mesuré sur un poste de développement ; les autres commandes Docker
  ne subissent aucun délai mesurable. En CI, dont les runners sont plus lents et variables, le
  même test sert de garde-fou de non-régression avec un seuil de 100 ms.
- **SC-004**: Dans 100 % des cas des tests de l'intégration shell (`run`, `create`, `exec` et
  `container …` réussies, en échec et interactives, avec et sans piège, autres commandes), la
  sortie standard, la sortie d'erreur de Docker et le code de sortie sont identiques avec et
  sans TELEMACH, sous bash et sous zsh.
- **SC-005**: Un débutant qui lit l'avertissement de l'issue #9 sait corriger sa commande du
  premier coup (validé sur l'exemple de l'issue par l'auteur de l'issue, porteur du projet,
  lors de la validation manuelle du quickstart V1 après publication).

## Assumptions

- La feature s'appuie sur l'intégration shell et le parseur de commandes existants (feature
  001) ; aucune nouvelle installation n'est demandée à l'apprenant au-delà de la mise à jour du
  plugin (et du rechargement de `shell-init`).
- Seules les commandes `run`, `create` et `exec` (et `docker container …`) sont concernées :
  ce sont celles où Docker arrête de lire ses options au premier argument positionnel (image ou
  conteneur). Pour les autres sous-commandes, une option placée après les arguments reste lue
  par Docker.
- La détection repose sur la ligne tapée, complétée par deux lectures locales sans exécution :
  l'existence d'un dossier dans le dossier courant (piège 1) et la présence d'une variable dans
  l'environnement de l'apprenant (piège 3, sans jamais lire ni afficher sa valeur). TELEMACH ne
  consulte ni l'image, ni le démon, ni le réseau (il ne peut donc pas savoir quelles options le programme du conteneur
  accepte) ; d'où la règle d'abstention en cas de doute (FR-003).
- Les avertissements ne remplacent pas les explications d'erreur existantes ; ils s'y ajoutent
  pour les commandes qui réussissent.
- Docker Compose et les fichiers Compose sont hors périmètre (déjà couverts par `check`).
