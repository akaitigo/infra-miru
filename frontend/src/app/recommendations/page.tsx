"use client";

import { useEffect, useState } from "react";
import { fetchRecommendations } from "@/lib/api";
import type { Recommendation, RecommendationsResponse } from "@/types/api";

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${String(bytes)} B`;
  if (bytes < 1024 * 1024) return `${String(Math.round(bytes / 1024))} KiB`;
  if (bytes < 1024 * 1024 * 1024)
    return `${String(Math.round(bytes / (1024 * 1024)))} MiB`;
  return `${String(Math.round(bytes / (1024 * 1024 * 1024)))} GiB`;
}

function formatMillicores(m: number): string {
  if (m >= 1000) return `${String(Math.round(m / 100) / 10)}c`;
  return `${String(m)}m`;
}

function formatJPY(amount: number): string {
  return new Intl.NumberFormat("ja-JP", {
    style: "currency",
    currency: "JPY",
    maximumFractionDigits: 0,
  }).format(amount);
}

export default function RecommendationsPage() {
  const [data, setData] = useState<RecommendationsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const load = async () => {
      try {
        const result = await fetchRecommendations();
        setData(result);
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

  const totalSavings =
    data?.recommendations.reduce(
      (sum, rec) => sum + rec.monthly_savings_jpy,
      0,
    ) ?? 0;

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          Cost-Saving Recommendations
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Optimize resource requests to reduce infrastructure costs
        </p>
      </div>

      {data && !loading && data.recommendations.length > 0 && (
        <div className="mb-6 bg-blue-50 border border-blue-200 rounded-lg p-4 dark:bg-blue-900/20 dark:border-blue-800">
          <div className="flex items-center gap-3">
            <svg
              className="w-8 h-8 text-blue-500"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
            <div>
              <p className="text-sm font-medium text-blue-800 dark:text-blue-300">
                Total Monthly Savings Potential
              </p>
              <p className="text-2xl font-bold text-blue-900 dark:text-blue-100">
                {formatJPY(totalSavings)}
              </p>
            </div>
          </div>
        </div>
      )}

      {loading && (
        <div className="text-center py-12 text-gray-500" role="status">
          Loading recommendations...
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

      {data && !loading && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {data.recommendations.length === 0 ? (
            <p className="col-span-full text-center py-12 text-gray-500 dark:text-gray-400">
              No recommendations available
            </p>
          ) : (
            data.recommendations.map((rec: Recommendation) => (
              <div
                key={`${rec.namespace}-${rec.deployment}`}
                className="bg-white rounded-lg border border-gray-200 p-5 shadow-sm dark:bg-gray-900 dark:border-gray-700"
              >
                <div className="flex items-center justify-between mb-3">
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                    {rec.deployment}
                  </h3>
                  <span className="text-xs text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">
                    {rec.namespace}
                  </span>
                </div>

                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                  {rec.message}
                </p>

                <div className="space-y-3">
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <div>
                      <p className="text-gray-500 dark:text-gray-400">
                        Current CPU
                      </p>
                      <p className="font-medium text-gray-900 dark:text-gray-100">
                        {formatMillicores(rec.current_request_cpu_millicores)}
                      </p>
                    </div>
                    <div>
                      <p className="text-gray-500 dark:text-gray-400">
                        Recommended CPU
                      </p>
                      <p className="font-medium text-green-600 dark:text-green-400">
                        {formatMillicores(rec.recommended_cpu_millicores)}
                      </p>
                    </div>
                    <div>
                      <p className="text-gray-500 dark:text-gray-400">
                        Current Memory
                      </p>
                      <p className="font-medium text-gray-900 dark:text-gray-100">
                        {formatBytes(rec.current_request_memory_bytes)}
                      </p>
                    </div>
                    <div>
                      <p className="text-gray-500 dark:text-gray-400">
                        Recommended Memory
                      </p>
                      <p className="font-medium text-green-600 dark:text-green-400">
                        {formatBytes(rec.recommended_memory_bytes)}
                      </p>
                    </div>
                  </div>

                  <div className="pt-3 border-t border-gray-100 dark:border-gray-800">
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-gray-500 dark:text-gray-400">
                        Monthly Savings
                      </span>
                      <span className="text-lg font-bold text-green-600 dark:text-green-400">
                        {formatJPY(rec.monthly_savings_jpy)}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
