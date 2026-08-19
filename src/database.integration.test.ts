import { afterAll, afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { run } from "./cli.js";
import {
  addPlan,
  claimRun,
  connectDatabase,
  listPlans,
  markRunCreated,
  removePlan,
  setupDatabase,
} from "./database.js";

const databaseUrl = process.env.TEST_DATABASE_URL;
const database = databaseUrl ? connectDatabase(databaseUrl) : undefined;
const describeDatabase = database ? describe : describe.skip;
const originalDatabaseUrl = process.env.SUPABASE_DB_URL;

const db = () => {
  if (!database) throw new Error("TEST_DATABASE_URL is required for database integration tests.");
  return database;
};

const plan = {
  id: "plan-the-day-6ae5c007",
  content: "Plan the day",
  period: "day" as const,
  projectId: "project-123",
  priority: 2 as const,
};

describeDatabase("Postgres shared scheduling", () => {
  beforeEach(async () => {
    process.env.SUPABASE_DB_URL = databaseUrl;
    await setupDatabase(db());
    await db()`truncate task_planner_runs, task_planner_plans`;
  });

  afterEach(() => vi.restoreAllMocks());

  afterAll(async () => {
    if (originalDatabaseUrl === undefined) delete process.env.SUPABASE_DB_URL;
    else process.env.SUPABASE_DB_URL = originalDatabaseUrl;
    await db().end({ timeout: 5 });
  });

  it("shares active plans and deletes them by exact task text", async () => {
    await addPlan(db(), plan);
    await expect(listPlans(db())).resolves.toEqual([plan]);
    await expect(addPlan(db(), plan)).rejects.toThrow("An active plan already has task text");

    await removePlan(db(), plan.content);
    await expect(listPlans(db())).resolves.toEqual([]);
  });

  it("allows exactly one concurrent scheduler to claim a task period", async () => {
    await addPlan(db(), plan);

    const claims = await Promise.all(
      Array.from({ length: 8 }, () => claimRun(db(), plan.content, "2026-08-19")),
    );

    expect(claims.filter(Boolean)).toHaveLength(1);
    await markRunCreated(db(), plan.content, "2026-08-19");
    await expect(claimRun(db(), plan.content, "2026-08-19")).resolves.toBe(false);
  });

  it("manages shared plans through the CLI", async () => {
    const log = vi.spyOn(console, "log").mockImplementation(() => undefined);

    await run([
      "add",
      "--text",
      plan.content,
      "--period",
      plan.period,
      "--project-id",
      plan.projectId,
      "--priority",
      String(plan.priority),
    ]);
    await run(["plans"]);
    await run(["delete", plan.content]);

    expect(log).toHaveBeenCalledWith(`Added plan: ${plan.content}`);
    expect(log).toHaveBeenCalledWith(`${plan.content} — ${plan.period}`);
    expect(log).toHaveBeenCalledWith(`Stopped plan: ${plan.content}`);
    await expect(listPlans(db())).resolves.toEqual([]);
  });
});
