import React from "react";

import { cn } from "@/utils/cn";

export const ProgressBar = ({
  value,
  className,
}: {
  value: number;
  className?: string;
}) => (
  <div
    aria-valuemax={100}
    aria-valuemin={0}
    aria-valuenow={Math.round(value)}
    className={cn(
      "h-2 w-full overflow-hidden rounded-full bg-surface-quaternary/50",
      className,
    )}
    role="progressbar"
  >
    <div
      className="h-full rounded-full bg-brand-500 transition-[width] duration-300"
      style={{ width: `${Math.min(100, Math.max(0, value))}%` }}
    />
  </div>
);
