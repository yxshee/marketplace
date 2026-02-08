import Link from "next/link";
import type { CatalogSearchParams } from "@marketplace/shared/contracts/api";
import { ProductCard } from "@/components/catalog/product-card";
import { getCatalogCategories, getCatalogProducts } from "@/lib/api-client";

export const dynamic = "force-dynamic";

interface SearchPageProps {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}

const first = (value: string | string[] | undefined): string | undefined => {
  if (Array.isArray(value)) {
    return value[0];
  }
  return value;
};

const parseNumber = (value: string | undefined): number | undefined => {
  if (!value) {
    return undefined;
  }
  const parsed = Number(value);
  if (Number.isNaN(parsed)) {
    return undefined;
  }
  return parsed;
};

export default async function SearchPage({ searchParams }: SearchPageProps) {
  const params = await searchParams;
  const queryParams: CatalogSearchParams = {
    q: first(params.q),
    category: first(params.category),
    sort: first(params.sort) as CatalogSearchParams["sort"],
    min_rating: parseNumber(first(params.min_rating)),
    price_min: parseNumber(first(params.price_min)),
    price_max: parseNumber(first(params.price_max)),
    limit: 18,
    offset: parseNumber(first(params.offset)),
  };

  const [catalog, categories] = await Promise.all([getCatalogProducts(queryParams), getCatalogCategories()]);

  return (
    <div className="space-y-8">
      <header className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted">Search and discovery</p>
        <h1 className="font-display text-4xl font-semibold leading-tight">Find products with clear filters</h1>
      </header>

      <section className="rounded-md border border-border bg-white p-5">
        <form action="/search" className="grid gap-3 md:grid-cols-4">
          <input
            className="rounded-sm border border-border px-3 py-2 text-sm outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
            defaultValue={queryParams.q}
            name="q"
            placeholder="Search products"
            type="search"
          />

          <select
            className="rounded-sm border border-border px-3 py-2 text-sm outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
            defaultValue={queryParams.category ?? ""}
            name="category"
          >
            <option value="">All categories</option>
            {categories.map((category) => (
              <option key={category.slug} value={category.slug}>
                {category.name}
              </option>
            ))}
          </select>

          <select
            className="rounded-sm border border-border px-3 py-2 text-sm outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
            defaultValue={queryParams.sort ?? "relevance"}
            name="sort"
          >
            <option value="relevance">Relevance</option>
            <option value="newest">Newest</option>
            <option value="price_low_high">Price low-high</option>
            <option value="price_high_low">Price high-low</option>
            <option value="rating">Rating</option>
          </select>

          <button
            className="rounded-sm border border-ink bg-ink px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-black"
            type="submit"
          >
            Apply filters
          </button>
        </form>
      </section>

      {catalog.items.length === 0 ? (
        <section className="rounded-md border border-border bg-white p-10 text-center">
          <h2 className="font-display text-2xl font-semibold">No products found</h2>
          <p className="mt-2 text-sm text-muted">Try a different query or clear category filters.</p>
          <Link className="mt-4 inline-block text-sm underline-offset-4 hover:underline" href="/search">
            Reset search
          </Link>
        </section>
      ) : (
        <section className="space-y-4">
          <p className="text-sm text-muted">
            Showing {catalog.items.length} of {catalog.total} products
          </p>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {catalog.items.map((product) => (
              <ProductCard key={product.id} product={product} />
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
