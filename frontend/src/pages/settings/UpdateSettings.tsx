import { useState } from "react";
import { useTranslation } from "react-i18next";
import { FaCloudDownloadAlt, FaSyncAlt } from "react-icons/fa";

import { ProgressBar } from "@/components/ProgressBar";
import { SectionCard, SettingRow } from "@/components/SettingRow";
import { Switch } from "@/components/Switch";
import { useAppStore } from "@/store/AppStore";
import type { ProxyTestResult, ReleaseInfo } from "@/types/maa";
import { cn } from "@/utils/cn";
import { formatBytes } from "@/utils/format";

const inputClass =
  "h-9 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none focus:border-brand-500/60";

export const UpdateSettings = () => {
  const { t } = useTranslation();
  const {
    config,
    updateConfig,
    core,
    updateStatus,
    checkCoreUpdate,
    startCoreUpdate,
    startResourceUpdate,
    testProxy,
    restartApp,
    openUpdatesDir,
  } = useAppStore();

  const [checking, setChecking] = useState(false);
  const [release, setRelease] = useState<ReleaseInfo | null>(null);
  const [proxyTesting, setProxyTesting] = useState(false);
  const [proxyResult, setProxyResult] = useState<ProxyTestResult | null>(null);

  const busy =
    updateStatus?.stage === "checking" ||
    updateStatus?.stage === "downloading" ||
    updateStatus?.stage === "extracting";
  const pending = Boolean(updateStatus?.needsRestart);

  const updateProxy = (patch: Partial<typeof config.proxy>) => {
    updateConfig((previous) => ({ ...previous, proxy: { ...previous.proxy, ...patch } }));
  };
  const updateUpdate = (patch: Partial<typeof config.update>) => {
    updateConfig((previous) => ({ ...previous, update: { ...previous.update, ...patch } }));
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

      <SectionCard title={t("settings.update.core_title")}>
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
          title={t("settings.update.mirror")}
        />

        <div className="flex flex-wrap items-center gap-2 py-3">
          <button
            className="flex items-center gap-2 rounded-full bg-surface-secondary px-4 py-2 text-xs font-medium text-muted hover:text-foreground disabled:opacity-60"
            disabled={checking || busy || pending}
            onClick={() => {
              setChecking(true);
              void checkCoreUpdate()
                .then(setRelease)
                .finally(() => setChecking(false));
            }}
            type="button"
          >
            <FaSyncAlt className={cn((checking || updateStatus?.stage === "checking") && "animate-spin")} size={10} />
            {t("settings.update.check")}
          </button>
          <button
            className="flex items-center gap-2 rounded-full bg-brand-500 px-4 py-2 text-xs font-semibold text-white hover:bg-brand-600 disabled:opacity-60"
            disabled={busy || pending}
            onClick={() => void startCoreUpdate(config.update.targetVersion || release?.tag || "")}
            type="button"
          >
            <FaCloudDownloadAlt size={11} />
            {t("settings.update.download")}
          </button>
          <button
            className="rounded-full bg-surface-secondary px-4 py-2 text-xs font-medium text-muted hover:text-foreground disabled:opacity-60"
            disabled={busy || pending}
            onClick={() => void startResourceUpdate()}
            type="button"
          >
            {t("settings.update.resource")}
          </button>
          <button
            className="rounded-full bg-surface-secondary px-4 py-2 text-xs font-medium text-muted hover:text-foreground"
            onClick={openUpdatesDir}
            type="button"
          >
            {t("settings.update.open_staging")}
          </button>
        </div>

        {release && (
          <p className="mb-2 rounded-xl bg-surface-secondary/50 px-3 py-2 text-[11px] text-muted">
            {t("settings.update.latest")}: <span className="font-semibold">{release.tag}</span>
            {release.size > 0 ? ` · ${formatBytes(release.size)}` : ""}
            {release.published ? ` · ${release.published}` : ""}
          </p>
        )}

        {updateStatus && updateStatus.stage !== "idle" && (
          <div className="rounded-2xl bg-surface-secondary/50 p-3">
            <div className="flex items-center justify-between gap-2 text-[11px]">
              <span className="font-medium">
                {t(`settings.update.stage.${updateStatus.stage}`, {
                  defaultValue: updateStatus.stage,
                })}
              </span>
              <span className="text-muted">
                {updateStatus.total > 0
                  ? `${formatBytes(updateStatus.downloaded)} / ${formatBytes(updateStatus.total)}`
                  : updateStatus.downloaded > 0
                    ? formatBytes(updateStatus.downloaded)
                    : ""}
              </span>
            </div>
            <ProgressBar className="mt-2" value={updateStatus.progress} />
            <p
              className={cn(
                "mt-2 break-all text-[10px] leading-relaxed",
                updateStatus.stage === "error"
                  ? "text-red-600 dark:text-red-400"
                  : "text-muted",
              )}
            >
              {updateStatus.error || updateStatus.message}
            </p>
            {updateStatus.needsRestart && (
              <button
                className="mt-3 rounded-full bg-brand-500 px-4 py-2 text-xs font-semibold text-white hover:bg-brand-600"
                onClick={() => void restartApp()}
                type="button"
              >
                {t("settings.update.restart")}
              </button>
            )}
          </div>
        )}
      </SectionCard>

      <SectionCard description={t("settings.update.resource_hint")} title={t("settings.update.resource_title")}>
        <p className="text-xs leading-relaxed text-muted">{t("settings.update.resource_note")}</p>
      </SectionCard>
    </div>
  );
};
