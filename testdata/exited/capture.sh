#!/usr/bin/env bash
# Capture de vrais conteneurs arrêtés en erreur pour le corpus de la feature 004 (SC-001).
# Usage : testdata/exited/capture.sh [nom-du-cas…]
# Chaque cas produit <cas>/case.sh, <cas>/inspect.json (champs utiles seulement, variables
# d'environnement réduites à leurs noms) et <cas>/logs.txt ; expected.yaml est écrit à la main.
# Les conteneurs créés sont préfixés « telemach-exited- » et supprimés à la fin.
# Tags vérifiés sur Docker Hub le 2026-10-16 (LL-004) : postgres:18-alpine, mariadb:13.0
# (13.0.2), alpine:3.24, nginxinc/nginx-unprivileged:1.30-alpine.
set -uo pipefail

cd "$(dirname "$0")"
P=telemach-exited
U=(--user 1000:1000)

# Nettoyage installé avant toute création (LL-003).
cleanup() {
	docker ps -aq --filter "name=^$P-" | xargs -r docker rm -f >/dev/null 2>&1
}
trap cleanup EXIT
cleanup

# reduce lit « docker inspect » sur stdin et n'écrit que les champs décodés par
# internal/dockerread, variables d'environnement réduites à leurs noms.
reduce() {
	python3 -I -c '
import json, sys
d = json.load(sys.stdin)[0]
labels = d["Config"].get("Labels") or {}
out = {
  "Id": d["Id"], "Name": d["Name"], "Path": d["Path"], "Args": d["Args"],
  "RestartCount": d["RestartCount"], "Created": d["Created"],
  "State": {k: d["State"][k] for k in ("Status", "Running", "Restarting", "OOMKilled", "ExitCode", "Error", "StartedAt", "FinishedAt")},
  "Config": {"Image": d["Config"]["Image"],
             "Env": [e.split("=", 1)[0] for e in (d["Config"].get("Env") or [])],
             "Labels": {k: v for k, v in labels.items() if k.startswith("com.docker.compose.")}},
  "HostConfig": {"RestartPolicy": {"Name": d["HostConfig"]["RestartPolicy"]["Name"]},
                 "Memory": d["HostConfig"]["Memory"],
                 "LogConfig": {"Type": d["HostConfig"]["LogConfig"]["Type"]}},
}
json.dump([out], sys.stdout, indent=2, ensure_ascii=False)
print()
'
}

# record <cas> <conteneur> <commande…> : enregistre le cas une fois le conteneur arrêté.
record() {
	local name="$1" ctr="$2"
	shift 2
	mkdir -p "$name"
	printf '#!/bin/sh\n# Reproduit le cas « %s » (capture du %s).\n%s\n' "$name" "$(date +%F)" "$*" >"$name/case.sh"
	docker inspect "$ctr" | reduce >"$name/inspect.json"
	docker logs --tail 200 "$ctr" >"$name/logs.txt" 2>&1
	echo "$name : $(docker inspect "$ctr" --format 'statut {{.State.Status}}, code {{.State.ExitCode}}, OOM {{.State.OOMKilled}}, redémarrages {{.RestartCount}}')"
}

# wait_stopped <conteneur> : attend l'arrêt (au plus 60 s).
wait_stopped() {
	timeout 60 docker wait "$1" >/dev/null 2>&1 || true
}

# run_case <cas> <arguments de docker run…> : lance en arrière-plan, attend, enregistre.
run_case() {
	local name="$1"
	shift
	local ctr="$P-$name"
	docker run -d --name "$ctr" "$@" >/dev/null 2>&1
	if [ "$(docker inspect "$ctr" --format '{{.State.Status}}' 2>/dev/null)" != "created" ]; then
		wait_stopped "$ctr"
	fi
	record "$name" "$ctr" "docker run -d --name $ctr $*"
}

want() { [ $# -eq 0 ] || [[ " $* " == *" $CASE "* ]]; }

for CASE in missing-env-postgres missing-env-mariadb permission-denied address-in-use file-not-found \
	not-executable command-not-found start-error oom stopped sigterm segfault generic-1 usage-2 unknown-42 \
	restart-loop address-retry-then-other option-after-image option-after-image-debian; do
	want "$@" || continue
	case "$CASE" in
	missing-env-postgres) run_case "$CASE" postgres:18-alpine ;;
	missing-env-mariadb) run_case "$CASE" mariadb:13.0 ;;
	permission-denied) run_case "$CASE" "${U[@]}" alpine:3.24 sh -c 'echo démarrage; echo x > /etc/app.conf' ;;
	address-in-use)
		docker run -d --name "$P-web" nginxinc/nginx-unprivileged:1.30-alpine >/dev/null
		sleep 1
		run_case "$CASE" --network "container:$P-web" nginxinc/nginx-unprivileged:1.30-alpine
		;;
	file-not-found) run_case "$CASE" "${U[@]}" alpine:3.24 cat /config/app.yaml ;;
	not-executable) run_case "$CASE" "${U[@]}" alpine:3.24 /etc/hostname ;;
	command-not-found) run_case "$CASE" "${U[@]}" alpine:3.24 commande-inexistante ;;
	start-error) run_case "$CASE" --user utilisateur-inexistant alpine:3.24 true ;;
	oom) run_case "$CASE" "${U[@]}" --memory 8m alpine:3.24 sh -c 'x=a; while :; do x=$x$x; done' ;;
	stopped)
		docker run -d --name "$P-$CASE" "${U[@]}" alpine:3.24 sleep 300 >/dev/null
		sleep 1
		docker stop -t 1 "$P-$CASE" >/dev/null
		record "$CASE" "$P-$CASE" "docker run -d --name $P-$CASE --user 1000:1000 alpine:3.24 sleep 300 ; docker stop -t 1 $P-$CASE"
		;;
	sigterm)
		docker run -d --name "$P-$CASE" "${U[@]}" alpine:3.24 sh -c 'trap "echo arrêt demandé; exit 143" TERM; echo prêt; while :; do sleep 1; done' >/dev/null
		sleep 1
		docker stop "$P-$CASE" >/dev/null
		record "$CASE" "$P-$CASE" "docker run -d --name $P-$CASE --user 1000:1000 alpine:3.24 sh -c 'trap \"echo arrêt demandé; exit 143\" TERM; echo prêt; while :; do sleep 1; done' ; docker stop $P-$CASE"
		;;
	# « ; exit $? » empêche busybox de remplacer le shell par le dernier processus : devenu PID 1,
	# celui-ci ignorerait SIGSEGV.
	segfault) run_case "$CASE" "${U[@]}" alpine:3.24 sh -c 'echo calcul; sh -c "kill -SEGV \$\$"; exit $?' ;;
	generic-1) run_case "$CASE" "${U[@]}" alpine:3.24 sh -c 'echo démarrage du service; echo "erreur : configuration invalide" >&2; exit 1' ;;
	usage-2) run_case "$CASE" "${U[@]}" alpine:3.24 sh -c 'echo "usage: app [--port PORT] FICHIER" >&2; exit 2' ;;
	unknown-42) run_case "$CASE" "${U[@]}" alpine:3.24 sh -c 'echo traitement terminé; exit 42' ;;
	restart-loop)
		docker run -d --name "$P-$CASE" "${U[@]}" --restart on-failure:3 alpine:3.24 sh -c 'echo tentative; exit 1' >/dev/null
		for _ in $(seq 60); do
			[ "$(docker inspect "$P-$CASE" --format '{{.State.Status}} {{.RestartCount}}')" = "exited 3" ] && break
			sleep 1
		done
		record "$CASE" "$P-$CASE" "docker run -d --name $P-$CASE --user 1000:1000 --restart on-failure:3 alpine:3.24 sh -c 'echo tentative; exit 1'"
		;;
	# Issue #9 vue après coup : option de docker run placée après l'image, prise pour la commande.
	# Deux entrypoints, deux messages : busybox (« illegal option -p », code 2) et Debian
	# (« exec: -p: not found », code 127).
	option-after-image) run_case "$CASE" nginxinc/nginx-unprivileged:1.30-alpine -p 8090:8080 ;;
	option-after-image-debian) run_case "$CASE" nginx:1.30 -p 8090:80 ;;
	# Cas légitime voisin (LL-010) : un « address already in use » ancien, suivi de 15 lignes
	# normales, ne doit pas être pris pour la cause d'un arrêt dû à autre chose.
	address-retry-then-other) run_case "$CASE" "${U[@]}" alpine:3.24 sh -c 'echo "bind: address already in use, nouvelle tentative"; i=0; while [ $i -lt 15 ]; do echo "traitement $i"; i=$((i+1)); done; echo "erreur : base de données injoignable"; exit 1' ;;
	esac
done

# Événements de « docker run -d » pour internal/exited/watch_test.go.
CASE=events
if want "$@"; then
	mkdir -p events
	t0=$(date +%s.%N)
	docker run -d --name "$P-ev-pg" postgres:18-alpine >/dev/null
	docker run -d --rm "${U[@]}" alpine:3.24 sh -c 'exit 3' >/dev/null
	docker run -d --rm "${U[@]}" alpine:3.24 true >/dev/null
	docker run -d --name "$P-ev-web" nginxinc/nginx-unprivileged:1.30-alpine >/dev/null
	sleep 4
	docker events --since "$t0" --until "$(date +%s)" --filter type=container --filter event=create \
		--filter event=die --format '{{json .}}' >events/run-d.jsonl
	echo "t0=$t0" >events/run-d.t0
	echo "events : $(wc -l <events/run-d.jsonl) événements"
fi
