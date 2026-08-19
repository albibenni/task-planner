import { mkdir, writeFile } from "node:fs/promises";
import { dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { rulesPath } from "./config.js";
import { main } from "./index.js";
import { listProjects, login, logout } from "./oauth.js";

export const usage = `task-planner — create distinct Todoist tasks on a schedule

Usage:
  task-planner auth login       Connect Todoist in a graphical desktop session
  task-planner auth projects    List Todoist projects
  task-planner auth logout      Remove the saved Todoist credentials
  task-planner run [--dry-run]  Process configured rules
  task-planner init             Create the default rules file, if absent
  task-planner completion <shell>
                                Print Bash or Zsh completion code
  task-planner help             Show this help

For a global installation, rules are stored in:
  macOS: ~/Library/Application Support/task-planner/rules.json
  Linux: $XDG_CONFIG_HOME/task-planner/rules.json (default: ~/.config/...)

Set TASK_PLANNER_RULES_PATH to override the rules-file location.`;

const bashCompletion = `# Bash completion for task-planner
_task_planner() {
  local cur command
  cur="\${COMP_WORDS[COMP_CWORD]}"
  command="\${COMP_WORDS[1]}"

  if (( COMP_CWORD == 1 )); then
    COMPREPLY=( $(compgen -W 'auth run init completion help --help -h' -- "$cur") )
    return
  fi

  case "$command" in
    auth)
      COMPREPLY=( $(compgen -W 'login logout projects' -- "$cur") )
      ;;
    run)
      COMPREPLY=( $(compgen -W '--dry-run --help -h' -- "$cur") )
      ;;
    completion)
      COMPREPLY=( $(compgen -W 'bash zsh' -- "$cur") )
      ;;
  esac
}
complete -F _task_planner task-planner`;

const zshCompletion = `_task_planner() {
  if (( CURRENT == 2 )); then
    _describe -t commands 'task-planner command' \\
      'auth:Connect or manage Todoist credentials' \\
      'run:Process configured rules' \\
      'init:Create the default rules file' \\
      'completion:Print shell completion code' \\
      'help:Show usage'
    return
  fi

  case "$words[2]" in
    auth) _values 'authentication command' login logout projects ;;
    run) _arguments '--dry-run[Show pending task creations without creating them]' ;;
    completion) _values 'shell' bash zsh ;;
  esac
}
compdef _task_planner task-planner`;

export const completionFor = (shell: string): string => {
  if (shell === "bash") return bashCompletion;
  if (shell === "zsh") return zshCompletion;
  throw new Error("Completion is available for Bash and Zsh.");
};

export const initializeRules = async (path = rulesPath()): Promise<string> => {
  await mkdir(dirname(path), { recursive: true });
  try {
    await writeFile(path, "[]\n", { encoding: "utf8", flag: "wx", mode: 0o600 });
    return path;
  } catch (error: unknown) {
    if ((error as NodeJS.ErrnoException).code === "EEXIST") {
      throw new Error(`Rules file already exists: ${path}`);
    }
    throw error;
  }
};

export const run = async (args = process.argv.slice(2)): Promise<void> => {
  const [command, subcommand] = args;
  if (!command || command === "help" || command === "--help" || command === "-h") {
    console.log(usage);
    return;
  }
  if (command === "auth" && subcommand === "login") return login();
  if (command === "auth" && subcommand === "logout") return logout();
  if (command === "auth" && subcommand === "projects") return listProjects();
  if (command === "init") {
    console.log(`Created ${await initializeRules()}`);
    return;
  }
  if (command === "completion") return console.log(completionFor(subcommand ?? ""));
  if (command === "run") return main();
  throw new Error(`Unknown command.\n\n${usage}`);
};

if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  void run().catch((error: unknown) => {
    console.error(error instanceof Error ? error.message : error);
    process.exitCode = 1;
  });
}
