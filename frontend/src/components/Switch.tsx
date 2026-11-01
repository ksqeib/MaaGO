import React from "react";

import { cn } from "@/utils/cn";

interface SwitchProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
  label?: string;
  className?: string;
}

export const Switch: React.FC<SwitchProps> = ({
  checked,
  onChange,
  disabled,
  label,
  className,
}) => (
  <button
    aria-checked={checked}
    aria-label={label}
    className={cn(
      "relative inline-flex h-6 w-11 shrink-0 items-center rounded-full border transition-colors",
      checked
        ? "border-brand-500/60 bg-brand-500"
        : "border-border bg-surface-quaternary/60",
      disabled && "cursor-not-allowed opacity-50",
      className,
    )}
    disabled={disabled}
    onClick={() => !disabled && onChange(!checked)}
    role="switch"
    type="button"
  >
    <span
      className={cn(
        "inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform",
        checked ? "translate-x-6" : "translate-x-1",
      )}
    />
  </button>
);
