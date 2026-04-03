import type {
  CronHPAResponse,
  RecommendationsResponse,
  ResourceFilterParams,
  ResourcesResponse,
  SchedulesResponse,
} from "@/types/api";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

class ApiClientError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(message: string, status: number, code: string) {
    super(message);
    this.name = "ApiClientError";
    this.status = status;
    this.code = code;
  }
}

function buildQueryString(params?: ResourceFilterParams): string {
  if (!params) return "";
  const searchParams = new URLSearchParams();
  if (params.namespace) searchParams.set("namespace", params.namespace);
  if (params.deployment) searchParams.set("deployment", params.deployment);
  const qs = searchParams.toString();
  return qs ? `?${qs}` : "";
}

async function fetchJson<T>(url: string): Promise<T> {
  const response = await fetch(url);
  if (!response.ok) {
    let code = "UNKNOWN_ERROR";
    let message = `API request failed with status ${String(response.status)}`;
    try {
      const body: unknown = await response.json();
      if (
        typeof body === "object" &&
        body !== null &&
        "error" in body &&
        "code" in body
      ) {
        const errorBody = body as { error: string; code: string };
        message = errorBody.error;
        code = errorBody.code;
      }
    } catch {
      // response body is not JSON, use default message
    }
    throw new ApiClientError(message, response.status, code);
  }
  return response.json() as Promise<T>;
}

export function fetchResources(
  params?: ResourceFilterParams,
): Promise<ResourcesResponse> {
  return fetchJson<ResourcesResponse>(
    `${API_BASE}/api/v1/resources${buildQueryString(params)}`,
  );
}

export function fetchRecommendations(
  params?: ResourceFilterParams,
): Promise<RecommendationsResponse> {
  return fetchJson<RecommendationsResponse>(
    `${API_BASE}/api/v1/recommendations${buildQueryString(params)}`,
  );
}

export function fetchSchedules(
  params?: ResourceFilterParams,
): Promise<SchedulesResponse> {
  return fetchJson<SchedulesResponse>(
    `${API_BASE}/api/v1/schedules${buildQueryString(params)}`,
  );
}

export function fetchCronHPA(
  deployment: string,
  namespace?: string,
): Promise<CronHPAResponse> {
  const params = new URLSearchParams();
  if (namespace) params.set("namespace", namespace);
  const qs = params.toString();
  const query = qs ? `?${qs}` : "";
  return fetchJson<CronHPAResponse>(
    `${API_BASE}/api/v1/cronhpa/${encodeURIComponent(deployment)}${query}`,
  );
}

export { ApiClientError };
