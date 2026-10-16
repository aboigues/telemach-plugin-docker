#!/usr/bin/env bash
# Mesure 1 de docs/couverture-catalogue.md : extrait les messages d'erreur du code source de
# Docker aux versions épinglées, puis les rapproche des motifs du catalogue.
#
# Usage : scripts/couverture/statique.sh <dossier de travail>
# Le dossier reçoit les sources clonées (src/) et les résultats (out/messages.tsv, out/matched.tsv).
set -euo pipefail

CLI_TAG=v29.9.1          # github.com/docker/cli
MOBY_TAG=docker-v29.9.1  # github.com/moby/moby (démon, client, BuildKit embarqué)
COMPOSE_TAG=v5.6.0       # github.com/docker/compose

[ $# -eq 1 ] || { echo "usage : $0 <dossier de travail>" >&2; exit 2; }
WORK=$(mkdir -p "$1" && cd "$1" && pwd)
HERE=$(cd "$(dirname "$0")" && pwd)
CAT=$(cd "$HERE/../../internal/catalog/data" && pwd)
SRC=$WORK/src
OUT=$WORK/out
mkdir -p "$SRC" "$OUT"

clone() { # dépôt tag dossier
  [ -d "$SRC/$3" ] || git -c advice.detachedHead=false clone -q --depth 1 --branch "$2" "https://github.com/$1.git" "$SRC/$3"
}
clone docker/cli "$CLI_TAG" cli
clone moby/moby "$MOBY_TAG" moby
clone docker/compose "$COMPOSE_TAG" compose

# compose-go (validation des fichiers Compose) : version exigée par Compose.
CG_VERSION=$(awk '$1 == "github.com/compose-spec/compose-go/v2" {print $2}' "$SRC/compose/go.mod")
[ -n "$CG_VERSION" ] || { echo "version de compose-go introuvable" >&2; exit 1; }
CG=$(cd "$WORK" && go mod download -json "github.com/compose-spec/compose-go/v2@$CG_VERSION" | awk -F'"' '$2 == "Dir" {print $4}')
[ -n "$CG" ] || { echo "téléchargement de compose-go impossible" >&2; exit 1; }

(cd "$HERE/cov" && go build -o "$WORK/cov" .)
COV=$WORK/cov

{
  "$COV" extract cli "$SRC/cli/cli" "$SRC/cli/cmd" "$SRC/cli/internal" "$SRC/cli/opts"
  "$COV" extract cli-deps "$SRC/cli/vendor/github.com/docker/docker-credential-helpers" \
    "$SRC/cli/vendor/github.com/spf13/cobra" "$SRC/cli/vendor/github.com/spf13/pflag" \
    "$SRC/cli/vendor/github.com/distribution/reference"
  "$COV" extract moby-client "$SRC/moby/client"
  "$COV" extract moby-daemon "$SRC/moby/daemon" "$SRC/moby/api" "$SRC/moby/errdefs" \
    "$SRC/moby/pkg" "$SRC/moby/cmd" "$SRC/moby/internal"
  "$COV" extract buildkit "$SRC/moby/vendor/github.com/moby/buildkit"
  "$COV" extract compose "$SRC/compose/cmd" "$SRC/compose/pkg" "$SRC/compose/internal"
  "$COV" extract compose-go "$CG"
} > "$OUT/messages.tsv"

n=$(wc -l < "$OUT/messages.tsv")
[ -n "$n" ] && [ "$n" -gt 0 ] || { echo "extraction vide" >&2; exit 1; }

"$COV" match "$CAT" < "$OUT/messages.tsv" > "$OUT/matched.tsv"

# Surface « utilisateur » : messages de la CLI et de ses dépendances, du client, de Compose,
# des Dockerfile, et ceux du démon classés comme causés par la requête (errdefs).
# « network %s … : %w » reconnu seulement grâce à l'erreur enveloppée « not found » : motif
# trop large de network.not-found, compté à part.
printf "%-20s %8s %9s\n" source messages reconnus
awk -F'\t' '
  { split($3, a, "@"); s = $1
    if (s == "moby-daemon") s = (a[2] ~ /^(InvalidParameter|NotFound|Conflict|Forbidden|Unauthorized)$/) ? "daemon-utilisateur" : "daemon-autre"
    if (s == "buildkit") s = ($2 ~ /^frontend\/dockerfile/) ? "buildkit-dockerfile" : "buildkit-autre"
    loose = ($5 == "network.not-found" && $4 !~ /not found/)
    t[s]++; if ($5 != "-" && !loose) m[s]++; if (loose) l++ }
  END { for (k in t) printf "%-20s %8d %9d\n", k, t[k], m[k]
        printf "motif trop large (network.not-found) : %d\n", l }' "$OUT/matched.tsv" | sort
