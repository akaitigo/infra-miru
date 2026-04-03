import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  ApiClientError,
  fetchCronHPA,
  fetchRecommendations,
  fetchResources,
  fetchSchedules,
} from "./api";

describe("API Client", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ pods: [], deployments: [] }),
      }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("fetchResources", () => {
    it("calls the resources endpoint", async () => {
      await fetchResources();

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/api/v1/resources",
      );
    });

    it("passes namespace and deployment as query params", async () => {
      await fetchResources({ namespace: "prod", deployment: "api" });

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/api/v1/resources?namespace=prod&deployment=api",
      );
    });

    it("returns parsed JSON response", async () => {
      const mockData = {
        pods: [{ pod_name: "test-pod" }],
        deployments: [],
      };
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(mockData),
        }),
      );

      const result = await fetchResources();
      expect(result).toEqual(mockData);
    });
  });

  describe("fetchRecommendations", () => {
    it("calls the recommendations endpoint", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve({ recommendations: [] }),
        }),
      );

      await fetchRecommendations();

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/api/v1/recommendations",
      );
    });

    it("passes filter params", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve({ recommendations: [] }),
        }),
      );

      await fetchRecommendations({ namespace: "staging" });

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/api/v1/recommendations?namespace=staging",
      );
    });
  });

  describe("fetchSchedules", () => {
    it("calls the schedules endpoint", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve({ schedules: [] }),
        }),
      );

      await fetchSchedules();

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/api/v1/schedules",
      );
    });
  });

  describe("fetchCronHPA", () => {
    it("calls the cronhpa endpoint with deployment and namespace", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve({ yaml: "", config: {} }),
        }),
      );

      await fetchCronHPA("my-app", "default");

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/api/v1/cronhpa/my-app?namespace=default",
      );
    });

    it("encodes deployment name in URL", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve({ yaml: "", config: {} }),
        }),
      );

      await fetchCronHPA("my app", "default");

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/api/v1/cronhpa/my%20app?namespace=default",
      );
    });

    it("works without namespace", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve({ yaml: "", config: {} }),
        }),
      );

      await fetchCronHPA("my-app");

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/api/v1/cronhpa/my-app",
      );
    });
  });

  describe("error handling", () => {
    it("throws ApiClientError on non-ok response with JSON body", async () => {
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

      try {
        await fetchResources();
        expect.fail("should have thrown");
      } catch (err: unknown) {
        expect(err).toBeInstanceOf(ApiClientError);
        if (err instanceof ApiClientError) {
          expect(err.message).toBe("Internal server error");
          expect(err.status).toBe(500);
          expect(err.code).toBe("INTERNAL_ERROR");
        }
      }
    });

    it("throws ApiClientError with default message for non-JSON error body", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: false,
          status: 502,
          json: () => Promise.reject(new Error("not json")),
        }),
      );

      try {
        await fetchResources();
        expect.fail("should have thrown");
      } catch (err: unknown) {
        expect(err).toBeInstanceOf(ApiClientError);
        if (err instanceof ApiClientError) {
          expect(err.status).toBe(502);
          expect(err.code).toBe("UNKNOWN_ERROR");
          expect(err.message).toContain("502");
        }
      }
    });

    it("throws ApiClientError for 400 Bad Request", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: false,
          status: 400,
          json: () =>
            Promise.resolve({
              error: "namespace is required",
              code: "MISSING_NAMESPACE",
            }),
        }),
      );

      try {
        await fetchCronHPA("my-app");
        expect.fail("should have thrown");
      } catch (err: unknown) {
        expect(err).toBeInstanceOf(ApiClientError);
        if (err instanceof ApiClientError) {
          expect(err.status).toBe(400);
          expect(err.code).toBe("MISSING_NAMESPACE");
        }
      }
    });

    it("propagates network errors from fetch", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockRejectedValue(new TypeError("Failed to fetch")),
      );

      try {
        await fetchResources();
        expect.fail("should have thrown");
      } catch (err: unknown) {
        expect(err).toBeInstanceOf(TypeError);
        if (err instanceof TypeError) {
          expect(err.message).toBe("Failed to fetch");
        }
      }
    });
  });

  describe("buildQueryString edge cases", () => {
    it("omits empty namespace from query string", async () => {
      await fetchResources({ namespace: "" });

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/api/v1/resources",
      );
    });

    it("omits undefined deployment from query string", async () => {
      await fetchResources({ namespace: "prod" });

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/api/v1/resources?namespace=prod",
      );
    });

    it("omits both when empty", async () => {
      await fetchResources({});

      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/api/v1/resources",
      );
    });
  });
});
