"use client";

import { useCallback, useEffect, useState } from "react";
import ProgressBar from "@/components/ProgressBar";
import { fetchResources } from "@/lib/api";
import type { Pod, ResourcesResponse } from "@/types/api";

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

export default function DashboardPage() {
  const [data, setData] = useState<ResourcesResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [namespaceFilter, setNamespaceFilter] = useState("");
  const [deploymentFilter, setDeploymentFilter] = useState("");

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params: Record<string, string> = {};
      if (namespaceFilter) params.namespace = namespaceFilter;
      if (deploymentFilter) params.deployment = deploymentFilter;
      const result = await fetchResources(
        Object.keys(params).length > 0 ? params : undefined,
      );
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
  }, [namespaceFilter, deploymentFilter]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          Resource Dashboard
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Monitor Kubernetes resource usage and identify over-provisioned
          workloads
        </p>
      </div>

      <div className="mb-6 flex gap-4">
        <div>
          <label
            htmlFor="namespace-filter"
            className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
          >
            Namespace
          </label>
          <input
            id="namespace-filter"
            type="text"
            placeholder="Filter by namespace"
            value={namespaceFilter}
            onChange={(e) => setNamespaceFilter(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-800 dark:border-gray-600 dark:text-gray-100"
          />
        </div>
        <div>
          <label
            htmlFor="deployment-filter"
            className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
          >
            Deployment
          </label>
          <input
            id="deployment-filter"
            type="text"
            placeholder="Filter by deployment"
            value={deploymentFilter}
            onChange={(e) => setDeploymentFilter(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-800 dark:border-gray-600 dark:text-gray-100"
          />
        </div>
      </div>

      {loading && (
        <div className="text-center py-12 text-gray-500" role="status">
          Loading resources...
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
        <div className="overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-100 dark:bg-gray-800">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider dark:text-gray-400">
                  Namespace
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider dark:text-gray-400">
                  Pod Name
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider dark:text-gray-400">
                  Deployment
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider dark:text-gray-400">
                  CPU (Request / Usage)
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider dark:text-gray-400 min-w-[160px]">
                  CPU Usage
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider dark:text-gray-400">
                  Memory (Request / Usage)
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider dark:text-gray-400 min-w-[160px]">
                  Memory Usage
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider dark:text-gray-400">
                  Status
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200 dark:bg-gray-900 dark:divide-gray-700">
              {data.pods.length === 0 ? (
                <tr>
                  <td
                    colSpan={8}
                    className="px-4 py-8 text-center text-gray-500 dark:text-gray-400"
                  >
                    No pods found
                  </td>
                </tr>
              ) : (
                data.pods.map((pod: Pod) => (
                  <tr
                    key={`${pod.namespace}-${pod.pod_name}`}
                    className="hover:bg-gray-50 dark:hover:bg-gray-800/50"
                  >
                    <td className="px-4 py-3 text-sm text-gray-900 dark:text-gray-100">
                      {pod.namespace}
                    </td>
                    <td className="px-4 py-3 text-sm font-mono text-gray-900 dark:text-gray-100">
                      {pod.pod_name}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-900 dark:text-gray-100">
                      {pod.deployment}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                      {formatMillicores(pod.cpu_request_millicores)} /{" "}
                      {formatMillicores(pod.cpu_usage_millicores)}
                    </td>
                    <td className="px-4 py-3">
                      <ProgressBar
                        value={pod.cpu_usage_millicores}
                        max={pod.cpu_request_millicores}
                        label={`CPU usage for ${pod.pod_name}`}
                      />
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                      {formatBytes(pod.memory_request_bytes)} /{" "}
                      {formatBytes(pod.memory_usage_bytes)}
                    </td>
                    <td className="px-4 py-3">
                      <ProgressBar
                        value={pod.memory_usage_bytes}
                        max={pod.memory_request_bytes}
                        label={`Memory usage for ${pod.pod_name}`}
                      />
                    </td>
                    <td className="px-4 py-3">
                      {pod.is_over_provisioned ? (
                        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400">
                          Over-provisioned
                        </span>
                      ) : (
                        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">
                          OK
                        </span>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
