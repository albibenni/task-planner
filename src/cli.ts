import { main } from "./index.js";
import { listProjects, login, logout } from "./oauth.js";

const [command, subcommand] = process.argv.slice(2);

const run = async (): Promise<void> => {
  if (command === "auth" && subcommand === "login") return login();
  if (command === "auth" && subcommand === "logout") return logout();
  if (command === "auth" && subcommand === "projects") return listProjects();
  if (command === "run") return main();
  throw new Error(
    "Use `pnpm auth login`, `pnpm auth projects`, `pnpm auth logout`, or `pnpm run run`.",
  );
};

void run().catch((error: unknown) => {
  console.error(error instanceof Error ? error.message : error);
  process.exitCode = 1;
});
