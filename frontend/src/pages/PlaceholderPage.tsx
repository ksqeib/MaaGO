import React from "react";
import { useTranslation } from "react-i18next";
import { FaHourglassHalf } from "react-icons/fa";

import { PageContainer } from "@/components/PageContainer";
import { PageHeader } from "@/components/PageHeader";

interface PlaceholderPageProps {
  titleKey: string;
  subtitleKey: string;
}

export const PlaceholderPage: React.FC<PlaceholderPageProps> = ({
  titleKey,
  subtitleKey,
}) => {
  const { t } = useTranslation();

  return (
    <PageContainer>
      <PageHeader description={t(subtitleKey)} title={t(titleKey)} />
      <div className="launcher-glass flex flex-1 items-center justify-center rounded-3xl p-10">
        <div className="max-w-md text-center">
          <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-brand-500/10 text-brand-600 dark:text-brand-400">
            <FaHourglassHalf size={24} />
          </div>
          <p className="mt-4 text-sm leading-relaxed text-muted">
            {t("common.coming_soon")}
          </p>
        </div>
      </div>
    </PageContainer>
  );
};
