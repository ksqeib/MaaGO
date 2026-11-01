import { useTranslation } from "react-i18next";
import { FaPlay } from "react-icons/fa";

import { Field } from "@/components/Field";
import { Switch } from "@/components/Switch";
import { TaskIcon } from "@/components/TaskIcon";
import type { TaskMeta } from "@/data/tasks";
import type { TaskConfig } from "@/types/maa";
import { pickText } from "@/utils/text";

interface TaskSettingsFormProps {
  task: TaskConfig;
  meta: TaskMeta;
  onChange: (options: Record<string, unknown>) => void;
  onToggle: (enabled: boolean) => void;
  onRun: () => void;
}

export const TaskSettingsForm = ({
  task,
  meta,
  onChange,
  onToggle,
  onRun,
}: TaskSettingsFormProps) => {
  const { t, i18n } = useTranslation();
  const language = i18n.resolvedLanguage;

  return (
    <div className="flex min-h-0 flex-col">
      <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border/70 p-4">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-brand-500/10 text-brand-600 dark:text-brand-400">
            <TaskIcon name={meta.icon} size={18} />
          </div>
          <div>
            <h2 className="text-lg font-bold">{pickText(meta.label, language)}</h2>
            <p className="mt-0.5 max-w-xl text-xs text-muted">
              {pickText(meta.description, language)}
            </p>
          </div>
        </div>
        <label className="flex items-center gap-2 text-sm font-medium">
          {t("tasks.enable")}
          <Switch checked={task.enabled} onChange={onToggle} />
        </label>
      </div>

      <div className="grid gap-4 p-4 sm:grid-cols-2">
        {meta.fields.map((field) => (
          <Field
            field={field}
            key={field.key}
            onChange={(value) => onChange({ ...task.options, [field.key]: value })}
            value={task.options[field.key]}
          />
        ))}
      </div>

      <div className="mt-auto flex items-center justify-between gap-3 border-t border-border/70 p-4">
        <span className="text-[11px] text-muted">{t("settings.saved")}</span>
        <button
          className="flex items-center gap-2 rounded-full bg-brand-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-brand-600"
          onClick={onRun}
          type="button"
        >
          <FaPlay size={11} />
          {t("tasks.run_this_task")}
        </button>
      </div>
    </div>
  );
};
