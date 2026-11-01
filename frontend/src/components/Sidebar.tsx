import { Button, Tooltip } from "@heroui/react";
import { motion } from "framer-motion";
import React from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "react-router-dom";
import {
  FaCog,
  FaInfoCircle,
  FaListUl,
  FaRocket,
  FaScroll,
} from "react-icons/fa";

import { ThemeSwitcher } from "@/components/ThemeSwitcher";
import { ROUTES, isRouteActive } from "@/constants/routes";

interface SidebarProps {
  tryNavigate: (path: string | number) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({ tryNavigate }) => {
  const { t } = useTranslation();
  const location = useLocation();

  const navItems = [
    {
      key: "home",
      label: t("nav.home"),
      path: ROUTES.home,
      icon: <FaRocket size={20} />,
    },
    {
      key: "tasks",
      label: t("nav.tasks"),
      path: ROUTES.tasks,
      icon: <FaListUl size={20} />,
    },
    {
      key: "logs",
      label: t("nav.logs"),
      path: ROUTES.logs,
      icon: <FaScroll size={20} />,
    },
    {
      key: "settings",
      label: t("nav.settings"),
      path: ROUTES.settings,
      icon: <FaCog size={20} />,
    },
    {
      key: "about",
      label: t("nav.about"),
      path: ROUTES.about,
      icon: <FaInfoCircle size={20} />,
    },
  ];

  return (
    <div className="fixed bottom-0 left-0 top-14 z-50 flex w-14 flex-col">
      <nav
        aria-label={t("nav.breadcrumb")}
        className="scrollbar-hide flex w-full flex-1 flex-col items-center gap-4 overflow-y-auto overflow-x-hidden px-2 py-4"
      >
        {navItems.map((item) => {
          const isActive = isRouteActive(location.pathname, item.path);
          return (
            <Tooltip key={item.key} closeDelay={0} delay={0}>
              <div className="group relative">
                {isActive && (
                  <motion.div
                    className="absolute -left-2 top-1/2 h-8 w-1 -translate-y-1/2 rounded-r-full bg-brand-500"
                    layoutId="active-pill"
                    transition={{ type: "spring", stiffness: 300, damping: 30 }}
                  />
                )}
                <Button
                  isIconOnly
                  aria-current={isActive ? "page" : undefined}
                  aria-label={item.label}
                  className={`h-12 w-12 rounded-xl transition-all duration-200 ${
                    isActive
                      ? "bg-brand-500/10 text-brand-600 dark:text-brand-400"
                      : "text-zinc-500 hover:bg-surface-secondary hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                  }`}
                  onPress={() => tryNavigate(item.path)}
                  variant={isActive ? "secondary" : "ghost"}
                >
                  {item.icon}
                </Button>
              </div>
              <Tooltip.Content placement="right">{item.label}</Tooltip.Content>
            </Tooltip>
          );
        })}
      </nav>

      <div className="flex w-full flex-col items-center gap-4 px-2 pb-6">
        <Tooltip closeDelay={0} delay={0}>
          <div>
            <ThemeSwitcher className="h-12 w-12 rounded-xl text-zinc-500 transition-all duration-200 hover:bg-surface-secondary hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100" />
          </div>
          <Tooltip.Content placement="right">{t("theme.toggle")}</Tooltip.Content>
        </Tooltip>
      </div>
    </div>
  );
};
