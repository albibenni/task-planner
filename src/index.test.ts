import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { deletePlan, parseRules, periodKey, readRules } from "./index.js";

const temporaryDirectories: string[] = [];

afterEach(async () => {
  await Promise.all(temporaryDirectories.splice(0).map((path) => rm(path, { recursive: true })));
});

describe("periodKey", () => {
  it("returns the ISO week across a year boundary", () => {
    expect(periodKey("week", new Date("2027-01-01T12:00:00Z"))).toBe("2026-W53");
  });

  it("returns a local calendar month", () => {
    expect(periodKey("month", new Date("2026-08-18T12:00:00Z"))).toBe("2026-08");
  });
});

describe("parseRules", () => {
  it("accepts a valid rule", () => {
    expect(
      parseRules([{ id: "weekly-review", period: "week", content: "Review", projectId: "123" }]),
    ).toHaveLength(1);
  });

  it("rejects unexpected fields before an API request", () => {
    expect(() =>
      parseRules([
        { id: "unsafe", period: "week", content: "Review", projectId: "123", extra: true },
      ]),
    ).toThrow();
  });

  it("requires task text to be unique across active plans", () => {
    expect(() =>
      parseRules([
        { id: "morning", period: "day", content: "Plan the day", projectId: "123" },
        { id: "evening", period: "day", content: "Plan the day", projectId: "456" },
      ]),
    ).toThrow("Task text must be unique: Plan the day");
  });
});

describe("deletePlan", () => {
  it("removes a plan by its exact unique task text", async () => {
    const directory = await mkdtemp(join(tmpdir(), "task-planner-test-"));
    temporaryDirectories.push(directory);
    const path = join(directory, "rules.json");
    await writeFile(
      path,
      JSON.stringify([
        { id: "morning", period: "day", content: "Plan the day", projectId: "123" },
        { id: "weekly", period: "week", content: "Review the week", projectId: "123" },
      ]),
    );

    await expect(deletePlan("Plan the day", path)).resolves.toMatchObject({ id: "morning" });
    await expect(readRules(path)).resolves.toEqual([
      { id: "weekly", period: "week", content: "Review the week", projectId: "123" },
    ]);
    await expect(readFile(path, "utf8")).resolves.toContain("Review the week");
  });

  it("does not remove a plan when task text does not match exactly", async () => {
    const directory = await mkdtemp(join(tmpdir(), "task-planner-test-"));
    temporaryDirectories.push(directory);
    const path = join(directory, "rules.json");
    await writeFile(
      path,
      JSON.stringify([{ id: "morning", period: "day", content: "Plan the day", projectId: "123" }]),
    );

    await expect(deletePlan("plan the day", path)).rejects.toThrow(
      "No active plan has task text: plan the day",
    );
  });
});
