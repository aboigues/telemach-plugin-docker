# Intégration TELEMACH pour zsh — à activer avec :
#   eval "$(docker telemach shell-init zsh)"
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

# Version du client Docker, lue une seule fois (commande locale, sans contact avec le démon).
__telemach_docker_version="$(command docker --version 2>/dev/null | sed -n 's/^Docker version \([^,]*\),.*/\1/p')"

# Fonction exécutée dans un sous-shell : ses pièges (trap) restent locaux.
docker() (
	# Sans MULTIOS, « 1>&3 » redirige stdout au lieu de le dupliquer (comportement de bash).
	# La fonction s'exécute dans un sous-shell : l'option du shell de l'utilisateur est intacte.
	setopt NO_MULTIOS
	# « docker telemach … » n'est jamais expliqué par TELEMACH lui-même.
	if [[ "${TELEMACH_DISABLE:-}" == "1" || "${1:-}" == "telemach" ]] || ! command -v "$__telemach_bin" >/dev/null 2>&1; then
		command docker "$@"
		exit $?
	fi

	# Feature 003 : run, create et exec (y compris « container … », après les options globales
	# de Docker) sont analysés après une exécution réussie ; les autres commandes, jamais.
	__telemach_lint=0
	if [ "${TELEMACH_LINT:-}" != "0" ]; then
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
	if [[ "$__telemach_login" == 1 && "$__telemach_password" == 0 ]]; then
		__telemach_interactive=1
	fi

	if [[ "$__telemach_interactive" == 1 ]]; then
		command docker "$@"
		__telemach_rc=$?
		if (( __telemach_rc != 0 )); then
			command "$__telemach_bin" explain --exit-code "$__telemach_rc" --interactive \
				--docker-version "$__telemach_docker_version" -- "$@" || true
		elif [[ "$__telemach_lint" == 1 ]]; then
			command "$__telemach_bin" lint --after -- "$@" || true
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
	__telemach_rc=${pipestatus[1]}

	if (( __telemach_rc != 0 )); then
		command "$__telemach_bin" explain --exit-code "$__telemach_rc" --stderr-file "$__telemach_tmp" \
			--docker-version "$__telemach_docker_version" -- "$@" || true
	elif [[ "$__telemach_lint" == 1 ]]; then
		command "$__telemach_bin" lint --after -- "$@" || true
	fi
	exit "$__telemach_rc"
)
