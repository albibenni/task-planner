import { join, resolve } from "node:path";

export const repositoryPath = (): string =>
  resolve(process.env.TASK_PLANNER_REPOSITORY?.trim() || process.cwd());

export const defaultRulesPath = (): string => join(repositoryPath(), "rules.json");

export const rulesPath = (): string =>
  process.env.TASK_PLANNER_RULES_PATH?.trim() || defaultRulesPath();

export const processedPath = (): string => join(repositoryPath(), "processed.json");
