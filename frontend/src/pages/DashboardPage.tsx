import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import {
  FaArrowRight,
  FaCalendarAlt,
  FaCheckCircle,
  FaClock,
  FaDesktop,
  FaPlay,
  FaPlug,
  FaStop,
  FaSyncAlt,
} from "react-icons/fa";
import { useState } from "react";

import { LogConsole } from "@/components/LogConsole";
import { PageHeader } from "@/components/PageHeader";
import { ProgressBar } from "@/components/ProgressBar";
import { RDPPreviewCard } from "@/components/RDPPreviewCard";
import { DiagnosticSteps } from "@/components/DiagnosticSteps";
import { TaskIcon } from "@/components/TaskIcon";
import { TaskToggle } from "@/components/TaskToggle";
import { TASK_META_BY_TYPE } from "@/data/tasks";
import { taskDisplayName } from "@/data/taskSummary";
import { ROUTES } from "@/constants/routes";
import { useAppStore } from "@/store/AppStore";
import type { ConnectionCandidate, DiagnosticResult } from "@/types/maa";
import { cn } from "@/utils/cn";
import { gameWindowTitle } from "@/utils/client";
import { formatCountdown, formatDateTime, formatDuration } from "@/utils/format";
import { pickText } from "@/utils/text";

function StatusPill({ status }: { status: string }) {
  const { t } = useTranslation();
  const tone =
    status === "running"
      ? "bg-sky-500/15 text-sky-600 dark:text-sky-400"
      : status === "success"
        ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400"
        : status === "failed" || status === "stopped"
          ? "bg-amber-500/15 text-amber-600 dark:text-amber-400"
          : "bg-surface-secondary text-muted";
  return (
    <span className={cn("rounded-full px-2.5 py-1 text-[11px] font-bold", tone)}>
      {t(`status.${status}`)}
    </span>
  );
}

export const DashboardPage = () => {
  const { t, i18n } = useTranslation();
  const {
    config,
    updateConfig,
    run,
    logs,
    taskStates,
    schedule,
    core,
    startQueue,
    stopQueue,
    diagnoseConnection,
    detectConnection,
    openLogsDir,
  } = useAppStore();

  const language = i18n.resolvedLanguage;
  const [testing, setTesting] = useState(false);
  const [detecting, setDetecting] = useState(false);
  const [candidates, setCandidates] = useState<ConnectionCandidate[]>([]);
  const [hasDetected, setHasDetected] = useState(false);
  const [diagnosis, setDiagnosis] = useState<DiagnosticResult | null>(null);

  const running = run.status === "running" || run.status === "stopping";
  const enabledCount = config.tasks.filter((task) => task.enabled).length;
  const pcMode = config.connection.mode === "win32";
  const rdpMode = config.connection.mode === "rdp";
  const remoteMode = pcMode || rdpMode;
  const pcWindowTitle =
    config.connection.win32?.windowTitle?.trim() || gameWindowTitle(config.run.clientType);
  const rdpTarget = config.connection.rdp
    ? `${config.connection.rdp.username}@${config.connection.rdp.host}:${config.connection.rdp.port}`
    : "";

  const handleTest = async () => {
    const address = config.connection.address.trim();
    setTesting(true);
    try {
      setDiagnosis(await diagnoseConnection(address));
    } finally {
      setTesting(false);
    }
  };

  const handleDetect = async () => {
    setDetecting(true);
    try {
      setCandidates(await detectConnection());
    } finally {
      setDetecting(false);
      setHasDetected(true);
    }
  };

  const useCandidate = (candidate: ConnectionCandidate) => {
    updateConfig((previous) => ({
      ...previous,
      connection: { ...previous.connection, address: candidate.address },
    }));
    setDiagnosis(null);
  };

  const hasResult = Boolean(diagnosis);
  const connectionOk = Boolean(diagnosis?.ok);

  return (
    <div className="scrollbar-hide h-full w-full overflow-y-auto px-4 pb-4 pt-[var(--content-pt)]">
      <div className="mx-auto flex max-w-[1400px] flex-col gap-4">
        <PageHeader
          description={t("dashboard.subtitle")}
          endContent={
            <button
              className={cn(
                "flex items-center gap-2 rounded-full px-5 py-2.5 text-sm font-semibold transition-colors",
                running
                  ? "bg-red-500/90 text-white hover:bg-red-500"
                  : "bg-brand-500 text-white hover:bg-brand-600",
              )}
              onClick={() => void (running ? stopQueue() : startQueue())}
              type="button"
            >
              {running ? <FaStop size={11} /> : <FaPlay size={11} />}
              {running ? t("dashboard.quick_stop") : t("dashboard.quick_start")}
            </button>
          }
          title={t("dashboard.title")}
        />

        <div className="grid gap-4 lg:grid-cols-3">
          {/* Run status */}
          <section className="launcher-glass rounded-3xl p-5 lg:col-span-2">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <div className="flex items-center gap-2">
                  <StatusPill status={run.status} />
                  <span className="text-xs text-muted">
                    {run.completed}/{run.total || enabledCount}
                  </span>
                </div>
                <h2 className="mt-3 text-xl font-bold">
                  {run.status === "running" || run.status === "stopping"
                    ? run.currentTaskName || t("dashboard.no_current_task")
                    : run.message || t("dashboard.no_current_task")}
                </h2>
                <p className="mt-1 text-xs text-muted">
                  {run.message ||
                    (core?.demo ? t("dashboard.core_demo") : t("dashboard.core_ready"))}
                </p>
              </div>
              <div className="rounded-2xl bg-surface-secondary/60 px-3 py-2 text-right">
                <p className="text-[10px] uppercase tracking-wider text-muted">
                  {t("dashboard.progress")}
                </p>
                <p className="text-xl font-black text-brand-600 dark:text-brand-400">
                  {run.progress}%
                </p>
              </div>
            </div>

            <ProgressBar className="mt-4" value={run.progress} />

            <div className="mt-4 flex flex-wrap items-center gap-2">
              <Link
                className="flex items-center gap-1.5 rounded-full bg-brand-500 px-3.5 py-2 text-xs font-semibold text-white hover:bg-brand-600"
                to={ROUTES.tasks}
              >
                {t("nav.tasks")}
                <FaArrowRight size={9} />
              </Link>
              <Link
                className="flex items-center gap-1.5 rounded-full bg-surface-secondary px-3.5 py-2 text-xs font-medium text-muted hover:text-foreground"
                to={ROUTES.logs}
              >
                {t("dashboard.open_logs")}
              </Link>
              {core && (
                <span
                  className={cn(
                    "rounded-full px-3 py-1.5 text-[11px] font-medium",
                    core.available
                      ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                      : "bg-amber-500/10 text-amber-600 dark:text-amber-400",
                  )}
                >
                  {core.available ? t("settings.core.available") : t("settings.core.missing")}
                </span>
              )}
            </div>
          </section>

          {/* Connection */}
          <section className="launcher-glass rounded-3xl p-5">
            <div className="flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-sm font-bold">
                <FaPlug className="text-brand-500" />
                {t("dashboard.connection_title")}
              </h2>
              <span
                className={cn(
                  "rounded-full px-2 py-0.5 text-[10px] font-semibold",
                  connectionOk
                    ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400"
                    : hasResult
                      ? "bg-red-500/15 text-red-600 dark:text-red-400"
                      : "bg-surface-secondary text-muted",
                )}
              >
                {connectionOk
                  ? t("dashboard.connection_ok")
                  : hasResult
                    ? t("dashboard.connection_failed")
                    : t("dashboard.connection_unknown")}
              </span>
            </div>

            <label className="mt-3 block text-[11px] font-medium text-muted">
              {pcMode
                ? t("dashboard.pc_window")
                : rdpMode
                  ? t("dashboard.rdp_host")
                  : t("dashboard.address")}
            </label>
            {remoteMode ? (
              <div className="mt-1 flex h-9 w-full items-center rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm">
                {pcMode ? pcWindowTitle : rdpTarget}
              </div>
            ) : (
              <input
                className="mt-1 h-9 w-full rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none focus:border-brand-500/60"
                onChange={(event) =>
                  updateConfig((previous) => ({
                    ...previous,
                    connection: { ...previous.connection, address: event.target.value },
                  }))
                }
                placeholder={t("dashboard.address_placeholder")}
                value={config.connection.address}
              />
            )}

            <div className="mt-2 flex gap-2">
              <button
                className="flex flex-1 items-center justify-center gap-1.5 rounded-full bg-brand-500 px-3 py-1.5 text-[11px] font-semibold text-white hover:bg-brand-600 disabled:opacity-60"
                disabled={testing}
                onClick={() => void handleTest()}
                type="button"
              >
                <FaSyncAlt className={cn(testing && "animate-spin")} size={9} />
                {testing ? t("settings.connection.testing") : t("dashboard.test_connection")}
              </button>
              {!remoteMode && (
                <button
                  className="flex flex-1 items-center justify-center gap-1.5 rounded-full bg-surface-secondary px-3 py-1.5 text-[11px] font-medium text-muted hover:text-foreground disabled:opacity-60"
                  disabled={detecting}
                  onClick={() => void handleDetect()}
                  type="button"
                >
                  <FaDesktop size={9} />
                  {t("dashboard.detect_emulators")}
                </button>
              )}
            </div>

            <DiagnosticSteps result={diagnosis} running={testing} />

            {candidates.length > 0 && (
              <div className="mt-2 flex flex-col gap-1">
                {candidates.slice(0, 4).map((candidate) => (
                  <div
                    className="flex items-center justify-between rounded-xl bg-surface-secondary/50 px-2.5 py-1.5"
                    key={candidate.address}
                  >
                    <div className="min-w-0">
                      <p className="truncate text-[11px] font-medium">{candidate.name}</p>
                      <p className="font-mono text-[10px] text-muted">{candidate.address}</p>
                    </div>
                    <button
                      className="rounded-full bg-brand-500/10 px-2 py-0.5 text-[10px] font-semibold text-brand-600 dark:text-brand-400"
                      onClick={() => useCandidate(candidate)}
                      type="button"
                    >
                      {t("dashboard.use_candidate")}
                    </button>
                  </div>
                ))}
              </div>
            )}

            {hasDetected && candidates.length === 0 && !detecting && (
              <p className="mt-2 text-[10px] text-muted">{t("dashboard.no_candidates")}</p>
            )}
          </section>

          {/* RDP preview */}
          {rdpMode && <RDPPreviewCard />}

          {/* Schedule */}
          <section className="launcher-glass rounded-3xl p-5 lg:col-span-1">
            <div className="flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-sm font-bold">
                <FaCalendarAlt className="text-brand-500" />
                {t("dashboard.schedule_title")}
              </h2>
              <span
                className={cn(
                  "rounded-full px-2 py-0.5 text-[10px] font-semibold",
                  schedule?.enabled
                    ? "bg-brand-500/15 text-brand-600 dark:text-brand-400"
                    : "bg-surface-secondary text-muted",
                )}
              >
                {schedule?.enabled ? t("common.enabled") : t("dashboard.schedule_disabled")}
              </span>
            </div>

            <div className="mt-3 flex items-end justify-between">
              <div>
                <p className="text-[10px] uppercase tracking-wider text-muted">
                  {t("dashboard.next_run")}
                </p>
                <p className="mt-0.5 text-lg font-bold">
                  {schedule?.enabled && schedule.nextRun
                    ? schedule.nextRun
                    : t("dashboard.no_next_run")}
                </p>
                {schedule?.enabled && schedule.nextRun && (
                  <p className="text-[11px] text-brand-600 dark:text-brand-400">
                    {formatCountdown(schedule.secondsUntilRun)}
                  </p>
                )}
              </div>
              <FaClock className="mb-1 text-2xl text-brand-500/30" />
            </div>

            <div className="mt-3 flex items-center justify-between text-[11px]">
              <span className="text-muted">{t("dashboard.last_triggered")}</span>
              <span className="font-medium">{schedule?.lastTriggered || t("dashboard.never")}</span>
            </div>

            <Link
              className="mt-3 flex items-center justify-center gap-1.5 rounded-full bg-surface-secondary px-3 py-1.5 text-[11px] font-medium text-muted hover:text-foreground"
              to={ROUTES.settings}
            >
              {t("nav.settings")}
              <FaArrowRight size={8} />
            </Link>
          </section>

          {/* Tasks */}
          <section className="launcher-glass rounded-3xl p-5 lg:col-span-2">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-bold">{t("dashboard.tasks_title")}</h2>
              <span className="text-[11px] text-muted">
                {t("dashboard.task_count", { enabled: enabledCount, total: config.tasks.length })}
              </span>
            </div>

            <div className="mt-3 grid gap-2 sm:grid-cols-2">
              {config.tasks.map((task) => {
                const meta = TASK_META_BY_TYPE[task.type];
                const state = taskStates[task.id]?.status;
                return (
                  <div
                    className={cn(
                      "flex items-center gap-2.5 rounded-2xl border border-transparent bg-surface-secondary/50 px-3 py-2",
                      !task.enabled && "opacity-55",
                    )}
                    key={task.id}
                  >
                    <TaskToggle
                      enabled={task.enabled}
                      onChange={(enabled) =>
                        updateConfig((previous) => ({
                          ...previous,
                          tasks: previous.tasks.map((item) =>
                            item.id === task.id ? { ...item, enabled } : item,
                          ),
                        }))
                      }
                    />
                    <span className="text-brand-600 dark:text-brand-400">
                      <TaskIcon name={meta.icon} size={12} />
                    </span>
                    <span className="min-w-0 flex-1 truncate text-xs font-medium">
                      {taskDisplayName(task, language)}
                    </span>
                    <span
                      className={cn(
                        "h-1.5 w-1.5 rounded-full",
                        state === "running"
                          ? "bg-sky-500"
                          : state === "completed"
                            ? "bg-emerald-500"
                            : state === "error"
                              ? "bg-red-500"
                              : "bg-zinc-300 dark:bg-zinc-600",
                      )}
                    />
                  </div>
                );
              })}
            </div>
          </section>

          {/* Stats */}
          <section className="launcher-glass rounded-3xl p-5">
            <h2 className="flex items-center gap-2 text-sm font-bold">
              <FaCheckCircle className="text-brand-500" />
              {t("dashboard.last_run")}
            </h2>
            {config.lastRun?.finishedAt ? (
              <>
                <p className="mt-2 text-sm font-medium">
                  {t("dashboard.last_run_summary", {
                    completed: config.lastRun.completedTasks,
                    total: config.lastRun.totalTasks,
                    duration: formatDuration(config.lastRun.durationSeconds),
                  })}
                </p>
                <p className="mt-1 text-[11px] text-muted">
                  {formatDateTime(config.lastRun.finishedAt)} ·{" "}
                  {t(`status.${config.lastRun.status}`)}
                </p>
              </>
            ) : (
              <p className="mt-2 text-xs text-muted">{t("dashboard.last_run_never")}</p>
            )}

            <div className="mt-4 rounded-2xl bg-surface-secondary/50 p-3">
              <p className="text-[10px] uppercase tracking-wider text-muted">
                {t("settings.core.title")}
              </p>
              <p className="mt-1 text-xs font-medium">
                {core?.available ? t("settings.core.available") : t("settings.core.missing")}
              </p>
              <p className="mt-1 text-[10px] leading-relaxed text-muted">
                {core?.available
                  ? core.version
                    ? `MaaCore ${core.version}`
                    : t("settings.core.available")
                  : core?.message || t("settings.core.demo")}
              </p>
            </div>
          </section>

          {/* Recent logs */}
          <section className="launcher-glass flex h-[260px] flex-col overflow-hidden rounded-3xl lg:col-span-3">
            <div className="flex items-center justify-between border-b border-border/70 px-4 py-2.5">
              <h2 className="text-sm font-bold">{t("dashboard.logs_title")}</h2>
              <div className="flex items-center gap-2 text-[11px]">
                <button
                  className="text-muted hover:text-foreground"
                  onClick={openLogsDir}
                  type="button"
                >
                  {t("logs.open_dir")}
                </button>
                <Link className="font-medium text-brand-600 dark:text-brand-400" to={ROUTES.logs}>
                  {t("dashboard.open_logs")}
                </Link>
              </div>
            </div>
            <LogConsole
              className="min-h-0 flex-1"
              emptyText={t("dashboard.no_logs")}
              logs={logs.slice(-80)}
            />
          </section>
        </div>
      </div>
    </div>
  );
};
