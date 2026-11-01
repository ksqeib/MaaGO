import { useTranslation } from "react-i18next";
import { FaFolderOpen, FaGithub } from "react-icons/fa";

import { PageHeader } from "@/components/PageHeader";
import { APP_VERSION } from "@/constants/app";
import { AppLogo } from "@/icons/AppLogo";
import { useAppStore } from "@/store/AppStore";

export const AboutPage = () => {
  const { t } = useTranslation();
  const { appInfo, openDataDir, openLogsDir } = useAppStore();

  const rows = [
    { key: "about.version", value: appInfo?.version || APP_VERSION },
    { key: "about.commit", value: appInfo?.commit || "dev" },
    { key: "about.stack", value: t("home.stack_value") },
    {
      key: "about.runtime",
      value: appInfo ? `${appInfo.goVersion} · ${appInfo.os}/${appInfo.arch}` : "Go",
    },
    { key: "about.data_dir", value: appInfo?.dataDir || "%APPDATA%\\MaaGO" },
    { key: "about.logs_dir", value: appInfo?.logsDir || "%APPDATA%\\MaaGO\\logs" },
    { key: "about.upstream", value: t("about.upstream_value") },
  ];

  return (
    <div className="scrollbar-hide h-full w-full overflow-y-auto px-4 pb-4 pt-[var(--content-pt)]">
      <div className="mx-auto flex max-w-3xl flex-col gap-4">
        <PageHeader description={t("about.subtitle")} title={t("about.title")} />

        <section className="launcher-glass rounded-3xl p-6">
          <div className="flex items-center gap-4">
            <AppLogo size={56} />
            <div>
              <h2 className="text-xl font-bold">MaaGO</h2>
              <p className="text-xs text-muted">{t("home.subtitle")}</p>
            </div>
          </div>

          <dl className="mt-6 divide-y divide-border">
            {rows.map((row) => (
              <div
                className="flex flex-wrap items-center justify-between gap-2 py-3 text-sm"
                key={row.key}
              >
                <dt className="text-muted">{t(row.key)}</dt>
                <dd className="max-w-[70%] break-all text-right font-medium">
                  {row.value}
                </dd>
              </div>
            ))}
          </dl>

          <div className="mt-5 flex flex-wrap items-center justify-between gap-3">
            <p className="flex items-center gap-2 text-xs leading-relaxed text-muted">
              <FaGithub className="shrink-0" />
              {t("about.note")}
            </p>
            <div className="flex gap-2">
              <button
                className="flex items-center gap-2 rounded-full bg-surface-secondary px-3.5 py-2 text-xs font-medium text-muted hover:text-foreground"
                onClick={openDataDir}
                type="button"
              >
                <FaFolderOpen size={10} />
                {t("about.open_data_dir")}
              </button>
              <button
                className="flex items-center gap-2 rounded-full bg-surface-secondary px-3.5 py-2 text-xs font-medium text-muted hover:text-foreground"
                onClick={openLogsDir}
                type="button"
              >
                <FaFolderOpen size={10} />
                {t("about.open_logs_dir")}
              </button>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
};
