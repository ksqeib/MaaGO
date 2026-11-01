import { useTranslation } from "react-i18next";

import { SectionCard, SettingRow } from "@/components/SettingRow";
import { Switch } from "@/components/Switch";
import { useAppStore } from "@/store/AppStore";

const selectClass =
  "h-9 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none";

const clientTypes = [
  { value: "Official", zh: "官服", en: "Official" },
  { value: "YoStarEN", zh: "美服", en: "YoStarEN" },
  { value: "YoStarJP", zh: "日服", en: "YoStarJP" },
  { value: "YoStarKR", zh: "韩服", en: "YoStarKR" },
  { value: "txwy", zh: "台服", en: "txwy" },
];

export const RunSettings = () => {
  const { t, i18n } = useTranslation();
  const { config, updateConfig } = useAppStore();
  const isZh = String(i18n.resolvedLanguage || "").startsWith("zh");

  const updateRun = (patch: Partial<typeof config.run>) => {
    updateConfig((previous) => ({ ...previous, run: { ...previous.run, ...patch } }));
  };

  return (
    <div className="flex flex-col gap-4">
      <SectionCard title={t("settings.tabs.run")}>
        <SettingRow
          control={
            <select
              className={selectClass}
              onChange={(event) =>
                updateRun({
                  clientType: event.target.value,
                })
              }
              value={config.run.clientType}
            >
              {clientTypes.map((client) => (
                <option key={client.value} value={client.value}>
                  {isZh ? client.zh : client.en}
                </option>
              ))}
            </select>
          }
          title={t("settings.run.client_type")}
        />

        <SettingRow
          control={
            <Switch
              checked={config.run.blockSleep}
              onChange={(blockSleep) => updateRun({ blockSleep })}
            />
          }
          title={t("settings.run.block_sleep")}
        />

        <SettingRow
          control={
            <Switch
              checked={config.run.keepScreenOn}
              onChange={(keepScreenOn) => updateRun({ keepScreenOn })}
            />
          }
          title={t("settings.run.keep_screen_on")}
        />

        <SettingRow
          control={
            <input
              className="h-9 w-24 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none"
              min={0}
              onChange={(event) =>
                updateRun({ runDurationMinutes: Number(event.target.value) || 0 })
              }
              type="number"
              value={config.run.runDurationMinutes}
            />
          }
          description={t("settings.run.duration_hint")}
          title={t("settings.run.duration_limit")}
        />
      </SectionCard>

      <SectionCard
        description={t("settings.run.post_action_note")}
        title={t("settings.run.post_action")}
      >
        <SettingRow
          control={
            <select
              className={selectClass}
              onChange={(event) => updateRun({ postAction: event.target.value })}
              value={config.run.postAction}
            >
              <option value="none">{t("settings.run.post_action_none")}</option>
              <option value="exitArknights">
                {t("settings.run.post_action_exit_arknights")}
              </option>
              <option value="exitEmulator">
                {t("settings.run.post_action_exit_emulator")}
              </option>
              <option value="exitSelf">{t("settings.run.post_action_exit_self")}</option>
              <option value="lockScreen">{t("settings.run.post_action_lock")}</option>
              <option value="sleep">{t("settings.run.post_action_sleep")}</option>
              <option value="hibernate">{t("settings.run.post_action_hibernate")}</option>
              <option value="shutdown">{t("settings.run.post_action_shutdown")}</option>
            </select>
          }
          title={t("settings.run.post_action")}
        />

        <SettingRow
          control={
            <Switch
              checked={config.run.postActionOnlyWhenNoOther}
              onChange={(postActionOnlyWhenNoOther) =>
                updateRun({ postActionOnlyWhenNoOther })
              }
            />
          }
          title={t("settings.run.only_no_other")}
        />

        <SettingRow
          control={
            <Switch
              checked={config.run.postActionSkipOnError}
              onChange={(postActionSkipOnError) => updateRun({ postActionSkipOnError })}
            />
          }
          title={t("settings.run.skip_on_error")}
        />
      </SectionCard>
    </div>
  );
};
