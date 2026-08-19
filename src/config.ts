import { homedir, platform } from "node:os";
import { join } from "node:path";

const configDirectory = (): string => {
  if (platform() === "darwin") return join(homedir(), "Library", "Application Support");
  return process.env.XDG_CONFIG_HOME?.trim() || join(homedir(), ".config");
};

export const defaultRulesPath = (): string => join(configDirectory(), "task-planner", "rules.json");

export const rulesPath = (): string =>
  process.env.TASK_PLANNER_RULES_PATH?.trim() || defaultRulesPath();
