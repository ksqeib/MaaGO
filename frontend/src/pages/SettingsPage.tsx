import { useState } from "react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import {
  FaBell,
  FaClock,
  FaCloudDownloadAlt,
  FaDesktop,
  FaPalette,
  FaPlayCircle,
  FaPlug,
} from "react-icons/fa";

import { PageHeader } from "@/components/PageHeader";
import { useAppStore } from "@/store/AppStore";
import { cn } from "@/utils/cn";

import { AppearanceSettings } from "./settings/AppearanceSettings";
import { ConnectionSettings } from "./settings/ConnectionSettings";
import { NotificationSettings } from "./settings/NotificationSettings";
import { RunSettings } from "./settings/RunSettings";
import { ScheduleSettings } from "./settings/ScheduleSettings";
import { TraySettings } from "./settings/TraySettings";
import { UpdateSettings } from "./settings/UpdateSettings";

type TabKey = "connection" | "run" | "schedule" | "notification" | "appearance" | "tray" | "update";

const tabs: { key: TabKey; icon: ReactNode }[] = [
  { key: "connection", icon: <FaPlug /> },
  { key: "run", icon: <FaPlayCircle /> },
  { key: "schedule", icon: <FaClock /> },
  { key: "notification", icon: <FaBell /> },
  { key: "appearance", icon: <FaPalette /> },
  { key: "tray", icon: <FaDesktop /> },
  { key: "update", icon: <FaCloudDownloadAlt /> },
];

export const SettingsPage = () => {
  const { t } = useTranslation();
  const { saveState } = useAppStore();
  const [activeTab, setActiveTab] = useState<TabKey>("connection");

  const content = (() => {
    switch (activeTab) {
      case "run":
        return <RunSettings />;
      case "schedule":
        return <ScheduleSettings />;
      case "notification":
        return <NotificationSettings />;
      case "appearance":
        return <AppearanceSettings />;
      case "tray":
        return <TraySettings />;
      case "update":
        return <UpdateSettings />;
      default:
        return <ConnectionSettings />;
    }
  })();

  return (
    <div className="flex h-full min-h-0 w-full flex-col gap-3 overflow-hidden px-4 pb-4 pt-[var(--content-pt)]">
      <PageHeader
        description={t("settings.subtitle")}
        endContent={
          <span
            className={cn(
              "rounded-full px-3 py-1 text-[11px] font-medium",
              saveState === "error"
                ? "bg-red-500/10 text-red-600 dark:text-red-400"
                : "bg-surface-secondary text-muted",
            )}
          >
            {saveState === "saving"
              ? t("common.saving")
              : saveState === "error"
                ? t("common.save_error")
                : t("common.saved")}
          </span>
        }
        title={t("settings.title")}
      />

      <div className="flex min-h-0 flex-1 flex-col gap-3 lg:flex-row">
        <aside className="launcher-glass scrollbar-hide flex shrink-0 flex-row gap-1 overflow-x-auto rounded-3xl p-2 lg:w-44 lg:flex-col lg:overflow-y-auto">
          {tabs.map((tab) => {
            const active = tab.key === activeTab;
            return (
              <button
                className={cn(
                  "flex shrink-0 items-center gap-2 rounded-2xl px-3 py-2.5 text-left text-sm font-medium transition-colors",
                  active
                    ? "bg-brand-500/10 text-brand-700 dark:text-brand-300"
                    : "text-muted hover:bg-surface-secondary/60 hover:text-foreground",
                )}
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                type="button"
              >
                <span className="text-xs">{tab.icon}</span>
                <span className="whitespace-nowrap">{t(`settings.tabs.${tab.key}`)}</span>
              </button>
            );
          })}
        </aside>

        <div className="min-h-0 flex-1 overflow-y-auto px-0.5 pb-2">{content}</div>
      </div>
    </div>
  );
};
