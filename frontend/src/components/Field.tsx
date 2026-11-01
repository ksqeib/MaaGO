import { useTranslation } from "react-i18next";

import { Switch } from "@/components/Switch";
import type { FieldDef } from "@/data/tasks";
import { cn } from "@/utils/cn";
import { pickText } from "@/utils/text";

interface FieldProps {
  field: FieldDef;
  value: unknown;
  onChange: (value: unknown) => void;
}

const inputClass =
  "h-10 w-full rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none transition-colors focus:border-brand-500/60 focus:bg-surface";

export const Field = ({ field, value, onChange }: FieldProps) => {
  const { i18n } = useTranslation();
  const language = i18n.resolvedLanguage;
  const label = pickText(field.label, language);
  const hint = field.hint ? pickText(field.hint, language) : "";
  const placeholder = field.placeholder
    ? pickText(field.placeholder, language)
    : "";

  const control = (() => {
    switch (field.type) {
      case "switch":
        return (
          <Switch
            checked={Boolean(value)}
            label={label}
            onChange={(checked) => onChange(checked)}
          />
        );
      case "number":
        return (
          <input
            className={inputClass}
            max={field.max}
            min={field.min}
            onChange={(event) => {
              const next = Number(event.target.value);
              onChange(Number.isFinite(next) ? next : 0);
            }}
            step={field.step ?? 1}
            type="number"
            value={typeof value === "number" ? value : Number(value) || 0}
          />
        );
      case "select":
        return (
          <select
            className={cn(inputClass, "appearance-none pr-8")}
            onChange={(event) => onChange(event.target.value)}
            value={String(value ?? "")}
          >
            {(field.options ?? []).map((option) => (
              <option key={option.value} value={option.value}>
                {pickText(option.label, language)}
              </option>
            ))}
          </select>
        );
      case "multiselect": {
        const selected = Array.isArray(value) ? (value as string[]) : [];
        return (
          <div className="flex flex-wrap gap-2">
            {(field.options ?? []).map((option) => {
              const active = selected.includes(option.value);
              return (
                <button
                  className={cn(
                    "rounded-full border px-3 py-1.5 text-xs font-medium transition-colors",
                    active
                      ? "border-brand-500/60 bg-brand-500/15 text-brand-700 dark:text-brand-300"
                      : "border-border bg-surface-secondary/60 text-muted hover:text-foreground",
                  )}
                  key={option.value}
                  onClick={() =>
                    onChange(
                      active
                        ? selected.filter((item) => item !== option.value)
                        : [...selected, option.value],
                    )
                  }
                  type="button"
                >
                  {pickText(option.label, language)}
                </button>
              );
            })}
          </div>
        );
      }
      default:
        return (
          <input
            className={inputClass}
            onChange={(event) => onChange(event.target.value)}
            placeholder={placeholder}
            type="text"
            value={String(value ?? "")}
          />
        );
    }
  })();

  const isWide = field.type === "multiselect";

  return (
    <div className={cn("flex flex-col gap-1.5", isWide && "sm:col-span-2")}>
      <div className="flex items-center justify-between gap-3">
        <label className="text-sm font-medium">{label}</label>
        {field.type !== "switch" && !isWide && hint && (
          <span className="text-[11px] text-muted">{hint}</span>
        )}
      </div>
      {control}
      {field.type !== "switch" && isWide && hint && (
        <span className="text-[11px] text-muted">{hint}</span>
      )}
    </div>
  );
};
