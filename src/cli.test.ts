import { mkdtemp, readFile, rm, stat } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, describe, expect, it, vi } from "vitest";
import { completionFor, initializeRules, run, usage } from "./cli.js";

const temporaryDirectories: string[] = [];

afterEach(async () => {
  await Promise.all(temporaryDirectories.splice(0).map((path) => rm(path, { recursive: true })));
  vi.restoreAllMocks();
});

describe("CLI help and completions", () => {
  it("shows usage when no command is supplied", async () => {
    const log = vi.spyOn(console, "log").mockImplementation(() => undefined);

    await run([]);

    expect(log).toHaveBeenCalledWith(usage);
  });

  it("generates Bash and Zsh completion definitions", () => {
    expect(completionFor("bash")).toContain("complete -F _task_planner task-planner");
    expect(completionFor("bash")).toContain("login logout projects");
    expect(completionFor("zsh")).toContain("compdef _task_planner task-planner");
    expect(completionFor("zsh")).toContain("--dry-run");
  });

  it("rejects unsupported completion shells", () => {
    expect(() => completionFor("fish")).toThrow("Completion is available for Bash and Zsh.");
  });
});

describe("initializeRules", () => {
  it("creates a private, empty rules file in a missing directory", async () => {
    const directory = await mkdtemp(join(tmpdir(), "task-planner-test-"));
    temporaryDirectories.push(directory);
    const path = join(directory, "config", "rules.json");

    await expect(initializeRules(path)).resolves.toBe(path);
    await expect(readFile(path, "utf8")).resolves.toBe("[]\n");
    expect((await stat(path)).mode & 0o777).toBe(0o600);
  });

  it("does not overwrite an existing rules file", async () => {
    const directory = await mkdtemp(join(tmpdir(), "task-planner-test-"));
    temporaryDirectories.push(directory);
    const path = join(directory, "rules.json");

    await initializeRules(path);

    await expect(initializeRules(path)).rejects.toThrow(`Rules file already exists: ${path}`);
  });
});
