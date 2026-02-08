import { SurfaceCard } from "@/components/ui/surface-card";

const filters = ["Category", "Price range", "Rating", "Vendor"];

export default function BuyerSearchPage() {
  return (
    <div className="space-y-6">
      <h1 className="font-display text-3xl font-semibold">Search and discovery</h1>
      <SurfaceCard>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {filters.map((filter) => (
            <button
              className="rounded-sm border border-border px-4 py-3 text-left text-sm text-muted transition-colors hover:border-ink hover:text-ink"
              key={filter}
              type="button"
            >
              {filter}
            </button>
          ))}
        </div>
      </SurfaceCard>
    </div>
  );
}
