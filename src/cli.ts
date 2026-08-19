import { mkdir, writeFile } from "node:fs/promises";
import { dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { repositoryPath, rulesPath } from "./config.js";
import { deletePlan, main, readRules } from "./index.js";
import { listProjects, login, logout } from "./oauth.js";
import { publishFile, updateFromRemote } from "./repository.js";

export const usage = `task-planner — create distinct Todoist tasks on a schedule

Usage:
  task-planner auth login       Connect Todoist in a graphical desktop session
  task-planner auth projects    List Todoist projects
  task-planner auth logout      Remove the saved Todoist credentials
  task-planner run [--dry-run]  Process configured rules
  task-planner init             Create the default rules file, if absent
  task-planner plans            List active plans
  task-planner delete <text>    Stop a plan by its unique task text
  task-planner completion <shell>
                                Print Bash or Zsh completion code
  task-planner help             Show this help

Plans and processed periods are stored in the Git checkout. Run the command from that
checkout, or set TASK_PLANNER_REPOSITORY. Set TASK_PLANNER_RULES_PATH to override only
the plans file.`;

const bashCompletion = `# Bash completion for task-planner
_task_planner() {
  local cur command
  cur="\${COMP_WORDS[COMP_CWORD]}"
  command="\${COMP_WORDS[1]}"

  if (( COMP_CWORD == 1 )); then
    COMPREPLY=( $(compgen -W 'auth run init plans delete completion help --help -h' -- "$cur") )
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
      'plans:List active plans' \\
      'delete:Stop a plan by its task text' \\
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
  if (command === "plans") {
    await updateFromRemote(repositoryPath());
    const plans = await readRules();
    if (plans.length === 0) {
      console.log("No active plans.");
      return;
    }
    for (const plan of plans) console.log(`${plan.content} — ${plan.period}`);
    return;
  }
  if (command === "delete") {
    const content = args.slice(1).join(" ").trim();
    if (!content) throw new Error("Provide the unique task text to delete.");
    const repository = repositoryPath();
    await updateFromRemote(repository);
    await deletePlan(content);
    await publishFile(repository, "rules.json", `task-planner: stop ${content}`);
    console.log(`Stopped plan: ${content}`);
    return;
  }
  if (command === "run") return main();
  throw new Error(`Unknown command.\n\n${usage}`);
};

if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  void run().catch((error: unknown) => {
    console.error(error instanceof Error ? error.message : error);
    process.exitCode = 1;
  });
}
