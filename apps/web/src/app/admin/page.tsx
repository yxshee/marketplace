import { SurfaceCard } from "@/components/ui/surface-card";

export default function AdminSurfacePage() {
  return (
    <div className="space-y-6">
      <h1 className="font-display text-3xl font-semibold">Admin surface</h1>
      <SurfaceCard>
        <p className="text-sm text-muted">
          Verification, moderation, order operations, promotions, analytics, and audit trails are API-first and role-scoped.
        </p>
      </SurfaceCard>
    </div>
  );
}
