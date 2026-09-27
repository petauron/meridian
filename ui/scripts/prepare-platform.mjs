import { lstatSync, realpathSync, symlinkSync, unlinkSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { reviewedCenterSource } from "../platform-source.mjs";

const target = reviewedCenterSource();
const link = fileURLToPath(new URL("../.center-src", import.meta.url));
try {
  const current = lstatSync(link);
  if (!current.isSymbolicLink()) throw new Error(".center-src must be a generated symlink");
  try {
    if (realpathSync(link) === realpathSync(target)) process.exit(0);
  } catch { /* Replace a broken generated symlink. */ }
  unlinkSync(link);
} catch (error) {
  if (error.code !== "ENOENT") throw error;
}
symlinkSync(target, link, "dir");
