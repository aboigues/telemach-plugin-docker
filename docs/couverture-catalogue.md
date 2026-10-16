# Couverture du catalogue par rapport aux erreurs de Docker

Étude du 2026-10-16, catalogue de la v0.3.0 (63 entrées, 52 motifs). Question posée : quelle
part des erreurs Docker TELEMACH sait-il expliquer ? Deux mesures y répondent, car un seul
pourcentage serait trompeur :

1. **Mesure statique** : tous les messages d'erreur du code source de Docker, rapprochés des
   motifs du catalogue. Elle donne l'étendue et sert à repérer les familles d'erreurs.
2. **Mesure pondérée** : les questions Stack Overflow les plus votées qui portent sur une erreur
   Docker. Elle dit si TELEMACH explique les erreurs **réellement rencontrées**.

TELEMACH ne collecte aucune donnée d'usage (principe III) : ces mesures n'utilisent que des
sources publiques.

## En bref

| Mesure | Base | Couvert |
| --- | --- | --- |
| Statique, tous messages | 7 303 messages | 29 (0,4 %) |
| Statique, surface utilisateur¹ | 1 940 messages | 16 (0,8 %) |
| Pondérée, toutes questions | 100 questions | **64 %** (78 % pondéré par les votes) |
| Pondérée, questions depuis 2024 | 30 questions | **43 %** (22 % pondéré par les votes) |

¹ Messages de la CLI, du client, de Compose et des Dockerfile, plus ceux du démon classés comme
causés par la requête (paramètre invalide, objet introuvable, conflit…).

- **Le taux statique ne mesure pas l'utilité.** La plupart des messages sont internes ou rares
  (pilotes de stockage, Swarm, plugins réseau, cache de BuildKit…). Un apprenant n'en verra
  presque aucun.
- **Les erreurs classiques sont bien couvertes** : 79 % des questions les plus votées de tous
  les temps, et 68 % pour Compose.
- **Les erreurs récentes le sont mal** : 43 % depuis 2024. Il manque notamment les échecs de
  BuildKit sur l'image de base, les assistants d'identifiants, les versions d'API et les
  constructions multi-plateformes.

## Lacunes à combler, par priorité

Les votes donnent un ordre de grandeur de l'audience, pas une fréquence exacte. Chaque message
actuel ci-dessous a été retrouvé dans le code source (fichier indiqué) ou reproduit sur
Docker 29.8.0.

| Priorité | Famille | Message actuel | Questions (votes) |
| --- | --- | --- | --- |
| 1 | `-it` sans terminal | `cannot attach stdin to a TTY-enabled container because stdin is not a terminal` (cli `streams/in.go`, reproduit) | 1 (1 013) |
| 1 | attacher un conteneur arrêté | `cannot attach to a stopped container, start it first` (cli `container/attach.go`) | 1 (651) |
| 1 | registre injoignable (proxy, IPv6) | `dial tcp …: connect: …` (réseau de Go, pas de message propre à Docker) | 2 (408) |
| 1 | volume mal écrit | `invalid spec: :/data: empty section between colons` (cli `volumespec.go`, reproduit) | 1 (399) |
| 1 | plages d'adresses épuisées | `all predefined address pools have been fully subnetted` (moby `ipamapi/contract.go`) | 1 (355) |
| 2 | registre sans identifiants (ECR) | `no basic auth credentials` (distribution) | 1 (296) |
| 2 | image sans commande | `no command specified` (moby `daemon/create.go`) | 1 (266) |
| 2 | conteneur arrêté avec le code 0 | aucun message : le conteneur « s'arrête tout de suite » | 2 (217) |
| 2 | arrêt impossible | `cannot stop container: …: tried to kill container, but did not receive an exit event` (moby `stop.go`, `kill.go`) | 2 (206) |
| 2 | image de base introuvable au build | `failed to resolve source metadata for …` (BuildKit `imageresolver.go`) | 3 (152) |
| 2 | assistant d'identifiants absent | `error getting credentials - err: …` (docker-credential-helpers) | 2 (151) |
| 2 | système de l'image | `image operating system "windows" cannot be used on this platform` (moby `distribution/config.go`) | 1 (151) |
| 2 | Compose : rien à démarrer | `service "web" has no container to start` (compose `service_containers.go`) | 1 (154) |
| 3 | version d'API | `client version 1.42 is too old. Minimum supported API version is …` (moby `middleware/version.go`) | 2 (102) |
| 3 | autres | lien hérité, `OCI runtime create failed`, contexte introuvable, pilote multi-plateforme, `unknown flag`, analyse du Dockerfile, options incompatibles, volume Compose sans cible… | 14 (115 au plus par famille) |

**Cas à trancher avant d'écrire une entrée** :

- **Code 0** : `docker telemach ps` ignore volontairement les conteneurs arrêtés sans erreur
  (feature 004). Expliquer pourquoi un conteneur « s'arrête tout de suite » sans erreur, faute
  de processus au premier plan, demanderait d'élargir ce choix.
- **Réseau vers le registre** : le message vient de Go, pas de Docker. Il faudrait un motif sur
  `dial tcp` combiné au contexte (`pull`, `build`, `login`).

## Défauts du catalogue révélés par l'étude

1. **`network.not-found` est trop large.** Le motif `network (?P<network>\S+) not found`
   reconnaît aussi `failed to enter network namespace: not found`, qui n'a rien à voir.
   23 messages du démon sont concernés. À resserrer, par exemple en excluant les deux-points
   avant `not found`, avec un cas adversarial (LL-010).
2. **`image.in-use` ignore une variante encore produite.** Avec le stockage d'images
   containerd, le message est `conflict: unable to delete x (must be forced) - container … is
   using its referenced image …` : reconnu, vérifié sur Docker 29.8.0. Avec le stockage
   d'images classique (`graphdriver`), il devient `conflict: unable to remove repository
   reference "x" (must force) - container …` : non reconnu (moby `images/image_delete.go`). Une
   question à 285 votes porte sur cette variante.
3. **Anciennes formulations.** Certaines formulations d'avant BuildKit 0.13 ou Docker 20 ne
   sont pas reconnues, par exemple `failed to compute cache key: "/x" not found: not found` ou
   `Is the docker daemon running on this host?`. Docker 29 ne les produit plus : à ne pas
   traiter.

## Mesure statique : méthode et résultats

**Sources épinglées** : docker/cli `v29.9.1`, moby `docker-v29.9.1` (démon, client et BuildKit
embarqué), docker/compose `v5.6.0` et compose-go `v2.16.1`. Ce sont les dernières versions
stables au 2026-10-16. Le poste de test utilise Docker 29.8.0.

**Extraction** (`scripts/couverture/cov`, par l'arbre syntaxique Go, sans compiler les sources) :

- chaînes littérales passées à `errors.New`, `fmt.Errorf`, `errors.Wrap(f)`,
  `types.*Errorf` (libnetwork) et `status.Errorf` ;
- `fmt.Sprintf` et chaînes retournées dans les méthodes `Error()` ;
- constantes de type erreur et champs `message:` ;
- concaténations : la partie variable devient `%v`.

Les tests, `testdata`, l'intégration et l'outillage sont exclus. Un message est reconnu si un
motif du catalogue correspond à l'une de ses variantes : chaque verbe de format est remplacé
par une valeur plausible (`nginx`, `0.0.0.0:8080`…), et `%w` par `not found`, la sentinelle la
plus courante.

| Source | Messages | Reconnus |
| --- | ---: | ---: |
| CLI (`docker/cli`) | 744 | 0 |
| Dépendances de la CLI (cobra, pflag, reference, credential-helpers) | 71 | 2 |
| Client de l'API (`moby/client`) | 66 | 5 |
| Démon, erreurs causées par la requête | 371 | 6 |
| Démon, autres erreurs | 2 931 | 12 |
| BuildKit, Dockerfile | 181 | 2 |
| BuildKit, autres | 2 432 | 1 |
| Compose | 331 | 0 |
| compose-go (fichier Compose) | 176 | 1 |
| **Total** | **7 303** | **29** |

Dans le catalogue, 14 des 38 entrées d'erreurs de commande et de Compose retrouvent leur
message dans ces sources. Les 24 autres ne sont pas des oublis :

- 5 reposent sur le **code de sortie** (`runtime.exit-126`, `-127`, `-137`, `-143`,
  `runtime.application-error`) ;
- 10 sont des **constats de `docker telemach check`** (tabulation, clé inconnue, secret en
  clair…), produits par TELEMACH lui-même ;
- 3 viennent de **l'extérieur des sources** : le système (`no space left on device`) ou la
  réponse du registre (`denied: requested access…`, `unauthorized: …`) ;
- 6 sont **assemblées à l'exécution** à partir de plusieurs messages, ce qu'une analyse
  statique ne reconstitue pas. Par exemple, `cannot remove container "x": container is
  running: stop the container…` vient de deux appels distincts (`build.*`,
  `container.remove-running`, `volume.in-use`, `image.in-use`).

**Limites** : les messages assemblés à l'exécution sont sous-comptés. Les messages hors des
sources (runc, containerd, système, registres) ne sont pas comptés.

## Mesure pondérée : méthode et résultats

**Échantillon** : API publique de Stack Exchange, relevé du 2026-10-16, trié par votes, en
trois strates. L'API renvoie 100 questions par page, moins celles qu'elle masque. Après
dédoublonnage, il reste 693 questions :

- `top` : 3 pages de questions `docker`, les plus votées de tous les temps (268 reçues) ;
- `recent` : 3 pages de questions `docker` créées depuis le 2024-01-01 (274 reçues) ;
- `compose` : 2 pages de questions `docker-compose`, les plus votées (200 reçues).

**Classement** : 272 questions contiennent une ligne d'erreur dans leur titre ou leurs blocs de
code. Elles ont été relues une à une et classées dans `docs/couverture/stackoverflow.tsv`
(identifiant, strate, votes, verdict) :

- `C` : couverte, l'entrée du catalogue explique la situation ;
- `G` : couverte de façon générique, une étape `RUN` a échoué et la cause est dans
  l'application (12 cas, `build.run-failed`) ;
- `N:<famille>` : erreur Docker non couverte ;
- `O` : situation disparue (Compose v1 en Python, docker-py, Docker Toolbox) ;
- `X` : hors périmètre (erreur de l'application, installation, Docker Desktop ou WSL, autre
  outil).

**Règle pour les anciennes formulations** : une question ancienne est jugée sur le message
qu'afficherait Docker 29 dans la même situation. Par exemple, `Got permission denied while
trying to connect to the Docker daemon socket` correspond aujourd'hui à `permission denied
while trying to connect to the docker API`, qui est couvert.

| Strate | Questions sur une erreur Docker | Couvertes | Pondéré par les votes |
| --- | ---: | ---: | ---: |
| `top` (tous les temps) | 39 | 31 (79 %) | 80 % |
| `recent` (depuis 2024) | 30 | 13 (43 %) | 22 % |
| `compose` | 37 | 25 (68 %) | 78 % |
| **Total** (dédoublonné) | **100** | **64 (64 %)** | **78 %** |

Verdicts : 52 C, 12 G, 36 N, 13 O, 580 X.

**Limites** :

- **Un seul relecteur.** Le classement est à relire : chaque ligne renvoie à
  `https://stackoverflow.com/q/<identifiant>`.
- **Biais des votes.** Les votes favorisent les questions anciennes, d'où l'intérêt de la
  strate `recent`.
- **Préfiltre.** Une question qui cite l'erreur seulement en prose, sans bloc de code, a pu être
  écartée.
- **Situation, pas texte exact.** Un verdict `C` dit que l'entrée explique la situation. Le
  texte exact de la question n'est pas toujours reconnu.

## Reproduire

```bash
scripts/couverture/statique.sh /tmp/couverture          # mesure statique (clone ~250 Mo)
scripts/couverture/stackoverflow.sh /tmp/couverture     # nouvel échantillon à relire
python3 -I scripts/couverture/score.py docs/couverture/stackoverflow.tsv
```

`scripts/couverture/cov` est un module Go à part, hors du plugin : il n'est ni livré ni
compilé avec le reste.

## Suite proposée

Une feature « couverture du catalogue », à lancer avec `/speckit-specify` :

1. Corriger les deux défauts (motif de `network.not-found`, variante de `image.in-use`), avec
   des cas réels et adversariaux.
2. Ajouter les lacunes de priorité 1 et 2, chacune avec une erreur capturée dans
   `testdata/errors/`, selon `docs/catalogue.md`.
3. Trancher les deux cas ouverts (code 0, réseau vers le registre).
4. Refaire la mesure pondérée. L'objectif est à fixer ensemble, par exemple au moins 70 % sur
   la strate `recent`.
