import { Button } from "@heroui/react";
import { useTheme } from "next-themes";
import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import { MoonIcon } from "@/icons/MoonIcon";
import { SunIcon } from "@/icons/SunIcon";
import { cn } from "@/utils/cn";

interface ThemeSwitcherProps {
  className?: string;
  iconSize?: number;
}

export const ThemeSwitcher: React.FC<ThemeSwitcherProps> = ({
  className,
  iconSize = 20,
}) => {
  const { t } = useTranslation();
  const { resolvedTheme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);

  useEffect(() => setMounted(true), []);

  if (!mounted) return <div className={className ?? "h-8 w-8"} />;

  const isDark = resolvedTheme === "dark";

  return (
    <Button
      isIconOnly
      aria-label={t("theme.toggle")}
      className={cn(
        "rounded-full",
        className ??
          "w-8 h-8 text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100",
      )}
      onPress={() => setTheme(isDark ? "light" : "dark")}
      size="sm"
      variant="ghost"
    >
      {isDark ? (
        <MoonIcon size={iconSize} />
      ) : (
        <SunIcon size={iconSize} />
      )}
    </Button>
  );
};
