# Feature Specification: Distribution du plugin sans Go

**Feature Branch**: `002-distribution`

**Created**: 2026-10-15

**Status**: Draft

**Input**: User description: "002-distribution — Diffuser le plugin TELEMACH plus facilement, sans exiger Go chez l'apprenant. 1. GitHub Releases = source de référence : à chaque tag v*, binaires linux/darwin (amd64, arm64), version injectée, sommes SHA-256, SBOM, signature keyless et attestation de provenance SLSA ; README d'installation depuis la Release. 2. Image Docker Hub telemachlearning/telemach : essai sans installation (`check`) et installation du plugin sur l'hôte (`install`, `HOST_OS=darwin` pour macOS) ; image minimale, non-root, épinglée par digest, multi-architecture, signée, avec gate de scan. 3. Jeton Docker Hub créé et stocké par le mainteneur dans un environment « release » protégé. Hors périmètre : Homebrew, Windows natif, deb/rpm."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Installer le plugin depuis une Release, sans Go (Priority: P1)

Un apprenant (ou un formateur qui prépare une salle de TP) veut installer TELEMACH sur une
machine Linux, macOS ou WSL2 qui n'a que Docker. Il ouvre la page de la dernière version du
projet, télécharge le binaire correspondant à son système et à son processeur, vérifie qu'il
n'a pas été altéré (somme de contrôle, signature), le place dans le dossier des plugins Docker
et constate avec `docker telemach --version` qu'il a bien la version publiée.

**Why this priority**: c'est la voie de référence. Aujourd'hui, installer TELEMACH exige Go
1.27 et une compilation, ce qui exclut une grande partie du public débutant. À elle seule,
cette histoire lève ce frein et constitue un MVP.

**Independent Test**: sur une machine propre sans Go (Linux amd64, Linux arm64, macOS arm64),
suivre uniquement la procédure du README et vérifier que `docker telemach --version` affiche la
version de la Release et que l'explication automatique d'une erreur fonctionne.

**Acceptance Scenarios**:

1. **Given** un tag de version `vX.Y.Z` poussé sur un commit de `main` dont les gates sont
   vertes, **When** la publication s'exécute, **Then** une Release `vX.Y.Z` apparaît avec un
   binaire pour chacune des quatre plateformes (linux/amd64, linux/arm64, darwin/amd64,
   darwin/arm64), un fichier de sommes SHA-256, un inventaire des composants (SBOM), une
   signature vérifiable et une attestation de provenance.
2. **Given** un binaire téléchargé depuis la Release, **When** l'apprenant lance la commande de
   vérification de somme documentée, **Then** elle confirme l'intégrité du fichier ; **and**
   la vérification de signature documentée confirme qu'il a été produit par le workflow
   officiel du dépôt.
3. **Given** le binaire installé dans le dossier des plugins Docker, **When** l'apprenant
   exécute `docker telemach --version`, **Then** la version affichée est exactement `X.Y.Z`.
4. **Given** un binaire volontairement modifié, **When** l'apprenant lance la vérification de
   somme ou de signature, **Then** la vérification échoue de façon visible, et le README
   explique qu'il ne faut pas installer ce fichier.

---

### User Story 2 - Installer le plugin avec Docker seulement (Priority: P2)

Un apprenant qui ne veut ni chercher la bonne archive ni manipuler de somme de contrôle installe
le plugin en une seule commande `docker run`, à partir de l'image officielle
`telemachlearning/telemach`. L'image copie dans son dossier des plugins Docker le binaire
adapté à son système, puis lui explique en français ce qu'elle a fait et comment activer
l'explication automatique.

**Why this priority**: c'est la voie la plus simple pour un public qui a déjà Docker, et elle
sert d'exemple concret des notions que le plugin enseigne (image, conteneur, montage). Elle
s'appuie sur les mêmes binaires que la Release (P1).

**Independent Test**: sur Linux (et sous macOS avec Docker Desktop), lancer la commande
d'installation documentée, puis vérifier que `docker telemach --version` affiche la version de
l'image et que le fichier installé appartient à l'utilisateur.

**Acceptance Scenarios**:

1. **Given** une machine Linux avec Docker, **When** l'apprenant exécute la commande
   d'installation documentée (dossier des plugins monté, tag de version explicite),
   **Then** le binaire linux de la bonne architecture est copié dans ce dossier, exécutable,
   appartenant à l'utilisateur, et un message en français résume ce qui a été fait et la
   prochaine étape (`shell-init`).
2. **Given** un Mac avec Docker Desktop, **When** l'apprenant exécute la commande documentée
   pour macOS, **Then** c'est le binaire darwin de l'architecture du Mac qui est copié, et
   `docker telemach --version` fonctionne sur le Mac.
3. **Given** l'apprenant oublie de monter le dossier de destination, **When** il lance
   l'installation, **Then** rien n'est copié et un message en français explique la notion de
   montage (bind mount) et donne la commande complète à copier.
4. **Given** une version du plugin déjà installée, **When** l'apprenant relance l'installation
   avec une image plus récente, **Then** le binaire est remplacé et le message indique
   l'ancienne et la nouvelle version.

---

### User Story 3 - Essayer TELEMACH sans rien installer (Priority: P3)

Une personne curieuse, ou un formateur pendant une démonstration, veut voir ce que TELEMACH dit
d'un fichier Compose sans rien installer sur sa machine. Elle lance l'image en montant son
dossier de projet en lecture seule et obtient la vérification pédagogique du fichier Compose.

**Why this priority**: utile pour découvrir le projet et pour les démonstrations, mais la
valeur principale du plugin (l'explication automatique des erreurs) nécessite l'installation.

**Independent Test**: dans un dossier contenant un fichier Compose fautif, lancer la commande
d'essai documentée et vérifier que le résultat est identique à celui de
`docker telemach check` installé, et qu'aucun fichier du projet n'est modifié.

**Acceptance Scenarios**:

1. **Given** un dossier contenant un `compose.yaml` comportant une erreur, **When**
   l'apprenant exécute la commande d'essai documentée, **Then** il obtient la même vérification
   pédagogique qu'avec le plugin installé, en couleur si son terminal le permet.
2. **Given** le dossier de projet monté en lecture seule, **When** la vérification s'exécute,
   **Then** aucun fichier du projet n'est créé ni modifié.
3. **Given** aucun fichier Compose dans le dossier monté, **When** l'apprenant lance l'essai,
   **Then** un message en français explique qu'aucun fichier n'a été trouvé et rappelle comment
   monter le bon dossier.

---

### User Story 4 - Publier une version en toute confiance (Priority: P2)

Le mainteneur décide de publier une version : il pose un tag `vX.Y.Z` sur `main` et le pousse.
Sans autre manipulation, la Release et l'image sont produites, signées et publiées. Il n'a jamais
à confier son jeton Docker Hub à quelqu'un d'autre, et une version publiée ne peut pas être
écrasée silencieusement.

**Why this priority**: sans publication automatisée et sûre, les histoires P1 à P3 n'ont pas de
contenu fiable à distribuer.

**Independent Test**: pousser un tag de test sur un dépôt de répétition (ou un tag de
pré-version) et vérifier que Release et image apparaissent, signées, et qu'une seconde exécution
sur le même tag n'écrase rien.

**Acceptance Scenarios**:

1. **Given** le secret Docker Hub configuré par le mainteneur dans l'environnement protégé
   « release », **When** un tag `vX.Y.Z` est poussé, **Then** la Release et l'image
   `telemachlearning/telemach:X.Y.Z` sont publiées sans intervention supplémentaire.
2. **Given** un tag qui pointe sur un commit absent de `main`, ou dont le nom n'est pas une
   version valide, **When** il est poussé, **Then** rien n'est publié et la raison est indiquée
   dans le journal de l'exécution.
3. **Given** une Release `vX.Y.Z` déjà publiée, **When** la publication est relancée pour le
   même tag, **Then** elle échoue sans modifier les artefacts existants.
4. **Given** un pull request qui modifie l'image, **When** les gates s'exécutent, **Then**
   l'image est construite et analysée (vulnérabilités, bonnes pratiques du Dockerfile), et une
   vulnérabilité critique ou haute bloque la fusion.

---

### Edge Cases

- **Dossier des plugins inexistant** (installation par l'image, Linux) : si Docker crée
  lui-même le dossier monté, il appartient à root et l'utilisateur ne pourra plus le gérer.
  La commande documentée crée donc le dossier avant de lancer l'image, et l'installation
  refuse d'écrire dans un dossier qui appartient à root alors qu'elle ne s'exécute pas en root
  (signe que Docker l'a créé), en expliquant pourquoi. La règle ne compare pas le propriétaire
  à l'utilisateur, car Docker Desktop (macOS) peut présenter les fichiers montés sous une autre
  identité.
- **Droits d'écriture** : l'image s'exécute sans privilèges root ; la commande documentée la
  lance avec l'identité de l'utilisateur, pour que le fichier installé lui appartienne. Sans
  droit d'écriture, l'installation s'arrête avec une explication en français (droits Unix,
  identité du processus dans le conteneur).
- **Système non pris en charge** : `HOST_OS` absent sur macOS → binaire linux copié par
  erreur. L'installation rappelle dans son message final quel système a été ciblé et comment
  cibler macOS ; une valeur de `HOST_OS` inconnue (ex. `windows`) est refusée avec un message
  qui renvoie vers WSL2.
- **Architecture** : l'image est multi-architecture ; sur un Mac Apple Silicon, l'image arm64
  est utilisée et copie le binaire darwin/arm64. Si l'apprenant force une autre plateforme
  d'image, le message final indique l'architecture installée.
- **Fichier existant** dans le dossier de destination qui n'est pas un binaire TELEMACH (ou un
  lien symbolique) : l'installation ne le remplace pas et explique pourquoi.
- **Essai et variables d'environnement** : dans le conteneur d'essai, les variables du shell de
  l'apprenant ne sont pas visibles ; un fichier Compose qui en dépend peut être signalé comme
  utilisant des variables non définies. Le README le signale et montre comment les transmettre.
- **Tag `latest`** : la documentation utilise toujours un tag de version explicite ; si
  l'apprenant utilise `latest` ou aucun tag, l'image fonctionne, et le README rappelle pourquoi
  un tag explicite est préférable (principe III).
- **Téléchargement par navigateur sous macOS** : un binaire téléchargé par un navigateur peut
  être bloqué par le système ; le README privilégie un téléchargement en ligne de commande et
  explique le message éventuel.
- **Tag `v0.1.0` déjà existant sans Release** : ce tag pointe sur un commit antérieur à la
  chaîne de publication (ni workflow ni configuration de publication) ; il ne peut donc pas
  être publié. Il reste dans l'historique comme tag sans Release, et la première Release est
  `v0.2.0`, posée après la fusion de cette feature. Le déclenchement manuel (FR-001) sert aux
  tags futurs dont une publication automatique aurait échoué.
- **Pré-versions** (`vX.Y.Z-rc.N`) : publiées comme pré-versions, sans déplacer le tag
  `latest` de l'image ni la « dernière version » de la page des Releases.

## Requirements *(mandatory)*

### Functional Requirements

**Publication des binaires (GitHub Releases)**

- **FR-001**: La publication MUST se déclencher à chaque push d'un tag au format `vX.Y.Z`
  (versionnement sémantique, pré-versions comprises) et pouvoir être relancée manuellement pour
  un tag existant.
- **FR-002**: La publication MUST refuser un tag qui ne pointe pas sur un commit de `main`, et
  MUST NOT publier si les gates obligatoires (lint, tests, vulnérabilités, secrets) échouent sur
  ce commit.
- **FR-003**: Chaque Release MUST contenir un binaire `docker-telemach` pour linux/amd64,
  linux/arm64, darwin/amd64 et darwin/arm64, dont `--version` affiche exactement la version du
  tag (sans le `v`).
- **FR-004**: Chaque Release MUST contenir un fichier de sommes SHA-256 couvrant tous les
  binaires, un SBOM, une signature sans clé gérée par le mainteneur (keyless, Sigstore) et une
  attestation de provenance (SLSA) liant chaque binaire au commit et au workflow qui l'ont
  produit.
- **FR-005**: Les binaires MUST être reproductibles à partir du tag (même source, même
  chaîne d'outils épinglée) et ne dépendre d'aucune bibliothèque système.
- **FR-006**: Une Release existante MUST NOT être écrasée ; une nouvelle exécution sur un tag
  déjà publié MUST échouer sans modifier les artefacts.
- **FR-007**: Les notes de Release MUST être rédigées en français et lister les changements
  depuis la version précédente, avec les commandes d'installation et de vérification.

**Image Docker Hub `telemachlearning/telemach`**

- **FR-008**: Chaque version MUST publier l'image sous les tags `X.Y.Z` et `X.Y` ; seule une
  version stable (hors pré-version) met à jour `latest`. L'image MUST être disponible pour
  linux/amd64 et linux/arm64.
- **FR-009**: L'image MUST contenir les binaires issus de la même Release (mêmes sommes
  SHA-256) : ceux de son architecture pour linux et pour darwin.
- **FR-010**: L'image MUST être minimale (aucun shell ni gestionnaire de paquets), s'exécuter
  avec un utilisateur non-root, partir d'une base épinglée par digest et ne contenir aucun
  secret.
- **FR-011**: L'image MUST être signée (keyless) et accompagnée d'un SBOM et d'une attestation
  de provenance vérifiables.
- **FR-012**: Lancée avec `check`, l'image MUST produire la même vérification qu'un plugin
  installé, sur un dossier monté en lecture seule, sans écrire nulle part.
- **FR-013**: Lancée avec `install`, l'image MUST copier dans le dossier monté de destination
  le binaire correspondant au système demandé (`HOST_OS`, linux par défaut ; darwin) et à
  l'architecture de l'image, avec les droits d'exécution, puis afficher en français : version
  installée (et version remplacée le cas échéant), système et architecture ciblés, prochaine
  étape (`shell-init`).
- **FR-014**: `install` MUST échouer avec une explication pédagogique en français, sans rien
  modifier, quand : le dossier de destination n'est pas monté, n'est pas accessible en
  écriture, appartient à root alors que l'installation ne s'exécute pas en root (dossier créé
  par Docker), contient un fichier homonyme qui n'est pas un
  binaire TELEMACH, ou quand `HOST_OS` a une valeur non prise en charge.
- **FR-015**: Lancée sans argument ou avec `--help`, l'image MUST afficher en français ses deux
  usages (essai et installation) avec les commandes complètes à copier.

**Gates et sécurité de la chaîne de publication**

- **FR-016**: Chaque pull request MUST construire l'image et l'analyser (vulnérabilités connues
  et bonnes pratiques du Dockerfile) ; une vulnérabilité critique ou haute MUST bloquer la
  fusion (principe IV), et cette gate MUST devenir un check obligatoire de `main`.
- **FR-017**: Le jeton Docker Hub MUST être créé et enregistré par le mainteneur lui-même,
  limité au dépôt `telemach` (lecture et écriture), et accessible uniquement depuis un
  environnement GitHub protégé « release » restreint aux tags `v*`. Il MUST NOT apparaître dans
  le dépôt, les journaux ni les artefacts.
- **FR-018**: Les workflows de publication MUST respecter le principe IV : actions épinglées par
  SHA complet, permissions minimales déclarées par job (écriture seulement là où elle est
  nécessaire), aucune exécution de code non fiable.
- **FR-019**: Tous les outils et images de base ajoutés MUST être épinglés sur la dernière
  version stable vérifiée au moment de l'ajout (règle globale et LL-004), et les outils compilés
  avec Go MUST l'être avec la chaîne d'outils du projet (LL-001).

**Documentation**

- **FR-020**: Le README MUST présenter, en français et dans cet ordre : installation par la
  Release (avec vérification de somme et de signature), installation par l'image, essai sans
  installation, puis compilation depuis les sources (procédure actuelle).
- **FR-021**: Chaque commande documentée MUST être exécutable telle quelle et utiliser un tag
  de version explicite ; les procédures documentées MUST être testées de bout en bout sans
  raccourci d'environnement (LL-006).
- **FR-022**: `CONTRIBUTING.md` MUST décrire la procédure de publication (poser et pousser un
  tag, relance manuelle, que faire en cas d'échec) et les prérequis côté dépôt et Docker Hub
  (environnement « release », secret, réglages), conformément à LL-005.

### Key Entities

- **Version** : un tag `vX.Y.Z` sur `main` ; détermine la Release, les tags de l'image et le
  texte de `--version`.
- **Release** : page de publication d'une version ; contient les binaires, les sommes, le SBOM,
  les signatures, l'attestation de provenance et les notes en français.
- **Binaire de plateforme** : `docker-telemach` pour un couple système/architecture ; identifié
  par sa somme SHA-256, identique dans la Release et dans l'image.
- **Image** : `telemachlearning/telemach`, multi-architecture, porteuse des binaires et des deux
  usages (`check`, `install`) ; identifiée par son digest, signée.
- **Environnement « release »** : contexte protégé qui seul donne accès au secret Docker Hub.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Sur une machine qui n'a que Docker, un débutant installe le plugin depuis la
  Release, vérification comprise, en moins de 5 minutes en suivant uniquement le README.
- **SC-002**: L'installation par l'image tient en une seule commande à copier (plus la création
  du dossier) et dure moins de 1 minute une fois l'image téléchargée.
- **SC-003**: 100 % des binaires et des images publiés sont vérifiables (somme, signature,
  provenance) avec les commandes du README, et toute altération est détectée.
- **SC-004**: Une version est disponible (Release et image) moins de 30 minutes après le push de
  son tag, sans autre action du mainteneur.
- **SC-005**: Aucune image n'est publiée avec une vulnérabilité connue critique ou haute.
- **SC-006**: Le téléchargement de l'image représente moins de 15 Mo par architecture
  (téléchargement rapide en salle de TP, y compris quand vingt postes le font en même temps).
- **SC-007**: La procédure d'installation est validée sur les quatre plateformes cibles avant la
  première Release ; `docker telemach --version` y affiche la version publiée.

## Assumptions

- Public visé : Linux (dont WSL2) et macOS, avec Docker Engine ou Docker Desktop ; Windows
  natif, Homebrew et les paquets deb/rpm sont hors périmètre (Homebrew prévu plus tard).
- La version provient uniquement du tag git, injectée à la compilation comme le fait déjà le
  job `build` de la CI ; aucun numéro n'est écrit en dur.
- La première Release sera `v0.2.0`, tag posé sur `main` après la fusion de cette feature.
  Le tag `v0.1.0` (antérieur à la chaîne de publication) n'est ni supprimé ni déplacé et reste
  sans Release.
- Le mainteneur crée lui-même le compte, le dépôt Docker Hub `telemachlearning/telemach`, le
  jeton et le secret GitHub ; Claude Code ne manipule jamais ce jeton.
- L'environnement « release » est restreint aux tags `v*` ; le push du tag par le mainteneur
  vaut décision de publier (pas d'approbation supplémentaire, cohérent avec le workflow de PR
  du projet).
- Les binaires macOS ne sont ni signés ni notarisés par Apple (compte développeur payant) ;
  l'installation en ligne de commande ou par l'image évite le blocage lié au téléchargement par
  navigateur.
- La description de la page Docker Hub est rédigée et mise à jour manuellement par le
  mainteneur à partir d'un texte fourni dans le dépôt.
- Les termes Docker Hub, GitHub Releases, SBOM, Sigstore et SLSA relèvent ici du besoin
  exprimé (où et comment l'apprenant obtient et vérifie le plugin), pas d'un choix
  d'implémentation.
