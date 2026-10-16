# Contrat CLI : `docker telemach`

**Feature** : `001-docker-error-guidance` | **Date** : 2026-10-14

Le binaire `docker-telemach` est appelé soit par la CLI Docker (`docker telemach <cmd>`), soit
directement par l'intégration shell (`docker-telemach <cmd>`). Tous les messages destinés à
l'apprenant sont en français ; les noms de commandes et d'options sont en anglais.

## Options globales

| Option | Valeurs | Défaut | Effet |
|--------|---------|--------|-------|
| `--color` | `auto`, `always`, `never` | `auto` | `auto` : couleur seulement si la sortie est un terminal, sauf si `NO_COLOR` est défini. `CLICOLOR_FORCE=1` équivaut à `always`. |
| `--width` | entier ≥ 40 | largeur du terminal, sinon 80 | Largeur de mise en page |

Sortie : l'explication est écrite sur **stderr** (pour ne pas polluer le stdout d'un pipeline).
Seul `shell-init` écrit sur stdout.

---

## `docker-cli-plugin-metadata` (protocole Docker CLI)

Sous-commande cachée, appelée par la CLI Docker pour découvrir le plugin.

```json
{"SchemaVersion":"0.1.0","Vendor":"TELEMACH","Version":"<version>","ShortDescription":"Explique en français les erreurs Docker et Docker Compose"}
```

Code de sortie `0`. Aucune autre sortie.

---

## `explain` — expliquer une erreur capturée (FR-001 à FR-010)

```text
docker-telemach explain --exit-code <n> [--stderr-file <chemin>] [--interactive] [--docker-version <x.y.z>] -- <args docker…>
```

| Option | Obligatoire | Description |
|--------|-------------|-------------|
| `--exit-code` | oui | Code de sortie de la commande Docker (1 à 255) |
| `--stderr-file` | non | Fichier contenant stderr. Lu une fois (1 Mio max.), jamais modifié ni supprimé par `explain` (la fonction shell s'en charge) |
| `--interactive` | non | La commande était interactive : stderr non capturé, explication fondée sur le code de sortie |
| `--docker-version` | non | Version du client Docker (FR-001b), mentionnée dans le bloc « Erreur non reconnue » ; ignorée si mal formée |
| `-- <args>` | oui | Arguments d'origine, sans `docker` |

Comportement :
- reconnue → explication complète (bloc résumé, concept, cause, corrections, anti-pattern
  éventuel, message d'origine) ;
- famille `compose` → analyse des fichiers Compose concernés, puis problèmes avec leurs
  extraits colorés ;
- non reconnue → bloc « Erreur non reconnue » : reformulation prudente et lien vers la
  documentation, **sans cause inventée** (FR-008) ;
- même empreinte depuis moins de 10 min → rendu court, avec renvoi à `docker telemach last`
  (explication complète) et à `deepen`.

Codes de sortie de `explain` : `0` dans tous les cas où une explication (même « non
reconnue ») a été produite ; `2` en cas d'usage invalide. `explain` ne doit **jamais** faire
échouer la fonction shell : elle ignore son code de sortie.

Exemple de rendu (`--color=never`) :

```text
✖ Le port 8080 de votre machine est déjà utilisé.

📘 Concept — port publishing
   « -p HÔTE:CONTENEUR » relie un port de votre machine à un port du conteneur.
   Un port de la machine ne peut être relié qu'une seule fois.

🔎 Cause probable
   Un autre conteneur ou programme écoute déjà sur le port 8080.

💡 Corrections
   1. Voir qui utilise le port :  docker ps --filter publish=8080
   2. Utiliser un autre port :    docker run -p 8081:80 nginx

── Message Docker d'origine ──
docker: Error response from daemon: ... Bind for 0.0.0.0:8080 failed: port is already allocated.

➜ Pour aller plus loin : docker telemach deepen
```

---

## `check` — vérifier un fichier Compose avant exécution (FR-014)

```text
docker telemach check [-f <fichier>]... [<répertoire>]
```

- Sans `-f` : recherche de `compose.yaml`, `compose.yml`, `docker-compose.yaml`,
  `docker-compose.yml` dans le répertoire (par défaut `.`), dans cet ordre, comme Docker
  Compose.
- Plusieurs `-f` : fusion dans l'ordre, chaque problème indique son fichier d'origine.
- Mêmes contrôles qu'après une erreur (FR-012), plus le signalement des secrets en clair.
- Aucun problème → message explicite « ✔ Aucun problème détecté dans compose.yaml ».

Codes de sortie : `0` aucun problème ; `1` au moins un problème ; `2` usage invalide ou
fichier introuvable (avec explication pédagogique de la recherche des fichiers par défaut).

Lecture seule stricte : aucun fichier n'est créé ni modifié (FR-018).

---

## `last` — réafficher l'explication complète de la dernière erreur (FR-009a)

```text
docker telemach last
```

- Lit `LastDiagnosis` et affiche exactement le même rendu complet que `explain` (jamais la
  version courte) ; pour une erreur Compose, les fichiers sont réanalysés dans leur état actuel.
- Aucune erreur récente → message expliquant comment en obtenir une. Code `0`.

---

## `deepen` — approfondir le dernier diagnostic (US3, FR-009)

```text
docker telemach deepen [--exercise]
```

- Lit `LastDiagnosis` et affiche `Deepen.Explanation` et `Deepen.Example` (coloration
  syntaxique), puis `Deepen.Exercise` avec `--exercise`.
- Aucun diagnostic récent → message expliquant comment en obtenir un. Code `0`.

---

## `lint` — ligne de commande (feature 003)

Vérification d'une ligne `docker run|create|exec`, à la main ou après une commande réussie
(`--after`) : voir `specs/003-command-line-analysis/contracts/cli-lint.md`.

---

## `shell-init` — intégration shell (R2)

```text
docker telemach shell-init bash|zsh
```

Écrit sur stdout le code shell à évaluer : `eval "$(docker telemach shell-init zsh)"`.

Garanties du code généré (vérifiées par les tests) :
1. définit uniquement une fonction `docker` et des variables préfixées `__telemach_` ;
2. stdin et stdout de `docker` ne sont jamais redirigés ;
3. le **code de sortie d'origine** est toujours renvoyé ;
4. le fichier temporaire est créé en `0600` et supprimé dans tous les cas (y compris Ctrl-C) ;
5. `TELEMACH_DISABLE=1` ou l'absence de `docker-telemach` → comportement identique à `docker` ;
6. commandes interactives → pas de redirection de stderr, passage de `--interactive` ;
7. la version du client est lue **une seule fois**, à l'évaluation du script, par
   `command docker --version` (lecture seule, sans contact avec le démon), stockée dans
   `__telemach_docker_version` et transmise par `--docker-version`. Aucune autre commande
   n'est exécutée.

Garanties 8 à 10 (appel de `lint` après une commande `run|create|exec` réussie,
`TELEMACH_LINT=0`) : voir `specs/003-command-line-analysis/contracts/cli-lint.md`.

Shell non pris en charge → message d'erreur en français, code `2`.

---

## Sécurité transverse (FR-015 à FR-018)

- Aucune sous-commande n'exécute de commande Docker ni ne modifie de fichier de l'apprenant
  (FR-015). Seule exception, hors du binaire : `docker --version`, lu par le script de
  `shell-init` (garantie 7).
  Les corrections sont **affichées**, jamais exécutées.
- Toute chaîne affichée ou persistée passe par le masquage (R7) et la neutralisation des
  séquences ANSI (R6).
- Aucun accès réseau.
