import { useTranslation } from "react-i18next";
import { FaCalendarCheck } from "react-icons/fa";

import { SectionCard, SettingRow } from "@/components/SettingRow";
import { Switch } from "@/components/Switch";
import { useAppStore } from "@/store/AppStore";
import { cn } from "@/utils/cn";
import { formatCountdown } from "@/utils/format";

const weekdayKeys = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];

export const ScheduleSettings = () => {
  const { t } = useTranslation();
  const { config, updateConfig, schedule } = useAppStore();

  const updateSchedule = (patch: Partial<typeof config.schedule>) => {
    updateConfig((previous) => ({
      ...previous,
      schedule: { ...previous.schedule, ...patch },
    }));
  };

  const toggleWeekday = (day: number) => {
    const selected = new Set(config.schedule.weekdays);
    if (selected.has(day)) {
      if (selected.size === 1) return;
      selected.delete(day);
    } else {
      selected.add(day);
    }
    updateSchedule({ weekdays: [...selected].sort((a, b) => a - b) });
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
            <input
              className="h-9 w-32 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none"
              onChange={(event) => updateSchedule({ time: event.target.value })}
              type="time"
              value={config.schedule.time}
            />
          }
          title={t("settings.schedule.time")}
        />

        <div className="border-b border-border/60 py-3">
          <p className="text-sm font-medium">{t("settings.schedule.weekdays")}</p>
          <div className="mt-2 flex flex-wrap gap-2">
            {weekdayKeys.map((key, index) => {
              const day = index + 1;
              const active = config.schedule.weekdays.includes(day);
              return (
                <button
                  className={cn(
                    "h-9 w-9 rounded-full border text-xs font-semibold transition-colors",
                    active
                      ? "border-brand-500/60 bg-brand-500 text-white"
                      : "border-border bg-surface-secondary/60 text-muted hover:text-foreground",
                  )}
                  key={key}
                  onClick={() => toggleWeekday(day)}
                  type="button"
                >
                  {t(`weekday.${key}`)}
                </button>
              );
            })}
          </div>
        </div>

        <SettingRow
          control={
            <Switch
              checked={config.schedule.force}
              onChange={(force) => updateSchedule({ force })}
            />
          }
          description={t("settings.schedule.force_hint")}
          title={t("settings.schedule.force")}
        />

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
