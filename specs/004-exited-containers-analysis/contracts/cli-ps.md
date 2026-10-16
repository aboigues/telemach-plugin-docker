# Contrat CLI : `docker telemach ps` et `watch`

**Feature** : `004-exited-containers-analysis` | **Date** : 2026-10-16

Complète `specs/001-docker-error-guidance/contracts/cli.md` et
`specs/003-command-line-analysis/contracts/cli-lint.md`. Messages en français, sur **stderr**
(comme toutes les explications) ; rien sur stdout.

## Options globales de Docker

`docker --context x telemach …`, `docker -H tcp://… telemach …` : la CLI transmet au plugin les
options globales placées avant `telemach`. Le plugin les retire avant d'analyser sa propre ligne
de commande, et les repasse en tête de chacune de ses lectures (`docker --context x ps …`).
Vaut pour **toutes** les sous-commandes (correctif d'un bug existant, research R3).

## `ps`

```text
docker telemach ps [CONTENEUR]
```

- Sans argument : explique le conteneur en erreur arrêté le plus récemment, puis liste les
  autres (10 au plus).
- `CONTENEUR` : nom ou préfixe d'identifiant ; explique ce conteneur.

Codes de sortie : `0` analyse affichée, ou aucun conteneur en erreur ; `1` démon injoignable,
droits insuffisants, ou conteneur désigné introuvable, en cours d'exécution ou arrêté avec 0
(explication affichée) ; `2` usage invalide.

Conteneurs retenus : arrêtés (`exited`) ou en boucle de redémarrage (`restarting`) avec un code
≠ 0, et conteneurs `created` dont le démarrage a échoué (code ≠ 0 et erreur enregistrée).

Lectures effectuées, et elles seules (research R2) : `ps -a … --format`, `inspect`,
`logs --tail 200`. Chaque lecture a un délai maximal de 5 s.

### Rendu (exemple)

```text
✖ Le conteneur « eager_pg » (postgres:18-alpine) s'est arrêté avec le code 1, 3 s après son démarrage (il y a 2 min).

── Fin du journal (5 dernières lignes sur 20) ───────────────────────────────
       You must specify POSTGRES_PASSWORD to a non-empty value for the
       superuser. For example, "-e POSTGRES_PASSWORD=password" on "docker run".
       …

📘 Concept — variable d'environnement obligatoire
   Certaines images refusent de démarrer sans une variable de configuration …

🔎 Cause probable
   L'image postgres exige POSTGRES_PASSWORD ; le conteneur a été lancé sans elle.

💡 Corrections
   1. Mettre le mot de passe dans un fichier lisible par toi seul, puis le passer avec --env-file.
      docker run -d --env-file db.env postgres:18-alpine
   2. Ou utiliser un secret monté et la variable POSTGRES_PASSWORD_FILE.

➜ Pour aller plus loin : docker telemach deepen

Autres conteneurs arrêtés en erreur :
   api        code 127   il y a 1 h    commande introuvable dans l'image
   worker     code 137   il y a 3 h    arrêt forcé (docker kill, ou docker stop après délai)
→ Pour en expliquer un : docker telemach ps <nom>
```

Aucun conteneur en erreur :

```text
✔ Aucun conteneur arrêté en erreur.
   Un conteneur lancé avec --rm est supprimé à l'arrêt : il ne peut plus être analysé.
```

## `watch` (sous-commande cachée, appelée par l'intégration shell)

```text
docker-telemach watch --since T0 --shell-pid PID -- <arguments de docker run…>
```

- Sort aussitôt, sans rien lire, si la ligne n'est pas un `run` (ou `container run`) en
  arrière-plan (`-d`, `--detach`), si `TELEMACH_WATCH=0` ou `TELEMACH_DISABLE=1`.
- Sinon attend 3 s, lit `events --since T0 --until <maintenant>` (types `create` et `die`), et,
  si le conteneur lancé (research R5) s'est arrêté avec un code ≠ 0, écrit **un fichier par
  signal**, `pending-PID-<horodatage>`, de façon atomique (fichier temporaire puis renommage) —
  seulement si le shell `PID` est vivant.
- `T0` : secondes depuis l'époque, décimales séparées par `.` ou `,`.
- Code de sortie toujours `0` ; aucune sortie (stdin, stdout et stderr reliés à `/dev/null` par
  le script).

Signal affiché à l'invite suivante :

```text
⚠ Le conteneur « eager_pg » s'est arrêté avec le code 1, 3 s après son lancement : docker telemach ps pour comprendre pourquoi.
```

Conteneur lancé avec `--rm` (supprimé depuis) :

```text
⚠ Le conteneur « eager_pg » s'est arrêté avec le code 1, 3 s après son lancement, et a été supprimé (--rm) : relance-le sans --rm pour lire son journal.
```

## Intégration shell (`shell-init`) — garanties ajoutées

S'ajoutent aux garanties 1 à 10 (001 et 003) :

11. avant chaque commande `docker`, le script relève `t0` (`$EPOCHREALTIME`) ; après un `run`
    ou `container run` **réussi** dans un shell **interactif**, si `TELEMACH_WATCH` ≠ `0`, il
    lance `watch` en arrière-plan, stdin, stdout et stderr redirigés vers `/dev/null` ; aucun
    message de tâche, aucune attente ;
12. un crochet d'invite (`precmd` en zsh via `add-zsh-hook`, `PROMPT_COMMAND` en bash, forme
    chaîne ou tableau) affiche sur stderr puis supprime les fichiers `pending-$$-*` ; sans
    fichier, il ne lance aucun processus ; il préserve le code de retour de la dernière
    commande ; le chargement répété de `shell-init` n'ajoute pas le crochet deux fois ;
13. au chargement (`shell-init`), les fichiers `pending-*` de shells qui ne sont plus vivants
    sont supprimés ;
14. `TELEMACH_WATCH=0` supprime le lancement de `watch` ; `TELEMACH_LINT=0` ne le supprime
    pas (désactivations séparées, FR-016) ; `TELEMACH_DISABLE=1` reste prioritaire ; un shell non
    interactif ne lance jamais `watch`.

## `last` et `deepen`

- `last` réaffiche la dernière analyse de `ps` (en-tête « analyse enregistrée le … :
  conteneur <nom> »), extrait de journal compris, **sans** interroger le démon.
- `deepen` approfondit l'entrée retenue.
- Le signal de `watch` n'est pas un diagnostic : il ne remplace pas `last`.
