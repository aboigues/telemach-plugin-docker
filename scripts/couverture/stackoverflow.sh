#!/usr/bin/env bash
# Mesure 2 de docs/couverture-catalogue.md : télécharge les questions Stack Overflow les plus
# votées (API publique, sans clé ni compte), puis prépare leur relecture manuelle.
#
# Usage : scripts/couverture/stackoverflow.sh <dossier de travail>
# Produit so/questions.tsv et so/review.txt (questions ayant une ligne d'erreur, avec les
# entrées du catalogue reconnues). Le classement relu va dans docs/couverture/stackoverflow.tsv ;
# le score se calcule avec scripts/couverture/score.py.
set -euo pipefail

[ $# -eq 1 ] || { echo "usage : $0 <dossier de travail>" >&2; exit 2; }
WORK=$(mkdir -p "$1" && cd "$1" && pwd)
HERE=$(cd "$(dirname "$0")" && pwd)
CAT=$(cd "$HERE/../../internal/catalog/data" && pwd)
SO=$WORK/so
mkdir -p "$SO"

API=https://api.stackexchange.com/2.3/questions
fetch() { # strate étiquette page [date de début, secondes]
  local url="$API?order=desc&sort=votes&tagged=$2&site=stackoverflow&pagesize=100&page=$3&filter=withbody"
  [ -n "${4:-}" ] && url="$url&fromdate=$4"
  curl -sSf --compressed "$url" -o "$SO/$1-$3.json"
  local n
  n=$(python3 -I -c 'import json,sys; print(len(json.load(open(sys.argv[1])).get("items", [])))' "$SO/$1-$3.json")
  [ -n "$n" ] && [ "$n" -gt 0 ] || { echo "page vide : $1 $3" >&2; exit 1; }
  echo "$1 page $3 : $n questions"
  sleep 1
}
for p in 1 2 3; do fetch top docker "$p"; done
for p in 1 2 3; do fetch recent docker "$p" 1704067200; done # depuis le 2024-01-01
for p in 1 2; do fetch compose docker-compose "$p"; done

python3 -I "$HERE/so-extract.py" "$SO"
(cd "$HERE/cov" && go build -o "$WORK/cov" .)
"$WORK/cov" text "$CAT" < "$SO/texts.txt" | awk -F'\t' '{print $NF}' > "$SO/match-col.txt"
paste "$SO/questions.tsv" "$SO/match-col.txt" |
  awk -F'\t' '$5 != "-" {printf "%s|%s|%s || %s || %s\n", $1, $3, substr($4, 1, 90), substr($5, 1, 230), $6}' > "$SO/review.txt"
echo "à relire : $(wc -l < "$SO/review.txt") questions ($SO/review.txt)"
