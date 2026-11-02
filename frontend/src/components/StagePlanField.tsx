import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  FaArrowDown,
  FaArrowUp,
  FaCheck,
  FaChevronDown,
  FaSearch,
  FaSyncAlt,
  FaTimes,
  FaTrash,
} from "react-icons/fa";

import * as api from "bindings/MaaGO/internal/app/service";
import { cn } from "@/utils/cn";

interface StageInfo {
  code: string;
  name?: string;
  ap?: number;
  group?: string;
  available: boolean;
  note?: string;
}

interface StageGroup {
  label: string;
  stages: StageInfo[];
}

interface StageTab {
  id: string;
  label: string;
  groups: StageGroup[];
}

interface StageCatalog {
  updatedAt: string;
  source: string;
  tabs: StageTab[];
}

interface StagePlanFieldProps {
  value: string[];
  onChange: (value: string[]) => void;
}

const SELECTED_TAB = "selected";

export const StagePlanField = ({ value, onChange }: StagePlanFieldProps) => {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [catalog, setCatalog] = useState<StageCatalog | null>(null);
  const [loadedAt, setLoadedAt] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [draft, setDraft] = useState<string[]>(value);
  const [tab, setTab] = useState("main");
  const [query, setQuery] = useState("");

  const load = useCallback(async (refresh: boolean) => {
    setLoading(true);
    setError("");
    try {
      const result = (await api.GetStageCatalog(refresh)) as StageCatalog;
      setCatalog(result);
      setLoadedAt(Date.now());
      const first = result.tabs.find((item) => item.id === "main") ?? result.tabs[0];
      if (first) {
        setTab((current) => (current === SELECTED_TAB ? current : first.id));
      }
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open && (!catalog || Date.now() - loadedAt > 5 * 60_000)) {
      void load(false);
    }
  }, [open, catalog, loadedAt, load]);

  const openPicker = () => {
    setDraft(Array.isArray(value) ? value : []);
    setQuery("");
    setOpen(true);
  };

  const index = useMemo(() => {
    const map = new Map<string, StageInfo>();
    catalog?.tabs.forEach((item) =>
      item.groups.forEach((group) =>
        group.stages.forEach((stage) => map.set(stage.code, stage)),
      ),
    );
    return map;
  }, [catalog]);

  const selectedSet = useMemo(() => new Set(draft), [draft]);

  const toggle = (code: string) => {
    setDraft((current) =>
      current.includes(code)
        ? current.filter((item) => item !== code)
        : [...current, code],
    );
  };

  const move = (position: number, delta: number) => {
    setDraft((current) => {
      const next = [...current];
      const target = position + delta;
      if (target < 0 || target >= next.length) {
        return current;
      }
      [next[position], next[target]] = [next[target], next[position]];
      return next;
    });
  };

  const groups = useMemo(() => {
    if (!catalog) {
      return [];
    }
    if (query.trim()) {
      const needle = query.trim().toLowerCase();
      const matched: StageInfo[] = [];
      catalog.tabs.forEach((item) =>
        item.groups.forEach((group) =>
          group.stages.forEach((stage) => {
            const haystack = `${stage.code} ${stage.name ?? ""} ${stage.group ?? ""}`.toLowerCase();
            if (haystack.includes(needle)) {
              matched.push(stage);
            }
          }),
        ),
      );
      return [{ label: t("stages.search_results"), stages: matched }];
    }
    if (tab === SELECTED_TAB) {
      return [];
    }
    return catalog.tabs.find((item) => item.id === tab)?.groups ?? [];
  }, [catalog, query, tab, t]);

  const confirm = () => {
    onChange(draft);
    setOpen(false);
  };

  const cancel = () => {
    setOpen(false);
  };

  const codes = Array.isArray(value) ? value : [];

  return (
    <>
      <button
        className="flex min-h-10 w-full items-center justify-between gap-3 rounded-xl border border-border bg-surface-secondary/60 px-3 py-2 text-left text-sm transition-colors hover:border-brand-500/50 hover:bg-surface"
        onClick={openPicker}
        type="button"
      >
        <span className="flex min-w-0 flex-1 flex-wrap items-center gap-1.5">
          {codes.length === 0 ? (
            <span className="text-muted">{t("stages.current_last")}</span>
          ) : (
            codes.map((code, position) => {
              const info = index.get(code);
              return (
                <span
                  className="inline-flex max-w-full items-center gap-1.5 rounded-full bg-brand-500/12 px-2.5 py-1 text-xs font-medium text-brand-700 dark:text-brand-300"
                  key={`${code}-${position}`}
                >
                  <span className="opacity-60">{position + 1}</span>
                  <span className="truncate">{code}</span>
                  {info?.name && (
                    <span className="truncate text-[11px] opacity-70">{info.name}</span>
                  )}
                </span>
              );
            })
          )}
        </span>
        <span className="flex shrink-0 items-center gap-1 text-xs text-muted">
          {codes.length === 0 ? t("stages.pick") : t("stages.edit")}
          <FaChevronDown size={10} />
        </span>
      </button>

      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
          <div className="launcher-glass flex h-[min(760px,92vh)] w-[min(1080px,94vw)] flex-col overflow-hidden rounded-3xl border border-border bg-surface shadow-2xl">
            <header className="flex items-center justify-between gap-4 border-b border-border px-5 py-4">
              <div>
                <h2 className="text-base font-semibold">{t("stages.title")}</h2>
                <p className="mt-0.5 text-xs text-muted">{t("stages.subtitle")}</p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  className="flex items-center gap-1.5 rounded-full bg-surface-secondary px-3 py-1.5 text-xs text-muted transition-colors hover:text-foreground disabled:opacity-50"
                  disabled={loading}
                  onClick={() => void load(true)}
                  type="button"
                >
                  <FaSyncAlt className={cn(loading && "animate-spin")} size={10} />
                  {t("stages.refresh")}
                </button>
                <button
                  className="flex h-8 w-8 items-center justify-center rounded-full bg-surface-secondary text-muted transition-colors hover:text-foreground"
                  onClick={cancel}
                  type="button"
                >
                  <FaTimes size={12} />
                </button>
              </div>
            </header>

            <div className="flex flex-wrap items-center gap-1.5 border-b border-border px-4 py-2.5">
              {catalog?.tabs.map((item) => (
                <button
                  className={cn(
                    "rounded-full px-3 py-1.5 text-xs font-medium transition-colors",
                    tab === item.id
                      ? "bg-brand-500 text-white"
                      : "bg-surface-secondary text-muted hover:text-foreground",
                  )}
                  key={item.id}
                  onClick={() => setTab(item.id)}
                  type="button"
                >
                  {t(`stages.tab.${item.id}`, { defaultValue: item.label })}
                </button>
              ))}
              <button
                className={cn(
                  "rounded-full px-3 py-1.5 text-xs font-medium transition-colors",
                  tab === SELECTED_TAB
                    ? "bg-brand-500 text-white"
                    : "bg-surface-secondary text-muted hover:text-foreground",
                )}
                onClick={() => setTab(SELECTED_TAB)}
                type="button"
              >
                {t("stages.selected")} ({draft.length})
              </button>
            </div>

            <div className="grid min-h-0 flex-1 grid-cols-1 md:grid-cols-[minmax(0,1fr)_320px]">
              <section className="flex min-h-0 flex-col border-b border-border md:border-b-0 md:border-r">
                <div className="px-4 py-3">
                  <label className="flex items-center gap-2 rounded-xl border border-border bg-surface-secondary/60 px-3 py-2 focus-within:border-brand-500/60">
                    <FaSearch className="text-muted" size={12} />
                    <input
                      className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-muted"
                      onChange={(event) => setQuery(event.target.value)}
                      placeholder={t("stages.search_placeholder")}
                      value={query}
                    />
                    {query && (
                      <button
                        className="text-muted hover:text-foreground"
                        onClick={() => setQuery("")}
                        type="button"
                      >
                        <FaTimes size={11} />
                      </button>
                    )}
                  </label>
                </div>

                <div className="min-h-0 flex-1 overflow-y-auto px-3 pb-4">
                  {loading && !catalog && (
                    <p className="px-2 py-10 text-center text-sm text-muted">
                      {t("stages.loading")}
                    </p>
                  )}
                  {error && !catalog && (
                    <div className="mx-2 mt-2 rounded-xl border border-red-500/30 bg-red-500/5 px-3 py-2 text-xs text-red-500">
                      {t("stages.load_failed")}: {error}
                    </div>
                  )}
                  {catalog && groups.length === 0 && tab !== SELECTED_TAB && (
                    <p className="px-2 py-10 text-center text-sm text-muted">
                      {t("stages.empty")}
                    </p>
                  )}
                  {groups.map((group) => (
                    <div className="mb-3" key={group.label}>
                      <h3 className="px-2 pb-1.5 pt-2 text-[11px] font-semibold uppercase tracking-wide text-muted">
                        {localizeGroup(group.label, t)}
                      </h3>
                      <div className="flex flex-col gap-1">
                        {group.stages.map((stage) => {
                          const selected = selectedSet.has(stage.code);
                          return (
                            <button
                              className={cn(
                                "flex items-center gap-3 rounded-xl border px-3 py-2 text-left transition-colors",
                                selected
                                  ? "border-brand-500/50 bg-brand-500/10"
                                  : "border-transparent hover:border-border hover:bg-surface-secondary/70",
                              )}
                              key={`${group.label}-${stage.code}`}
                              onClick={() => toggle(stage.code)}
                              type="button"
                            >
                              <span
                                className={cn(
                                  "flex h-5 w-5 shrink-0 items-center justify-center rounded-md border",
                                  selected
                                    ? "border-brand-500 bg-brand-500 text-white"
                                    : "border-border text-transparent",
                                )}
                              >
                                <FaCheck size={9} />
                              </span>
                              <span className="min-w-0 flex-1">
                                <span className="flex items-baseline gap-2">
                                  <span className="text-sm font-semibold">{stage.code}</span>
                                  {stage.name && (
                                    <span className="truncate text-xs text-muted">{stage.name}</span>
                                  )}
                                </span>
                                <span className="mt-0.5 flex items-center gap-2 text-[11px]">
                                  {stage.ap ? (
                                    <span className="text-muted">{stage.ap} 理智</span>
                                  ) : null}
                                  <span
                                    className={cn(
                                      stage.available ? "text-emerald-600" : "text-amber-600",
                                    )}
                                  >
                                    {stage.note || (stage.available ? t("stages.open") : t("stages.closed"))}
                                  </span>
                                </span>
                              </span>
                            </button>
                          );
                        })}
                      </div>
                    </div>
                  ))}
                </div>
              </section>

              <aside className="flex min-h-0 flex-col bg-surface-secondary/30">
                <div className="flex items-center justify-between px-4 py-3">
                  <span className="text-xs font-semibold">
                    {t("stages.plan")} ({draft.length})
                  </span>
                  <button
                    className="flex items-center gap-1 text-[11px] text-muted transition-colors hover:text-red-500 disabled:opacity-40"
                    disabled={draft.length === 0}
                    onClick={() => setDraft([])}
                    type="button"
                  >
                    <FaTrash size={9} />
                    {t("stages.clear")}
                  </button>
                </div>
                <div className="min-h-0 flex-1 overflow-y-auto px-3 pb-3">
                  {draft.length === 0 && (
                    <p className="rounded-xl border border-dashed border-border px-3 py-6 text-center text-xs leading-5 text-muted">
                      {t("stages.plan_empty")}
                    </p>
                  )}
                  {draft.map((code, position) => {
                    const info = index.get(code);
                    return (
                      <div
                        className="mb-1.5 flex items-center gap-2 rounded-xl border border-border bg-surface px-2.5 py-2"
                        key={`${code}-${position}`}
                      >
                        <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-md bg-brand-500/15 text-[10px] font-semibold text-brand-700 dark:text-brand-300">
                          {position + 1}
                        </span>
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-xs font-semibold">{code}</p>
                          <p className="truncate text-[10px] text-muted">
                            {info?.name || (info ? "" : t("stages.unknown"))}
                          </p>
                        </div>
                        <button
                          className="text-muted hover:text-foreground disabled:opacity-30"
                          disabled={position === 0}
                          onClick={() => move(position, -1)}
                          type="button"
                        >
                          <FaArrowUp size={9} />
                        </button>
                        <button
                          className="text-muted hover:text-foreground disabled:opacity-30"
                          disabled={position === draft.length - 1}
                          onClick={() => move(position, 1)}
                          type="button"
                        >
                          <FaArrowDown size={9} />
                        </button>
                        <button
                          className="text-muted hover:text-red-500"
                          onClick={() => toggle(code)}
                          type="button"
                        >
                          <FaTimes size={10} />
                        </button>
                      </div>
                    );
                  })}
                </div>
                <div className="border-t border-border px-4 py-2 text-[11px] text-muted">
                  {draft.length === 0
                    ? t("stages.plan_hint_empty")
                    : t("stages.plan_hint")}
                </div>
              </aside>
            </div>

            <footer className="flex items-center justify-between gap-3 border-t border-border px-5 py-3">
              <span className="truncate text-[11px] text-muted">
                {catalog ? `${t("stages.source")}: ${catalog.source}` : ""}
              </span>
              <div className="flex gap-2">
                <button
                  className="rounded-full bg-surface-secondary px-4 py-2 text-xs font-medium text-muted transition-colors hover:text-foreground"
                  onClick={cancel}
                  type="button"
                >
                  {t("stages.cancel")}
                </button>
                <button
                  className="rounded-full bg-brand-500 px-5 py-2 text-xs font-semibold text-white transition-opacity hover:opacity-90"
                  onClick={confirm}
                  type="button"
                >
                  {t("stages.confirm")}
                </button>
              </div>
            </footer>
          </div>
        </div>
      )}
    </>
  );
};

function localizeGroup(
  label: string,
  t: (key: string, options?: Record<string, unknown>) => string,
): string {
  const fixed: Record<string, string> = {
    主线: t("stages.group.main"),
    资源收集: t("stages.group.resource"),
    芯片: t("stages.group.chips"),
    剿灭: t("stages.group.annihilation"),
    其他: t("stages.group.other"),
  };
  if (fixed[label]) {
    return fixed[label];
  }
  const chapter = /^第 (\d+) 章$/.exec(label);
  if (chapter) {
    return t("stages.group.chapter", { number: chapter[1] });
  }
  const event = /^活动 ([A-Za-z]+)$/.exec(label);
  if (event) {
    return `${t("stages.group.event")} ${event[1].toUpperCase()}`;
  }
  return label;
}
