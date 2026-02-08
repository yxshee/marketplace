import { SurfaceCard } from "@/components/ui/surface-card";
import { roleList } from "@marketplace/shared/contracts/api";

export default function BuyerHomePage() {
  return (
    <div className="space-y-8">
      <section className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted">Buyer Surface</p>
        <h1 className="font-display text-4xl font-semibold leading-tight sm:text-5xl">
          Discover products from trusted independent vendors.
        </h1>
        <p className="max-w-2xl text-base text-muted sm:text-lg">
          Clean, fast, and distraction-free marketplace foundations with multi-shipment architecture.
        </p>
      </section>

      <SurfaceCard>
        <h2 className="font-display text-2xl font-semibold">Platform roles wired for RBAC</h2>
        <ul className="mt-4 grid gap-2 text-sm text-muted sm:grid-cols-2">
          {roleList.map((role) => (
            <li className="rounded-sm border border-border px-3 py-2" key={role}>
              {role}
            </li>
          ))}
        </ul>
      </SurfaceCard>
    </div>
  );
}
