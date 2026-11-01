import { useTranslation } from "react-i18next";
import { useTheme } from "next-themes";
import { FaDesktop, FaMoon, FaSun } from "react-icons/fa";
import type { ReactNode } from "react";

import { SectionCard } from "@/components/SettingRow";
import { useAppStore } from "@/store/AppStore";
import type { ThemeMode } from "@/types/maa";
import { cn } from "@/utils/cn";

const themes: { value: ThemeMode; icon: ReactNode; label: string }[] = [
  { value: "light", icon: <FaSun size={13} />, label: "settings.appearance.theme_light" },
  { value: "dark", icon: <FaMoon size={13} />, label: "settings.appearance.theme_dark" },
  { value: "system", icon: <FaDesktop size={13} />, label: "settings.appearance.theme_system" },
];

export const AppearanceSettings = () => {
  const { t, i18n } = useTranslation();
  const { setTheme } = useTheme();
  const { config, updateConfig } = useAppStore();

  const changeTheme = (theme: ThemeMode) => {
    setTheme(theme);
    updateConfig((previous) => ({ ...previous, theme }));
  };

  const changeLanguage = (language: "zh_CN" | "en_US") => {
    void i18n.changeLanguage(language);
    updateConfig((previous) => ({ ...previous, language }));
  };

  return (
    <div className="flex flex-col gap-4">
      <SectionCard title={t("settings.appearance.theme")}>
        <div className="grid gap-2 sm:grid-cols-3">
          {themes.map((theme) => {
            const active = config.theme === theme.value;
            return (
              <button
                className={cn(
                  "flex items-center gap-2.5 rounded-2xl border px-4 py-3 text-sm font-medium transition-colors",
                  active
                    ? "border-brand-500/60 bg-brand-500/10 text-brand-700 dark:text-brand-300"
                    : "border-border bg-surface-secondary/50 text-muted hover:text-foreground",
                )}
                key={theme.value}
                onClick={() => changeTheme(theme.value)}
                type="button"
              >
                {theme.icon}
                {t(theme.label)}
              </button>
            );
          })}
        </div>
      </SectionCard>

      <SectionCard title={t("settings.appearance.language")}>
        <div className="flex gap-2">
          <button
            className={cn(
              "rounded-full px-4 py-2 text-sm font-medium transition-colors",
              String(i18n.resolvedLanguage || "").startsWith("zh")
                ? "bg-brand-500 text-white"
                : "bg-surface-secondary text-muted hover:text-foreground",
            )}
            onClick={() => changeLanguage("zh_CN")}
            type="button"
          >
            {t("settings.appearance.language_zh")}
          </button>
          <button
            className={cn(
              "rounded-full px-4 py-2 text-sm font-medium transition-colors",
              String(i18n.resolvedLanguage || "").startsWith("en")
                ? "bg-brand-500 text-white"
                : "bg-surface-secondary text-muted hover:text-foreground",
            )}
            onClick={() => changeLanguage("en_US")}
            type="button"
          >
            {t("settings.appearance.language_en")}
          </button>
        </div>
      </SectionCard>
    </div>
  );
};
