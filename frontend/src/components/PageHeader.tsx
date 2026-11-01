import React from "react";

interface PageHeaderProps {
  title: React.ReactNode;
  description?: React.ReactNode;
  className?: string;
  titleClassName?: string;
  descriptionClassName?: string;
  startContent?: React.ReactNode;
  endContent?: React.ReactNode;
}

export const PageHeader: React.FC<PageHeaderProps> = ({
  title,
  description,
  className = "",
  titleClassName = "",
  descriptionClassName = "",
  startContent,
  endContent,
}) => (
  <div className={className}>
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-center gap-4">
        {startContent}
        <div>
          <h1
            className={`text-left text-3xl font-black tracking-tight text-brand-600 dark:text-brand-500 pb-1 ${titleClassName}`}
          >
            {title}
          </h1>
          {description && (
            <div
              className={`mt-1 text-base font-medium text-muted sm:text-lg ${descriptionClassName}`}
            >
              {description}
            </div>
          )}
        </div>
      </div>
      {endContent && <div className="flex items-center gap-2">{endContent}</div>}
    </div>
  </div>
);

interface SectionHeaderProps {
  title: React.ReactNode;
  description?: React.ReactNode;
  icon?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
}

export const SectionHeader: React.FC<SectionHeaderProps> = ({
  title,
  description,
  icon,
  action,
  className = "",
}) => (
  <div className={`flex flex-col gap-1 ${className}`}>
    <div className="flex w-full items-center justify-between">
      <div className="flex items-center gap-2">
        {icon && (
          <div className="rounded-xl bg-brand-500/10 p-2 text-brand-600 dark:text-brand-500">
            {icon}
          </div>
        )}
        <h2 className="text-xl font-bold text-foreground">{title}</h2>
      </div>
      {action}
    </div>
    {description && (
      <p className={`text-sm text-muted ${icon ? "ml-12" : ""}`}>
        {description}
      </p>
    )}
  </div>
);
