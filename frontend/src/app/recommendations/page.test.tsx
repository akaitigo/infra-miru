import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { RecommendationsResponse } from "@/types/api";

vi.mock("next/navigation", () => ({
  usePathname: () => "/recommendations",
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), prefetch: vi.fn() }),
  useSearchParams: () => new URLSearchParams(),
}));

vi.mock("next/font/google", () => ({
  Geist: () => ({ variable: "mock-geist-sans" }),
  Geist_Mono: () => ({ variable: "mock-geist-mono" }),
}));

const mockRecommendationsResponse: RecommendationsResponse = {
  recommendations: [
    {
      namespace: "default",
      deployment: "my-app",
      message:
        "This Deployment uses only 15% of requests - reduce by 42% to save 8000/month",
      current_request_cpu_millicores: 3000,
      current_request_memory_bytes: 3221225472,
      recommended_cpu_millicores: 540,
      recommended_memory_bytes: 773094211,
      monthly_savings_jpy: 8000,
    },
    {
      namespace: "production",
      deployment: "api-server",
      message: "Minor optimization possible",
      current_request_cpu_millicores: 500,
      current_request_memory_bytes: 536870912,
      recommended_cpu_millicores: 300,
      recommended_memory_bytes: 322122547,
      monthly_savings_jpy: 2000,
    },
  ],
};

describe("RecommendationsPage", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockRecommendationsResponse),
      }),
    );
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("renders the page title", async () => {
    const { default: RecommendationsPage } = await import("./page");
    render(<RecommendationsPage />);
    expect(screen.getByText("Cost-Saving Recommendations")).toBeInTheDocument();
  });

  it("renders recommendation cards after loading", async () => {
    const { default: RecommendationsPage } = await import("./page");
    render(<RecommendationsPage />);

    await waitFor(() => {
      expect(screen.getByText("my-app")).toBeInTheDocument();
    });

    expect(screen.getByText("api-server")).toBeInTheDocument();
  });

  it("displays total monthly savings", async () => {
    const { default: RecommendationsPage } = await import("./page");
    render(<RecommendationsPage />);

    await waitFor(() => {
      expect(
        screen.getByText("Total Monthly Savings Potential"),
      ).toBeInTheDocument();
    });
  });

  it("shows recommendation messages", async () => {
    const { default: RecommendationsPage } = await import("./page");
    render(<RecommendationsPage />);

    await waitFor(() => {
      expect(
        screen.getByText(
          "This Deployment uses only 15% of requests - reduce by 42% to save 8000/month",
        ),
      ).toBeInTheDocument();
    });
  });

  it("displays loading state initially", async () => {
    vi.stubGlobal("fetch", vi.fn().mockReturnValue(new Promise(() => {})));
    const { default: RecommendationsPage } = await import("./page");
    render(<RecommendationsPage />);

    expect(screen.getByText("Loading recommendations...")).toBeInTheDocument();
  });

  it("displays error on fetch failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: () =>
          Promise.resolve({
            error: "Server error",
            code: "INTERNAL_ERROR",
          }),
      }),
    );
    const { default: RecommendationsPage } = await import("./page");
    render(<RecommendationsPage />);

    await waitFor(() => {
      expect(screen.getByText("Server error")).toBeInTheDocument();
    });
  });
});
