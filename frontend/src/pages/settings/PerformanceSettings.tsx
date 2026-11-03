import { useTranslation } from "react-i18next";
import { FaMicrochip } from "react-icons/fa";

import { SectionCard, SettingRow } from "@/components/SettingRow";
import { Switch } from "@/components/Switch";
import { useAppStore } from "@/store/AppStore";

export const PerformanceSettings = () => {
  const { t } = useTranslation();
  const { config, updateConfig } = useAppStore();
  const performance = config.performance ?? { gpuOcr: false, gpuDevice: "0" };

  const updatePerformance = (patch: Partial<typeof performance>) => {
    updateConfig((previous) => ({
      ...previous,
      performance: { ...previous.performance, ...patch },
    }));
  };

  return (
    <div className="flex flex-col gap-4">
      <SectionCard
        action={<FaMicrochip className="text-brand-500" />}
        description={t("settings.performance.note")}
        title={t("settings.performance.title")}
      >
        <SettingRow
          control={
            <Switch
              checked={performance.gpuOcr}
              onChange={(gpuOcr) => updatePerformance({ gpuOcr })}
            />
          }
          description={t("settings.performance.gpu_ocr_hint")}
          title={t("settings.performance.gpu_ocr")}
        />
        <SettingRow
          control={
            <input
              className="h-9 w-40 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none focus:border-brand-500/60 disabled:opacity-50"
              disabled={!performance.gpuOcr}
              onChange={(event) => updatePerformance({ gpuDevice: event.target.value })}
              placeholder="0"
              value={performance.gpuDevice}
            />
          }
          description={t("settings.performance.gpu_device_hint")}
          title={t("settings.performance.gpu_device")}
        />
      </SectionCard>

      <SectionCard title={t("settings.performance.about_title")}>
        <ul className="list-disc space-y-1.5 py-1 pl-5 text-xs leading-5 text-muted">
          <li>{t("settings.performance.about_ocr")}</li>
          <li>{t("settings.performance.about_restart")}</li>
          <li>{t("settings.performance.about_fallback")}</li>
          <li>{t("settings.performance.about_device")}</li>
        </ul>
      </SectionCard>
    </div>
  );
};
