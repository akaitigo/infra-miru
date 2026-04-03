"use client";

interface ProgressBarProps {
  value: number;
  max: number;
  label: string;
}

function getBarColor(percent: number): string {
  if (percent >= 80) return "bg-red-500";
  if (percent >= 50) return "bg-yellow-500";
  return "bg-green-500";
}

export default function ProgressBar({ value, max, label }: ProgressBarProps) {
  const percent = max > 0 ? Math.min((value / max) * 100, 100) : 0;
  const rounded = Math.round(percent * 10) / 10;

  return (
    <div className="flex items-center gap-2 min-w-0">
      <div
        className="flex-1 h-4 bg-gray-200 rounded-full overflow-hidden dark:bg-gray-700"
        role="progressbar"
        aria-valuenow={rounded}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={label}
      >
        <div
          className={`h-full rounded-full transition-all ${getBarColor(percent)}`}
          style={{ width: `${String(rounded)}%` }}
        />
      </div>
      <span className="text-xs text-gray-600 dark:text-gray-400 w-12 text-right shrink-0">
        {rounded}%
      </span>
    </div>
  );
}
