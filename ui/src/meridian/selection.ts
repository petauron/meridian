import type { IPQualityCheck } from "@/ip-quality-types";
import { publicationNeedsAttention, serviceNeedsAttention } from "@/views/installed-apps-model";
import type { InstalledAppInstance } from "@/views/installed-apps-model";

export function usableReport(check?: IPQualityCheck) {
  return Boolean(check?.state === "succeeded" && check.report && !check.error && !check.stale && check.assessment?.status !== "ip_changed");
}

export function historicalReport(check?: IPQualityCheck) {
  const checkedAt = Date.parse(check?.checkedAt ?? "");
  return check?.assessment?.status === "expired" || !Number.isFinite(checkedAt) || Date.now() - checkedAt > 24 * 60 * 60 * 1000;
}

export function matchesPurpose(check: IPQualityCheck | undefined, purpose: string) {
  if (purpose === "all") return true;
  if (!usableReport(check)) return false;
  if (purpose === "residential") return check?.assessment?.ipType === "residential";
  const services = purpose === "streaming" ? ["Netflix", "DisneyPlus"] : ["ChatGPT"];
  return services.every((name) => check?.report?.services.some((service) => service.name === name && service.status.trim().toLowerCase() === "yes"));
}

export function entryAvailable(instance: InstalledAppInstance) {
  const service = instance.realityServices[0];
  const hy2Only = service?.protocols?.includes("hy2") && !service.protocols.includes("vless");
  return Boolean(instance.agent?.connected && instance.agent.status === "active" && !instance.agent.credentialRevoked && instance.application.status === "running" && !instance.activeChange && service && !instance.realityServices.some((value) => serviceNeedsAttention(value) || value.guardStatus === "pending" || value.guardStatus === "hardening") && !instance.realityPublications.some(publicationNeedsAttention) &&
    (instance.application.role !== "worker" || Boolean(instance.controller && instance.application.nodeSyncStatus === "ready")) &&
    (hy2Only || instance.realityPublications.some((publication) => publication.status === "ready" && !publication.actionRequired && !publication.lastError)));
}

// Keep unavailable nodes last; historical reports never outrank current reports.
export function compareSelection(a: { ready: boolean; check?: IPQualityCheck; name: string }, b: { ready: boolean; check?: IPQualityCheck; name: string }, order: string) {
  if (a.ready !== b.ready) return a.ready ? -1 : 1;
  if (order === "quality") {
    const tier = (check?: IPQualityCheck) => !usableReport(check) ? 0 : historicalReport(check) ? 1 : check?.assessment?.status === "complete" ? 3 : 2;
    const difference = tier(b.check) - tier(a.check);
    if (difference) return difference;
    const score = (check?: IPQualityCheck) => usableReport(check) ? (check?.assessment?.status === "partial" ? -1 : check?.assessment?.score ?? -1) : -1;
    const differenceScore = score(b.check) - score(a.check);
    if (differenceScore) return differenceScore;
  }
  return a.name.localeCompare(b.name);
}
