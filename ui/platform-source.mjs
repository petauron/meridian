import { execFileSync } from "node:child_process";
import { resolve } from "node:path";

export const pinnedCenterCommit = "e7a314ad7c3e48092686ce10d6215cfe1d6a6421";

export function reviewedCenterSource() {
  const source = process.env.VASTORA_WEB_SOURCE;
  if (!source) throw new Error("VASTORA_WEB_SOURCE must point to the pinned Vastora web/src checkout");
  const directory = resolve(source);
  const commit = execFileSync("git", ["-C", directory, "rev-parse", "HEAD"], { encoding: "utf8" }).trim();
  if (commit !== pinnedCenterCommit) throw new Error("Meridian UI requires its reviewed Vastora UI dependency revision");
  return directory;
}
