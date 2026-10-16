# Corpus de conteneurs arrêtés en erreur (feature 004)

Chaque cas est un dossier `testdata/exited/<cas>/`, produit par `capture.sh` à partir d'un
vrai conteneur :

- `case.sh` : la commande qui reproduit le cas ;
- `inspect.json` : la sortie de `docker inspect` du conteneur, **variables d'environnement
  réduites à leurs noms avant écriture** (aucune valeur n'est versionnée) ;
- `logs.txt` : la fin du journal (`docker logs --tail 200`) ;
- `expected.yaml` : le résultat attendu du diagnostic :

  ```yaml
  entry_id: exited.missing-required-env
  source: log            # log, state, code ou unknown
  captures:
    variable: POSTGRES_PASSWORD
  ```

Le dossier `events/` contient des captures de `docker events` (une ligne JSON par
événement) pour la vérification après `docker run -d`.

Le test `internal/exited/diagnose_test.go` charge tous les cas (SC-001).
