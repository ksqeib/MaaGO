import { useTranslation } from "react-i18next";
import { FaFolderOpen, FaImage } from "react-icons/fa";

import { LogConsole } from "@/components/LogConsole";
import { PageHeader } from "@/components/PageHeader";
import { useAppStore } from "@/store/AppStore";

export const LogsPage = () => {
  const { t } = useTranslation();
  const { logs, clearLogs, openLogsDir, openCoreLogsDir, openFailureShotsDir } = useAppStore();

  return (
    <div className="flex h-full min-h-0 w-full flex-col gap-3 overflow-hidden px-4 pb-4 pt-[var(--content-pt)]">
      <PageHeader
        description={t("logs.subtitle")}
        endContent={
          <div className="flex gap-2">
            <button
              className="flex items-center gap-2 rounded-full bg-surface-secondary px-3.5 py-2 text-xs font-medium text-muted transition-colors hover:text-foreground"
              onClick={openCoreLogsDir}
              type="button"
            >
              <FaFolderOpen size={11} />
              {t("logs.core_dir")}
            </button>
            <button
              className="flex items-center gap-2 rounded-full bg-surface-secondary px-3.5 py-2 text-xs font-medium text-muted transition-colors hover:text-foreground"
              onClick={openFailureShotsDir}
              type="button"
            >
              <FaImage size={11} />
              {t("logs.shots_dir")}
            </button>
            <button
              className="flex items-center gap-2 rounded-full bg-surface-secondary px-3.5 py-2 text-xs font-medium text-muted transition-colors hover:text-foreground"
              onClick={openLogsDir}
              type="button"
            >
              <FaFolderOpen size={11} />
              {t("logs.open_dir")}
            </button>
          </div>
        }
        title={t("logs.title")}
      />

      <section className="launcher-glass flex min-h-0 flex-1 flex-col overflow-hidden rounded-3xl">
        <header className="flex items-center justify-between border-b border-border/70 px-4 py-2.5">
          <span className="text-xs font-medium text-muted">
            {t("logs.count", { count: logs.length })}
          </span>
        </header>
        <LogConsole
          className="min-h-0 flex-1"
          emptyText={t("dashboard.no_logs")}
          logs={logs}
          onClear={clearLogs}
          showControls
        />
      </section>
    </div>
  );
};
