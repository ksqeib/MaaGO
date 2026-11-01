import { useTheme } from "next-themes";
import { useTranslation } from "react-i18next";
import { FaLanguage, FaMoon, FaSun, FaDesktop } from "react-icons/fa";

import { PageContainer } from "@/components/PageContainer";
import { PageHeader } from "@/components/PageHeader";
import { cn } from "@/utils/cn";

const themeOptions = [
  { value: "light", icon: <FaSun />, label: "theme.light" },
  { value: "dark", icon: <FaMoon />, label: "theme.dark" },
  { value: "system", icon: <FaDesktop />, label: "theme.system" },
];

export const SettingsPage = () => {
  const { t, i18n } = useTranslation();
  const { theme, setTheme } = useTheme();

  const changeLanguage = (language: "zh_CN" | "en_US") => {
    void i18n.changeLanguage(language);
  };

  return (
    <PageContainer>
      <PageHeader
        description={t("settings.subtitle")}
        title={t("settings.title")}
      />

      <div className="launcher-glass rounded-3xl p-6">
        <h2 className="text-lg font-bold">{t("settings.appearance")}</h2>

        <div className="mt-4 grid gap-3 sm:grid-cols-3">
          {themeOptions.map((option) => {
            const active = (theme ?? "system") === option.value;
            return (
              <button
                key={option.value}
                className={cn(
                  "flex items-center gap-3 rounded-2xl border px-4 py-3 text-left text-sm font-medium transition-colors",
                  active
                    ? "border-brand-500/60 bg-brand-500/10 text-brand-700 dark:text-brand-300"
                    : "border-border bg-surface-secondary/60 text-muted hover:text-foreground",
                )}
                onClick={() => setTheme(option.value)}
                type="button"
              >
                <span className="text-base">{option.icon}</span>
                {t(option.label)}
              </button>
            );
          })}
        </div>

        <div className="mt-6 flex flex-wrap items-center justify-between gap-3 border-t border-border pt-5">
          <div className="flex items-center gap-2 text-sm font-medium">
            <FaLanguage />
            {t("settings.language")}
          </div>
          <div className="flex gap-2">
            <button
              className={cn(
                "rounded-full px-4 py-2 text-sm font-medium transition-colors",
                i18n.resolvedLanguage?.startsWith("zh")
                  ? "bg-brand-500 text-white"
                  : "bg-surface-secondary text-muted hover:text-foreground",
              )}
              onClick={() => changeLanguage("zh_CN")}
              type="button"
            >
              {t("settings.language_zh")}
            </button>
            <button
              className={cn(
                "rounded-full px-4 py-2 text-sm font-medium transition-colors",
                i18n.resolvedLanguage?.startsWith("en")
                  ? "bg-brand-500 text-white"
                  : "bg-surface-secondary text-muted hover:text-foreground",
              )}
              onClick={() => changeLanguage("en_US")}
              type="button"
            >
              {t("settings.language_en")}
            </button>
          </div>
        </div>

        <p className="mt-5 text-xs text-muted">{t("settings.saved_locally")}</p>
      </div>
    </PageContainer>
  );
};
