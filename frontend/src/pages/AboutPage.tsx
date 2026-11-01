import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { FaFolderOpen, FaGithub, FaInfoCircle } from "react-icons/fa";

import { PageContainer } from "@/components/PageContainer";
import { PageHeader } from "@/components/PageHeader";
import { APP_VERSION } from "@/constants/app";
import {
  GetAppInfo,
  OpenDataDir,
} from "bindings/MaaGO/internal/app/service";
import type { AppInfo } from "bindings/MaaGO/internal/app/models";

export const AboutPage = () => {
  const { t } = useTranslation();
  const [info, setInfo] = useState<AppInfo | null>(null);

  useEffect(() => {
    let disposed = false;

    GetAppInfo()
      .then((value) => {
        if (!disposed) setInfo(value);
      })
      .catch(() => {
        // Running outside Wails (browser preview) has no backend.
      });

    return () => {
      disposed = true;
    };
  }, []);

  const rows = [
    { key: "about.version", value: info?.version || APP_VERSION },
    { key: "about.stack", value: t("home.stack_value") },
    {
      key: "about.runtime",
      value: info ? `${info.goVersion} · ${info.os}/${info.arch}` : "Go",
    },
    { key: "about.data_dir", value: info?.dataDir || t("about.data_dir_value") },
    { key: "about.upstream", value: t("about.upstream_value") },
  ];

  return (
    <PageContainer>
      <PageHeader description={t("about.subtitle")} title={t("about.title")} />

      <div className="launcher-glass rounded-3xl p-6">
        <div className="flex items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-brand-500/10 text-brand-600 dark:text-brand-400">
            <FaInfoCircle size={20} />
          </div>
          <div>
            <h2 className="text-lg font-bold">MaaGO</h2>
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
          <p className="flex items-center gap-2 text-xs text-muted">
            <FaGithub />
            {t("about.note")}
          </p>
          <button
            className="flex items-center gap-2 rounded-full bg-surface-secondary px-4 py-2 text-sm font-medium text-muted transition-colors hover:text-foreground"
            onClick={() => {
              void OpenDataDir().catch(() => undefined);
            }}
            type="button"
          >
            <FaFolderOpen />
            {t("about.open_data_dir")}
          </button>
        </div>
      </div>
    </PageContainer>
  );
};
