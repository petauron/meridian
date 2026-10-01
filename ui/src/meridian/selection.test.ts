import { describe, expect, it } from "vitest";
import type { IPQualityCheck } from "@/ip-quality-types";
import type { InstalledAppInstance } from "@/views/installed-apps-model";
import { compareSelection, entryAvailable, matchesPurpose } from "./selection";

function report(score: number, overrides: Partial<IPQualityCheck> = {}): IPQualityCheck {
  return { state: "succeeded", checkedAt: new Date().toISOString(), stale: false,
    report: { address: "192.0.2.1", version: "test", scores: [], services: [
      { name: "ChatGPT", status: "Yes" }, { name: "Netflix", status: "Yes" }, { name: "DisneyPlus", status: "Yes" },
    ] },
    assessment: { score, status: "complete", ipType: "hosting" }, ...overrides,
  } as IPQualityCheck;
}
const choice = (name: string, score: number, ready = true, overrides: Partial<IPQualityCheck> = {}) => ({ name, ready, check: report(score, overrides) });

describe("node selection evidence", () => {
  it("keeps an offline high scorer below ready nodes", () => {
    const values = [choice("offline", 99, false), choice("ready", 60)];
    expect(values.sort((a, b) => compareSelection(a, b, "quality")).map((item) => item.name)).toEqual(["ready", "offline"]);
  });
  it("keeps historical results below current results even before backend expiry refresh", () => {
    const values = [choice("old", 99, true, { checkedAt: "2020-01-01T00:00:00Z" }), choice("current", 60)];
    expect(values.sort((a, b) => compareSelection(a, b, "quality"))[0].name).toBe("current");
  });
  it("does not give incomplete evidence priority over a complete score", () => {
    const conservative = report(90);
    conservative.assessment!.status = "conservative";
    const values = [{ name: "incomplete", ready: true, check: conservative }, choice("complete", 60)];
    expect(values.sort((a, b) => compareSelection(a, b, "quality"))[0].name).toBe("complete");
  });
  it("does not match a changed IP or failed report to a service filter", () => {
    expect(matchesPurpose(report(90, { stale: true }), "ai")).toBe(false);
    expect(matchesPurpose(report(90, { state: "failed" }), "ai")).toBe(false);
    const changed = report(90); changed.assessment!.status = "ip_changed";
    expect(matchesPurpose(changed, "ai")).toBe(false);
  });
  it("requires both full streaming unlocks and excludes app-only ChatGPT", () => {
    const check = report(80);
    expect(matchesPurpose(check, "streaming")).toBe(true);
    check.report!.services[1].status = "Org";
    expect(matchesPurpose(check, "streaming")).toBe(false);
    check.report!.services[0].status = "AppOnly";
    expect(matchesPurpose(check, "ai")).toBe(false);
  });
  it("retains unmeasured nodes in the unfiltered view", () => {
    expect(matchesPurpose(undefined, "all")).toBe(true);
    expect(matchesPurpose(undefined, "ai")).toBe(false);
  });
  it("does not call a guarded or unsynchronized entry ready", () => {
    const instance = { agent: { connected: true, status: "active" }, application: { status: "running", role: "worker", nodeSyncStatus: "ready" }, controller: {}, realityServices: [{ protocols: ["vless"] }], realityPublications: [{ status: "ready" }] } as InstalledAppInstance;
    expect(entryAvailable(instance)).toBe(true);
    instance.application.nodeSyncStatus = "pending";
    expect(entryAvailable(instance)).toBe(false);
    instance.application.nodeSyncStatus = "ready";
    instance.realityServices[0].guardStatus = "hardening";
    expect(entryAvailable(instance)).toBe(false);
    instance.realityServices[0].guardStatus = "action_required";
    expect(entryAvailable(instance)).toBe(false);
  });
});
