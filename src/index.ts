import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { z } from "zod";
import { claimRun, connectDatabase, listPlans, markRunCreated, setupDatabase } from "./database.js";
import { getAccessToken } from "./oauth.js";

type Period = "day" | "week" | "month";

export const PlanSchema = z
  .object({
    id: z.string().regex(/^[a-z0-9-]+$/, "must use lowercase letters, numbers, and hyphens"),
    period: z.enum(["day", "week", "month"]),
    content: z.string().trim().min(1).max(500),
    projectId: z.string().trim().min(1),
    dueString: z.string().trim().min(1).max(500).optional(),
    priority: z.union([z.literal(1), z.literal(2), z.literal(3), z.literal(4)]).optional(),
  })
  .strict();

const PlansSchema = z.array(PlanSchema);
export type Plan = z.infer<typeof PlanSchema>;
type TodoistTask = { content: string; description: string };

const apiBase = "https://api.todoist.com/api/v1";
const dryRun = process.argv.includes("--dry-run");

export const periodKey = (period: Period, now = new Date()): string => {
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  if (period === "day") return `${year}-${month}-${day}`;
  if (period === "month") return `${year}-${month}`;
  const date = new Date(Date.UTC(year, now.getMonth(), now.getDate()));
  const weekday = date.getUTCDay() || 7;
  date.setUTCDate(date.getUTCDate() + 4 - weekday);
  const weekYear = date.getUTCFullYear();
  const firstDayOfYear = new Date(Date.UTC(weekYear, 0, 1));
  const week = Math.ceil(((date.getTime() - firstDayOfYear.getTime()) / 86_400_000 + 1) / 7);
  return `${weekYear}-W${String(week).padStart(2, "0")}`;
};

const marker = (plan: Plan) => `[task-planner:${plan.id}:${periodKey(plan.period)}]`;

const request = async <T>(path: string, init: RequestInit = {}): Promise<T> => {
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    headers: {
      Authorization: `Bearer ${await getAccessToken()}`,
      "Content-Type": "application/json",
      ...init.headers,
    },
  });
  if (!response.ok)
    throw new Error(`Todoist API returned ${response.status}: ${await response.text()}`);
  return response.json() as Promise<T>;
};

const taskExists = async (plan: Plan): Promise<boolean> => {
  const ruleMarker = marker(plan);
  let cursor: string | undefined;
  do {
    const query = new URLSearchParams({ project_id: plan.projectId });
    if (cursor) query.set("cursor", cursor);
    const result = await request<{ results: TodoistTask[]; next_cursor?: string }>(
      `/tasks?${query}`,
    );
    if (
      result.results.some(
        (task) => task.content.includes(ruleMarker) || task.description.includes(ruleMarker),
      )
    )
      return true;
    cursor = result.next_cursor;
  } while (cursor);
  return false;
};

const createTask = async (plan: Plan): Promise<void> => {
  const description = `${marker(plan)}\nCreated automatically by task-planner.`;
  if (dryRun) return console.log(`[dry-run] Would create: ${plan.content} (${description})`);
  await request("/tasks", {
    method: "POST",
    body: JSON.stringify({
      content: plan.content,
      description,
      project_id: plan.projectId,
      due_string: plan.dueString,
      priority: plan.priority,
    }),
  });
  console.log(`Created: ${plan.content}`);
};

export const parsePlans = (input: unknown): Plan[] => {
  const plans = PlansSchema.parse(input);
  const contents = new Set<string>();
  for (const plan of plans) {
    if (contents.has(plan.content)) throw new Error(`Task text must be unique: ${plan.content}`);
    contents.add(plan.content);
  }
  return plans;
};

export const main = async (): Promise<void> => {
  const database = connectDatabase();
  try {
    await setupDatabase(database);
    for (const plan of await listPlans(database)) {
      const key = periodKey(plan.period);
      if (dryRun) {
        await createTask(plan);
        continue;
      }
      if (!(await claimRun(database, plan.content, key))) {
        console.log(`Already claimed for this ${plan.period}: ${plan.content}`);
        continue;
      }
      if (await taskExists(plan))
        console.log(`Recorded existing task for this ${plan.period}: ${plan.content}`);
      else await createTask(plan);
      await markRunCreated(database, plan.content, key);
    }
  } finally {
    await database.end({ timeout: 5 });
  }
};

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  void main().catch((error: unknown) => {
    console.error(error instanceof Error ? error.message : error);
    process.exitCode = 1;
  });
}
