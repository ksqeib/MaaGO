import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { FaPlug, FaShieldAlt, FaSyncAlt } from "react-icons/fa";

import { SectionCard, SettingRow } from "@/components/SettingRow";
import { Switch } from "@/components/Switch";
import { DiagnosticSteps } from "@/components/DiagnosticSteps";
import { useAppStore } from "@/store/AppStore";
import type { ConnectionCandidate, DiagnosticResult } from "@/types/maa";
import { cn } from "@/utils/cn";
import { gameWindowTitle } from "@/utils/client";
import * as api from "bindings/MaaGO/internal/app/service";

const emulators = [
  { value: "auto", zh: "自动", en: "Auto" },
  { value: "mumu", zh: "MuMu 模拟器", en: "MuMu" },
  { value: "ld", zh: "雷电模拟器", en: "LDPlayer" },
  { value: "bluestacks", zh: "蓝叠模拟器", en: "BlueStacks" },
  { value: "nox", zh: "夜神模拟器", en: "Nox" },
  { value: "memu", zh: "逍遥模拟器", en: "MEmu" },
  { value: "custom", zh: "自定义", en: "Custom" },
];

const screencapOptions = [
  { value: "PrintWindow", label: "PrintWindow" },
  { value: "DesktopDupWindow", label: "DesktopDupWindow" },
  { value: "FramePool", label: "FramePool" },
  { value: "ScreenDC", label: "ScreenDC" },
];

const mouseOptions = [
  { value: "PostMessageWithWindowPos", label: "PostMessageWithWindowPos" },
  { value: "SendMessageWithWindowPos", label: "SendMessageWithWindowPos" },
  { value: "PostMessageWithCursorPos", label: "PostMessageWithCursorPos" },
  { value: "SendMessageWithCursorPos", label: "SendMessageWithCursorPos" },
  { value: "PostMessage", label: "PostMessage" },
  { value: "SendMessage", label: "SendMessage" },
  { value: "Seize", label: "Seize" },
];

const keyboardOptions = [
  { value: "SendMessage", label: "SendMessage" },
  { value: "PostMessage", label: "PostMessage" },
  { value: "Seize", label: "Seize" },
];

const defaultWin32 = {
  windowTitle: "",
  screencap: "PrintWindow",
  mouse: "PostMessageWithWindowPos",
  keyboard: "SendMessage",
};

const defaultRDP = {
  host: "",
  port: 3389,
  username: "",
  password: "",
  domain: "",
  width: 1280,
  height: 720,
};

export const ConnectionSettings = () => {
  const { t, i18n } = useTranslation();
  const { config, updateConfig, core, diagnoseConnection, detectConnection, openCoreLogsDir, openDiagnosticsDir } = useAppStore();
  const [testing, setTesting] = useState(false);
  const [detecting, setDetecting] = useState(false);
  const [diagnosis, setDiagnosis] = useState<DiagnosticResult | null>(null);
  const [candidates, setCandidates] = useState<ConnectionCandidate[]>([]);
  const [hasDetected, setHasDetected] = useState(false);
  const [elevated, setElevated] = useState<boolean | null>(null);
  const [elevating, setElevating] = useState(false);
  const isZh = String(i18n.resolvedLanguage || "").startsWith("zh");

  useEffect(() => {
    void api
      .GetPrivilegeStatus()
      .then((status) => setElevated(status.elevated))
      .catch(() => setElevated(null));
  }, []);

  const updateConnection = (patch: Partial<typeof config.connection>) => {
    updateConfig((previous) => ({
      ...previous,
      connection: { ...previous.connection, ...patch },
    }));
  };

  const win32 = config.connection.win32 ?? defaultWin32;
  const rdp = config.connection.rdp ?? defaultRDP;

  const updateWin32 = (patch: Partial<typeof config.connection.win32>) => {
    updateConnection({
      win32: { ...(config.connection.win32 ?? defaultWin32), ...patch },
    });
  };

  const updateRDP = (patch: Partial<typeof defaultRDP>) => {
    updateConnection({
      rdp: { ...(config.connection.rdp ?? defaultRDP), ...patch },
    });
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
        <div className="mt-3 flex flex-wrap gap-2">
          <button
            className="rounded-full bg-surface-secondary px-3 py-1.5 text-[10px] font-medium text-muted hover:text-foreground"
            onClick={openCoreLogsDir}
            type="button"
          >
            {t("settings.core.open_logs")}
          </button>
          <button
            className="rounded-full bg-surface-secondary px-3 py-1.5 text-[10px] font-medium text-muted hover:text-foreground"
            onClick={openDiagnosticsDir}
            type="button"
          >
            {t("settings.core.open_screenshots")}
          </button>
        </div>
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
              <option value="rdp">{t("settings.connection.mode_rdp")}</option>
            </select>
          }
          description={t("settings.connection.mode_hint")}
          title={t("settings.connection.mode")}
        />

        {config.connection.mode === "rdp" ? (
          <>
            <SettingRow
              control={
                <input
                  className="h-9 w-52 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none focus:border-brand-500/60"
                  onChange={(event) => updateRDP({ host: event.target.value })}
                  placeholder="192.168.10.180"
                  value={rdp.host}
                />
              }
              description={t("settings.connection.rdp_host_hint")}
              title={t("settings.connection.rdp_host")}
            />
            <SettingRow
              control={
                <input
                  className="h-9 w-24 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none focus:border-brand-500/60"
                  min={1}
                  onChange={(event) => updateRDP({ port: Number(event.target.value) || 3389 })}
                  type="number"
                  value={rdp.port}
                />
              }
              description={t("settings.connection.rdp_port_hint")}
              title={t("settings.connection.rdp_port")}
            />
            <SettingRow
              control={
                <input
                  autoComplete="off"
                  className="h-9 w-52 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none focus:border-brand-500/60"
                  onChange={(event) => updateRDP({ username: event.target.value })}
                  placeholder="bieqsk"
                  value={rdp.username}
                />
              }
              description={t("settings.connection.rdp_user_hint")}
              title={t("settings.connection.rdp_user")}
            />
            <SettingRow
              control={
                <input
                  autoComplete="new-password"
                  className="h-9 w-52 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none focus:border-brand-500/60"
                  onChange={(event) => updateRDP({ password: event.target.value })}
                  placeholder="••••••••"
                  type="password"
                  value={rdp.password}
                />
              }
              description={t("settings.connection.rdp_password_hint")}
              title={t("settings.connection.rdp_password")}
            />
            <SettingRow
              control={
                <input
                  className="h-9 w-52 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none focus:border-brand-500/60"
                  onChange={(event) => updateRDP({ domain: event.target.value })}
                  placeholder={t("settings.connection.rdp_domain_placeholder")}
                  value={rdp.domain}
                />
              }
              description={t("settings.connection.rdp_domain_hint")}
              title={t("settings.connection.rdp_domain")}
            />
            <SettingRow
              control={
                <select
                  className="h-9 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none"
                  onChange={(event) => {
                    const [width, height] = event.target.value.split("x").map(Number);
                    updateRDP({ width, height });
                  }}
                  value={`${rdp.width}x${rdp.height}`}
                >
                  <option value="1280x720">1280 × 720</option>
                  <option value="1600x900">1600 × 900</option>
                  <option value="1920x1080">1920 × 1080</option>
                </select>
              }
              description={t("settings.connection.rdp_resolution_hint")}
              title={t("settings.connection.rdp_resolution")}
            />
            <p className="py-2 text-[11px] leading-relaxed text-muted">
              {t("settings.connection.rdp_note")}
            </p>
            <div className="flex flex-wrap gap-2 py-2">
              <button
                className="flex items-center gap-2 rounded-full bg-brand-500 px-4 py-2 text-xs font-semibold text-white hover:bg-brand-600 disabled:opacity-60"
                disabled={testing}
                onClick={() => {
                  setTesting(true);
                  void diagnoseConnection(config.connection.address)
                    .then(setDiagnosis)
                    .finally(() => setTesting(false));
                }}
                type="button"
              >
                <FaSyncAlt className={cn(testing && "animate-spin")} size={10} />
                {testing ? t("settings.connection.testing") : t("settings.connection.test")}
              </button>
            </div>
            {diagnosis && <DiagnosticSteps result={diagnosis} running={testing} />}
          </>
        ) : config.connection.mode === "win32" ? (
          <>
            <SettingRow
              control={
                <input
                  className="h-9 w-52 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none focus:border-brand-500/60"
                  onChange={(event) => updateWin32({ windowTitle: event.target.value })}
                  placeholder={gameWindowTitle(config.run.clientType)}
                  value={win32.windowTitle}
                />
              }
              description={t("settings.connection.pc_window_hint")}
              title={t("settings.connection.pc_window")}
            />
            <SettingRow
              control={
                <select
                  className="h-9 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none"
                  onChange={(event) => updateWin32({ screencap: event.target.value })}
                  value={win32.screencap}
                >
                  {screencapOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              }
              description={t("settings.connection.pc_screencap_hint")}
              title={t("settings.connection.pc_screencap")}
            />
            <SettingRow
              control={
                <select
                  className="h-9 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none"
                  onChange={(event) => updateWin32({ mouse: event.target.value })}
                  value={win32.mouse}
                >
                  {mouseOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              }
              description={t("settings.connection.pc_mouse_hint")}
              title={t("settings.connection.pc_mouse")}
            />
            <SettingRow
              control={
                <select
                  className="h-9 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none"
                  onChange={(event) => updateWin32({ keyboard: event.target.value })}
                  value={win32.keyboard}
                >
                  {keyboardOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              }
              description={t("settings.connection.pc_keyboard_hint")}
              title={t("settings.connection.pc_keyboard")}
            />
            <p className="py-2 text-[11px] leading-relaxed text-muted">
              {t("settings.connection.pc_note")}
            </p>
            {elevated === false && (
              <div className="flex flex-wrap items-center gap-3 rounded-xl border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-[11px] leading-relaxed text-amber-700 dark:text-amber-300">
                <FaShieldAlt />
                <span className="min-w-[12rem] flex-1">
                  {t("settings.connection.pc_admin_hint")}
                </span>
                <button
                  className="rounded-full bg-amber-500 px-3 py-1.5 text-[11px] font-semibold text-white hover:bg-amber-600 disabled:opacity-60"
                  disabled={elevating}
                  onClick={() => {
                    setElevating(true);
                    void api.RestartAsAdmin().catch(() => setElevating(false));
                  }}
                  type="button"
                >
                  {elevating
                    ? t("settings.connection.pc_admin_restarting")
                    : t("settings.connection.pc_admin_restart")}
                </button>
              </div>
            )}
            <div className="flex flex-wrap gap-2 py-2">
              <button
                className="flex items-center gap-2 rounded-full bg-brand-500 px-4 py-2 text-xs font-semibold text-white hover:bg-brand-600 disabled:opacity-60"
                disabled={testing}
                onClick={() => {
                  setTesting(true);
                  void diagnoseConnection(config.connection.address)
                    .then(setDiagnosis)
                    .finally(() => setTesting(false));
                }}
                type="button"
              >
                <FaSyncAlt className={cn(testing && "animate-spin")} size={10} />
                {testing ? t("settings.connection.testing") : t("settings.connection.test")}
              </button>
            </div>
            {diagnosis && <DiagnosticSteps result={diagnosis} running={testing} />}
          </>
        ) : (
          <>

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
              void diagnoseConnection(config.connection.address)
                .then(setDiagnosis)
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

        {diagnosis && <DiagnosticSteps result={diagnosis} running={testing} />}

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
                    setDiagnosis(null);
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
          </>
        )}
      </SectionCard>
    </div>
  );
};
