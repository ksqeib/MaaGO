import { useTranslation } from "react-i18next";
import {
  FaChevronDown,
  FaChevronUp,
  FaGripVertical,
  FaPlay,
  FaPlus,
  FaStop,
  FaTrash,
} from "react-icons/fa";
import { useEffect, useState } from "react";

import { LogConsole } from "@/components/LogConsole";
import { PageHeader } from "@/components/PageHeader";
import { ProgressBar } from "@/components/ProgressBar";
import { TaskIcon } from "@/components/TaskIcon";
import { TaskSettingsForm } from "@/components/TaskSettingsForm";
import { TaskToggle } from "@/components/TaskToggle";
import { TASK_META, TASK_META_BY_TYPE, createTask } from "@/data/tasks";
import { taskDisplayName, taskSummary } from "@/data/taskSummary";
import { useAppStore } from "@/store/AppStore";
import type { TaskConfig, TaskType } from "@/types/maa";
import { cn } from "@/utils/cn";
import { pickText } from "@/utils/text";

const statusDot: Record<string, string> = {
  running: "bg-sky-500",
  completed: "bg-emerald-500",
  skipped: "bg-zinc-400",
  error: "bg-red-500",
  pending: "bg-zinc-300 dark:bg-zinc-600",
};

export const TasksPage = () => {
  const { t, i18n } = useTranslation();
  const {
    config,
    updateConfig,
    logs,
    run,
    taskStates,
    startQueue,
    stopQueue,
    startTask,
    clearLogs,
    saveState,
  } = useAppStore();

  const language = i18n.resolvedLanguage;
  const [selectedId, setSelectedId] = useState<string>(config.tasks[0]?.id ?? "");
  const [dragIndex, setDragIndex] = useState<number | null>(null);

  useEffect(() => {
    if (config.tasks.length === 0) {
      setSelectedId("");
      return;
    }
    if (!config.tasks.some((task) => task.id === selectedId)) {
      setSelectedId(config.tasks[0].id);
    }
  }, [config.tasks, selectedId]);

  const selectedTask = config.tasks.find((task) => task.id === selectedId);

  const updateTasks = (mutate: (tasks: TaskConfig[]) => TaskConfig[]) => {
    updateConfig((previous) => ({ ...previous, tasks: mutate([...previous.tasks]) }));
  };

  const addTask = (type: TaskType) => {
    const task = createTask(type);
    updateConfig((previous) => ({ ...previous, tasks: [...previous.tasks, task] }));
    setSelectedId(task.id);
  };

  const removeTask = (id: string) => {
    updateTasks((tasks) => tasks.filter((task) => task.id !== id));
  };

  const moveTask = (from: number, to: number) => {
    if (from === to || from < 0 || to < 0) return;
    updateTasks((tasks) => {
      const next = [...tasks];
      const [moved] = next.splice(from, 1);
      next.splice(to, 0, moved);
      return next;
    });
  };

  const running = run.status === "running" || run.status === "stopping";

  return (
    <div className="flex h-full min-h-0 w-full flex-col gap-3 overflow-hidden px-4 pb-4 pt-[var(--content-pt)]">
      <PageHeader
        description={t("tasks.subtitle")}
        endContent={
          <span className="text-xs text-muted">
            {saveState === "saving"
              ? t("common.saving")
              : saveState === "error"
                ? t("common.save_error")
                : t("common.saved")}
          </span>
        }
        title={t("tasks.title")}
      />

      <div className="grid min-h-0 flex-1 gap-3 lg:grid-cols-[270px_minmax(0,1fr)_360px]">
        {/* Queue */}
        <section className="launcher-glass flex min-h-0 flex-col rounded-3xl">
          <header className="flex items-center justify-between border-b border-border/70 px-3 py-2.5">
            <h2 className="text-sm font-bold">{t("tasks.queue")}</h2>
            <details className="relative">
              <summary className="flex cursor-pointer list-none items-center gap-1 rounded-full bg-surface-secondary px-2.5 py-1 text-[11px] font-medium text-muted transition-colors hover:text-foreground">
                <FaPlus size={9} />
                {t("tasks.add_task")}
              </summary>
              <div className="absolute right-0 top-full z-30 mt-1 w-48 rounded-2xl border border-border bg-overlay p-1.5 shadow-xl">
                {TASK_META.map((meta) => {
                  const exists = config.tasks.some((task) => task.type === meta.type);
                  return (
                    <button
                      className={cn(
                        "flex w-full items-center justify-between rounded-xl px-2.5 py-2 text-left text-xs transition-colors",
                        exists
                          ? "cursor-not-allowed text-muted/50"
                          : "hover:bg-surface-secondary",
                      )}
                      disabled={exists}
                      key={meta.type}
                      onClick={(event) => {
                        addTask(meta.type);
                        const details = event.currentTarget.closest("details");
                        if (details) details.open = false;
                      }}
                      type="button"
                    >
                      <span className="flex items-center gap-2">
                        <TaskIcon name={meta.icon} size={12} />
                        {pickText(meta.label, language)}
                      </span>
                      {exists && <span className="text-[10px]">{t("tasks.already_added")}</span>}
                    </button>
                  );
                })}
              </div>
            </details>
          </header>

          <div
            className="min-h-0 flex-1 overflow-y-auto p-2"
            onDragOver={(event) => event.preventDefault()}
          >
            {config.tasks.length === 0 ? (
              <p className="p-4 text-center text-xs text-muted">{t("tasks.no_tasks")}</p>
            ) : (
              config.tasks.map((task, index) => {
                const meta = TASK_META_BY_TYPE[task.type];
                const state = taskStates[task.id]?.status;
                const active = task.id === selectedId;
                return (
                  <div
                    className={cn(
                      "group mb-1 flex items-center gap-1.5 rounded-2xl border px-2 py-2 transition-colors",
                      active
                        ? "border-brand-500/40 bg-brand-500/10"
                        : "border-transparent hover:bg-surface-secondary/70",
                      dragIndex === index && "opacity-40",
                    )}
                    draggable
                    key={task.id}
                    onClick={() => setSelectedId(task.id)}
                    onDragEnd={() => setDragIndex(null)}
                    onDragOver={(event) => event.preventDefault()}
                    onDragStart={() => setDragIndex(index)}
                    onDrop={() => {
                      if (dragIndex !== null) moveTask(dragIndex, index);
                      setDragIndex(null);
                    }}
                  >
                    <FaGripVertical className="shrink-0 cursor-grab text-[10px] text-muted/40" />
                    <TaskToggle
                      enabled={task.enabled}
                      onChange={(enabled) =>
                        updateTasks((tasks) =>
                          tasks.map((item) =>
                            item.id === task.id ? { ...item, enabled } : item,
                          ),
                        )
                      }
                    />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-1.5">
                        <span
                          className={cn(
                            "h-1.5 w-1.5 shrink-0 rounded-full",
                            statusDot[state ?? "pending"],
                          )}
                        />
                        <span className="truncate text-xs font-semibold">
                          {taskDisplayName(task, language)}
                        </span>
                      </div>
                      <p className="truncate pl-3 text-[10px] text-muted">
                        {taskSummary(task, language)}
                      </p>
                    </div>

                    <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
                      <button
                        className="rounded p-1 text-[9px] text-muted hover:text-foreground"
                        disabled={index === 0}
                        onClick={(event) => {
                          event.stopPropagation();
                          moveTask(index, index - 1);
                        }}
                        title={t("tasks.move_up")}
                        type="button"
                      >
                        <FaChevronUp />
                      </button>
                      <button
                        className="rounded p-1 text-[9px] text-muted hover:text-foreground"
                        disabled={index === config.tasks.length - 1}
                        onClick={(event) => {
                          event.stopPropagation();
                          moveTask(index, index + 1);
                        }}
                        title={t("tasks.move_down")}
                        type="button"
                      >
                        <FaChevronDown />
                      </button>
                      <button
                        className="rounded p-1 text-[9px] text-muted hover:text-red-500"
                        onClick={(event) => {
                          event.stopPropagation();
                          removeTask(task.id);
                        }}
                        title={t("tasks.remove_task")}
                        type="button"
                      >
                        <FaTrash />
                      </button>
                    </div>
                  </div>
                );
              })
            )}
          </div>

          <div className="border-t border-border/70 p-2">
            <button
              className={cn(
                "flex w-full items-center justify-center gap-2 rounded-2xl px-3 py-2.5 text-sm font-semibold transition-colors",
                running
                  ? "bg-red-500/90 text-white hover:bg-red-500"
                  : "bg-brand-500 text-white hover:bg-brand-600",
              )}
              onClick={() => {
                void (running ? stopQueue() : startQueue());
              }}
              type="button"
            >
              {running ? <FaStop size={11} /> : <FaPlay size={11} />}
              {running ? t("tasks.stop_queue") : t("tasks.start_queue")}
            </button>
          </div>
        </section>

        {/* Settings */}
        <section className="launcher-glass flex min-h-0 flex-col overflow-hidden rounded-3xl">
          {selectedTask ? (
            <div className="min-h-0 flex-1 overflow-y-auto">
              <TaskSettingsForm
                meta={TASK_META_BY_TYPE[selectedTask.type]}
                onChange={(options) =>
                  updateTasks((tasks) =>
                    tasks.map((item) =>
                      item.id === selectedTask.id ? { ...item, options } : item,
                    ),
                  )
                }
                onRun={() => void startTask(selectedTask.id)}
                onToggle={(enabled) =>
                  updateTasks((tasks) =>
                    tasks.map((item) =>
                      item.id === selectedTask.id ? { ...item, enabled } : item,
                    ),
                  )
                }
                task={selectedTask}
              />
            </div>
          ) : (
            <div className="flex flex-1 items-center justify-center p-6 text-sm text-muted">
              {t("tasks.select_task")}
            </div>
          )}
        </section>

        {/* Logs */}
        <section className="launcher-glass flex min-h-0 flex-col overflow-hidden rounded-3xl">
          <header className="border-b border-border/70 p-3">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <span
                  className={cn(
                    "rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider",
                    run.status === "running"
                      ? "bg-sky-500/15 text-sky-600 dark:text-sky-400"
                      : run.status === "success"
                        ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400"
                        : run.status === "failed"
                          ? "bg-red-500/15 text-red-600 dark:text-red-400"
                          : "bg-surface-secondary text-muted",
                  )}
                >
                  {t(`status.${run.status}`)}
                </span>
                <span className="text-xs text-muted">
                  {run.completed}/{run.total || config.tasks.filter((task) => task.enabled).length}
                </span>
              </div>
              <button
                className="text-[11px] text-muted hover:text-foreground"
                onClick={clearLogs}
                type="button"
              >
                {t("tasks.clear_logs")}
              </button>
            </div>
            <p className="mt-1 truncate text-xs font-medium">
              {run.currentTaskName || t("dashboard.no_current_task")}
            </p>
            <ProgressBar className="mt-2" value={run.progress} />
          </header>

          <div className="flex items-start gap-2 border-b border-amber-500/20 bg-amber-500/10 px-3 py-2 text-[10px] leading-relaxed text-amber-700 dark:text-amber-300">
            {t("tasks.demo_warning")}
          </div>

          <LogConsole
            className="min-h-0 flex-1"
            emptyText={t("dashboard.no_logs")}
            logs={logs.slice(-300)}
          />
        </section>
      </div>
    </div>
  );
};
