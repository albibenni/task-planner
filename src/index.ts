import { readFile, rename, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { z } from "zod";
import { processedPath, repositoryPath, rulesPath } from "./config.js";
import { getAccessToken } from "./oauth.js";
import { publishFile, updateFromRemote } from "./repository.js";

type Period = "day" | "week" | "month";

export const RuleSchema = z
  .object({
    id: z.string().regex(/^[a-z0-9-]+$/, "must use lowercase letters, numbers, and hyphens"),
    period: z.enum(["day", "week", "month"]),
    content: z.string().trim().min(1).max(500),
    projectId: z.string().trim().min(1),
    dueString: z.string().trim().min(1).max(500).optional(),
    priority: z.union([z.literal(1), z.literal(2), z.literal(3), z.literal(4)]).optional(),
  })
  .strict();

const RulesSchema = z.array(RuleSchema);
export type Rule = z.infer<typeof RuleSchema>;

const ProcessedStateSchema = z
  .object({
    version: z.literal(1),
    processed: z.record(z.string(), z.array(z.string())),
  })
  .strict();
type ProcessedState = z.infer<typeof ProcessedStateSchema>;

type TodoistTask = { content: string; description: string };

const apiBase = "https://api.todoist.com/api/v1";
const dryRun = process.argv.includes("--dry-run");

export const periodKey = (period: Period, now = new Date()): string => {
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");

  if (period === "day") return `${year}-${month}-${day}`;
  if (period === "month") return `${year}-${month}`;

  // ISO week, so the key is stable across the Monday-to-Sunday week.
  const date = new Date(Date.UTC(year, now.getMonth(), now.getDate()));
  const weekday = date.getUTCDay() || 7;
  date.setUTCDate(date.getUTCDate() + 4 - weekday);
  const weekYear = date.getUTCFullYear();
  const firstDayOfYear = new Date(Date.UTC(weekYear, 0, 1));
  const week = Math.ceil(((date.getTime() - firstDayOfYear.getTime()) / 86_400_000 + 1) / 7);
  return `${weekYear}-W${String(week).padStart(2, "0")}`;
};

const marker = (rule: Rule) => `[task-planner:${rule.id}:${periodKey(rule.period)}]`;

const request = async <T>(path: string, init: RequestInit = {}): Promise<T> => {
  const token = await getAccessToken();

  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
      ...init.headers,
    },
  });
  if (!response.ok)
    throw new Error(`Todoist API returned ${response.status}: ${await response.text()}`);
  return response.json() as Promise<T>;
};

const taskExists = async (rule: Rule): Promise<boolean> => {
  const ruleMarker = marker(rule);
  let cursor: string | undefined;
  do {
    const query = new URLSearchParams({ project_id: rule.projectId });
    if (cursor) query.set("cursor", cursor);
    const result = await request<{ results: TodoistTask[]; next_cursor?: string }>(
      `/tasks?${query}`,
    );
    if (
      result.results.some(
        (task) => task.content.includes(ruleMarker) || task.description.includes(ruleMarker),
      )
    ) {
      return true;
    }
    cursor = result.next_cursor;
  } while (cursor);
  return false;
};

const createTask = async (rule: Rule): Promise<void> => {
  const description = `${marker(rule)}\nCreated automatically by task-planner.`;
  if (dryRun) {
    console.log(`[dry-run] Would create: ${rule.content} (${description})`);
    return;
  }
  await request("/tasks", {
    method: "POST",
    body: JSON.stringify({
      content: rule.content,
      description,
      project_id: rule.projectId,
      due_string: rule.dueString,
      priority: rule.priority,
    }),
  });
  console.log(`Created: ${rule.content}`);
};

export const parseRules = (input: unknown): Rule[] => {
  const rules = RulesSchema.parse(input);
  const contents = new Set<string>();
  for (const rule of rules) {
    if (contents.has(rule.content)) throw new Error(`Task text must be unique: ${rule.content}`);
    contents.add(rule.content);
  }
  return rules;
};

export const readRules = async (path = rulesPath()): Promise<Rule[]> =>
  parseRules(JSON.parse(await readFile(path, "utf8")));

export const deletePlan = async (content: string, path = rulesPath()): Promise<Rule> => {
  const rules = await readRules(path);
  const removed = rules.find((rule) => rule.content === content);
  if (!removed) throw new Error(`No active plan has task text: ${content}`);
  await writeFile(
    path,
    `${JSON.stringify(
      rules.filter((rule) => rule.content !== content),
      null,
      2,
    )}\n`,
    "utf8",
  );
  return removed;
};

const readProcessedState = async (path = processedPath()): Promise<ProcessedState> => {
  try {
    return ProcessedStateSchema.parse(JSON.parse(await readFile(path, "utf8")));
  } catch (error: unknown) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") return { version: 1, processed: {} };
    throw error;
  }
};

const markProcessed = (state: ProcessedState, rule: Rule): void => {
  const key = periodKey(rule.period);
  const periods = state.processed[rule.content] ?? [];
  if (!periods.includes(key)) periods.push(key);
  state.processed[rule.content] = periods;
};

const isProcessed = (state: ProcessedState, rule: Rule): boolean =>
  state.processed[rule.content]?.includes(periodKey(rule.period)) ?? false;

const writeProcessedState = async (
  state: ProcessedState,
  path = processedPath(),
): Promise<void> => {
  const temporaryPath = `${path}.tmp`;
  await writeFile(temporaryPath, `${JSON.stringify(state, null, 2)}\n`, "utf8");
  await rename(temporaryPath, path);
};

export const main = async (): Promise<void> => {
  const repository = repositoryPath();
  if (!dryRun) await updateFromRemote(repository);
  const rules = await readRules();
  const processed = await readProcessedState();
  for (const rule of rules) {
    if (isProcessed(processed, rule)) {
      console.log(`Already processed for this ${rule.period}: ${rule.content}`);
      continue;
    }
    if (!dryRun && (await taskExists(rule))) {
      markProcessed(processed, rule);
      await writeProcessedState(processed);
      await publishFile(
        repository,
        "processed.json",
        `task-planner: record ${rule.content} for ${periodKey(rule.period)}`,
      );
      console.log(`Recorded existing task for this ${rule.period}: ${rule.content}`);
      continue;
    }
    await createTask(rule);
    if (!dryRun) {
      markProcessed(processed, rule);
      await writeProcessedState(processed);
      await publishFile(
        repository,
        "processed.json",
        `task-planner: record ${rule.content} for ${periodKey(rule.period)}`,
      );
    }
  }
};

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  void main().catch((error: unknown) => {
    console.error(error instanceof Error ? error.message : error);
    process.exitCode = 1;
  });
}
