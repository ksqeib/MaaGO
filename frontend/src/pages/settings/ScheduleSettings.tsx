import { useTranslation } from "react-i18next";
import { FaCalendarCheck, FaPlus, FaTrash } from "react-icons/fa";

import { SectionCard, SettingRow } from "@/components/SettingRow";
import { Switch } from "@/components/Switch";
import { useAppStore } from "@/store/AppStore";
import type { ScheduleEntry } from "@/types/maa";
import { cn } from "@/utils/cn";
import { formatCountdown } from "@/utils/format";

const weekdayKeys = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
const allWeekdays = [1, 2, 3, 4, 5, 6, 7];

const sortEntries = (entries: ScheduleEntry[]) =>
  [...entries].sort((a, b) => a.time.localeCompare(b.time));

const newEntry = (existing: ScheduleEntry[]): ScheduleEntry => {
  const used = new Set(existing.map((entry) => entry.time));
  let hour = 4;
  const clock = (value: number) => `${String(value).padStart(2, "0")}:00`;
  while (hour < 23 && used.has(clock(hour))) {
    hour += 1;
  }
  return {
    id: `schedule-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
    time: clock(hour),
    weekdays: [...allWeekdays],
    enabled: true,
    force: false,
    lastTriggered: "",
    lastNotified: "",
  };
};

export const ScheduleSettings = () => {
  const { t } = useTranslation();
  const { config, updateConfig, schedule } = useAppStore();
  const entries = config.schedule.entries ?? [];

  const updateSchedule = (patch: Partial<typeof config.schedule>) => {
    updateConfig((previous) => ({
      ...previous,
      schedule: { ...previous.schedule, ...patch },
    }));
  };

  const updateEntries = (next: ScheduleEntry[]) => {
    updateSchedule({ entries: sortEntries(next) });
  };

  const patchEntry = (id: string, patch: Partial<ScheduleEntry>) => {
    updateEntries(
      entries.map((entry) => (entry.id === id ? { ...entry, ...patch } : entry)),
    );
  };

  const toggleWeekday = (entry: ScheduleEntry, day: number) => {
    const selected = new Set(entry.weekdays);
    if (selected.has(day)) {
      if (selected.size === 1) return;
      selected.delete(day);
    } else {
      selected.add(day);
    }
    patchEntry(entry.id, { weekdays: [...selected].sort((a, b) => a - b) });
  };

  return (
    <div className="flex flex-col gap-4">
      <SectionCard
        action={
          <Switch
            checked={config.schedule.enabled}
            onChange={(enabled) => updateSchedule({ enabled })}
          />
        }
        description={t("settings.schedule.note")}
        title={t("settings.schedule.enabled")}
      >
        <SettingRow
          control={
            <Switch
              checked={config.schedule.notifyBefore}
              onChange={(notifyBefore) => updateSchedule({ notifyBefore })}
            />
          }
          title={t("settings.schedule.notify_before")}
        />
        <SettingRow
          control={
            <input
              className="h-9 w-24 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none"
              min={0}
              onChange={(event) =>
                updateSchedule({ notifyMinutes: Number(event.target.value) || 0 })
              }
              type="number"
              value={config.schedule.notifyMinutes}
            />
          }
          title={t("settings.schedule.notify_minutes")}
        />
      </SectionCard>

      <SectionCard
        action={
          <button
            className="flex items-center gap-2 rounded-full bg-brand-500 px-3.5 py-2 text-xs font-semibold text-white transition-opacity hover:opacity-90"
            onClick={() => updateEntries([...entries, newEntry(entries)])}
            type="button"
          >
            <FaPlus size={10} />
            {t("settings.schedule.add_time")}
          </button>
        }
        description={t("settings.schedule.entries_note")}
        title={t("settings.schedule.entries_title")}
      >
        {entries.length === 0 && (
          <p className="py-6 text-center text-sm text-muted">
            {t("settings.schedule.no_entries")}
          </p>
        )}
        <div className="flex flex-col gap-3 py-1">
          {entries.map((entry) => (
            <div
              className="rounded-2xl border border-border/70 bg-surface-secondary/35 p-3"
              key={entry.id}
            >
              <div className="flex flex-wrap items-center gap-3">
                <input
                  className="h-9 w-28 rounded-xl border border-border bg-surface px-3 text-sm outline-none focus:border-brand-500/60"
                  onChange={(event) => patchEntry(entry.id, { time: event.target.value })}
                  type="time"
                  value={entry.time}
                />
                <div className="flex items-center gap-2 text-xs text-muted">
                  <Switch
                    checked={entry.enabled}
                    onChange={(enabled) => patchEntry(entry.id, { enabled })}
                  />
                  <span>{t("settings.schedule.entry_enabled")}</span>
                </div>
                <button
                  className="ml-auto flex items-center gap-1 rounded-full px-2.5 py-1.5 text-[11px] text-muted transition-colors hover:bg-red-500/10 hover:text-red-500"
                  onClick={() =>
                    updateEntries(entries.filter((item) => item.id !== entry.id))
                  }
                  type="button"
                >
                  <FaTrash size={9} />
                  {t("settings.schedule.remove")}
                </button>
              </div>

              <div className="mt-3 flex flex-wrap gap-1.5">
                {weekdayKeys.map((key, index) => {
                  const day = index + 1;
                  const active = entry.weekdays.includes(day);
                  return (
                    <button
                      className={cn(
                        "h-7 min-w-7 rounded-full border px-2 text-[11px] font-semibold transition-colors",
                        active
                          ? "border-brand-500/60 bg-brand-500 text-white"
                          : "border-border bg-surface text-muted hover:text-foreground",
                      )}
                      key={key}
                      onClick={() => toggleWeekday(entry, day)}
                      type="button"
                    >
                      {t(`weekday.${key}`)}
                    </button>
                  );
                })}
              </div>

              <div className="mt-3 flex items-center justify-between gap-3">
                <div>
                  <p className="text-xs font-medium">{t("settings.schedule.force")}</p>
                  <p className="text-[11px] text-muted">
                    {t("settings.schedule.force_hint")}
                  </p>
                </div>
                <Switch
                  checked={entry.force}
                  onChange={(force) => patchEntry(entry.id, { force })}
                />
              </div>
            </div>
          ))}
        </div>
      </SectionCard>

      <SectionCard title={t("dashboard.schedule_title")}>
        <div className="flex items-center gap-3 rounded-2xl bg-surface-secondary/50 p-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-brand-500/10 text-brand-600 dark:text-brand-400">
            <FaCalendarCheck />
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-[10px] uppercase tracking-wider text-muted">
              {t("settings.schedule.next_run")}
            </p>
            <p className="mt-0.5 text-sm font-semibold">
              {schedule?.enabled && schedule.nextRun
                ? schedule.nextRun
                : t("settings.schedule.none")}
            </p>
            {schedule?.enabled && schedule.nextRun && (
              <p className="text-[11px] text-brand-600 dark:text-brand-400">
                {formatCountdown(schedule.secondsUntilRun)}
              </p>
            )}
          </div>
          <div className="text-right">
            <p className="text-[10px] uppercase tracking-wider text-muted">
              {t("settings.schedule.last_triggered")}
            </p>
            <p className="mt-0.5 text-xs font-medium">
              {schedule?.lastTriggered || t("dashboard.never")}
            </p>
          </div>
        </div>
      </SectionCard>
    </div>
  );
};
