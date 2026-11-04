import { Events } from "@wailsio/runtime";
import { useTheme } from "next-themes";
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";

import * as api from "bindings/MaaGO/internal/app/service";
import { DEFAULT_CONFIG, DEFAULT_RUN_STATE } from "@/data/defaults";
import type {
  AppConfig,
  AppInfo,
  ConnectionCandidate,
  CoreStatus,
  CoreRuntimeStatus,
  DiagnosticResult,
  LogEntry,
  ProxyTestResult,
  ReleaseInfo,
  ResourceVersions,
  RunState,
  ScheduleStatus,
  TaskState,
  UpdateStatus,
} from "@/types/maa";

type SaveState = "idle" | "saving" | "saved" | "error";

interface AppStoreValue {
  config: AppConfig;
  updateConfig: (updater: (previous: AppConfig) => AppConfig) => void;
  ready: boolean;
  saveState: SaveState;
  run: RunState;
  logs: LogEntry[];
  taskStates: Record<string, TaskState>;
  schedule: ScheduleStatus | null;
  core: CoreStatus | null;
  appInfo: AppInfo | null;
  updateStatus: UpdateStatus | null;
  resourceVersions: ResourceVersions | null;
  runtimeStatus: CoreRuntimeStatus | null;
  error: string;
  clearError: () => void;
  startQueue: () => Promise<void>;
  stopQueue: () => Promise<void>;
  startTask: (taskId: string) => Promise<void>;
  clearLogs: () => void;
  diagnoseConnection: (address: string) => Promise<DiagnosticResult>;
  detectConnection: () => Promise<ConnectionCandidate[]>;
  testNotification: () => Promise<void>;
  refreshSchedule: () => Promise<void>;
  openDataDir: () => void;
  openLogsDir: () => void;
  openCoreLogsDir: () => void;
  openFailureShotsDir: () => void;
  openDiagnosticsDir: () => void;
  checkCoreUpdate: () => Promise<ReleaseInfo | null>;
  startCoreUpdate: (version: string) => Promise<void>;
  checkBaseResourceUpdate: () => Promise<ReleaseInfo | null>;
  startBaseResourceUpdate: () => Promise<void>;
  checkResourceOverlayUpdate: () => Promise<ReleaseInfo | null>;
  startResourceOverlayUpdate: () => Promise<void>;
  refreshResourceVersions: () => Promise<void>;
  refreshCoreRuntime: () => Promise<void>;
  repairCoreRuntime: () => Promise<void>;
  testProxy: (url: string) => Promise<ProxyTestResult>;
  restartApp: () => Promise<void>;
  openUpdatesDir: () => void;
}

const AppStoreContext = createContext<AppStoreValue | null>(null);

const CONFIG_FALLBACK_KEY = "maago.config";

function withTimeout<T>(promise: Promise<T>, fallback: T, ms = 5000): Promise<T> {
  return Promise.race([
    promise,
    new Promise<T>((resolve) => {
      window.setTimeout(() => resolve(fallback), ms);
    }),
  ]).catch(() => fallback);
}

function cloneConfig(config: AppConfig): AppConfig {
  return structuredClone(config);
}

export const AppStoreProvider = ({ children }: { children: React.ReactNode }) => {
  const { t, i18n } = useTranslation();
  const { setTheme } = useTheme();

  const [config, setConfig] = useState<AppConfig>(DEFAULT_CONFIG);
  const [ready, setReady] = useState(false);
  const [saveState, setSaveState] = useState<SaveState>("idle");
  const [run, setRun] = useState<RunState>(DEFAULT_RUN_STATE as unknown as RunState);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [taskStates, setTaskStates] = useState<Record<string, TaskState>>({});
  const [schedule, setSchedule] = useState<ScheduleStatus | null>(null);
  const [core, setCore] = useState<CoreStatus | null>(null);
  const [appInfo, setAppInfo] = useState<AppInfo | null>(null);
  const [updateStatus, setUpdateStatus] = useState<UpdateStatus | null>(null);
  const [resourceVersions, setResourceVersions] = useState<ResourceVersions | null>(null);
  const [runtimeStatus, setRuntimeStatus] = useState<CoreRuntimeStatus | null>(null);
  const [error, setError] = useState("");

  const saveReady = useRef(false);
  const lastRunStarted = useRef("");

  useEffect(() => {
    let disposed = false;

    const load = async () => {
      const [loadedConfig, state, initialLogs, scheduleStatus, coreStatus, info, update, versions, runtime] =
        await Promise.all([
          withTimeout(api.GetConfig() as unknown as Promise<AppConfig>, DEFAULT_CONFIG),
          withTimeout(api.GetRunState() as unknown as Promise<RunState>, DEFAULT_RUN_STATE as unknown as RunState),
          withTimeout(api.GetLogs() as unknown as Promise<LogEntry[]>, []),
          withTimeout(api.GetScheduleStatus() as unknown as Promise<ScheduleStatus>, null as unknown as ScheduleStatus),
          withTimeout(api.GetCoreStatus() as unknown as Promise<CoreStatus>, null as unknown as CoreStatus),
          withTimeout(api.GetAppInfo() as unknown as Promise<AppInfo>, null as unknown as AppInfo),
          withTimeout(api.GetUpdateStatus() as unknown as Promise<UpdateStatus>, null as unknown as UpdateStatus),
          withTimeout(api.GetResourceVersions() as unknown as Promise<ResourceVersions>, null as unknown as ResourceVersions),
          withTimeout(api.GetCoreRuntimeStatus() as unknown as Promise<CoreRuntimeStatus>, null as unknown as CoreRuntimeStatus),
        ]);
      if (disposed) return;

      let effectiveConfig = loadedConfig;
      if (!effectiveConfig || !effectiveConfig.tasks) {
        try {
          const cached = localStorage.getItem(CONFIG_FALLBACK_KEY);
          effectiveConfig = cached ? (JSON.parse(cached) as AppConfig) : DEFAULT_CONFIG;
        } catch {
          effectiveConfig = DEFAULT_CONFIG;
        }
      }

      setConfig(effectiveConfig);
      setRun(state ?? (DEFAULT_RUN_STATE as unknown as RunState));
      setLogs(initialLogs ?? []);
      setSchedule(scheduleStatus ?? null);
      setCore(coreStatus ?? null);
      setAppInfo(info ?? null);
      setUpdateStatus(update ?? null);
      setResourceVersions(versions ?? null);
      setRuntimeStatus(runtime ?? null);
      setReady(true);
    };

    void load();
    return () => {
      disposed = true;
    };
  }, []);

  // Persist config changes after a short debounce.
  useEffect(() => {
    if (!ready) return;
    if (!saveReady.current) {
      saveReady.current = true;
      return;
    }

    setSaveState("saving");
    const timer = window.setTimeout(() => {
      void (async () => {
        try {
          await Promise.race([
            api.SaveConfig(config as never),
            new Promise((_, reject) =>
              window.setTimeout(() => reject(new Error("timeout")), 1500),
            ),
          ]);
          setSaveState("saved");
        } catch {
          try {
            localStorage.setItem(CONFIG_FALLBACK_KEY, JSON.stringify(config));
            setSaveState("saved");
          } catch {
            setSaveState("error");
          }
        }
      })();
    }, 350);

    return () => window.clearTimeout(timer);
  }, [config, ready]);

  // Apply the persisted theme once the config is available.
  useEffect(() => {
    if (!ready) return;
    setTheme(config.theme);
  }, [config.theme, ready, setTheme]);

  // Apply the persisted language.
  useEffect(() => {
    if (!ready || !config.language) return;
    if (i18n.resolvedLanguage !== config.language) {
      void i18n.changeLanguage(config.language);
    }
  }, [config.language, i18n, ready]);

  // Keep tray labels in sync with the rendered language.
  useEffect(() => {
    if (!ready) return;
    try {
      void api
        .SetTrayLabels(t("tray.show"), t("tray.start"), t("tray.stop"), t("tray.quit"))
        .catch(() => undefined);
    } catch {
      // Browser preview has no tray.
    }
  }, [i18n.resolvedLanguage, ready, t]);

  useEffect(() => {
    const offLog = Events.On("maago:log", (event) => {
      const entry = event.data as unknown as LogEntry;
      if (!entry) return;
      setLogs((previous) => [...previous.slice(-999), entry]);
    });
    const offRun = Events.On("maago:run", (event) => {
      const state = event.data as unknown as RunState;
      if (!state) return;
      if (state.status === "running" && state.startedAt !== lastRunStarted.current) {
        lastRunStarted.current = state.startedAt;
        setTaskStates({});
      }
      if (
        state.finishedAt &&
        (state.status === "success" || state.status === "failed" || state.status === "stopped")
      ) {
        const started = Date.parse(state.startedAt);
        const finished = Date.parse(state.finishedAt);
        const durationSeconds =
          Number.isFinite(started) && Number.isFinite(finished)
            ? Math.max(0, Math.round((finished - started) / 1000))
            : 0;
        setConfig((previous) => {
          if (previous.lastRun?.finishedAt === state.finishedAt) return previous;
          return {
            ...previous,
            lastRun: {
              startedAt: state.startedAt,
              finishedAt: state.finishedAt,
              status: state.status,
              durationSeconds,
              completedTasks: state.completed,
              totalTasks: state.total,
            },
          };
        });
      }
      setRun(state);
    });
    const offTask = Events.On("maago:task", (event) => {
      const state = event.data as unknown as TaskState;
      if (!state?.taskId) return;
      setTaskStates((previous) => ({ ...previous, [state.taskId]: state }));
    });
    const offSchedule = Events.On("maago:schedule", (event) => {
      const status = event.data as unknown as ScheduleStatus;
      if (status) setSchedule(status);
    });
    const offUpdate = Events.On("maago:update", (event) => {
      const status = event.data as unknown as UpdateStatus;
      if (status) setUpdateStatus(status);
    });

    return () => {
      offLog();
      offRun();
      offTask();
      offSchedule();
      offUpdate();
    };
  }, []);

  const updateConfig = useCallback((updater: (previous: AppConfig) => AppConfig) => {
    setConfig((previous) => updater(cloneConfig(previous)));
  }, []);

  const guard = useCallback(async (action: () => Promise<unknown>, fallback: string) => {
    try {
      await action();
      setError("");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : fallback);
    }
  }, []);

  const saveNow = useCallback(async (value: AppConfig) => {
    try {
      await Promise.race([
        api.SaveConfig(value as never),
        new Promise((_, reject) =>
          window.setTimeout(() => reject(new Error("timeout")), 1500),
        ),
      ]);
    } catch {
      try {
        localStorage.setItem(CONFIG_FALLBACK_KEY, JSON.stringify(value));
      } catch {
        // Ignore storage failures; the next save attempt will retry.
      }
    }
  }, []);

  const startQueue = useCallback(async () => {
    await saveNow(config);
    await guard(() => api.StartQueue(), "启动任务失败");
  }, [config, guard, saveNow]);

  const stopQueue = useCallback(async () => {
    await guard(() => api.StopQueue(), "停止任务失败");
  }, [guard]);

  const startTask = useCallback(
    async (taskId: string) => {
      await saveNow(config);
      await guard(() => api.StartTask(taskId), "启动任务失败");
    },
    [config, guard, saveNow],
  );

  const clearLogs = useCallback(() => {
    setLogs([]);
    void api.ClearLogs().catch(() => undefined);
  }, []);

  const diagnoseConnection = useCallback(async (address: string) => {
    const fallback: DiagnosticResult = {
      ok: false,
      address,
      steps: [
        {
          key: "core",
          status: "fail",
          detail: "backend is unavailable",
          durationMs: 0,
        },
      ],
      system: {
        address,
        adbPath: "",
        serial: "",
        androidVersion: "",
        resolution: "",
        density: "",
        uuid: "",
        screenshotPath: "",
        screenshotSize: 0,
      },
      message: "backend is unavailable",
    };
    return withTimeout(
      api.DiagnoseConnection(address) as unknown as Promise<DiagnosticResult>,
      fallback,
      95000,
    );
  }, []);

  const detectConnection = useCallback(async () => {
    return withTimeout(
      api.DetectConnection() as unknown as Promise<ConnectionCandidate[]>,
      [],
      5000,
    );
  }, []);

  const testNotification = useCallback(async () => {
    await guard(() => api.TestNotification(), "发送测试通知失败");
  }, [guard]);

  const refreshSchedule = useCallback(async () => {
    const status = await withTimeout(
      api.GetScheduleStatus() as unknown as Promise<ScheduleStatus>,
      null as unknown as ScheduleStatus,
    );
    if (status) setSchedule(status);
  }, []);

  const openDataDir = useCallback(() => {
    void api.OpenDataDir().catch(() => undefined);
  }, []);

  const openLogsDir = useCallback(() => {
    void api.OpenLogsDir().catch(() => undefined);
  }, []);

  const openCoreLogsDir = useCallback(() => {
    void api.OpenCoreLogsDir().catch(() => undefined);
  }, []);

  const openFailureShotsDir = useCallback(() => {
    void api.OpenFailureShotsDir().catch(() => undefined);
  }, []);

  const openDiagnosticsDir = useCallback(() => {
    void api.OpenDiagnosticsDir().catch(() => undefined);
  }, []);

  const checkCoreUpdate = useCallback(async () => {
    await saveNow(config);
    const info = await withTimeout(
      api.CheckCoreUpdate() as unknown as Promise<ReleaseInfo>,
      null as unknown as ReleaseInfo,
      30000,
    );
    if (!info) {
      setError("无法检查更新，请检查代理或网络");
    } else {
      setError("");
    }
    return info ?? null;
  }, [config, saveNow]);

  const startCoreUpdate = useCallback(
    async (version: string) => {
      await saveNow(config);
      await guard(() => api.StartCoreUpdate(version), "启动核心更新失败");
    },
    [config, guard, saveNow],
  );

  const checkBaseResourceUpdate = useCallback(async () => {
    await saveNow(config);
    const info = await withTimeout(
      api.CheckBaseResourceUpdate() as unknown as Promise<ReleaseInfo>,
      null as unknown as ReleaseInfo,
      35000,
    );
    if (!info) {
      setError("无法检查资源底包，请检查代理或网络");
    } else {
      setError("");
    }
    return info ?? null;
  }, [config, saveNow]);

  const startBaseResourceUpdate = useCallback(async () => {
    await saveNow(config);
    await guard(() => api.StartBaseResourceUpdate(), "启动资源底包更新失败");
  }, [config, guard, saveNow]);

  const checkResourceOverlayUpdate = useCallback(async () => {
    await saveNow(config);
    const info = await withTimeout(
      api.CheckResourceOverlayUpdate() as unknown as Promise<ReleaseInfo>,
      null as unknown as ReleaseInfo,
      35000,
    );
    if (!info) {
      setError("无法检查资源覆盖，请检查代理或网络");
    } else {
      setError("");
    }
    return info ?? null;
  }, [config, saveNow]);

  const startResourceOverlayUpdate = useCallback(async () => {
    await saveNow(config);
    await guard(() => api.StartResourceOverlayUpdate(), "启动资源覆盖更新失败");
  }, [config, guard, saveNow]);

  const refreshResourceVersions = useCallback(async () => {
    const versions = await withTimeout(
      api.GetResourceVersions() as unknown as Promise<ResourceVersions>,
      null as unknown as ResourceVersions,
      10000,
    );
    if (versions) setResourceVersions(versions);
  }, []);

  const refreshCoreRuntime = useCallback(async () => {
    const status = await withTimeout(
      api.GetCoreRuntimeStatus() as unknown as Promise<CoreRuntimeStatus>,
      null as unknown as CoreRuntimeStatus,
      15000,
    );
    if (status) setRuntimeStatus(status);
  }, []);

  const repairCoreRuntime = useCallback(async () => {
    await saveNow(config);
    await guard(() => api.RepairCoreRuntime(), "启动前置库修复失败");
  }, [config, guard, saveNow]);

  const testProxy = useCallback(async (proxyURL: string) => {
    return withTimeout(
      api.TestProxy(proxyURL) as unknown as Promise<ProxyTestResult>,
      { ok: false, status: 0, latencyMs: 0, message: "后端不可用" },
      20000,
    );
  }, []);

  const restartApp = useCallback(async () => {
    await guard(() => api.RestartApp(), "重启失败，请手动退出后重新打开");
  }, [guard]);

  const openUpdatesDir = useCallback(() => {
    void api.OpenUpdatesDir().catch(() => undefined);
  }, []);

  const value: AppStoreValue = {
    config,
    updateConfig,
    ready,
    saveState,
    run,
    logs,
    taskStates,
    schedule,
    core,
    appInfo,
    updateStatus,
    resourceVersions,
    runtimeStatus,
    error,
    clearError: () => setError(""),
    startQueue,
    stopQueue,
    startTask,
    clearLogs,
    diagnoseConnection,
    detectConnection,
    testNotification,
    refreshSchedule,
    openDataDir,
    openLogsDir,
    openCoreLogsDir,
    openFailureShotsDir,
    openDiagnosticsDir,
    checkCoreUpdate,
    startCoreUpdate,
    checkBaseResourceUpdate,
    startBaseResourceUpdate,
    checkResourceOverlayUpdate,
    startResourceOverlayUpdate,
    refreshResourceVersions,
    refreshCoreRuntime,
    repairCoreRuntime,
    testProxy,
    restartApp,
    openUpdatesDir,
  };

  return (
    <AppStoreContext.Provider value={value}>{children}</AppStoreContext.Provider>
  );
};

export function useAppStore(): AppStoreValue {
  const value = useContext(AppStoreContext);
  if (!value) {
    throw new Error("useAppStore must be used inside AppStoreProvider");
  }
  return value;
}
