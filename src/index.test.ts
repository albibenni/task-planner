import { describe, expect, it } from "vitest";
import { parsePlans, periodKey } from "./index.js";

describe("periodKey", () => {
  it("returns the ISO week across a year boundary", () => {
    expect(periodKey("week", new Date("2027-01-01T12:00:00Z"))).toBe("2026-W53");
  });

  it("returns a local calendar month", () => {
    expect(periodKey("month", new Date("2026-08-18T12:00:00Z"))).toBe("2026-08");
  });
});

describe("parsePlans", () => {
  it("accepts a valid rule", () => {
    expect(
      parsePlans([{ id: "weekly-review", period: "week", content: "Review", projectId: "123" }]),
    ).toHaveLength(1);
  });

  it("rejects unexpected fields before an API request", () => {
    expect(() =>
      parsePlans([
        { id: "unsafe", period: "week", content: "Review", projectId: "123", extra: true },
      ]),
    ).toThrow();
  });

  it("requires task text to be unique across active plans", () => {
    expect(() =>
      parsePlans([
        { id: "morning", period: "day", content: "Plan the day", projectId: "123" },
        { id: "evening", period: "day", content: "Plan the day", projectId: "456" },
      ]),
    ).toThrow("Task text must be unique: Plan the day");
  });
});
