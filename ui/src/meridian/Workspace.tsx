import { useState } from "react";
import { EllipsisIcon, SearchIcon } from "lucide-react";
import type { AppWorkspaceProps } from "@/app-workspaces/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useIPQuality } from "@/views/IPQuality";
import { LandingNotice, useLanding } from "@/views/LandingControls";
import { LandingTableRows } from "./LandingNodes";
import { NodeLocation } from "./NodeLocation";
import { RegionFlag } from "./RegionFlag";
import { ApplicationStatus, ApplicationUpdate, ApplicationPrimaryStatus, AccessStatus } from "@/views/apps/InstalledApplicationPrimitives";
import { api } from "@/api";
import type { InstalledAppInstance } from "@/views/installed-apps-model";
import { publicationNeedsAttention, showInstalledNode } from "@/views/installed-apps-model";
import { copy } from "@/views/shared";
import { MeridianLinkTests } from "./LinkTests";
import { SelectControl } from "@/components/SelectControl";
import { ipQualityCheckForAddress } from "@/views/ipQualityModel";
import { compareSelection, entryAvailable, matchesPurpose } from "./selection";
import { NodeQualityCells, UnlockHeaders, nodeTableColumns } from "./NodeQuality";
import { manifest } from "./manifest";

export function MeridianWorkspace({ group, data, language, mutate, onManage, onUpgrade, onClients, showSite }: AppWorkspaceProps) {
  const landing = useLanding();
  const [query, setQuery] = useState("");
  const [page, setPage] = useState<string>(manifest.pages[0].id);
  const quality = useIPQuality();
  const [purpose, setPurpose] = useState("all");
  const [order, setOrder] = useState("quality");
  const checkFor = (instance: InstalledAppInstance) => quality?.error ? undefined : ipQualityCheckForAddress(quality?.checks ?? [], instance.application.nodeId, instance.agent?.publicEgress?.address ?? "");
  const search = query.trim().toLocaleLowerCase();
  const entries = group.instances.filter(showInstalledNode);
  const instances = entries.filter((instance) => !search || [instance.agent?.name, instance.application.nodeId, instance.siteName, ...instance.realityServices.map((service) => service.displayName)].some((value) => value?.toLocaleLowerCase().includes(search))).filter((instance) => matchesPurpose(checkFor(instance), purpose)).sort((a, b) => compareSelection({ ready: entryAvailable(a), check: checkFor(a), name: a.agent?.name ?? a.application.nodeId }, { ready: entryAvailable(b), check: checkFor(b), name: b.agent?.name ?? b.application.nodeId }, order));
  const siteNames = Object.fromEntries(data.agents.map((agent) => [agent.id, data.sites.find((site) => site.id === agent.siteId)?.name ?? ""]));
  const controller = group.controller;
  const controllerWebIDs = new Set(controller?.services.filter((service) => service.protocol === "http" || service.protocol === "https").map((service) => service.id));
  const controllerAttention = controller?.publications.some((publication) => controllerWebIDs.has(publication.serviceId) && publicationNeedsAttention(publication));
  const accounts = manifest.pages.find((item) => item.surface === "manager")!;
  return <section aria-label="Meridian" className="apps-three-xui flex min-w-0 flex-col gap-3" data-app-workspace={manifest.appKey}>
    <header className="flex items-baseline gap-3"><h2 className="text-base font-medium">Meridian</h2><p className="text-xs text-muted-foreground">{copy(language, `${entries.length} 个线路机 · ${landing?.view?.servers.length ?? "—"} 个落地机`, `${entries.length} entry nodes · ${landing?.view?.servers.length ?? "—"} landing nodes`)}</p></header>
    <LandingNotice language={language} />
    <div className="apps-three-xui-toolbar flex flex-wrap items-center gap-3 py-2">
      {page === "nodes" ? <><InputGroup className="w-full sm:w-48"><InputGroupInput type="search" value={query} onChange={(event) => setQuery(event.target.value)} aria-label={copy(language, "搜索节点", "Search nodes")} placeholder={copy(language, "搜索节点…", "Search nodes…")} /><InputGroupAddon><SearchIcon aria-hidden="true" /></InputGroupAddon></InputGroup>
      <SelectControl className="sm:w-40" aria-label={copy(language, "按用途筛选", "Filter by purpose")} value={purpose} onValueChange={setPurpose} options={[
        { value: "all", label: copy(language, "全部用途", "All uses") },
        { value: "ai", label: "ChatGPT" },
        { value: "streaming", label: copy(language, "Netflix + Disney+", "Netflix + Disney+") },
        { value: "residential", label: copy(language, "家宽出口", "Residential exits") },
      ]} />
      <SelectControl className="sm:w-40" aria-label={copy(language, "节点排序", "Node order")} value={order} onValueChange={setOrder} options={[
        { value: "quality", label: copy(language, "IP 质量优先", "IP quality first") },
        { value: "name", label: copy(language, "按名称", "By name") },
      ]} />
      {query || purpose !== "all" ? <Button variant="ghost" size="sm" onClick={() => { setQuery(""); setPurpose("all"); }}>{copy(language, "清除筛选", "Clear filters")}</Button> : null}</> : null}
      {controller ? <div className="ml-auto flex min-w-0 flex-wrap items-center gap-2">{controller.activeChange || controller.application.status !== "running" ? <ApplicationPrimaryStatus instance={controller} language={language} /> : null}{controllerAttention ? <span className="text-xs text-destructive">{copy(language, "入口待处理", "Access needs attention")}</span> : null}<div className="ml-auto flex items-center gap-2"><ApplicationUpdate instance={controller} language={language} onUpgrade={onUpgrade} /><Button disabled={controller.locked} size="sm" variant="secondary" onClick={() => onClients(controller.application)}>{accounts.title[language]}</Button><Button size="icon-sm" variant="ghost" aria-label={copy(language, "管理订阅主机", "Manage subscription controller")} onClick={() => onManage(controller.application)}><EllipsisIcon aria-hidden="true" /></Button></div></div> : null}
    </div>
    <Tabs value={page} onValueChange={(value) => { if (typeof value === "string") setPage(value); }}>
      <TabsList variant="line" aria-label={copy(language, "Meridian 视图", "Meridian views")}>{manifest.pages.filter((item) => item.surface === "tab").map((item) => <TabsTrigger key={item.id} value={item.id}>{item.title[language]}</TabsTrigger>)}</TabsList>
      <TabsContent value="nodes"><div className="mb-2 flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground"><span>{copy(language, "✓ 同区 · US 等地区码为异区 · × 未解锁 · ? 未确认", "✓ Matching region · Region codes differ · × Blocked · ? Unknown")}</span><Button variant="ghost" size="sm" onClick={() => setPage("network")}>{copy(language, "比较线路速度 →", "Compare route speeds →")}</Button></div><div className="meridian-node-list"><Table aria-label={copy(language, "Meridian 节点", "Meridian nodes")} className="apps-instance-table meridian-comparison-table">
        <colgroup><col className="meridian-node-column" /><col className="meridian-quality-column" /><col span={nodeTableColumns - 5} className="meridian-service-column" /><col className="meridian-state-column" /><col className="meridian-connection-column" /><col className="meridian-actions-column" /></colgroup>
        <TableHeader><TableRow><TableHead scope="col">{copy(language, "节点", "Node")}</TableHead><TableHead scope="col" title={copy(language, "分数仅表示出口 IP 质量，不代表速度。", "Scores describe exit IP quality, not speed.")}>{copy(language, "IP 质量", "IP quality")}</TableHead><UnlockHeaders /><TableHead scope="col">{copy(language, "状态", "Status")}</TableHead><TableHead scope="col">{copy(language, "连接", "Connections")}</TableHead><TableHead scope="col"><span className="sr-only">{copy(language, "操作", "Actions")}</span></TableHead></TableRow></TableHeader>
        <TableBody><TableRow className="meridian-node-group"><TableCell colSpan={nodeTableColumns} className="text-xs">{copy(language, "线路机", "Entry nodes")} {instances.length}</TableCell></TableRow>
          {instances.map((instance) => {
            const name = instance.agent?.name ?? instance.application.nodeId;
            const service = instance.realityServices[0];
            const hy2Only = service?.protocols?.includes("hy2") && !service.protocols.includes("vless");
            return <TableRow key={instance.application.id} data-application-id={instance.application.id}>
              <TableCell className="whitespace-normal"><div className="flex items-center gap-2"><RegionFlag code={service?.regionCode} language={language} /><span className="font-medium">{name}</span></div><p className="mt-1 text-xs text-muted-foreground"><NodeLocation regionCode={service?.regionCode} siteName={showSite ? instance.siteName : undefined} language={language} />{service?.protocols?.length ? ` · ${service.protocols.map((protocol) => protocol.toUpperCase()).join(" / ")}` : ""}</p></TableCell>
              <NodeQualityCells nodeId={instance.application.nodeId} name={name} language={language} address={instance.agent?.publicEgress?.address} check={checkFor(instance)} />
              <TableCell className="whitespace-normal">{instance.agent?.connected ? <ApplicationStatus instance={instance} language={language} onUpgrade={onUpgrade} /> : <div className="flex flex-col gap-1"><span className="text-destructive">{copy(language, "节点离线", "Node offline")}</span>{instance.activeChange ? <span className="text-xs text-muted-foreground">{copy(language, "有待处理任务", "Pending operation")}</span> : null}</div>}</TableCell>
              <TableCell className="whitespace-normal">{hy2Only ? <Badge variant="outline">{copy(language, "HY2 已配置", "HY2 configured")}</Badge> : <AccessStatus services={instance.realityServices} publications={instance.realityPublications} language={language} threeXUI />}<div className="mt-1 flex flex-wrap gap-1"><VerifyEntry instance={instance} language={language} mutate={mutate} />{!service ? <Button disabled={instance.locked} variant="outline" size="sm" onClick={() => onManage(instance.application)}>{copy(language, "配置入口", "Configure entry")}</Button> : null}</div></TableCell>
              <TableCell><div className="flex justify-end gap-2"><Button aria-label={copy(language, `管理 ${name} 应用`, `Manage ${name} application`)} className="max-lg:min-h-11 max-lg:min-w-11" size="icon-sm" variant="ghost" onClick={() => onManage(instance.application)}><EllipsisIcon aria-hidden="true" /></Button></div></TableCell>
            </TableRow>;
          })}
          {!instances.length ? <TableRow><TableCell colSpan={nodeTableColumns} className="py-8 text-center text-muted-foreground">{copy(language, "没有匹配的线路机", "No matching entry nodes")}</TableCell></TableRow> : null}
          <LandingTableRows language={language} search={search} purpose={purpose} order={order} siteNames={siteNames} data={data} mutate={mutate} />
        </TableBody>
      </Table></div></TabsContent>
      <TabsContent value="network"><MeridianLinkTests instances={entries} language={language} /></TabsContent>
    </Tabs>
  </section>;
}

function VerifyEntry({ instance, language, mutate }: Pick<AppWorkspaceProps, "language" | "mutate"> & { instance: InstalledAppInstance }) {
  const [busy, setBusy] = useState(false);
  const publication = instance.realityPublications.find((value) => value.status !== "ready" && value.status !== "stopped");
  const protocols = instance.realityServices[0]?.protocols;
  if (!publication || protocols?.includes("hy2") && !protocols.includes("vless")) return null;
  const verify = async () => {
    if (busy || instance.locked) return;
    setBusy(true);
    try { await mutate(() => api.verifyPublication(publication.id), copy(language, "入口检查已完成。", "Access point checked.")); }
    catch { /* The platform mutation notice reports failures. */ }
    finally { setBusy(false); }
  };
  return <Button disabled={busy || instance.locked || !instance.agent?.connected} size="sm" variant="outline" onClick={() => void verify()}>{copy(language, "检查", "Check")}</Button>;
}
