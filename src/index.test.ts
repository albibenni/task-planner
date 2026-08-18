import { describe, expect, it } from "vitest";
import { parseRules, periodKey } from "./index.js";

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
});
