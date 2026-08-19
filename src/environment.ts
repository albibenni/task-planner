import { chmod, mkdir, writeFile } from "node:fs/promises";
import { homedir } from "node:os";
import { dirname, join } from "node:path";

export const environmentPath = (): string =>
  process.env.TASK_PLANNER_ENVIRONMENT_PATH?.trim() ||
  join(homedir(), ".config", "task-planner", "environment");

const validateDatabaseUrl = (value: string): string => {
  const url = value.trim();
  if (/\r|\n/.test(url)) throw new Error("The Supabase connection URL must be one line.");
  try {
    const parsed = new URL(url);
    if (!parsed.hostname || !["postgres:", "postgresql:"].includes(parsed.protocol)) {
      throw new Error();
    }
  } catch {
    throw new Error("Enter a valid postgres:// or postgresql:// Supabase connection URL.");
  }
  return url;
};

export const saveDatabaseUrl = async (url: string, path = environmentPath()): Promise<string> => {
  const value = validateDatabaseUrl(url);
  await mkdir(dirname(path), { recursive: true, mode: 0o700 });
  await chmod(dirname(path), 0o700);
  await writeFile(path, `SUPABASE_DB_URL=${value}\n`, { encoding: "utf8", mode: 0o600 });
  await chmod(path, 0o600);
  return path;
};
