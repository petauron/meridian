import { execFileSync } from "node:child_process";
import { resolve } from "node:path";

export const pinnedCenterCommit = "583e2f23ae3fbc91671ed6633b8776dbce4f9b95";

export function reviewedCenterSource() {
  const source = process.env.VASTORA_WEB_SOURCE;
  if (!source) throw new Error("VASTORA_WEB_SOURCE must point to the pinned Vastora web/src checkout");
  const directory = resolve(source);
  const commit = execFileSync("git", ["-C", directory, "rev-parse", "HEAD"], { encoding: "utf8" }).trim();
  if (commit !== pinnedCenterCommit) throw new Error("Meridian UI requires its reviewed Vastora UI dependency revision");
  return directory;
}
