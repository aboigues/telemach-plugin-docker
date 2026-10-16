# Enrichir le catalogue d'erreurs

Le catalogue est le cœur pédagogique de TELEMACH : chaque entrée associe une erreur Docker à
un concept, une cause, des corrections et un approfondissement, **en français**. Il se trouve
dans `internal/catalog/data/`, un fichier YAML par catégorie.

## Ajouter une erreur en 4 étapes

1. **Capturer l'erreur réelle.** Ajoute un cas à `testdata/errors/capture.sh`, puis lance le
   script : il crée `<cas>.stderr.txt`, `<cas>.args` et `<cas>.exit`. N'invente pas de
   message : seul un texte réellement produit par Docker est accepté (une erreur non
   reproductible peut être retranscrite depuis le code de Docker Engine, en le signalant dans
   le script).
2. **Écrire le résultat attendu** dans `<cas>.expected.yaml` :

   ```yaml
   entry_id: "run.port-already-allocated"
   origin: docker            # docker, application ou unknown
   captures:
     port: "18080"
   ```

3. **Ajouter l'entrée** dans le fichier de sa catégorie (voir le format ci-dessous).
4. **Lancer les tests** : `go test ./internal/catalog/ ./internal/diagnose/ ./internal/render/`.

## Format d'une entrée

```yaml
- id: run.port-already-allocated          # unique, format ^[a-z]+(\.[a-z0-9-]+)+$
  category: network                       # network, image, container, volume, daemon,
                                          # permission, build, compose, runtime
  applies: [run, create, start, compose]  # sous-commandes concernées (vide = toutes)
  exit_codes: []                          # codes de sortie concernés (facultatif)
  patterns:                               # expressions régulières RE2, groupes nommés
    - 'Bind for [0-9.:]+:(?P<port>\d+) failed: port is already allocated'
  concept:
    name: port publishing                 # terme Docker d'origine
    definition: >-                        # une ou deux phrases en français
      …
  summary: Le port {{.port}} de ta machine est déjà utilisé.   # 160 caractères maximum
  cause: >-
    …
  fixes:                                  # au moins une
    - description: Voir quel conteneur publie déjà ce port.
      command: docker ps --filter publish={{.port}}
      safe: true                          # false = solution dangereuse (anti-pattern requis)
  anti_pattern:                           # obligatoire si une correction n'est pas sûre
    description: >-
      …
  deepen:
    explanation: >-                       # le « pourquoi » du concept
      …
    example: …                            # exécutable tel quel
    exercise: >-                          # court, sans risque, vérifiable
      …
  doc_url: https://docs.docker.com/…      # documentation officielle uniquement
  tests: [run-port-allocated]             # cas du corpus testdata/errors/
```

Les textes peuvent utiliser les valeurs capturées (`{{.port}}`, `{{.image}}`…) et des
conditions (`{{if .suggestion}}…{{end}}`). Pour écrire littéralement `{{`, par exemple dans
un format `docker ps --format`, utilise `{{"{{"}}`.

Deux valeurs de `fallback` existent pour des entrées sans motif :

- `application` : repli quand le code de sortie montre que l'erreur vient du programme du
  conteneur ;
- `compose-issue` : pédagogie d'un problème détecté par l'analyseur Compose
  (`compose.<type>`) ; ses `tests` désignent alors des dossiers de `testdata/compose/`.
- `cmdline` : piège de la ligne de commande (`cmdline.<piège>`), une écriture que Docker
  accepte mais qui ne fait pas ce qui est voulu ; voir ci-dessous.

## Ajouter un piège de la ligne de commande

Les pièges sont repérés par `internal/lint`, sans rien exécuter, sur `docker run`, `create` et
`exec` (même quand la commande réussit, ou à la main avec `docker telemach lint -- …`).

1. **Écrire les cas** dans `testdata/cmdline/` (format dans son `README.md`) : des lignes
   fautives dans un fichier `*-faulty.tsv` et des lignes **correctes voisines** dans un
   `*-legit.tsv`. En cas de doute, le piège doit se taire : une ligne correcte signalée est
   un défaut.
2. **Écrire la règle** dans `internal/lint/pitfalls.go` (une fonction enregistrée dans
   `init`) : elle lit les options placées avant l'image, renvoie des captures nettoyées et
   masquées et une empreinte qui ne contient **jamais** de valeur saisie.
3. **Ajouter l'entrée** dans `internal/catalog/data/cmdline.yaml`, avec `fallback: cmdline`,
   `severity: warning` (⚠) ou `severity: info` (ℹ), sans `patterns`, et `tests` qui nomme
   le fichier de corpus (sans `.tsv`). Captures disponibles en plus de celles de la règle :
   `{{.cleanup}}` et `{{.cleanup_command}}`, renseignées seulement après exécution
   (conseil de suppression du conteneur déjà créé).
4. **Lancer les tests** : `go test ./internal/lint/ ./internal/catalog/ ./cmd/...`.

## Règles vérifiées automatiquement

- **Langue** : `summary`, `cause` et `concept.definition` doivent être en français.
- **Honnêteté** : une erreur inconnue ne reçoit jamais de cause (corpus de référence).
- **Sécurité des exemples** (`internal/catalog/safety_test.go`, constitution v1.1.0) :
  - interdit dans un exemple, un exercice ou une correction sûre : `--privileged`,
    `chmod 666`, l'utilisateur root (`--user 0`, `-u root`), `--network host`, un secret
    écrit en clair ;
  - chaque image doit figurer dans `internal/catalog/data/example-images.txt`, avec sa
    justification ; une image qui tourne en root par défaut (comme `alpine`) doit toujours
    être utilisée avec `--user` (ou `USER` dans un Dockerfile) ;
  - une image sans tag explicite, ou en `latest`, produit un **avertissement non bloquant**.
- **Versions** : avant d'écrire un tag d'image, vérifie sur Docker Hub la dernière version
  stable (voir `LESSON-LEARNED.md`, LL-004).
