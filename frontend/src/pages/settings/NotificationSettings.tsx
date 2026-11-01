import { useState } from "react";
import { useTranslation } from "react-i18next";
import { FaBell } from "react-icons/fa";

import { SectionCard, SettingRow } from "@/components/SettingRow";
import { Switch } from "@/components/Switch";
import { useAppStore } from "@/store/AppStore";

export const NotificationSettings = () => {
  const { t } = useTranslation();
  const { config, updateConfig, testNotification } = useAppStore();
  const [testing, setTesting] = useState(false);

  const updateNotification = (patch: Partial<typeof config.notification>) => {
    updateConfig((previous) => ({
      ...previous,
      notification: { ...previous.notification, ...patch },
    }));
  };

  return (
    <div className="flex flex-col gap-4">
      <SectionCard
        action={
          <Switch
            checked={config.notification.systemEnabled}
            onChange={(systemEnabled) => updateNotification({ systemEnabled })}
          />
        }
        description={t("settings.notification.note")}
        title={t("settings.notification.system")}
      >
        <SettingRow
          control={
            <Switch
              checked={config.notification.onComplete}
              onChange={(onComplete) => updateNotification({ onComplete })}
            />
          }
          title={t("settings.notification.on_complete")}
        />
        <SettingRow
          control={
            <Switch
              checked={config.notification.onError}
              onChange={(onError) => updateNotification({ onError })}
            />
          }
          title={t("settings.notification.on_error")}
        />
        <SettingRow
          control={
            <Switch
              checked={config.notification.onSchedule}
              onChange={(onSchedule) => updateNotification({ onSchedule })}
            />
          }
          title={t("settings.notification.on_schedule")}
        />

        <div className="pt-3">
          <button
            className="flex items-center gap-2 rounded-full bg-brand-500 px-4 py-2 text-xs font-semibold text-white hover:bg-brand-600 disabled:opacity-60"
            disabled={testing}
            onClick={() => {
              setTesting(true);
              void testNotification().finally(() => setTesting(false));
            }}
            type="button"
          >
            <FaBell size={10} />
            {t("settings.notification.test")}
          </button>
        </div>
      </SectionCard>
    </div>
  );
};
