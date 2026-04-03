export interface Pod {
  namespace: string;
  pod_name: string;
  deployment: string;
  cpu_request_millicores: number;
  cpu_usage_millicores: number;
  cpu_divergence_percent: number;
  memory_request_bytes: number;
  memory_usage_bytes: number;
  memory_divergence_percent: number;
  is_over_provisioned: boolean;
}

export interface DeploymentSummary {
  namespace: string;
  deployment: string;
  pod_count: number;
  total_cpu_request_millicores: number;
  total_cpu_usage_millicores: number;
  avg_cpu_divergence_percent: number;
  total_memory_request_bytes: number;
  total_memory_usage_bytes: number;
  avg_memory_divergence_percent: number;
  is_over_provisioned: boolean;
}

export interface ResourcesResponse {
  pods: Pod[];
  deployments: DeploymentSummary[];
}

export interface Recommendation {
  namespace: string;
  deployment: string;
  message: string;
  current_request_cpu_millicores: number;
  current_request_memory_bytes: number;
  recommended_cpu_millicores: number;
  recommended_memory_bytes: number;
  monthly_savings_jpy: number;
}

export interface RecommendationsResponse {
  recommendations: Recommendation[];
}

export interface HourlyLoad {
  hour: number;
  avg_cpu_usage: number;
  avg_memory_usage: number;
  sample_count: number;
}

export interface Schedule {
  namespace: string;
  deployment: string;
  hourly_loads: HourlyLoad[];
  low_load_hours: number[];
  is_weekend_low_load: boolean;
}

export interface SchedulesResponse {
  schedules: Schedule[];
}

export interface CronHPAConfig {
  namespace: string;
  deployment: string;
  scale_down_time: string;
  scale_up_time: string;
  min_replicas: number;
  max_replicas: number;
}

export interface CronHPAResponse {
  yaml: string;
  config: CronHPAConfig;
}

export interface ResourceFilterParams {
  namespace?: string;
  deployment?: string;
}

export interface ApiError {
  error: string;
  code: string;
}
