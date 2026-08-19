package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

func usage(writer io.Writer) {
	fmt.Fprint(writer, `task-planner — shared Todoist plans

Usage:
  task-planner config          Configure Supabase interactively
  task-planner auth login      Connect Todoist
  task-planner auth projects   List projects
  task-planner status          Check this computer's setup
  task-planner add             Add a plan in the guided TUI
  task-planner plans           List active plans
  task-planner delete          Search and delete a plan in the guided TUI
  task-planner run [--dry-run] Run shared plans
  task-planner completion bash|zsh
`)
}

func main() {
	args := os.Args[1:]
	var err error
	switch {
	case len(args) == 0 || args[0] == "help" || args[0] == "--help":
		usage(os.Stdout)
	case len(args) == 1 && args[0] == "config":
		err = config()
	case len(args) == 2 && args[0] == "auth" && args[1] == "login":
		err = login()
	case len(args) == 2 && args[0] == "auth" && args[1] == "logout":
		err = deleteSecret()
	case len(args) == 2 && args[0] == "auth" && args[1] == "projects":
		var projects []project
		projects, err = todoistProjects()
		if err == nil {
			err = json.NewEncoder(os.Stdout).Encode(projects)
		}
	case len(args) == 1 && args[0] == "status":
		err = showStatus()
	case len(args) == 1 && args[0] == "add":
		err = guidedAdd()
	case len(args) == 1 && args[0] == "plans":
		var activePlans []plan
		activePlans, err = plans()
		if err == nil {
			for _, activePlan := range activePlans {
				fmt.Printf("%s — %s\n", activePlan.Content, activePlan.Period)
			}
		}
	case len(args) == 1 && args[0] == "delete":
		err = guidedDelete()
	case len(args) >= 1 && args[0] == "run":
		err = schedule(len(args) == 2 && args[1] == "--dry-run")
	case len(args) == 2 && args[0] == "completion":
		err = completion(args[1])
	default:
		usage(os.Stdout)
		err = errors.New("unknown command")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
