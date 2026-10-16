# Intégration TELEMACH pour bash — à activer avec :
#   eval "$(docker telemach shell-init bash)"
# Définit une fonction « docker » qui explique en français les commandes en échec.
# Garanties : stdin et stdout ne sont jamais redirigés, le code de sortie d'origine est
# toujours renvoyé, aucun autre programme n'est observé.

# Binaire docker-telemach, à l'emplacement où il était installé lors de la génération de ce
# script (souvent ~/.docker/cli-plugins/, qui n'est pas dans le PATH).
__telemach_bin=__TELEMACH_BIN__

# Intégration active et version du plugin qui l'a générée : « docker telemach last » repère
# ainsi un terminal sans intégration, ou dont la fonction date d'une version antérieure
# (issue #7). Seule variable exportée par ce script.
export TELEMACH_SHELL=__TELEMACH_SHELL__

# Feature 004 : après un « docker run -d » réussi, « docker telemach watch » vérifie en
# arrière-plan, environ 3 s plus tard, que le conteneur ne s'est pas arrêté en erreur ; son
# signal est déposé dans __telemach_pending et affiché à l'invite suivante, jamais pendant une
# saisie. Seulement dans un shell interactif (il faut une invite où l'afficher).
__telemach_pending=__TELEMACH_PENDING__
__telemach_shell_pid=$$
__telemach_watch_on=0
[[ $- == *i* ]] && __telemach_watch_on=1

# Lance la vérification sans attendre ni rien afficher (entrées et sorties sur /dev/null).
__telemach_watch() {
	[[ "$__telemach_watch_on" == 1 && "${TELEMACH_WATCH:-}" != "0" && -n "${__telemach_t0:-}" ]] || return 0
	command "$__telemach_bin" watch --since "$__telemach_t0" --shell-pid "$__telemach_shell_pid" -- "$@" </dev/null >/dev/null 2>&1 &
}

# Crochet d'invite : affiche puis supprime les signaux de ce shell ; sans signal, aucun
# programme n'est lancé. Le code de retour de la dernière commande est préservé.
__telemach_prompt() {
	local __telemach_status=$? __telemach_f
	for __telemach_f in "$__telemach_pending"/pending-"$__telemach_shell_pid"-*; do
		[[ -e "$__telemach_f" ]] || continue
		printf '%s\n' "$(<"$__telemach_f")" >&2
		command rm -f -- "$__telemach_f"
	done
	return $__telemach_status
}
if [[ "$__telemach_watch_on" == 1 ]]; then
	# PROMPT_COMMAND peut être une chaîne ou (bash ≥ 5.1) un tableau ; ajout idempotent.
	if [[ "$(declare -p PROMPT_COMMAND 2>/dev/null)" == "declare -a"* ]]; then
		[[ " ${PROMPT_COMMAND[*]} " == *" __telemach_prompt "* ]] || PROMPT_COMMAND+=(__telemach_prompt)
	else
		case ";${PROMPT_COMMAND:-};" in
		*";__telemach_prompt;"*) ;;
		*) PROMPT_COMMAND="__telemach_prompt${PROMPT_COMMAND:+;$PROMPT_COMMAND}" ;;
		esac
	fi
fi

# Version du client Docker, lue une seule fois (commande locale, sans contact avec le démon).
__telemach_docker_version="$(command docker --version 2>/dev/null | sed -n 's/^Docker version \([^,]*\),.*/\1/p')"

# Fonction exécutée dans un sous-shell : ses pièges (trap) restent locaux.
docker() (
	# « docker telemach … » n'est jamais expliqué par TELEMACH lui-même.
	if [ "${TELEMACH_DISABLE:-}" = "1" ] || [ "${1:-}" = "telemach" ] || ! command -v "$__telemach_bin" >/dev/null 2>&1; then
		command docker "$@"
		exit $?
	fi

	# Heure de début, pour retrouver le conteneur lancé (feature 004).
	__telemach_t0=${EPOCHREALTIME:-}

	# Feature 003 : run, create et exec (y compris « container … », après les options globales
	# de Docker) sont analysés après une exécution réussie ; les autres commandes, jamais.
	# La détection a lieu même si TELEMACH_LINT=0 : la vérification de la feature 004 se
	# désactive séparément (TELEMACH_WATCH=0).
	__telemach_lint=0
	if :; then
		__telemach_skip=0
		__telemach_mgmt=0
		for __telemach_arg in "$@"; do
			if [ "$__telemach_skip" = 1 ]; then
				__telemach_skip=0
				continue
			fi
			case "$__telemach_arg" in
			-c | --context | -H | --host | --config | -l | --log-level | --tlscacert | --tlscert | --tlskey) __telemach_skip=1 ;;
			-*) [ "$__telemach_mgmt" = 1 ] && break ;;
			container)
				[ "$__telemach_mgmt" = 1 ] && break
				__telemach_mgmt=1
				;;
			run | create | exec)
				__telemach_lint=1
				break
				;;
			*) break ;;
			esac
		done
	fi

	# Commande interactive : on ne touche pas à stderr pour ne pas perturber le terminal.
	__telemach_interactive=0
	__telemach_login=0
	__telemach_password=0
	for __telemach_arg in "$@"; do
		case "$__telemach_arg" in
		-i | -t | -it | -ti | -ai | -ia | --interactive | --tty | attach) __telemach_interactive=1 ;;
		login) __telemach_login=1 ;;
		--password-stdin | -p | --password | --password=*) __telemach_password=1 ;;
		esac
	done
	if [ "$__telemach_login" = 1 ] && [ "$__telemach_password" = 0 ]; then
		__telemach_interactive=1
	fi

	if [ "$__telemach_interactive" = 1 ]; then
		command docker "$@"
		__telemach_rc=$?
		if [ "$__telemach_rc" -ne 0 ]; then
			command "$__telemach_bin" explain --exit-code "$__telemach_rc" --interactive \
				--docker-version "$__telemach_docker_version" -- "$@" || true
		elif [ "$__telemach_lint" = 1 ]; then
			if [ "${TELEMACH_LINT:-}" != "0" ]; then
				command "$__telemach_bin" lint --after -- "$@" || true
			fi
			__telemach_watch "$@"
		fi
		exit "$__telemach_rc"
	fi

	# Pièges installés avant la création du fichier : une interruption (Ctrl-C) à n'importe
	# quel moment ne laisse aucun fichier temporaire derrière elle.
	__telemach_tmp=""
	trap 'rm -f -- "$__telemach_tmp"' EXIT
	trap 'exit 130' INT
	trap 'exit 143' TERM
	__telemach_tmp="$(umask 077 && mktemp "${XDG_RUNTIME_DIR:-${TMPDIR:-/tmp}}/telemach.XXXXXX")" || {
		command docker "$@"
		exit $?
	}

	# stderr de docker passe par tee (copie dans le fichier temporaire) ; stdout reste intact
	# grâce au descripteur 3.
	{ command docker "$@" 2>&1 1>&3 3>&- | tee -- "$__telemach_tmp" >&2; } 3>&1
	__telemach_rc=${PIPESTATUS[0]}

	if [ "$__telemach_rc" -ne 0 ]; then
		command "$__telemach_bin" explain --exit-code "$__telemach_rc" --stderr-file "$__telemach_tmp" \
			--docker-version "$__telemach_docker_version" -- "$@" || true
	elif [ "$__telemach_lint" = 1 ]; then
		if [ "${TELEMACH_LINT:-}" != "0" ]; then
			command "$__telemach_bin" lint --after -- "$@" || true
		fi
		__telemach_watch "$@"
	fi
	exit "$__telemach_rc"
)
