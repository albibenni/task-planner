import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { z } from "zod";
import { getAccessToken } from "./oauth.js";

type Period = "day" | "week" | "month";

const RuleSchema = z
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
type Rule = z.infer<typeof RuleSchema>;

type TodoistTask = { content: string; description: string };

const root = fileURLToPath(new URL("../", import.meta.url));
const rulesPath = resolve(root, "rules.json");
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

export const parseRules = (input: unknown): Rule[] => RulesSchema.parse(input);

export const main = async (): Promise<void> => {
  const rules = parseRules(JSON.parse(await readFile(rulesPath, "utf8")));
  for (const rule of rules) {
    if (!dryRun && (await taskExists(rule))) {
      console.log(`Already created for this ${rule.period}: ${rule.id}`);
      continue;
    }
    await createTask(rule);
  }
};

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  void main().catch((error: unknown) => {
    console.error(error instanceof Error ? error.message : error);
    process.exitCode = 1;
  });
}
