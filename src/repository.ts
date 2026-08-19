import { execFile } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);

const runGit = async (repository: string, args: string[]): Promise<void> => {
  try {
    await execFileAsync("git", args, { cwd: repository });
  } catch (error: unknown) {
    const detail = error instanceof Error ? error.message : String(error);
    throw new Error(`Git ${args.join(" ")} failed in ${repository}: ${detail}`);
  }
};

export const updateFromRemote = async (repository: string): Promise<void> => {
  await runGit(repository, ["pull", "--ff-only"]);
};

export const publishFile = async (
  repository: string,
  relativeProcessedPath: string,
  message: string,
): Promise<void> => {
  await runGit(repository, ["add", "--", relativeProcessedPath]);
  await runGit(repository, ["commit", "--only", "-m", message, "--", relativeProcessedPath]);
  await runGit(repository, ["push"]);
};
