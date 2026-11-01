import { Button } from "@heroui/react";
import { Events, Window } from "@wailsio/runtime";
import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import { cn } from "@/utils/cn";

const controlButtonClassName =
  "wails-no-drag size-8 min-w-8 shrink-0 rounded-lg p-0 hover:bg-default data-[hovered=true]:bg-default";

const WindowControlIcon = ({ children }: { children: React.ReactNode }) => (
  <svg
    aria-hidden="true"
    className="size-4 shrink-0"
    fill="none"
    focusable="false"
    height="16"
    stroke="currentColor"
    strokeLinecap="round"
    strokeLinejoin="round"
    strokeWidth="1.25"
    viewBox="0 0 16 16"
    width="16"
  >
    {children}
  </svg>
);

interface WindowControlsProps {
  className?: string;
  hideSeparator?: boolean;
}

export const WindowControls: React.FC<WindowControlsProps> = ({
  className,
  hideSeparator,
}) => {
  const { t } = useTranslation();
  const [isMaximized, setIsMaximized] = useState(false);

  useEffect(() => {
    let disposed = false;
    const sync = async () => {
      if (disposed) return;
      try {
        setIsMaximized(await Window.IsMaximised());
      } catch {
        // Not running inside Wails (for example `npm run dev` in a browser).
      }
    };

    const events = [
      Events.Types.Common.WindowMaximise,
      Events.Types.Common.WindowUnMaximise,
      Events.Types.Common.WindowRestore,
    ];
    const offs = events.map((name) => Events.On(name, () => void sync()));
    void sync();

    return () => {
      disposed = true;
      offs.forEach((off) => off());
    };
  }, []);

  const safeWindowCall = (fn: () => unknown) => {
    void Promise.resolve(fn()).catch(() => undefined);
  };

  return (
    <div className={cn("flex shrink-0 items-center gap-1", className)}>
      {!hideSeparator && <div className="mx-2 h-6 w-px bg-surface-quaternary/60" />}

      <Button
        isIconOnly
        aria-label={t("window.minimize")}
        className={controlButtonClassName}
        onPress={() => safeWindowCall(() => Window.Minimise())}
        size="sm"
        variant="ghost"
      >
        <WindowControlIcon>
          <path d="M3 8h10" />
        </WindowControlIcon>
      </Button>

      <Button
        isIconOnly
        aria-label={isMaximized ? t("window.restore") : t("window.maximize")}
        className={controlButtonClassName}
        onPress={() => safeWindowCall(() => Window.ToggleMaximise())}
        size="sm"
        variant="ghost"
      >
        <WindowControlIcon>
          {isMaximized ? (
            <>
              <path d="M5 5V3h8v8h-2" />
              <rect height="8" rx="0.75" width="8" x="3" y="5" />
            </>
          ) : (
            <rect height="10" rx="0.75" width="10" x="3" y="3" />
          )}
        </WindowControlIcon>
      </Button>

      <Button
        isIconOnly
        aria-label={t("window.close")}
        className={cn(
          controlButtonClassName,
          "hover:bg-danger data-[hovered=true]:bg-danger",
        )}
        onPress={() => safeWindowCall(() => Window.Close())}
        size="sm"
        variant="ghost"
      >
        <WindowControlIcon>
          <path d="m3 3 10 10M13 3 3 13" />
        </WindowControlIcon>
      </Button>
    </div>
  );
};
