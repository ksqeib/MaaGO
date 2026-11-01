import React from "react";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "react-router-dom";
import { IoChevronForward } from "react-icons/io5";

import { WindowControls } from "@/components/WindowControls";
import { getRouteLabelKey } from "@/constants/routeLabels";
import { ROUTES } from "@/constants/routes";
import { AppLogo } from "@/icons/AppLogo";

interface TopBarProps {
  tryNavigate: (path: string | number) => void;
}

export const TopBar: React.FC<TopBarProps> = ({ tryNavigate }) => {
  const { t } = useTranslation();
  const location = useLocation();
  const pathnames = location.pathname.split("/").filter(Boolean);
  const isHome = pathnames.length === 0;

  return (
    <>
      <div
        id="wails-draggable"
        className="fixed left-0 right-0 top-0 z-[60] flex h-14 items-center justify-between pr-4"
      >
        <div className="flex min-w-0 items-center gap-2 overflow-hidden">
          <div className="flex w-14 shrink-0 items-center justify-center">
            <AppLogo size={30} />
          </div>

          <div className="mx-1 h-5 w-px shrink-0 bg-surface-quaternary/50" />

          <nav
            aria-label={t("nav.breadcrumb")}
            className="flex min-w-0 items-center overflow-hidden whitespace-nowrap text-sm font-medium text-ellipsis"
          >
            <Link
              className={`transition-colors hover:opacity-80 ${
                isHome
                  ? "text-lg font-bold text-brand-700 dark:text-brand-300"
                  : "text-muted hover:text-foreground"
              }`}
              onClick={(event) => {
                event.preventDefault();
                tryNavigate(ROUTES.home);
              }}
              to={ROUTES.home}
            >
              {isHome ? t("common.app_name") : t("nav.home")}
            </Link>

            {pathnames.map((_, index) => {
              const to = `/${pathnames.slice(0, index + 1).join("/")}`;
              const isLast = index === pathnames.length - 1;
              const name = t(getRouteLabelKey(to));

              return (
                <React.Fragment key={to}>
                  <IoChevronForward className="mx-1 shrink-0 text-muted" />
                  {isLast ? (
                    <span
                      aria-current="page"
                      className="truncate font-bold text-foreground"
                      title={name}
                    >
                      {name}
                    </span>
                  ) : (
                    <Link
                      className="truncate text-muted transition-colors hover:text-foreground"
                      onClick={(event) => {
                        event.preventDefault();
                        tryNavigate(to);
                      }}
                      title={name}
                      to={to}
                    >
                      {name}
                    </Link>
                  )}
                </React.Fragment>
              );
            })}
          </nav>
        </div>

        <div className="flex shrink-0 items-center gap-2">
          <WindowControls hideSeparator />
        </div>
      </div>
    </>
  );
};
