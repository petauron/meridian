import type { IPQualityCheck } from "@/ip-quality-types";
import { TableCell, TableHead } from "@/components/ui/table";
import type { Language } from "@/translations";
import { AssessmentBadge, AssessmentTypeBadge } from "@/views/IPAssessment";
import { IPQualityButton } from "@/views/IPQuality";
import { cleanIPQualityValue, unlockLabel, unlockServiceLabel, unlockServices } from "@/views/ipQualityModel";
import { copy } from "@/views/shared";
import { historicalReport, usableReport } from "./selection";

export const nodeTableColumns = 5 + unlockServices.length;

export function UnlockHeaders() {
  return <>{unlockServices.map((name) => <TableHead key={name} scope="col" className="meridian-service-cell" title={unlockServiceLabel(name)}>{name === "ChatGPT" || name === "AmazonPrimeVideo" ? unlockServiceLabel(name, true) : unlockServiceLabel(name)}</TableHead>)}</>;
}

export function NodeQualityCells({ nodeId, name, address, check, language, landing = false }: {
  nodeId: string; name: string; address?: string; check?: IPQualityCheck; language: Language; landing?: boolean;
}) {
  const usable = usableReport(check);
  const assessment = usable ? check?.assessment : undefined;
  const historical = usable && historicalReport(check);
  const date = check?.checkedAt ? new Date(check.checkedAt) : undefined;
  const dateLabel = date && Number.isFinite(date.getTime()) ? date.toLocaleDateString(language, { month: "numeric", day: "numeric" }) : "";
  const family = address ? address.includes(":") ? "IPv6" : "IPv4" : "IP";
  const descriptionDate = historical ? copy(language, `历史检测 ${dateLabel} · 建议复测`, `Historical ${dateLabel} · Recheck`) : usable ? copy(language, `检测 ${dateLabel}`, `Checked ${dateLabel}`) : copy(language, "尚无有效检测", "No valid check");
  return <>
    <TableCell className="meridian-quality-cell">
      <div className="meridian-quality-score">
        <AssessmentBadge language={language} assessment={assessment} />
        <AssessmentTypeBadge language={language} assessment={assessment} checkedAt={check?.checkedAt} />
        <IPQualityButton nodeId={nodeId} name={name} language={language} egressAddress={address} landingEgress={landing} compact />
      </div>
      <div className="meridian-quality-meta text-muted-foreground" title={descriptionDate}>
        {family} · {usable ? `${dateLabel}${historical ? copy(language, " 历史", " Past") : ""}` : copy(language, "待检测", "Pending")}
      </div>
      {landing && address?.includes(":") ? <div className="meridian-quality-meta text-muted-foreground">{copy(language, "仅 IPv6 目标", "IPv6 targets only")}</div> : null}
    </TableCell>
    {unlockServices.map((serviceName) => {
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
        return <TableCell key={serviceName} className="meridian-service-cell">
          <span className={matched ? "text-latency-fast" : mismatch || limited ? "text-latency-medium" : blocked ? "text-destructive" : "text-muted-foreground"} title={description} aria-label={`${name} · ${description}`}><strong>{mark}</strong></span>
        </TableCell>;
    })}
  </>;
}
