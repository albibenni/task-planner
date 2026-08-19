import { afterEach, describe, expect, it, vi } from "vitest";
import { completionFor, run, usage } from "./cli.js";

afterEach(() => vi.restoreAllMocks());

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
