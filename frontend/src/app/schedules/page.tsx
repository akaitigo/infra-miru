"use client";

import { useCallback, useEffect, useState } from "react";
import CopyButton from "@/components/CopyButton";
import { fetchCronHPA, fetchSchedules } from "@/lib/api";
import type { CronHPAResponse, Schedule } from "@/types/api";

function HourlyLoadChart({ schedule }: { schedule: Schedule }) {
  const maxCpuLoad = Math.max(
    ...schedule.hourly_loads.map((h) => h.avg_cpu_usage),
    1,
  );

  return (
    <div className="mt-3">
      <p className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">
        Hourly CPU Load (24h)
      </p>
      <div className="flex items-end gap-px h-24">
        {schedule.hourly_loads.map((load) => {
          const heightPercent = (load.avg_cpu_usage / maxCpuLoad) * 100;
          const isLowLoad = schedule.low_load_hours.includes(load.hour);
          return (
            <div
              key={load.hour}
              className="flex-1 flex flex-col items-center justify-end"
              title={`${String(load.hour)}:00 - CPU: ${String(Math.round(load.avg_cpu_usage * 10) / 10)}%, Memory: ${String(Math.round(load.avg_memory_usage * 10) / 10)}%`}
            >
              <div
                className={`w-full rounded-t transition-all ${
                  isLowLoad
                    ? "bg-green-400 dark:bg-green-600"
                    : "bg-blue-400 dark:bg-blue-600"
                }`}
                style={{
                  height: `${String(Math.max(heightPercent, 2))}%`,
                }}
              />
            </div>
          );
        })}
      </div>
      <div className="flex justify-between mt-1">
        <span className="text-[10px] text-gray-400">0h</span>
        <span className="text-[10px] text-gray-400">6h</span>
        <span className="text-[10px] text-gray-400">12h</span>
        <span className="text-[10px] text-gray-400">18h</span>
        <span className="text-[10px] text-gray-400">23h</span>
      </div>
      <div className="flex items-center gap-4 mt-2 text-xs text-gray-500 dark:text-gray-400">
        <span className="flex items-center gap-1">
          <span className="inline-block w-3 h-3 bg-green-400 dark:bg-green-600 rounded" />
          Low load
        </span>
        <span className="flex items-center gap-1">
          <span className="inline-block w-3 h-3 bg-blue-400 dark:bg-blue-600 rounded" />
          Normal
        </span>
      </div>
    </div>
  );
}

interface CronHPASectionProps {
  deployment: string;
  namespace: string;
}

function CronHPASection({ deployment, namespace }: CronHPASectionProps) {
  const [expanded, setExpanded] = useState(false);
  const [cronData, setCronData] = useState<CronHPAResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadCronHPA = useCallback(async () => {
    if (cronData) {
      setExpanded(!expanded);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const result = await fetchCronHPA(deployment, namespace);
      setCronData(result);
      setExpanded(true);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("Failed to load CronHPA template");
      }
    } finally {
      setLoading(false);
    }
  }, [deployment, namespace, cronData, expanded]);

  return (
    <div className="mt-4">
      <button
        type="button"
        onClick={() => void loadCronHPA()}
        className="flex items-center gap-2 text-sm font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 transition-colors"
        disabled={loading}
      >
        <svg
          className={`w-4 h-4 transition-transform ${expanded ? "rotate-90" : ""}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 5l7 7-7 7"
          />
        </svg>
        {loading ? "Loading..." : "CronHPA Template"}
      </button>

      {error && (
        <p className="mt-2 text-sm text-red-600 dark:text-red-400">{error}</p>
      )}

      {expanded && cronData && (
        <div className="mt-3 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
          <div className="flex items-center justify-between px-4 py-2 bg-gray-100 dark:bg-gray-800">
            <span className="text-xs font-medium text-gray-600 dark:text-gray-400">
              Scale: {cronData.config.scale_up_time} -{" "}
              {cronData.config.scale_down_time} | Replicas:{" "}
              {String(cronData.config.min_replicas)}-
              {String(cronData.config.max_replicas)}
            </span>
            <CopyButton text={cronData.yaml} />
          </div>
          <pre className="p-4 text-sm font-mono text-gray-800 dark:text-gray-200 bg-gray-50 dark:bg-gray-900 overflow-x-auto">
            <code>{cronData.yaml}</code>
          </pre>
        </div>
      )}
    </div>
  );
}

export default function SchedulesPage() {
  const [data, setData] = useState<Schedule[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const load = async () => {
      try {
        const result = await fetchSchedules();
        setData(result.schedules);
      } catch (err: unknown) {
        if (err instanceof Error) {
          setError(err.message);
        } else {
          setError("An unexpected error occurred");
        }
      } finally {
        setLoading(false);
      }
    };
    void load();
  }, []);

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          Schedule Analysis
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Analyze workload patterns and generate CronHPA templates for
          auto-scaling
        </p>
      </div>

      {loading && (
        <div className="text-center py-12 text-gray-500" role="status">
          Loading schedules...
        </div>
      )}

      {error && (
        <div
          className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg dark:bg-red-900/20 dark:border-red-800 dark:text-red-400"
          role="alert"
        >
          {error}
        </div>
      )}

      {!loading && !error && (
        <div className="grid gap-6 lg:grid-cols-2">
          {data.length === 0 ? (
            <p className="col-span-full text-center py-12 text-gray-500 dark:text-gray-400">
              No schedule data available
            </p>
          ) : (
            data.map((schedule: Schedule) => (
              <div
                key={`${schedule.namespace}-${schedule.deployment}`}
                className="bg-white rounded-lg border border-gray-200 p-5 shadow-sm dark:bg-gray-900 dark:border-gray-700"
              >
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                    {schedule.deployment}
                  </h3>
                  <span className="text-xs text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">
                    {schedule.namespace}
                  </span>
                </div>

                <div className="flex items-center gap-3 mb-3 text-sm">
                  <span className="text-gray-600 dark:text-gray-400">
                    Low-load hours:{" "}
                    <span className="font-medium text-gray-900 dark:text-gray-100">
                      {schedule.low_load_hours.length > 0
                        ? schedule.low_load_hours
                            .map((h) => `${String(h)}:00`)
                            .join(", ")
                        : "None"}
                    </span>
                  </span>
                </div>

                {schedule.is_weekend_low_load && (
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400 mb-3">
                    Weekend low-load detected
                  </span>
                )}

                <HourlyLoadChart schedule={schedule} />

                <CronHPASection
                  deployment={schedule.deployment}
                  namespace={schedule.namespace}
                />
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
