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
  task-planner config default-project set [name-or-id]
                               Choose the default Todoist destination project
  task-planner config default-project clear
                               Clear the default destination project
  task-planner auth login      Connect Todoist
  task-planner auth logout     Remove the local Todoist login
  task-planner auth projects   List projects
  task-planner status          Check this computer's setup
  task-planner check           List past schedules still in Supabase
  task-planner add             Add a plan in the guided TUI
  task-planner plans           List active plans
  task-planner delete          Search and delete a plan in the guided TUI
  task-planner completion bash|zsh
  task-planner help [command [subcommand ...]]

Run task-planner help <command> or append --help for more detail.
`)
}

func isHelpFlag(value string) bool {
	return value == "--help" || value == "-h"
}

func main() {
	args := os.Args[1:]
	var err error
	switch {
	case len(args) == 0:
		usage(os.Stdout)
	case args[0] == "help":
		path := args[1:]
		if len(path) > 0 && isHelpFlag(path[len(path)-1]) {
			path = path[:len(path)-1]
		}
		err = commandHelp(os.Stdout, path)
	case isHelpFlag(args[len(args)-1]):
		err = commandHelp(os.Stdout, args[:len(args)-1])
	case len(args) == 1 && args[0] == "config":
		err = config()
	case len(args) == 3 && args[0] == "config" && args[1] == "default-project" && args[2] == "set":
		err = selectDefaultProject()
	case len(args) == 4 && args[0] == "config" && args[1] == "default-project" && args[2] == "set":
		err = setDefaultProject(args[3])
	case len(args) == 3 && args[0] == "config" && args[1] == "default-project" && args[2] == "clear":
		err = clearDefaultProject()
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
	case len(args) == 1 && args[0] == "check":
		err = checkPastSchedules()
	case len(args) == 1 && args[0] == "add":
		err = guidedAdd()
	case len(args) == 1 && args[0] == "plans":
		var activePlans []plan
		activePlans, err = plans()
		if err == nil {
			for _, activePlan := range activePlans {
				fmt.Printf("%s — %s to %s\n", activePlan.Content, activePlan.StartDate.Format("2006-01-02"), activePlan.EndDate.Format("2006-01-02"))
			}
		}
	case len(args) == 1 && args[0] == "delete":
		err = guidedDelete()
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
