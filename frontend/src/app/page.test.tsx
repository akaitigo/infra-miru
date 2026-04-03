import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ResourcesResponse } from "@/types/api";

vi.mock("next/navigation", () => ({
  usePathname: () => "/",
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), prefetch: vi.fn() }),
  useSearchParams: () => new URLSearchParams(),
}));

vi.mock("next/font/google", () => ({
  Geist: () => ({ variable: "mock-geist-sans" }),
  Geist_Mono: () => ({ variable: "mock-geist-mono" }),
}));

const mockResourcesResponse: ResourcesResponse = {
  pods: [
    {
      namespace: "default",
      pod_name: "my-app-abc123",
      deployment: "my-app",
      cpu_request_millicores: 1000,
      cpu_usage_millicores: 150,
      cpu_divergence_percent: 85.0,
      memory_request_bytes: 1073741824,
      memory_usage_bytes: 214748364,
      memory_divergence_percent: 80.0,
      is_over_provisioned: true,
    },
    {
      namespace: "production",
      pod_name: "api-server-xyz789",
      deployment: "api-server",
      cpu_request_millicores: 500,
      cpu_usage_millicores: 400,
      cpu_divergence_percent: 20.0,
      memory_request_bytes: 536870912,
      memory_usage_bytes: 429496729,
      memory_divergence_percent: 20.0,
      is_over_provisioned: false,
    },
  ],
  deployments: [],
};

describe("DashboardPage", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResourcesResponse),
      }),
    );
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("renders the page title", async () => {
    const { default: DashboardPage } = await import("./page");
    render(<DashboardPage />);
    expect(screen.getByText("Resource Dashboard")).toBeInTheDocument();
  });

  it("renders pod data after loading", async () => {
    const { default: DashboardPage } = await import("./page");
    render(<DashboardPage />);

    await waitFor(() => {
      expect(screen.getByText("my-app-abc123")).toBeInTheDocument();
    });

    expect(screen.getByText("api-server-xyz789")).toBeInTheDocument();
    expect(screen.getByText("default")).toBeInTheDocument();
    expect(screen.getByText("production")).toBeInTheDocument();
  });

  it("shows over-provisioned badge for flagged pods", async () => {
    const { default: DashboardPage } = await import("./page");
    render(<DashboardPage />);

    await waitFor(() => {
      expect(screen.getByText("Over-provisioned")).toBeInTheDocument();
    });

    expect(screen.getByText("OK")).toBeInTheDocument();
  });

  it("displays loading state initially", async () => {
    vi.stubGlobal("fetch", vi.fn().mockReturnValue(new Promise(() => {})));
    const { default: DashboardPage } = await import("./page");
    render(<DashboardPage />);

    expect(screen.getByText("Loading resources...")).toBeInTheDocument();
  });

  it("displays error state on fetch failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: () =>
          Promise.resolve({
            error: "Internal server error",
            code: "INTERNAL_ERROR",
          }),
      }),
    );
    const { default: DashboardPage } = await import("./page");
    render(<DashboardPage />);

    await waitFor(() => {
      expect(screen.getByText("Internal server error")).toBeInTheDocument();
    });
  });

  it("renders filter inputs", async () => {
    const { default: DashboardPage } = await import("./page");
    render(<DashboardPage />);

    expect(screen.getByLabelText("Namespace")).toBeInTheDocument();
    expect(screen.getByLabelText("Deployment")).toBeInTheDocument();
  });

  it("renders progress bars for CPU and Memory", async () => {
    const { default: DashboardPage } = await import("./page");
    render(<DashboardPage />);

    await waitFor(() => {
      expect(screen.getByText("my-app-abc123")).toBeInTheDocument();
    });

    const cpuBars = screen.getAllByRole("progressbar");
    expect(cpuBars.length).toBeGreaterThanOrEqual(2);
  });
});
