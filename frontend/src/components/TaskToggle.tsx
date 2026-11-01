import { cn } from "@/utils/cn";

export const TaskToggle = ({
  enabled,
  onChange,
}: {
  enabled: boolean;
  onChange: (next: boolean) => void;
}) => (
  <button
    aria-checked={enabled}
    className={cn(
      "flex h-4 w-4 shrink-0 items-center justify-center rounded border transition-colors",
      enabled
        ? "border-brand-500 bg-brand-500 text-white"
        : "border-zinc-400/60 bg-transparent",
    )}
    onClick={(event) => {
      event.stopPropagation();
      onChange(!enabled);
    }}
    role="checkbox"
    type="button"
  >
    {enabled && (
      <svg fill="none" height="10" viewBox="0 0 12 12" width="10">
        <path
          d="M2 6.5 4.6 9 10 3"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="2"
        />
      </svg>
    )}
  </button>
);
