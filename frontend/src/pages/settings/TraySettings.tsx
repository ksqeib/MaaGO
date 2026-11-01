import { useTranslation } from "react-i18next";

import { SectionCard, SettingRow } from "@/components/SettingRow";
import { Switch } from "@/components/Switch";
import { useAppStore } from "@/store/AppStore";

export const TraySettings = () => {
  const { t } = useTranslation();
  const { config, updateConfig } = useAppStore();

  return (
    <div className="flex flex-col gap-4">
      <SectionCard description={t("settings.tray.note")} title={t("settings.tabs.tray")}>
        <SettingRow
          control={
            <Switch
              checked={config.minimizeToTray}
              onChange={(minimizeToTray) =>
                updateConfig((previous) => ({ ...previous, minimizeToTray }))
              }
            />
          }
          title={t("settings.tray.minimize")}
        />
        <SettingRow
          control={
            <Switch
              checked={config.closeToTray}
              onChange={(closeToTray) =>
                updateConfig((previous) => ({ ...previous, closeToTray }))
              }
            />
          }
          title={t("settings.tray.close")}
        />
      </SectionCard>
    </div>
  );
};
