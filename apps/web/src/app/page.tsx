import Link from "next/link";
import { ProductCard } from "@/components/catalog/product-card";
import { getCatalogCategories, getCatalogProducts } from "@/lib/api-client";

export const dynamic = "force-dynamic";

export default async function BuyerHomePage() {
  const [catalog, categories] = await Promise.all([
    getCatalogProducts({ sort: "newest", limit: 6, offset: 0 }),
    getCatalogCategories(),
  ]);

  return (
    <div className="space-y-10">
      <section className="space-y-4">
        <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted">Buyer Surface</p>
        <h1 className="max-w-4xl font-display text-4xl font-semibold leading-tight sm:text-5xl">
          Discover original products from independent vendors.
        </h1>
        <p className="max-w-2xl text-base text-muted sm:text-lg">
          Browse a clean shared catalog with fast filtering, simple checkout preparation, and no visual clutter.
        </p>
        <div className="pt-2">
          <Link
            className="inline-flex items-center rounded-sm border border-ink bg-ink px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-black"
            href="/search"
          >
            Browse catalog
          </Link>
        </div>
      </section>

      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="font-display text-2xl font-semibold">Popular categories</h2>
          <Link className="text-sm text-muted underline-offset-4 hover:underline" href="/search">
            View all
          </Link>
        </div>
        <div className="grid gap-3 sm:grid-cols-3">
          {categories.map((category) => (
            <Link
              className="rounded-sm border border-border px-4 py-3 text-sm transition-colors hover:border-black/20 hover:bg-black/5"
              href={`/categories/${category.slug}`}
              key={category.slug}
            >
              {category.name}
            </Link>
          ))}
        </div>
      </section>

      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="font-display text-2xl font-semibold">Newest listings</h2>
          <Link className="text-sm text-muted underline-offset-4 hover:underline" href="/search?sort=newest">
            See more
          </Link>
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {catalog.items.map((product) => (
            <ProductCard key={product.id} product={product} />
          ))}
        </div>
      </section>
    </div>
  );
}
