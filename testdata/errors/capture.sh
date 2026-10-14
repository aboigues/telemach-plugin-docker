#!/usr/bin/env bash
# Capture de vraies erreurs Docker pour le corpus de référence (SC-001).
# Usage : testdata/errors/capture.sh [nom-du-cas…]
# Chaque cas produit <nom>.stderr.txt, <nom>.args et <nom>.exit dans ce répertoire.
# Les ressources créées sont préfixées « telemach-corpus » et supprimées à la fin.
set -uo pipefail

cd "$(dirname "$0")"
WORK="$(mktemp -d)"
trap 'cleanup' EXIT

cleanup() {
	docker rm -f telemach-corpus-web telemach-corpus-busy telemach-corpus-stopped telemach-corpus-user >/dev/null 2>&1
	docker network rm telemach-corpus-net >/dev/null 2>&1
	docker volume rm telemach-corpus-vol >/dev/null 2>&1
	docker rmi telemach-corpus-img:1 >/dev/null 2>&1
	rm -rf "$WORK"
}

# capture <nom> <args docker…> : exécute la commande et enregistre stderr et le code.
capture() {
	local name="$1"
	shift
	printf '%s\n' "$*" >"$name.args"
	docker "$@" >/dev/null 2>"$name.stderr.txt"
	echo $? >"$name.exit"
	echo "$name : code $(cat "$name.exit")"
}

# capture_env <nom> <VAR=valeur> <args docker…> : idem avec une variable d'environnement.
capture_env() {
	local name="$1" var="$2"
	shift 2
	printf '%s\n' "$*" >"$name.args"
	env "$var" docker "$@" >/dev/null 2>"$name.stderr.txt"
	echo $? >"$name.exit"
	echo "$name : code $(cat "$name.exit")"
}

docker pull -q alpine:3.22 >/dev/null
docker pull -q nginx:1.29-alpine >/dev/null

# --- Réseau ---
docker run -d --name telemach-corpus-web -p 18080:80 nginx:1.29-alpine >/dev/null
capture run-port-allocated run --rm -p 18080:80 nginx:1.29-alpine
capture run-invalid-port-spec run --rm -p 99999:80 nginx:1.29-alpine
capture network-not-found run --rm --network telemach-corpus-absent alpine:3.22 true
docker network create telemach-corpus-net >/dev/null
docker network connect telemach-corpus-net telemach-corpus-web
capture network-active-endpoints network rm telemach-corpus-net

# --- Images ---
capture image-not-found-typo run --rm ngnix
capture image-pull-denied pull telemach-corpus/inexistante:1
capture image-invalid-reference run --rm Nginx
capture image-manifest-unknown pull alpine:9.99.99
docker run --name telemach-corpus-stopped alpine:3.22 true >/dev/null
docker commit telemach-corpus-stopped telemach-corpus-img:1 >/dev/null
docker create --name telemach-corpus-user telemach-corpus-img:1 true >/dev/null
capture image-in-use rmi telemach-corpus-img:1
capture image-push-denied push telemach-corpus-img:1
# « exec format error » n'est pas reproductible quand l'émulation qemu (binfmt) est active.

# --- Conteneurs ---
capture container-name-conflict run --name telemach-corpus-web nginx:1.29-alpine
capture container-not-found-logs logs telemach-corpus-absent
capture container-not-found-stop stop telemach-corpus-absent
capture container-not-found-start start telemach-corpus-absent
capture container-not-found-exec exec telemach-corpus-absent sh
capture container-not-running-exec exec telemach-corpus-stopped sh
capture container-remove-running rm telemach-corpus-web

# --- Démon et permissions ---
capture_env daemon-not-running DOCKER_HOST=unix:///tmp/telemach-corpus-absent.sock ps
python3 -c "import socket,os; s=socket.socket(socket.AF_UNIX); s.bind('$WORK/denied.sock')"
chmod 000 "$WORK/denied.sock"
capture_env permission-socket-denied "DOCKER_HOST=unix://$WORK/denied.sock" ps
printf 'mauvais-mot-de-passe' | docker --config "$WORK/cfg" login -u telemach-corpus-x --password-stdin >/dev/null 2>"permission-registry-unauthorized.stderr.txt"
echo $? >permission-registry-unauthorized.exit
echo 'login -u telemach-corpus-x --password-stdin' >permission-registry-unauthorized.args

# Disque plein : non reproductible sans risque ; texte retranscrit depuis Docker Engine.
printf 'pull nginx:1.29-alpine\n' >daemon-no-space-left.args
printf 'failed to register layer: write /var/lib/docker/overlay2/l/ABCDEF/usr/lib/libssl.so.3: no space left on device\n' >daemon-no-space-left.stderr.txt
echo 1 >daemon-no-space-left.exit

# --- Volumes ---
# Docker Desktop crée les dossiers de bind mount manquants : texte retranscrit depuis
# Docker Engine (moby, daemon/volumes) tel qu'affiché sur un moteur Linux.
printf 'run --rm --mount type=bind,src=/srv/absent,dst=/data alpine:3.22 true\n' >volume-bind-source-missing.args
printf 'docker: Error response from daemon: invalid mount config for type "bind": bind source path does not exist: /srv/absent\n\nRun %sdocker run --help%s for more information\n' "'" "'" >volume-bind-source-missing.stderr.txt
echo 125 >volume-bind-source-missing.exit
docker volume create telemach-corpus-vol >/dev/null
docker run -d --name telemach-corpus-busy -v telemach-corpus-vol:/data alpine:3.22 sleep 300 >/dev/null
capture volume-in-use volume rm telemach-corpus-vol

# --- Build ---
mkdir -p "$WORK/empty" "$WORK/copy" "$WORK/runfail"
capture build-dockerfile-not-found build "$WORK/empty"
printf 'FROM alpine:3.22\nCOPY absent.txt /app/\n' >"$WORK/copy/Dockerfile"
capture build-copy-missing build --progress=plain "$WORK/copy"
printf 'FROM alpine:3.22\nRUN echo préparation\nRUN commande-inexistante --version\n' >"$WORK/runfail/Dockerfile"
capture build-run-failed build --progress=plain "$WORK/runfail"

# --- Exécution (codes de sortie) ---
capture runtime-exit-126 run --rm alpine:3.22 /etc/passwd
capture runtime-exit-127 run --rm alpine:3.22 commande-inexistante
# --init : le shell n'est plus le PID 1 et peut donc recevoir le signal.
capture runtime-exit-137 run --rm --init alpine:3.22 sh -c 'kill -9 $$'
capture runtime-exit-143 run --rm --init alpine:3.22 sh -c 'kill -15 $$'
capture runtime-application-error run --rm alpine:3.22 sh -c 'echo "base de données injoignable" >&2; exit 3'

# --- Compose hors d'un projet ---
for sub in ps logs down; do
	name="compose-no-project-$sub"
	printf 'compose %s\n' "$sub" >"$name.args"
	docker compose --project-directory "$WORK/empty" "$sub" >/dev/null 2>"$name.stderr.txt"
	echo $? >"$name.exit"
	echo "$name : code $(cat "$name.exit")"
done

# --- Non reconnu (volontairement) ---
printf 'pull alpine:3.22\n' >unknown-invented.args
printf 'Erreur interne simulée : le flux quantique est désaligné.\n' >unknown-invented.stderr.txt
echo 1 >unknown-invented.exit
