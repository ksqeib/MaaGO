import { useState } from "react";
import { useTranslation } from "react-i18next";
import { FaCloudDownloadAlt, FaSyncAlt } from "react-icons/fa";

import { ProgressBar } from "@/components/ProgressBar";
import { SectionCard, SettingRow } from "@/components/SettingRow";
import { Switch } from "@/components/Switch";
import { useAppStore } from "@/store/AppStore";
import type { ProxyTestResult, ReleaseInfo, UpdateStatus } from "@/types/maa";
import { cn } from "@/utils/cn";
import { formatBytes } from "@/utils/format";

const inputClass =
  "h-9 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none focus:border-brand-500/60";

interface StatusPanelProps {
  status: UpdateStatus | null;
  target: string;
  onRestart: () => void;
}

const StatusPanel = ({ status, target, onRestart }: StatusPanelProps) => {
  const { t } = useTranslation();
  if (!status || status.target !== target || status.stage === "idle") {
    return null;
  }
  return (
    <div className="rounded-2xl bg-surface-secondary/50 p-3">
      <div className="flex items-center justify-between gap-2 text-[11px]">
        <span className="font-medium">
          {t(`settings.update.stage.${status.stage}`, { defaultValue: status.stage })}
        </span>
        <span className="text-muted">
          {status.total > 0
            ? `${formatBytes(status.downloaded)} / ${formatBytes(status.total)}`
            : status.downloaded > 0
              ? formatBytes(status.downloaded)
              : ""}
        </span>
      </div>
      <ProgressBar className="mt-2" value={status.progress} />
      <p
        className={cn(
          "mt-2 break-all text-[10px] leading-relaxed",
          status.stage === "error" ? "text-red-600 dark:text-red-400" : "text-muted",
        )}
      >
        {status.error || status.message}
      </p>
      {status.needsRestart && (
        <button
          className="mt-3 rounded-full bg-brand-500 px-4 py-2 text-xs font-semibold text-white hover:bg-brand-600"
          onClick={onRestart}
          type="button"
        >
          {t("settings.update.restart")}
        </button>
      )}
    </div>
  );
};

export const UpdateSettings = () => {
  const { t } = useTranslation();
  const {
    config,
    updateConfig,
    core,
    updateStatus,
    resourceVersions,
    runtimeStatus,
    checkCoreUpdate,
    startCoreUpdate,
    checkBaseResourceUpdate,
    startBaseResourceUpdate,
    checkResourceOverlayUpdate,
    startResourceOverlayUpdate,
    repairCoreRuntime,
    testProxy,
    restartApp,
    openUpdatesDir,
  } = useAppStore();

  const [checkingCore, setCheckingCore] = useState(false);
  const [checkingBase, setCheckingBase] = useState(false);
  const [checkingOverlay, setCheckingOverlay] = useState(false);
  const [coreRelease, setCoreRelease] = useState<ReleaseInfo | null>(null);
  const [baseRelease, setBaseRelease] = useState<ReleaseInfo | null>(null);
  const [overlayRelease, setOverlayRelease] = useState<ReleaseInfo | null>(null);
  const [proxyTesting, setProxyTesting] = useState(false);
  const [proxyResult, setProxyResult] = useState<ProxyTestResult | null>(null);

  const busy =
    updateStatus?.stage === "checking" ||
    updateStatus?.stage === "downloading" ||
    updateStatus?.stage === "extracting";
  const pending = Boolean(updateStatus?.needsRestart);
  const runtimeAvailable = runtimeStatus?.available ?? true;
  const runtimeVerified = runtimeStatus?.verified ?? true;
  const runtimeOk = runtimeStatus?.ok ?? true;
  const runtimePending = pending && updateStatus?.target === "runtime";
  const runtimeDescription = runtimePending
    ? t("settings.update.runtime_pending")
    : !runtimeAvailable
      ? t("settings.update.runtime_unavailable")
      : !runtimeVerified
        ? t("settings.update.runtime_unverified_hint")
        : runtimeOk
          ? t("settings.update.runtime_ok_hint")
          : runtimeStatus?.missing.length
            ? t("settings.update.runtime_missing", { files: runtimeStatus.missing.join(", ") })
            : runtimeStatus?.mismatched.length
              ? t("settings.update.runtime_mismatch", { files: runtimeStatus.mismatched.join(", ") })
              : runtimeStatus?.message || t("settings.update.runtime_hint");

  const updateProxy = (patch: Partial<typeof config.proxy>) => {
    updateConfig((previous) => ({ ...previous, proxy: { ...previous.proxy, ...patch } }));
  };
  const updateUpdate = (patch: Partial<typeof config.update>) => {
    updateConfig((previous) => ({ ...previous, update: { ...previous.update, ...patch } }));
  };

  const latestLine = (release: ReleaseInfo | null) => {
    if (!release) return null;
    return (
      <p className="mb-2 rounded-xl bg-surface-secondary/50 px-3 py-2 text-[11px] text-muted">
        {t("settings.update.latest")}: <span className="font-semibold">{release.tag}</span>
        {release.size > 0 ? ` · ${formatBytes(release.size)}` : ""}
        {release.published ? ` · ${release.published}` : ""}
      </p>
    );
  };

  return (
    <div className="flex flex-col gap-4">
      <SectionCard
        description={t("settings.update.note")}
        title={t("settings.update.proxy_title")}
      >
        <SettingRow
          control={
            <Switch
              checked={config.proxy.enabled}
              onChange={(enabled) => updateProxy({ enabled })}
            />
          }
          description={t("settings.update.proxy_hint")}
          title={t("settings.update.proxy_enable")}
        />
        <SettingRow
          control={
            <input
              className={cn(inputClass, "w-72")}
              disabled={!config.proxy.enabled}
              onChange={(event) => updateProxy({ url: event.target.value })}
              placeholder="http://127.0.0.1:7890"
              value={config.proxy.url}
            />
          }
          title={t("settings.update.proxy_url")}
        />
        <div className="flex flex-wrap items-center gap-2 py-3">
          <button
            className="flex items-center gap-2 rounded-full bg-brand-500 px-4 py-2 text-xs font-semibold text-white hover:bg-brand-600 disabled:opacity-60"
            disabled={proxyTesting}
            onClick={() => {
              setProxyTesting(true);
              void testProxy(config.proxy.url)
                .then(setProxyResult)
                .finally(() => setProxyTesting(false));
            }}
            type="button"
          >
            <FaSyncAlt className={cn(proxyTesting && "animate-spin")} size={10} />
            {proxyTesting ? t("settings.update.proxy_testing") : t("settings.update.proxy_test")}
          </button>
          {proxyResult && (
            <span
              className={cn(
                "text-[11px]",
                proxyResult.ok
                  ? "text-emerald-600 dark:text-emerald-400"
                  : "text-red-600 dark:text-red-400",
              )}
            >
              {proxyResult.message}
            </span>
          )}
        </div>
      </SectionCard>

      <SectionCard
        description={t("settings.update.core_hint")}
        title={t("settings.update.core_title")}
      >
        <SettingRow
          control={
            <span className="rounded-full bg-surface-secondary px-3 py-1 text-[11px] font-medium">
              {core?.version || "—"}
            </span>
          }
          title={t("settings.update.current")}
        />
        <SettingRow
          control={
            <input
              className={cn(inputClass, "w-40")}
              onChange={(event) => updateUpdate({ targetVersion: event.target.value })}
              placeholder={t("settings.update.target_ph")}
              value={config.update.targetVersion}
            />
          }
          description={t("settings.update.target_hint")}
          title={t("settings.update.target")}
        />
        <SettingRow
          control={
            <input
              className={cn(inputClass, "w-72")}
              onChange={(event) => updateUpdate({ mirror: event.target.value })}
              placeholder="https://your-mirror.example/"
              value={config.update.mirror}
            />
          }
          description={t("settings.update.mirror_hint")}
          title={t("settings.update.mirror")}
        />
        <SettingRow
          control={
            <span
              className={cn(
                "rounded-full px-3 py-1 text-[11px] font-medium",
                !runtimeAvailable || !runtimeVerified
                  ? "bg-surface-secondary text-muted"
                  : runtimeOk
                    ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                    : "bg-amber-500/10 text-amber-700 dark:text-amber-300",
              )}
            >
              {!runtimeAvailable
                ? t("settings.update.runtime_unchecked")
                : !runtimeVerified
                  ? t("settings.update.runtime_unverified")
                  : runtimePending
                    ? t("settings.update.runtime_pending_badge")
                    : runtimeOk
                      ? t("settings.update.runtime_ok")
                      : t("settings.update.runtime_bad")}
            </span>
          }
          description={runtimeDescription}
          title={t("settings.update.runtime_title")}
        />
        <div className="flex flex-wrap items-center gap-2 py-3">
          <button
            className="flex items-center gap-2 rounded-full bg-surface-secondary px-4 py-2 text-xs font-medium text-muted hover:text-foreground disabled:opacity-60"
            disabled={checkingCore || busy || pending}
            onClick={() => {
              setCheckingCore(true);
              void checkCoreUpdate()
                .then(setCoreRelease)
                .finally(() => setCheckingCore(false));
            }}
            type="button"
          >
            <FaSyncAlt className={cn(checkingCore && "animate-spin")} size={10} />
            {t("settings.update.check")}
          </button>
          <button
            className="flex items-center gap-2 rounded-full bg-brand-500 px-4 py-2 text-xs font-semibold text-white hover:bg-brand-600 disabled:opacity-60"
            disabled={busy || pending}
            onClick={() => void startCoreUpdate(config.update.targetVersion || coreRelease?.tag || "")}
            type="button"
          >
            <FaCloudDownloadAlt size={11} />
            {t("settings.update.download")}
          </button>
          {runtimeStatus && !runtimeStatus.ok && !runtimePending && (
            <button
              className="flex items-center gap-2 rounded-full bg-amber-500 px-4 py-2 text-xs font-semibold text-white hover:bg-amber-600 disabled:opacity-60"
              disabled={busy || pending || !runtimeStatus.canRepair}
              onClick={() => void repairCoreRuntime()}
              title={runtimeStatus.canRepair ? runtimeStatus.message : t("settings.update.runtime_no_source")}
              type="button"
            >
              <FaCloudDownloadAlt size={11} />
              {t("settings.update.runtime_repair")}
            </button>
          )}
        </div>
        {latestLine(coreRelease)}
        <StatusPanel onRestart={() => void restartApp()} status={updateStatus} target="core" />
        <StatusPanel onRestart={() => void restartApp()} status={updateStatus} target="runtime" />
      </SectionCard>

      <SectionCard
        description={t("settings.update.base_hint")}
        title={t("settings.update.base_title")}
      >
        <SettingRow
          control={
            <span className="rounded-full bg-surface-secondary px-3 py-1 text-[11px] font-medium">
              {resourceVersions?.base || "—"}
            </span>
          }
          description={t("settings.update.base_current_hint")}
          title={t("settings.update.base_current")}
        />
        <p className="py-2 text-[11px] leading-relaxed text-muted">
          {t("settings.update.base_note")}
        </p>
        <div className="flex flex-wrap items-center gap-2 pb-3">
          <button
            className="flex items-center gap-2 rounded-full bg-surface-secondary px-4 py-2 text-xs font-medium text-muted hover:text-foreground disabled:opacity-60"
            disabled={checkingBase || busy || pending}
            onClick={() => {
              setCheckingBase(true);
              void checkBaseResourceUpdate()
                .then(setBaseRelease)
                .finally(() => setCheckingBase(false));
            }}
            type="button"
          >
            <FaSyncAlt className={cn(checkingBase && "animate-spin")} size={10} />
            {t("settings.update.check")}
          </button>
          <button
            className="flex items-center gap-2 rounded-full bg-brand-500 px-4 py-2 text-xs font-semibold text-white hover:bg-brand-600 disabled:opacity-60"
            disabled={busy || pending}
            onClick={() => void startBaseResourceUpdate()}
            type="button"
          >
            <FaCloudDownloadAlt size={11} />
            {t("settings.update.base_update")}
          </button>
        </div>
        {latestLine(baseRelease)}
        <StatusPanel onRestart={() => void restartApp()} status={updateStatus} target="resource-base" />
      </SectionCard>

      <SectionCard
        description={t("settings.update.overlay_hint")}
        title={t("settings.update.overlay_title")}
      >
        <SettingRow
          control={
            <span className="rounded-full bg-surface-secondary px-3 py-1 text-[11px] font-medium">
              {resourceVersions?.overlay || "—"}
            </span>
          }
          description={
            resourceVersions?.updated
              ? t("settings.update.overlay_updated", { date: resourceVersions.updated })
              : t("settings.update.overlay_current_hint")
          }
          title={t("settings.update.overlay_current")}
        />
        <p className="py-2 text-[11px] leading-relaxed text-muted">
          {t("settings.update.overlay_note")}
        </p>
        <div className="flex flex-wrap items-center gap-2 pb-3">
          <button
            className="flex items-center gap-2 rounded-full bg-surface-secondary px-4 py-2 text-xs font-medium text-muted hover:text-foreground disabled:opacity-60"
            disabled={checkingOverlay || busy || pending}
            onClick={() => {
              setCheckingOverlay(true);
              void checkResourceOverlayUpdate()
                .then(setOverlayRelease)
                .finally(() => setCheckingOverlay(false));
            }}
            type="button"
          >
            <FaSyncAlt className={cn(checkingOverlay && "animate-spin")} size={10} />
            {t("settings.update.check")}
          </button>
          <button
            className="flex items-center gap-2 rounded-full bg-brand-500 px-4 py-2 text-xs font-semibold text-white hover:bg-brand-600 disabled:opacity-60"
            disabled={busy || pending}
            onClick={() => void startResourceOverlayUpdate()}
            type="button"
          >
            <FaCloudDownloadAlt size={11} />
            {t("settings.update.overlay_update")}
          </button>
        </div>
        {latestLine(overlayRelease)}
        <StatusPanel onRestart={() => void restartApp()} status={updateStatus} target="resource-overlay" />
      </SectionCard>

      <SectionCard
        action={
          <button
            className="rounded-full bg-surface-secondary px-3 py-1 text-[11px] font-medium text-muted hover:text-foreground"
            onClick={openUpdatesDir}
            type="button"
          >
            {t("settings.update.open_staging")}
          </button>
        }
        title={t("settings.update.resource_title")}
      >
        <p className="text-xs leading-relaxed text-muted">{t("settings.update.resource_note")}</p>
        {pending && (
          <p className="mt-2 rounded-xl bg-amber-500/10 px-3 py-2 text-[11px] text-amber-700 dark:text-amber-300">
            {t("settings.update.pending_hint")}
          </p>
        )}
      </SectionCard>
    </div>
  );
};
