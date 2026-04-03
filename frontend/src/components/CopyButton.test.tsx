import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import CopyButton from "./CopyButton";

const mockCopyToClipboard = vi
  .fn<(text: string) => Promise<void>>()
  .mockResolvedValue(undefined);

vi.mock("@/lib/clipboard", () => ({
  copyToClipboard: (...args: [string]) => mockCopyToClipboard(...args),
}));

describe("CopyButton", () => {
  afterEach(() => {
    cleanup();
    mockCopyToClipboard.mockClear();
  });

  it("renders with Copy text", () => {
    render(<CopyButton text="test content" />);
    expect(screen.getByText("Copy")).toBeInTheDocument();
  });

  it("copies text to clipboard on click", async () => {
    const user = userEvent.setup();
    render(<CopyButton text="apiVersion: batch/v1" />);

    await user.click(screen.getByRole("button"));

    await waitFor(() => {
      expect(mockCopyToClipboard).toHaveBeenCalledWith("apiVersion: batch/v1");
    });
  });

  it("shows Copied! after click", async () => {
    const user = userEvent.setup();
    render(<CopyButton text="test" />);

    await user.click(screen.getByRole("button"));

    await waitFor(() => {
      expect(screen.getByText("Copied!")).toBeInTheDocument();
    });
  });

  it("has accessible label", () => {
    render(<CopyButton text="test" />);
    expect(screen.getByLabelText("Copy to clipboard")).toBeInTheDocument();
  });
});
