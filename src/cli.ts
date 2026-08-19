import { createHash } from "node:crypto";
import { createInterface } from "node:readline/promises";
import { fileURLToPath } from "node:url";
import { addPlan, connectDatabase, listPlans, removePlan, setupDatabase } from "./database.js";
import { saveDatabaseUrl } from "./environment.js";
import { main, type Plan, PlanSchema } from "./index.js";
import { listProjects, login, logout } from "./oauth.js";

export const usage = `task-planner — create distinct Todoist tasks on a schedule

Usage:
  task-planner auth login       Connect Todoist in a graphical desktop session
  task-planner auth projects    List Todoist projects
  task-planner auth logout      Remove the saved Todoist credentials
  task-planner run [--dry-run]  Process shared plans
  task-planner config           Save and validate the Supabase connection URL
  task-planner db setup         Create/update the shared Supabase tables
  task-planner add <options>    Add a shared plan; use help for options
  task-planner plans            List active plans
  task-planner delete <text>    Stop a plan by its unique task text
  task-planner completion <shell>
                                Print Bash or Zsh completion code
  task-planner help             Show this help

Set SUPABASE_DB_URL to the Supabase Session Pooler connection string. The shared
database stores active plans and processed periods, so any device can run the scheduler.`;

const bashCompletion = `# Bash completion for task-planner
_task_planner() {
  local cur command
  cur="\${COMP_WORDS[COMP_CWORD]}"
  command="\${COMP_WORDS[1]}"

  if (( COMP_CWORD == 1 )); then
    COMPREPLY=( $(compgen -W 'auth run config db add plans delete completion help --help -h' -- "$cur") )
    return
  fi

  case "$command" in
    auth)
      COMPREPLY=( $(compgen -W 'login logout projects' -- "$cur") )
      ;;
    run)
      COMPREPLY=( $(compgen -W '--dry-run --help -h' -- "$cur") )
      ;;
    db)
      COMPREPLY=( $(compgen -W 'setup' -- "$cur") )
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
      'run:Process shared plans' \\
      'config:Save the Supabase connection URL' \\
      'db:Create/update shared database tables' \\
      'add:Add a shared plan' \\
      'plans:List active plans' \\
      'delete:Stop a plan by its task text' \\
      'completion:Print shell completion code' \\
      'help:Show usage'
    return
  fi

  case "$words[2]" in
    auth) _values 'authentication command' login logout projects ;;
    run) _arguments '--dry-run[Show pending task creations without creating them]' ;;
    db) _values 'database command' setup ;;
    completion) _values 'shell' bash zsh ;;
  esac
}
compdef _task_planner task-planner`;

export const completionFor = (shell: string): string => {
  if (shell === "bash") return bashCompletion;
  if (shell === "zsh") return zshCompletion;
  throw new Error("Completion is available for Bash and Zsh.");
};

const configure = async (): Promise<void> => {
  const prompt = createInterface({ input: process.stdin, output: process.stdout });
  try {
    const url = await prompt.question("Paste the Supabase Session Pooler URL: ");
    const path = await saveDatabaseUrl(url);
    process.env.SUPABASE_DB_URL = url.trim();
    const database = connectDatabase();
    try {
      await setupDatabase(database);
    } finally {
      await database.end({ timeout: 5 });
    }
    console.log(`Saved the Supabase connection and initialized shared tables in ${path}.`);
  } finally {
    prompt.close();
  }
};

const valueFor = (args: string[], option: string): string | undefined => {
  const index = args.indexOf(option);
  return index === -1 ? undefined : args[index + 1];
};

const idFor = (content: string): string => {
  const slug = content
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "")
    .slice(0, 40);
  return `${slug || "plan"}-${createHash("sha256").update(content).digest("hex").slice(0, 8)}`;
};

const addFromArgs = (args: string[]) => {
  const content = valueFor(args, "--text");
  const period = valueFor(args, "--period");
  const projectId = valueFor(args, "--project-id");
  const dueString = valueFor(args, "--due");
  const priorityValue = valueFor(args, "--priority");
  if (!content || !period || !projectId) {
    throw new Error(
      "Usage: task-planner add --text <text> --period <day|week|month> --project-id <id> [--due <date>] [--priority <1-4>]",
    );
  }
  return PlanSchema.parse({
    id: idFor(content),
    content,
    period,
    projectId,
    dueString,
    priority: priorityValue ? Number(priorityValue) : undefined,
  });
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
  if (command === "config") return configure();
  if (command === "completion") return console.log(completionFor(subcommand ?? ""));
  if (command === "db" && subcommand === "setup") {
    const database = connectDatabase();
    try {
      await setupDatabase(database);
    } finally {
      await database.end({ timeout: 5 });
    }
    console.log("Shared Supabase tables are ready.");
    return;
  }
  if (command === "add") {
    const database = connectDatabase();
    try {
      await setupDatabase(database);
      await addPlan(database, addFromArgs(args.slice(1)));
    } finally {
      await database.end({ timeout: 5 });
    }
    console.log(`Added plan: ${valueFor(args, "--text")}`);
    return;
  }
  if (command === "plans") {
    const database = connectDatabase();
    let plans: Plan[];
    try {
      await setupDatabase(database);
      plans = await listPlans(database);
    } finally {
      await database.end({ timeout: 5 });
    }
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
    const database = connectDatabase();
    try {
      await setupDatabase(database);
      await removePlan(database, content);
    } finally {
      await database.end({ timeout: 5 });
    }
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
