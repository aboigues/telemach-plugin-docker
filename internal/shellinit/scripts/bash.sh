# Intégration TELEMACH pour bash — à activer avec :
#   eval "$(docker telemach shell-init bash)"
# Définit une fonction « docker » qui explique en français les commandes en échec.
# Garanties : stdin et stdout ne sont jamais redirigés, le code de sortie d'origine est
# toujours renvoyé, aucun autre programme n'est observé.

# Version du client Docker, lue une seule fois (commande locale, sans contact avec le démon).
__telemach_docker_version="$(command docker --version 2>/dev/null | sed -n 's/^Docker version \([^,]*\),.*/\1/p')"

# Fonction exécutée dans un sous-shell : ses pièges (trap) restent locaux.
docker() (
	if [ "${TELEMACH_DISABLE:-}" = "1" ] || ! command -v docker-telemach >/dev/null 2>&1; then
		command docker "$@"
		exit $?
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
			command docker-telemach explain --exit-code "$__telemach_rc" --interactive \
				--docker-version "$__telemach_docker_version" -- "$@" || true
		fi
		exit "$__telemach_rc"
	fi

	__telemach_tmp="$(umask 077 && mktemp "${XDG_RUNTIME_DIR:-${TMPDIR:-/tmp}}/telemach.XXXXXX")" || {
		command docker "$@"
		exit $?
	}
	trap 'rm -f -- "$__telemach_tmp"' EXIT
	trap 'exit 130' INT
	trap 'exit 143' TERM

	# stderr de docker passe par tee (copie dans le fichier temporaire) ; stdout reste intact
	# grâce au descripteur 3.
	{ command docker "$@" 2>&1 1>&3 3>&- | tee -- "$__telemach_tmp" >&2; } 3>&1
	__telemach_rc=${PIPESTATUS[0]}

	if [ "$__telemach_rc" -ne 0 ]; then
		command docker-telemach explain --exit-code "$__telemach_rc" --stderr-file "$__telemach_tmp" \
			--docker-version "$__telemach_docker_version" -- "$@" || true
	fi
	exit "$__telemach_rc"
)
