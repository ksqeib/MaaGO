export type ThemeMode = "light" | "dark" | "system";
export type LayoutMode = "sidebar" | "navbar";

export type TaskType =
  | "startup"
  | "fight"
  | "infrast"
  | "recruit"
  | "mall"
  | "award";

export type RunStatus =
  | "idle"
  | "running"
  | "stopping"
  | "success"
  | "failed"
  | "stopped";

export type TaskStatus =
  | "pending"
  | "running"
  | "completed"
  | "skipped"
  | "error";

export interface TaskConfig {
  id: string;
  type: TaskType;
  name: string;
  enabled: boolean;
  options: Record<string, unknown>;
}

export interface ConnectionConfig {
  mode: string;
  address: string;
  adbPath: string;
  autoDetect: boolean;
  emulator: string;
  touchMode: string;
  screenshotEnhance: boolean;
  clientType: string;
}

export interface RunConfig {
  clientType: string;
  blockSleep: boolean;
  keepScreenOn: boolean;
  runDurationMinutes: number;
  postAction: string;
  postActionOnlyWhenNoOther: boolean;
  postActionSkipOnError: boolean;
}

export interface ScheduleConfig {
  enabled: boolean;
  time: string;
  weekdays: number[];
  force: boolean;
  notifyBefore: boolean;
  notifyMinutes: number;
  lastTriggered: string;
  lastNotified: string;
}

export interface NotificationConfig {
  systemEnabled: boolean;
  onComplete: boolean;
  onError: boolean;
  onSchedule: boolean;
}

export interface ProxyConfig {
  enabled: boolean;
  url: string;
}

export interface UpdateConfig {
  mirror: string;
  targetVersion: string;
}

export interface LastRun {
  startedAt: string;
  finishedAt: string;
  status: string;
  durationSeconds: number;
  completedTasks: number;
  totalTasks: number;
}

export interface AppConfig {
  configVersion: number;
  windowWidth: number;
  windowHeight: number;
  windowMaximised: boolean;
  minimizeToTray: boolean;
  closeToTray: boolean;
  theme: ThemeMode;
  accentColor: string;
  language: string;
  layout: LayoutMode;
  connection: ConnectionConfig;
  run: RunConfig;
  schedule: ScheduleConfig;
  notification: NotificationConfig;
  proxy: ProxyConfig;
  update: UpdateConfig;
  tasks: TaskConfig[];
  lastRun: LastRun;
}

export interface RunState {
  status: RunStatus;
  currentTaskId: string;
  currentTaskName: string;
  completed: number;
  total: number;
  progress: number;
  startedAt: string;
  finishedAt: string;
  lastError: string;
  message: string;
  demo: boolean;
}

export interface TaskState {
  taskId: string;
  status: TaskStatus;
  progress: number;
}

export interface LogEntry {
  id: number;
  time: string;
  timestamp: number;
  level: "info" | "success" | "warn" | "error" | string;
  taskId: string;
  message: string;
}

export interface ScheduleStatus {
  enabled: boolean;
  time: string;
  weekdays: number[];
  force: boolean;
  notifyBefore: boolean;
  notifyMinutes: number;
  nextRun: string;
  nextRunUnix: number;
  secondsUntilRun: number;
  lastTriggered: string;
}

export interface CoreStatus {
  available: boolean;
  path: string;
  message: string;
  demo: boolean;
  version?: string;
}

export interface AppInfo {
  name: string;
  version: string;
  commit: string;
  buildDate: string;
  goVersion: string;
  os: string;
  arch: string;
  dataDir: string;
  logsDir: string;
  coreLogsDir: string;
  diagnosticsDir: string;
}

export interface ConnectionCandidate {
  name: string;
  address: string;
  latencyMs: number;
}

export interface DiagnosticStep {
  key: string;
  status: "ok" | "fail" | "skip";
  detail: string;
  durationMs: number;
}

export interface DiagnosticSystemInfo {
  address: string;
  adbPath: string;
  serial: string;
  androidVersion: string;
  resolution: string;
  density: string;
  uuid: string;
  screenshotPath: string;
  screenshotSize: number;
}

export interface DiagnosticResult {
  ok: boolean;
  address: string;
  steps: DiagnosticStep[];
  system: DiagnosticSystemInfo;
  message: string;
}

export interface ReleaseInfo {
  tag: string;
  version: string;
  assetName: string;
  assetUrl: string;
  size: number;
  published: string;
}

export interface UpdateStatus {
  stage: "idle" | "checking" | "downloading" | "extracting" | "ready" | "error" | string;
  message: string;
  progress: number;
  downloaded: number;
  total: number;
  version: string;
  needsRestart: boolean;
  error: string;
}

export interface ProxyTestResult {
  ok: boolean;
  status: number;
  latencyMs: number;
  message: string;
}

export interface LocalizedText {
  zh: string;
  en: string;
}
