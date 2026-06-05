import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { ChartShell } from "./ChartShell";

const rechartsMocks = vi.hoisted(() => ({
  responsiveContainer: vi.fn(({ children }: { children?: ReactNode }) => <div data-testid="responsive-container">{children}</div>),
}));

vi.mock("recharts", () => ({
  ResponsiveContainer: rechartsMocks.responsiveContainer,
}));

describe("ChartShell", () => {
  it("renders the shared finance chart surface and accessible summary", () => {
    render(
      <ChartShell title="Cashflow" meta="Real data" summary="Cashflow chart summary">
        <div data-testid="chart-child" />
      </ChartShell>,
    );

    expect(screen.getByText("Cashflow")).toBeInTheDocument();
    expect(screen.getByText("Real data")).toBeInTheDocument();
    expect(screen.getByText("Cashflow chart summary")).toHaveClass("sr-only");
    expect(screen.getByTestId("chart-child")).toBeInTheDocument();
    expect(screen.getByTestId("responsive-container").closest(".chart-shell")).toHaveClass("chart", "chart-regular", "chart-shell");
    expect(rechartsMocks.responsiveContainer).toHaveBeenCalledWith(expect.objectContaining({ debounce: 80 }), undefined);
  });
});
