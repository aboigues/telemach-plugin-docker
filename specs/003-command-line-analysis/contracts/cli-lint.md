# Contrat — Analyse de la ligne de commande

Complète `specs/001-docker-error-guidance/contracts/cli.md`.

## `lint` — analyser une ligne `docker run|create|exec`

```text
docker telemach lint [--after] [--color auto|always|never] [--width N] -- <arguments docker…>
```

- Les arguments sont ceux qui suivent `docker` (exemple :
  `docker telemach lint -- run -d nginx -p 8090:80`).
- **Sans `--after`** : vérification à la main, **sans exécuter** la commande. Formulations
  au conditionnel (« le port ne serait pas publié »).
- **Avec `--after`** (appel par l'intégration shell après une commande réussie) :
  formulations au passé (« le conteneur a démarré, mais le port 8090 n'est pas publié ») et
  règle de répétition (`hints.json`).
- Sortie sur **stderr** uniquement ; rien sur stdout.
- Sous-commande autre que `run`, `create`, `exec` (ou `container …`) : aucune sortie, code 0.
- `TELEMACH_LINT=0` : aucune analyse, code 0.

Codes de sortie : `0` aucun piège ou informations seulement ; `1` au moins un avertissement ;
`2` usage invalide. L'intégration shell ignore ce code.

### Rendu (sans couleur) — exemple de l'issue #9, `--after`

```text
⚠ Option Docker placée après l'image : -p 8090:80 a été transmis au conteneur, pas à Docker.
   Le conteneur a démarré, mais le port 8090 de ta machine n'est pas publié.

📘 Concept — ordre des arguments de docker run
   docker run [OPTIONS] IMAGE [COMMANDE] [ARG...] : les options de Docker se placent avant
   l'image ; tout ce qui suit l'image est donné au programme du conteneur.

💡 Correction
   docker rm -f <conteneur>          (arrête le conteneur lancé sans le bon port)
   docker run -d -p 8090:80 nginx

➜ Pour aller plus loin : docker telemach deepen
```

Plusieurs pièges : un seul affichage ; avertissements (⚠) d'abord, informations (ℹ) ensuite,
chacun avec son concept et sa correction.

### Rappel (même piège affiché en complet il y a moins de 10 minutes)

```text
↺ Encore une option après l'image (-p 8090:80) : docker run -d -p 8090:80 nginx — détails : docker telemach last
```

### Information (exemple)

```text
ℹ Image sans tag explicite : nginx désigne nginx:latest, qui change à chaque publication.
   💡 Préfère un tag précis, par exemple nginx:1.30-alpine.
```

## Intégration shell (`shell-init`) — garanties ajoutées

S'ajoutent aux 7 garanties de la 001 :

8. le plugin n'est appelé après une commande **réussie** que si sa sous-commande est `run`,
   `create` ou `exec` (y compris `container run|create|exec`, après les options globales de
   Docker) ; toute autre commande n'entraîne aucun appel supplémentaire ;
9. l'appel `lint --after` a lieu après la fin de la commande (après la session pour une
   commande interactive), sa sortie va sur stderr, son code est ignoré ;
10. `TELEMACH_LINT=0` supprime l'appel de `lint` et l'ajout des pièges dans `explain` ;
    `TELEMACH_DISABLE=1` reste prioritaire.

Les garanties 1 à 7 (stdin/stdout jamais redirigés, code de sortie d'origine, fichier
temporaire sûr, etc.) sont inchangées et retestées sous bash et zsh.

## `explain` — pièges intégrés (FR-010)

Quand la commande échoue (code non nul) et que l'analyse trouve au moins un avertissement :

- erreur reconnue : une section placée **avant** l'explication habituelle (le piège est la
  cause la plus probable, et l'explication générique du message Docker peut alors être
  trompeuse), suivie de « Explication du message d'erreur de Docker : » et de l'explication
  habituelle :

  ```text
  ⚠ Dans ta commande — cause probable
     <résumé du piège>
     📘 <concept> …
     💡 <commande corrigée>
  ```

- erreur non reconnue : le piège devient l'explication principale, le message Docker
  d'origine reste affiché en dessous.

Les informations (ℹ) ne sont pas ajoutées à une explication d'erreur. `TELEMACH_LINT=0` :
`explain` se comporte exactement comme avant.

## `last` et `deepen`

- `last` réaffiche le dernier affichage complet, erreur **ou** avertissement, avec l'en-tête de
  date et de commande (PR #10).
- `deepen` approfondit le premier piège du dernier avertissement (champ `deepen` de son entrée
  de catalogue).

## Sécurité

- Arguments nettoyés (`sanitize`) puis masqués (`redact`) avant tout affichage et toute
  écriture ; `hints.json` ne contient que des empreintes sans valeur.
- Lectures locales autorisées : présence d'une variable d'environnement (jamais sa valeur),
  existence d'un dossier relatif au dossier courant. Aucune exécution, aucun réseau.
