import React from "react";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { SurfaceCard } from "./surface-card";

describe("SurfaceCard", () => {
  it("renders children", () => {
    render(
      <SurfaceCard>
        <span>Card body</span>
      </SurfaceCard>,
    );

    expect(screen.getByText("Card body")).toBeInTheDocument();
  });
});
