import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { SchedulesResponse } from "@/types/api";

vi.mock("next/navigation", () => ({
  usePathname: () => "/schedules",
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), prefetch: vi.fn() }),
  useSearchParams: () => new URLSearchParams(),
}));

vi.mock("next/font/google", () => ({
  Geist: () => ({ variable: "mock-geist-sans" }),
  Geist_Mono: () => ({ variable: "mock-geist-mono" }),
}));

const mockSchedulesResponse: SchedulesResponse = {
  schedules: [
    {
      namespace: "default",
      deployment: "web",
      hourly_loads: Array.from({ length: 24 }, (_, i) => ({
        hour: i,
        avg_cpu_usage: i >= 9 && i <= 18 ? 70 : 10,
        avg_memory_usage: i >= 9 && i <= 18 ? 60 : 15,
        sample_count: 10,
      })),
      low_load_hours: [0, 1, 2, 3, 4, 5, 22, 23],
      is_weekend_low_load: true,
    },
    {
      namespace: "production",
      deployment: "api-server",
      hourly_loads: Array.from({ length: 24 }, (_, i) => ({
        hour: i,
        avg_cpu_usage: 50,
        avg_memory_usage: 40,
        sample_count: 5,
      })),
      low_load_hours: [],
      is_weekend_low_load: false,
    },
  ],
};

describe("SchedulesPage", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockSchedulesResponse),
      }),
    );
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("renders the page title", async () => {
    const { default: SchedulesPage } = await import("./page");
    render(<SchedulesPage />);
    expect(screen.getByText("Schedule Analysis")).toBeInTheDocument();
  });

  it("renders schedule cards after loading", async () => {
    const { default: SchedulesPage } = await import("./page");
    render(<SchedulesPage />);

    await waitFor(() => {
      expect(screen.getByText("web")).toBeInTheDocument();
    });

    expect(screen.getByText("api-server")).toBeInTheDocument();
  });

  it("displays namespace labels", async () => {
    const { default: SchedulesPage } = await import("./page");
    render(<SchedulesPage />);

    await waitFor(() => {
      expect(screen.getByText("default")).toBeInTheDocument();
    });

    expect(screen.getByText("production")).toBeInTheDocument();
  });

  it("shows weekend low-load badge when detected", async () => {
    const { default: SchedulesPage } = await import("./page");
    render(<SchedulesPage />);

    await waitFor(() => {
      expect(screen.getByText("Weekend low-load detected")).toBeInTheDocument();
    });
  });

  it("displays loading state initially", async () => {
    vi.stubGlobal("fetch", vi.fn().mockReturnValue(new Promise(() => {})));
    const { default: SchedulesPage } = await import("./page");
    render(<SchedulesPage />);

    expect(screen.getByText("Loading schedules...")).toBeInTheDocument();
  });

  it("displays error on fetch failure", async () => {
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
    const { default: SchedulesPage } = await import("./page");
    render(<SchedulesPage />);

    await waitFor(() => {
      expect(screen.getByText("Internal server error")).toBeInTheDocument();
    });
  });

  it("shows no data message for empty schedules", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ schedules: [] }),
      }),
    );
    const { default: SchedulesPage } = await import("./page");
    render(<SchedulesPage />);

    await waitFor(() => {
      expect(
        screen.getByText("No schedule data available"),
      ).toBeInTheDocument();
    });
  });

  it("renders CronHPA template buttons", async () => {
    const { default: SchedulesPage } = await import("./page");
    render(<SchedulesPage />);

    await waitFor(() => {
      expect(screen.getByText("web")).toBeInTheDocument();
    });

    const buttons = screen.getAllByText("CronHPA Template");
    expect(buttons).toHaveLength(2);
  });
});
