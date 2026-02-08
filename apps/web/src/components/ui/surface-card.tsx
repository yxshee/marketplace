import type { PropsWithChildren } from "react";

export const SurfaceCard = ({ children }: PropsWithChildren) => {
  return <section className="surface-card p-6 sm:p-8">{children}</section>;
};
