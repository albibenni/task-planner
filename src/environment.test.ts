import { mkdtemp, readFile, rm, stat } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { saveDatabaseUrl } from "./environment.js";

const temporaryDirectories: string[] = [];

afterEach(async () => {
  await Promise.all(temporaryDirectories.splice(0).map((path) => rm(path, { recursive: true })));
});

describe("saveDatabaseUrl", () => {
  it("writes a private scheduler environment file", async () => {
    const directory = await mkdtemp(join(tmpdir(), "task-planner-test-"));
    temporaryDirectories.push(directory);
    const path = join(directory, "config", "environment");
    const url = "postgres://postgres.example:password@pooler.example.com:5432/postgres";

    await expect(saveDatabaseUrl(url, path)).resolves.toBe(path);
    await expect(readFile(path, "utf8")).resolves.toBe(`SUPABASE_DB_URL=${url}\n`);
    expect((await stat(path)).mode & 0o777).toBe(0o600);
  });

  it("rejects invalid and multi-line connection URLs", async () => {
    await expect(saveDatabaseUrl("https://example.com", join(tmpdir(), "unused"))).rejects.toThrow(
      "Enter a valid postgres:// or postgresql:// Supabase connection URL.",
    );
    await expect(
      saveDatabaseUrl("postgres://example.com\nvalue", join(tmpdir(), "unused")),
    ).rejects.toThrow("The Supabase connection URL must be one line.");
  });
});
