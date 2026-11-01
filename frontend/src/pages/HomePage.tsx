import { useTranslation } from "react-i18next";
import { FaCog, FaCube, FaLanguage, FaLayerGroup, FaWindowRestore } from "react-icons/fa";

import { PageContainer } from "@/components/PageContainer";
import { PageHeader } from "@/components/PageHeader";

const features = [
  { key: "window", icon: <FaWindowRestore /> },
  { key: "nav", icon: <FaLayerGroup /> },
  { key: "theme", icon: <FaCube /> },
  { key: "i18n", icon: <FaLanguage /> },
  { key: "config", icon: <FaCog /> },
] as const;

const featureLabels: Record<string, { zh: string; en: string }> = {
  window: { zh: "无边框窗口与自绘标题栏", en: "Frameless window and custom title bar" },
  nav: { zh: "侧边导航与圆角联体背景", en: "Side rail with joined rounded backdrop" },
  theme: { zh: "浅色 / 深色 / 跟随系统", en: "Light / dark / system theme" },
  i18n: { zh: "简体中文 / English", en: "Simplified Chinese / English" },
  config: { zh: "窗口尺寸与设置持久化", en: "Window size and settings persistence" },
};

export const HomePage = () => {
  const { t, i18n } = useTranslation();
  const isZh = i18n.resolvedLanguage?.startsWith("zh");

  return (
    <PageContainer>
      <PageHeader description={t("home.subtitle")} title={t("home.title")} />

      <div className="grid gap-4 lg:grid-cols-3">
        <div className="launcher-glass rounded-3xl p-6 lg:col-span-2">
          <div className="flex items-center gap-2 text-brand-600 dark:text-brand-400">
            <FaLayerGroup />
            <h2 className="text-lg font-bold">{t("home.ready_title")}</h2>
          </div>
          <p className="mt-2 text-sm leading-relaxed text-muted">
            {t("home.ready_body")}
          </p>

          <ul className="mt-5 grid gap-2 sm:grid-cols-2">
            {features.map((feature) => (
              <li
                key={feature.key}
                className="flex items-center gap-3 rounded-2xl bg-surface-secondary/70 px-3 py-2.5 text-sm"
              >
                <span className="text-brand-600 dark:text-brand-400">
                  {feature.icon}
                </span>
                <span>
                  {isZh ? featureLabels[feature.key].zh : featureLabels[feature.key].en}
                </span>
              </li>
            ))}
          </ul>
        </div>

        <div className="flex flex-col gap-4">
          <div className="launcher-glass rounded-3xl p-6">
            <h2 className="text-lg font-bold text-brand-600 dark:text-brand-400">
              {t("home.next_title")}
            </h2>
            <p className="mt-2 text-sm leading-relaxed text-muted">
              {t("home.next_body")}
            </p>
          </div>

          <div className="launcher-glass rounded-3xl p-6">
            <h2 className="text-sm font-bold uppercase tracking-wider text-muted">
              {t("home.stack")}
            </h2>
            <p className="mt-2 text-sm leading-relaxed">
              {t("home.stack_value")}
            </p>
          </div>
        </div>
      </div>
    </PageContainer>
  );
};
