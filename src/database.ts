import postgres from "postgres";
import { savedDatabaseUrl } from "./environment.js";
import type { Plan } from "./index.js";

export type Database = ReturnType<typeof postgres>;

const databaseUrl = (): string => {
  const url = process.env.SUPABASE_DB_URL?.trim() || savedDatabaseUrl();
  if (!url) throw new Error("SUPABASE_DB_URL is required. Run `task-planner db setup` first.");
  return url;
};

export const connectDatabase = (url = databaseUrl()): Database =>
  postgres(url, { max: 1, prepare: false });

export const setupDatabase = async (database: Database): Promise<void> => {
  await database`
    create table if not exists task_planner_plans (
      content text primary key,
      id text not null unique,
      period text not null check (period in ('day', 'week', 'month')),
      project_id text not null,
      due_string text,
      priority smallint check (priority between 1 and 4),
      created_at timestamptz not null default now()
    )
  `;
  await database`
    create table if not exists task_planner_runs (
      content text not null references task_planner_plans(content) on delete cascade,
      period_key text not null,
      status text not null check (status in ('claimed', 'created')),
      claimed_at timestamptz not null default now(),
      created_at timestamptz,
      primary key (content, period_key)
    )
  `;
};

export const listPlans = async (database: Database): Promise<Plan[]> =>
  (
    await database<Plan[]>`
    select id, period, content, project_id as "projectId", due_string as "dueString", priority
    from task_planner_plans
    order by content
  `
  ).map((plan) => ({
    ...plan,
    dueString: plan.dueString ?? undefined,
    priority: plan.priority ?? undefined,
  }));

export const addPlan = async (database: Database, plan: Plan): Promise<void> => {
  const result = await database`
    insert into task_planner_plans (content, id, period, project_id, due_string, priority)
    values (${plan.content}, ${plan.id}, ${plan.period}, ${plan.projectId}, ${plan.dueString ?? null}, ${plan.priority ?? null})
    on conflict (content) do nothing
    returning content
  `;
  if (result.count === 0) throw new Error(`An active plan already has task text: ${plan.content}`);
};

export const removePlan = async (database: Database, content: string): Promise<void> => {
  const result = await database`
    delete from task_planner_plans where content = ${content} returning content
  `;
  if (result.count === 0) throw new Error(`No active plan has task text: ${content}`);
};

export const claimRun = async (
  database: Database,
  content: string,
  periodKey: string,
): Promise<boolean> => {
  const inserted = await database`
    insert into task_planner_runs (content, period_key, status)
    values (${content}, ${periodKey}, 'claimed')
    on conflict (content, period_key) do nothing
    returning content
  `;
  if (inserted.count > 0) return true;

  const recovered = await database`
    update task_planner_runs
    set claimed_at = now()
    where content = ${content}
      and period_key = ${periodKey}
      and status = 'claimed'
      and claimed_at < now() - interval '15 minutes'
    returning content
  `;
  return recovered.count > 0;
};

export const markRunCreated = async (
  database: Database,
  content: string,
  periodKey: string,
): Promise<void> => {
  await database`
    update task_planner_runs
    set status = 'created', created_at = now()
    where content = ${content} and period_key = ${periodKey}
  `;
};
