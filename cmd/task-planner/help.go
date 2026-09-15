package main

import (
	"fmt"
	"io"
	"strings"
)

type helpTopic struct {
	usage       string
	description string
	details     string
	subcommands string
	example     string
}

var helpTopics = map[string]helpTopic{
	"config": {
		usage:       "task-planner config",
		description: "Configure the Supabase connection interactively.",
		details:     "Paste a PostgreSQL Session Pooler URL. The command saves it in ~/.config/task-planner/environment and initializes the shared tables.",
		subcommands: "task-planner config default-project set [name-or-id]  Choose a default project.\ntask-planner config default-project clear             Remove the saved default.",
	},
	"config default-project": {
		usage:       "task-planner config default-project set [name-or-id]\n  task-planner config default-project clear",
		description: "Manage the project preselected by task-planner add.",
		details:     "The setting is a Todoist project ID in ~/.config/task-planner/default-project-id. You can edit that file manually; find IDs with task-planner auth projects.",
		subcommands: "task-planner config default-project set [name-or-id]  Choose a project or provide its name or ID.\ntask-planner config default-project clear             Remove the saved default.",
	},
	"config default-project set": {
		usage:       "task-planner config default-project set [name-or-id]",
		description: "Save a default Todoist destination project for future schedules.",
		details:     "Opens a picker of your current Todoist projects. Use the arrow keys and Enter to save; Esc cancels. The current default is highlighted. For scripts, you can pass a unique name or Todoist project ID instead. The selected ID is written to ~/.config/task-planner/default-project-id. Todoist must be connected.",
		example:     "task-planner config default-project set",
	},
	"config default-project clear": {
		usage:       "task-planner config default-project clear",
		description: "Remove the saved default destination project.",
		details:     "Future task-planner add sessions start with the first available Todoist project selected.",
	},
	"auth": {
		usage:       "task-planner auth login|logout|projects",
		description: "Manage this computer's Todoist connection and list projects.",
		subcommands: "task-planner auth login     Connect Todoist in a browser.\ntask-planner auth logout    Remove the locally stored Todoist login.\ntask-planner auth projects  Print Todoist project names and IDs as JSON.",
	},
	"auth login": {
		usage:       "task-planner auth login",
		description: "Connect Todoist through a browser on this computer.",
		details:     "Starts an OAuth login and stores the resulting credentials in the system keychain or Secret Service. Use a graphical desktop session.",
	},
	"auth logout": {
		usage:       "task-planner auth logout",
		description: "Remove the Todoist login stored on this computer.",
		details:     "A TODOIST_API_TOKEN environment variable, if set, can still provide access.",
	},
	"auth projects": {
		usage:       "task-planner auth projects",
		description: "List current Todoist projects as JSON with their names and IDs.",
		details:     "Use the ID to set or manually configure a default destination project.",
	},
	"status": {
		usage:       "task-planner status",
		description: "Check this computer's Supabase connection and Todoist login.",
		details:     "Shows missing setup steps or a connection error and the command to fix them.",
	},
	"check": {
		usage:       "task-planner check",
		description: "List schedules whose end date is before today and still remain in Supabase.",
		details:     "This report is read-only. It does not inspect whether their Todoist tasks are complete; review past schedules with task-planner delete old.",
	},
	"add": {
		usage:       "task-planner add",
		description: "Create a shared schedule and its Todoist tasks in the guided TUI.",
		details:     "Enter task text, an inclusive date range, repeat pattern, priority, and destination project. Tasks for the whole range are created immediately. Shift+Tab returns to a previous input or choice.",
	},
	"plans": {
		usage:       "task-planner plans",
		description: "List schedules saved in Supabase with their date ranges.",
	},
	"delete": {
		usage:       "task-planner delete",
		description: "Search for and delete a shared schedule in the guided TUI.",
		details:     "Type at least two characters; after a brief pause, matching pages are fetched from Supabase. Confirm full deletion to remove the schedule and its Todoist tasks. For a past schedule, database only removes the Supabase schedule while leaving Todoist tasks unchanged. After deletion, choose another search or close.",
		subcommands: "task-planner delete old  Preview and remove all past schedules from Supabase only.",
	},
	"delete old": {
		usage:       "task-planner delete old",
		description: "Preview every schedule whose end date is before today, then remove them all from Supabase only.",
		details:     "Opens a paginated preview without a text search. Review the schedules, then explicitly confirm one bulk deletion. Only the previewed schedules and their stored task IDs are removed from Supabase. If a previewed schedule has been removed or is no longer past, nothing is deleted and you must reload the preview. Todoist tasks stay unchanged.",
	},
	"completion": {
		usage:       "task-planner completion bash|zsh",
		description: "Print shell completion code for Bash or Zsh.",
		subcommands: "task-planner completion bash  Print Bash completion code.\ntask-planner completion zsh   Print Zsh completion code.",
	},
	"completion bash": {
		usage:       "task-planner completion bash",
		description: "Print Bash completion code to standard output.",
		example:     "eval \"$(task-planner completion bash)\"",
	},
	"completion zsh": {
		usage:       "task-planner completion zsh",
		description: "Print Zsh completion code to standard output.",
		example:     "eval \"$(task-planner completion zsh)\"",
	},
	"help": {
		usage:       "task-planner help [command [subcommand ...]]",
		description: "Show the command list or detailed help for a command path.",
		details:     "You can also append --help or -h to a command path.",
		example:     "task-planner help config default-project set",
	},
}

func commandHelp(writer io.Writer, path []string) error {
	if len(path) == 0 {
		usage(writer)
		return nil
	}
	key := strings.Join(path, " ")
	topic, ok := helpTopics[key]
	if !ok {
		return fmt.Errorf("unknown help topic %q; run `task-planner help` for commands", key)
	}
	fmt.Fprintf(writer, "Usage:\n  %s\n\n%s\n", topic.usage, topic.description)
	if topic.details != "" {
		fmt.Fprintf(writer, "\n%s\n", topic.details)
	}
	if topic.subcommands != "" {
		fmt.Fprintf(writer, "\nSubcommands:\n%s\n", topic.subcommands)
	}
	if topic.example != "" {
		fmt.Fprintf(writer, "\nExample:\n  %s\n", topic.example)
	}
	return nil
}
