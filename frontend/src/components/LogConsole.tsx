import React, { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { FaSearch } from "react-icons/fa";

import { Switch } from "@/components/Switch";
import type { LogEntry } from "@/types/maa";
import { cn } from "@/utils/cn";

interface LogConsoleProps {
  logs: LogEntry[];
  className?: string;
  showControls?: boolean;
  onClear?: () => void;
  emptyText?: string;
}

const levels = ["info", "success", "warn", "error"] as const;

const levelColor: Record<string, string> = {
  info: "text-foreground/80",
  success: "text-emerald-600 dark:text-emerald-400",
  warn: "text-amber-600 dark:text-amber-400",
  error: "text-red-600 dark:text-red-400",
};

const dotColor: Record<string, string> = {
  info: "bg-zinc-400",
  success: "bg-emerald-500",
  warn: "bg-amber-500",
  error: "bg-red-500",
};

export const LogConsole: React.FC<LogConsoleProps> = ({
  logs,
  className,
  showControls,
  onClear,
  emptyText,
}) => {
  const { t } = useTranslation();
  const [activeLevels, setActiveLevels] = useState<string[]>([...levels]);
  const [search, setSearch] = useState("");
  const [autoScroll, setAutoScroll] = useState(true);
  const bodyRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    const needle = search.trim().toLowerCase();
    return logs.filter((entry) => {
      if (!activeLevels.includes(entry.level)) return false;
      if (!needle) return true;
      return entry.message.toLowerCase().includes(needle);
    });
  }, [activeLevels, logs, search]);

  useEffect(() => {
    if (!autoScroll || !bodyRef.current) return;
    bodyRef.current.scrollTop = bodyRef.current.scrollHeight;
  }, [autoScroll, filtered.length]);

  const toggleLevel = (level: string) => {
    setActiveLevels((previous) =>
      previous.includes(level)
        ? previous.filter((item) => item !== level)
        : [...previous, level],
    );
  };

  return (
    <div className={cn("flex min-h-0 flex-col", className)}>
      {showControls && (
        <div className="flex flex-wrap items-center gap-2 border-b border-border/70 p-3">
          <div className="flex min-w-[10rem] flex-1 items-center gap-2 rounded-xl border border-border bg-surface-secondary/60 px-3">
            <FaSearch className="text-xs text-muted" />
            <input
              className="h-9 w-full bg-transparent text-sm outline-none placeholder:text-muted"
              onChange={(event) => setSearch(event.target.value)}
              placeholder={t("logs.search_placeholder")}
              value={search}
            />
          </div>

          <div className="flex flex-wrap items-center gap-1">
            {levels.map((level) => {
              const active = activeLevels.includes(level);
              return (
                <button
                  className={cn(
                    "rounded-full border px-2.5 py-1 text-[11px] font-medium transition-colors",
                    active
                      ? "border-brand-500/50 bg-brand-500/10 text-brand-700 dark:text-brand-300"
                      : "border-border text-muted hover:text-foreground",
                  )}
                  key={level}
                  onClick={() => toggleLevel(level)}
                  type="button"
                >
                  {t(`level.${level}`)}
                </button>
              );
            })}
          </div>

          <label className="flex items-center gap-2 text-xs text-muted">
            {t("logs.auto_scroll")}
            <Switch checked={autoScroll} onChange={setAutoScroll} />
          </label>

          {onClear && (
            <button
              className="rounded-full border border-border px-3 py-1 text-[11px] font-medium text-muted transition-colors hover:text-foreground"
              onClick={onClear}
              type="button"
            >
              {t("logs.clear")}
            </button>
          )}
        </div>
      )}

      <div
        className="min-h-0 flex-1 overflow-y-auto p-3 font-mono text-[12px] leading-relaxed"
        ref={bodyRef}
      >
        {filtered.length === 0 ? (
          <div className="flex h-full items-center justify-center p-6 text-center font-sans text-sm text-muted">
            {emptyText ?? t("logs.empty")}
          </div>
        ) : (
          <div className="flex flex-col gap-0.5">
            {filtered.map((entry) => (
              <div className="flex items-start gap-2" key={entry.id}>
                <span className="shrink-0 text-[11px] text-muted">
                  {entry.time}
                </span>
                <span
                  className={cn(
                    "mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full",
                    dotColor[entry.level] ?? dotColor.info,
                  )}
                />
                <span
                  className={cn(
                    "min-w-0 break-words whitespace-pre-wrap",
                    levelColor[entry.level] ?? levelColor.info,
                  )}
                >
                  {entry.message}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
