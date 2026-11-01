import { useTranslation } from "react-i18next";
import { FaCheckCircle, FaMinusCircle, FaSpinner, FaTimesCircle } from "react-icons/fa";

import type { DiagnosticResult } from "@/types/maa";
import { cn } from "@/utils/cn";

interface DiagnosticStepsProps {
  result: DiagnosticResult | null;
  running: boolean;
}

const statusIcon = (status: string) => {
  switch (status) {
    case "ok":
      return <FaCheckCircle className="text-emerald-500" />;
    case "fail":
      return <FaTimesCircle className="text-red-500" />;
    default:
      return <FaMinusCircle className="text-zinc-400" />;
  }
};

export const DiagnosticSteps = ({ result, running }: DiagnosticStepsProps) => {
  const { t } = useTranslation();

  if (running) {
    return (
      <div className="mt-3 flex items-center gap-2 rounded-xl bg-surface-secondary/50 px-3 py-2 text-[11px] text-muted">
        <FaSpinner className="animate-spin text-brand-500" />
        {t("diag.running")}
      </div>
    );
  }

  if (!result) return null;

  return (
    <div className="mt-3 flex flex-col gap-1.5">
      <div className="flex items-center justify-between">
        <span
          className={cn(
            "rounded-full px-2 py-0.5 text-[10px] font-semibold",
            result.ok
              ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400"
              : "bg-red-500/15 text-red-600 dark:text-red-400",
          )}
        >
          {result.ok ? t("diag.passed") : t("diag.failed")}
        </span>
        {result.system.screenshotPath && (
          <span className="max-w-[60%] truncate font-mono text-[9px] text-muted">
            {result.system.screenshotPath}
          </span>
        )}
      </div>

      {result.steps.map((step) => (
        <div
          className="flex items-start gap-2 rounded-xl bg-surface-secondary/45 px-2.5 py-1.5"
          key={step.key}
        >
          <span className="mt-0.5 text-[11px]">{statusIcon(step.status)}</span>
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline justify-between gap-2">
              <span className="text-[11px] font-medium">{t(`diag.step.${step.key}`)}</span>
              <span className="shrink-0 text-[9px] text-muted">
                {t("diag.duration", { ms: step.durationMs })}
              </span>
            </div>
            <p className="break-all text-[10px] leading-relaxed text-muted">
              {step.detail === "blocked by an earlier step"
                ? t("diag.blocked")
                : step.detail}
            </p>
          </div>
        </div>
      ))}
    </div>
  );
};
