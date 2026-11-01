import { useState } from "react";
import { useTranslation } from "react-i18next";
import { FaPlug, FaSyncAlt } from "react-icons/fa";

import { SectionCard, SettingRow } from "@/components/SettingRow";
import { Switch } from "@/components/Switch";
import { useAppStore } from "@/store/AppStore";
import type { ConnectionCandidate, ConnectionTestResult } from "@/types/maa";
import { cn } from "@/utils/cn";

const emulators = [
  { value: "auto", zh: "自动", en: "Auto" },
  { value: "mumu", zh: "MuMu 模拟器", en: "MuMu" },
  { value: "ld", zh: "雷电模拟器", en: "LDPlayer" },
  { value: "bluestacks", zh: "蓝叠模拟器", en: "BlueStacks" },
  { value: "nox", zh: "夜神模拟器", en: "Nox" },
  { value: "memu", zh: "逍遥模拟器", en: "MEmu" },
  { value: "custom", zh: "自定义", en: "Custom" },
];

export const ConnectionSettings = () => {
  const { t, i18n } = useTranslation();
  const { config, updateConfig, core, testConnection, detectConnection } = useAppStore();
  const [testing, setTesting] = useState(false);
  const [detecting, setDetecting] = useState(false);
  const [result, setResult] = useState<ConnectionTestResult | null>(null);
  const [candidates, setCandidates] = useState<ConnectionCandidate[]>([]);
  const [hasDetected, setHasDetected] = useState(false);
  const isZh = String(i18n.resolvedLanguage || "").startsWith("zh");

  const updateConnection = (patch: Partial<typeof config.connection>) => {
    updateConfig((previous) => ({
      ...previous,
      connection: { ...previous.connection, ...patch },
    }));
  };

  return (
    <div className="flex flex-col gap-4">
      <SectionCard
        action={
          <span
            className={cn(
              "rounded-full px-2.5 py-1 text-[10px] font-semibold",
              core?.available
                ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400"
                : "bg-amber-500/15 text-amber-600 dark:text-amber-400",
            )}
          >
            {core?.available ? t("settings.core.available") : t("settings.core.missing")}
          </span>
        }
        title={t("settings.core.title")}
      >
        <p className="text-xs leading-relaxed text-muted">
          {core?.available
            ? core.version
              ? `MaaCore ${core.version}`
              : t("settings.core.available")
            : core?.message || t("settings.core.demo")}
        </p>
        {core?.path && (
          <p className="mt-2 break-all rounded-xl bg-surface-secondary/60 px-3 py-2 font-mono text-[10px] text-muted">
            {core.path}
          </p>
        )}
      </SectionCard>

      <SectionCard title={t("settings.tabs.connection")}>
        <SettingRow
          control={
            <select
              className="h-9 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none"
              onChange={(event) => updateConnection({ mode: event.target.value })}
              value={config.connection.mode}
            >
              <option value="adb">{t("settings.connection.mode_adb")}</option>
              <option value="win32">{t("settings.connection.mode_win32")}</option>
            </select>
          }
          description={t("settings.connection.address_hint")}
          title={t("settings.connection.mode")}
        />

        <SettingRow
          control={
            <Switch
              checked={config.connection.autoDetect}
              onChange={(autoDetect) => updateConnection({ autoDetect })}
            />
          }
          title={t("settings.connection.auto_detect")}
        />

        <SettingRow
          control={
            <input
              className="h-9 w-52 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none focus:border-brand-500/60"
              onChange={(event) => updateConnection({ address: event.target.value })}
              placeholder="127.0.0.1:16384"
              value={config.connection.address}
            />
          }
          description={t("settings.connection.address_hint")}
          title={t("settings.connection.address")}
        />

        <div className="flex flex-wrap gap-2 py-3">
          <button
            className="flex items-center gap-2 rounded-full bg-brand-500 px-4 py-2 text-xs font-semibold text-white hover:bg-brand-600 disabled:opacity-60"
            disabled={testing}
            onClick={() => {
              setTesting(true);
              void testConnection(config.connection.address)
                .then(setResult)
                .finally(() => setTesting(false));
            }}
            type="button"
          >
            <FaSyncAlt className={cn(testing && "animate-spin")} size={10} />
            {testing ? t("settings.connection.testing") : t("settings.connection.test")}
          </button>
          <button
            className="flex items-center gap-2 rounded-full bg-surface-secondary px-4 py-2 text-xs font-medium text-muted hover:text-foreground disabled:opacity-60"
            disabled={detecting}
            onClick={() => {
              setDetecting(true);
              void detectConnection()
                .then(setCandidates)
                .finally(() => {
                  setDetecting(false);
                  setHasDetected(true);
                });
            }}
            type="button"
          >
            <FaPlug size={10} />
            {t("settings.connection.detect")}
          </button>
        </div>

        {result && (
          <p
            className={cn(
              "rounded-xl px-3 py-2 text-[11px]",
              result.ok
                ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
                : "bg-red-500/10 text-red-700 dark:text-red-300",
            )}
          >
            {result.address} · {result.message}
          </p>
        )}

        {candidates.length > 0 && (
          <div className="mt-2 flex flex-col gap-1.5">
            <p className="text-[11px] font-medium text-muted">
              {t("settings.connection.candidates")}
            </p>
            {candidates.map((candidate) => (
              <div
                className="flex items-center justify-between rounded-xl bg-surface-secondary/50 px-3 py-2"
                key={candidate.address}
              >
                <div>
                  <p className="text-xs font-medium">{candidate.name}</p>
                  <p className="font-mono text-[10px] text-muted">{candidate.address}</p>
                </div>
                <button
                  className="rounded-full bg-brand-500/10 px-3 py-1 text-[10px] font-semibold text-brand-600 dark:text-brand-400"
                  onClick={() => {
                    updateConnection({ address: candidate.address, autoDetect: true });
                    setResult({
                      ok: true,
                      address: candidate.address,
                      latencyMs: candidate.latencyMs,
                      message: candidate.name,
                    });
                  }}
                  type="button"
                >
                  {t("settings.connection.use")}
                </button>
              </div>
            ))}
          </div>
        )}

        {hasDetected && candidates.length === 0 && !detecting && (
          <p className="text-[10px] text-muted">{t("settings.connection.no_candidates")}</p>
        )}

        <SettingRow
          control={
            <select
              className="h-9 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none"
              onChange={(event) => updateConnection({ emulator: event.target.value })}
              value={config.connection.emulator}
            >
              {emulators.map((emulator) => (
                <option key={emulator.value} value={emulator.value}>
                  {isZh ? emulator.zh : emulator.en}
                </option>
              ))}
            </select>
          }
          title={t("settings.connection.emulator")}
        />

        <SettingRow
          control={
            <input
              className="h-9 w-52 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none focus:border-brand-500/60"
              onChange={(event) => updateConnection({ adbPath: event.target.value })}
              placeholder={t("settings.connection.adb_path_placeholder")}
              value={config.connection.adbPath}
            />
          }
          title={t("settings.connection.adb_path")}
        />

        <SettingRow
          control={
            <select
              className="h-9 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none"
              onChange={(event) => updateConnection({ touchMode: event.target.value })}
              value={config.connection.touchMode}
            >
              <option value="minitouch">minitouch</option>
              <option value="maatouch">MaaTouch</option>
              <option value="adb">ADB</option>
            </select>
          }
          title={t("settings.connection.touch_mode")}
        />

        <SettingRow
          control={
            <Switch
              checked={config.connection.screenshotEnhance}
              onChange={(screenshotEnhance) => updateConnection({ screenshotEnhance })}
            />
          }
          title={t("settings.connection.screenshot_enhance")}
        />
      </SectionCard>
    </div>
  );
};
