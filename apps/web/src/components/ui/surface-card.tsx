import type { PropsWithChildren } from "react";

export const SurfaceCard = ({ children }: PropsWithChildren) => {
  return (
    <section className="rounded-md border border-border bg-white p-6 shadow-crisp sm:p-8">{children}</section>
  );
};
