import React, { useId } from "react";

/**
 * Original MaaGO mark: a rounded emerald tile with a stylised M and a small
 * status dot. Deliberately does not reuse the MAA logo.
 */
export const AppLogo = ({
  size = 32,
  className,
}: {
  size?: number;
  className?: string;
}) => {
  const id = useId().replace(/:/g, "");
  const gradient = `maago-gradient-${id}`;
  const shine = `maago-shine-${id}`;

  return (
    <svg
      aria-label="MaaGO"
      className={className}
      height={size}
      role="img"
      viewBox="0 0 64 64"
      width={size}
    >
      <defs>
        <linearGradient id={gradient} x1="10" x2="54" y1="6" y2="60">
          <stop offset="0" stopColor="#34d399" />
          <stop offset="0.55" stopColor="#10b981" />
          <stop offset="1" stopColor="#047857" />
        </linearGradient>
        <linearGradient id={shine} x1="0" x2="0" y1="0" y2="1">
          <stop offset="0" stopColor="#ffffff" stopOpacity="0.28" />
          <stop offset="1" stopColor="#ffffff" stopOpacity="0" />
        </linearGradient>
      </defs>

      <rect fill={`url(#${gradient})`} height="64" rx="16" width="64" />
      <rect
        fill={`url(#${shine})`}
        height="30"
        rx="16"
        width="64"
        x="0"
        y="0"
      />
      <path
        d="M15 45V21.5c0-1.6 1.9-2.4 3-1.2L32 35l14-14.7c1.1-1.2 3-.4 3 1.2V45"
        fill="none"
        stroke="#ffffff"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="6.5"
      />
      <circle cx="50" cy="15" fill="#ecfdf5" r="4" />
    </svg>
  );
};
