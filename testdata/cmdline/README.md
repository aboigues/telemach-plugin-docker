# Corpus de lignes de commande (feature 003)

Chaque fichier `*.tsv` contient un cas par ligne, champs séparés par une tabulation :

```text
ATTENDU	VARIABLES	DOSSIERS	ARGUMENTS
```

- `ATTENDU` : identifiants `cmdline.*` attendus, séparés par des virgules, ou `-` pour aucun ;
- `VARIABLES` : variables d'environnement présentes, séparées par des virgules, ou `-` ;
- `DOSSIERS` : dossiers existants dans le dossier courant, séparés par des virgules, ou `-` ;
- `ARGUMENTS` : la ligne après `docker`, découpée comme le ferait le shell (guillemets
  simples et doubles pris en charge).

Les lignes vides et celles qui commencent par `#` sont ignorées. Les fichiers
`*-faulty.tsv` contiennent des lignes fautives (au moins un identifiant attendu), les
fichiers `*-legit.tsv` des lignes correctes (`-`). Le test `internal/lint/corpus_test.go`
les charge tous (SC-001, SC-002).
