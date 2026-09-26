# Wharnex-owned project terminal init. Regenerated on every terminal start.
if [ -r /etc/profile ]; then . /etc/profile; fi
if [ -r "$HOME/.bash_profile" ]; then . "$HOME/.bash_profile"
elif [ -r "$HOME/.bash_login" ]; then . "$HOME/.bash_login"
elif [ -r "$HOME/.profile" ]; then . "$HOME/.profile"
fi
if [ -n "$WHARNEX_PROJECT_HISTFILE" ]; then
	HISTFILE="$WHARNEX_PROJECT_HISTFILE"
else
	unset HISTFILE
fi
shopt -s histappend
__wharnex_project_history() {
	[ -n "$WHARNEX_PROJECT_HISTFILE" ] || return 0
	HISTFILE="$WHARNEX_PROJECT_HISTFILE"
	history -a
	history -c
	history -r
}
if [[ "$(declare -p PROMPT_COMMAND 2>/dev/null)" == "declare -a"* ]]; then
	PROMPT_COMMAND=(__wharnex_project_history "${PROMPT_COMMAND[@]}")
else
	PROMPT_COMMAND="__wharnex_project_history${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
fi
