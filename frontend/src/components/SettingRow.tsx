import React from "react";

import { cn } from "@/utils/cn";

interface SettingRowProps {
  title: React.ReactNode;
  description?: React.ReactNode;
  control: React.ReactNode;
  className?: string;
}

export const SettingRow: React.FC<SettingRowProps> = ({
  title,
  description,
  control,
  className,
}) => (
  <div
    className={cn(
      "flex flex-wrap items-center justify-between gap-3 border-b border-border/60 py-3 last:border-0",
      className,
    )}
  >
    <div className="min-w-0 flex-1">
      <p className="text-sm font-medium">{title}</p>
      {description && (
        <p className="mt-0.5 text-[11px] leading-relaxed text-muted">{description}</p>
      )}
    </div>
    <div className="shrink-0">{control}</div>
  </div>
);

export const SectionCard = ({
  title,
  description,
  children,
  action,
}: {
  title: React.ReactNode;
  description?: React.ReactNode;
  children: React.ReactNode;
  action?: React.ReactNode;
}) => (
  <section className="launcher-glass rounded-3xl p-5">
    <div className="flex flex-wrap items-start justify-between gap-2">
      <div>
        <h2 className="text-base font-bold">{title}</h2>
        {description && <p className="mt-0.5 text-[11px] text-muted">{description}</p>}
      </div>
      {action}
    </div>
    <div className="mt-3">{children}</div>
  </section>
);
