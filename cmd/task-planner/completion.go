package main

import (
	"errors"
	"fmt"
)

func completion(shell string) error {
	script, err := completionScript(shell)
	if err != nil {
		return err
	}
	fmt.Print(script)
	return nil
}

func completionScript(shell string) (string, error) {
	switch shell {
	case "bash":
		return `_task_planner() {
  local cur command
  cur="${COMP_WORDS[COMP_CWORD]}"
  command="${COMP_WORDS[1]}"
  if (( COMP_CWORD == 1 )); then
    COMPREPLY=( $(compgen -W 'config auth status add plans delete completion help' -- "$cur") )
    return
  fi
  case "$command" in
    config)
      if (( COMP_CWORD == 2 )); then
        COMPREPLY=( $(compgen -W 'default-project' -- "$cur") )
      elif (( COMP_CWORD == 3 )) && [[ "${COMP_WORDS[2]}" == 'default-project' ]]; then
        COMPREPLY=( $(compgen -W 'set clear' -- "$cur") )
      fi
      ;;
    auth) COMPREPLY=( $(compgen -W 'login logout projects' -- "$cur") ) ;;
    completion) COMPREPLY=( $(compgen -W 'bash zsh' -- "$cur") ) ;;
  esac
}
complete -F _task_planner task-planner
`, nil
	case "zsh":
		return `_task_planner() {
  if (( CURRENT == 2 )); then
    _describe -t commands 'task-planner command' 'config:Manage local settings' 'auth:Manage Todoist login' 'status:Check local setup' 'add:Add a guided plan' 'plans:List plans' 'delete:Delete a plan' 'completion:Print completion code' 'help:Show help'
    return
  fi
  if [[ "$words[2]" == 'config' ]]; then
    if (( CURRENT == 3 )); then
      _values 'config command' default-project
    elif (( CURRENT == 4 )) && [[ "$words[3]" == 'default-project' ]]; then
      _values 'default project command' set clear
    fi
    return
  fi
  case "$words[2]" in
    auth) _values 'auth command' login logout projects ;;
    completion) _values 'shell' bash zsh ;;
  esac
}
compdef _task_planner task-planner
`, nil
	default:
		return "", errors.New("completion is available for bash and zsh")
	}
}
