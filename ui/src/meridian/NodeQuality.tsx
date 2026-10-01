import type { IPQualityCheck } from "@/ip-quality-types";
import type { Language } from "@/translations";
import { AssessmentBadge, AssessmentTypeBadge } from "@/views/IPAssessment";
import { IPQualityButton } from "@/views/IPQuality";
import { cleanIPQualityValue, unlockLabel, unlockServiceLabel, unlockServices } from "@/views/ipQualityModel";
import { copy } from "@/views/shared";
import { historicalReport, usableReport } from "./selection";

export function NodeQuality({ nodeId, name, address, check, language, landing = false }: {
  nodeId: string; name: string; address?: string; check?: IPQualityCheck; language: Language; landing?: boolean;
}) {
  const usable = usableReport(check);
  const assessment = usable ? check?.assessment : undefined;
  const historical = usable && historicalReport(check);
  const date = check?.checkedAt ? new Date(check.checkedAt) : undefined;
  const dateLabel = date && Number.isFinite(date.getTime()) ? date.toLocaleDateString(language, { month: "numeric", day: "numeric" }) : "";
  const family = address ? address.includes(":") ? "IPv6" : "IPv4" : "IP";
  return <div className="meridian-quality">
    <div className="meridian-quality-score">
      <AssessmentBadge language={language} assessment={assessment} />
      <span className="text-xs text-muted-foreground">{family}</span>
      <AssessmentTypeBadge language={language} assessment={assessment} checkedAt={check?.checkedAt} />
    </div>
    <div className="min-w-0">
      <div className="meridian-unlocks" aria-label={copy(language, `${name} 解锁状态`, `${name} service availability`)}>{unlockServices.map((serviceName) => {
        const service = usable ? check?.report?.services.find((item) => item.name === serviceName) : undefined;
        const status = cleanIPQualityValue(service?.status).toLowerCase();
        const region = cleanIPQualityValue(service?.regionCode).toUpperCase();
        const exit = cleanIPQualityValue(check?.report?.regionCode).toUpperCase();
        const known = (value: string) => /^[A-Z]{2}$/.test(value) && !["XX", "ZZ", "UN"].includes(value);
        const yes = status === "yes";
        const mismatch = yes && known(region) && known(exit) && region !== exit;
        const matched = yes && known(region) && known(exit) && region === exit;
        const blocked = ["no", "block"].includes(status);
        const limited = ["org", "originals only", "nf.only", "apponly", "webonly", "noprem."].includes(status);
        const label = unlockLabel(language, service?.status);
        const mark = matched ? "✓" : mismatch ? region : yes ? "✓?" : blocked ? "×" : limited ? copy(language, "受限", "Limited") : ["fail", "failed", "error"].includes(status) ? "!" : "?";
        const description = [unlockServiceLabel(serviceName), label, region, mismatch ? copy(language, "与出口地区不同", "Different from exit region") : "", historical ? copy(language, "历史检测", "Historical result") : ""].filter(Boolean).join(" · ");
        return <span key={serviceName} className={matched ? "text-latency-fast" : mismatch || limited ? "text-latency-medium" : blocked ? "text-destructive" : "text-muted-foreground"} title={description} aria-label={description}>
          <span>{unlockServiceLabel(serviceName, true)}</span><strong>{mark}</strong>
        </span>;
      })}</div>
      <div className="meridian-quality-meta text-xs text-muted-foreground">
        <span>{historical ? copy(language, `历史 ${dateLabel} · 需复测`, `Historical ${dateLabel} · Recheck`) : usable ? copy(language, `检测 ${dateLabel}`, `Checked ${dateLabel}`) : copy(language, "尚无有效检测", "No valid check")}</span>
        {landing && address?.includes(":") ? <span>{copy(language, "仅 IPv6 目标", "IPv6 destinations only")}</span> : null}
      </div>
    </div>
    <IPQualityButton nodeId={nodeId} name={name} language={language} egressAddress={address} landingEgress={landing} compact />
  </div>;
}
