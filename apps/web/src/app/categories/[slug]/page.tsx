import Link from "next/link";
import { notFound } from "next/navigation";
import { ProductCard } from "@/components/catalog/product-card";
import { getCatalogCategories, getCatalogProducts } from "@/lib/api-client";

export const dynamic = "force-dynamic";

interface CategoryPageProps {
  params: Promise<{ slug: string }>;
}

export default async function CategoryPage({ params }: CategoryPageProps) {
  const { slug } = await params;
  const [categories, catalog] = await Promise.all([
    getCatalogCategories(),
    getCatalogProducts({ category: slug, sort: "newest", limit: 24 }),
  ]);

  const category = categories.find((item) => item.slug === slug);
  if (!category) {
    notFound();
  }

  return (
    <div className="space-y-8">
      <header className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted">Category</p>
        <h1 className="font-display text-4xl font-semibold leading-tight">{category.name}</h1>
        <p className="text-sm text-muted">{catalog.total} products available</p>
      </header>

      <section className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {catalog.items.map((product) => (
          <ProductCard key={product.id} product={product} />
        ))}
      </section>

      <footer>
        <Link className="text-sm text-muted underline-offset-4 hover:underline" href="/search">
          Back to search
        </Link>
      </footer>
    </div>
  );
}
