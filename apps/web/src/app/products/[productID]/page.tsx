import Link from "next/link";
import { notFound } from "next/navigation";
import { addCartItemAction } from "@/app/actions/cart";
import { getCatalogProductById } from "@/lib/api-client";
import { formatUSD } from "@/lib/formatters";

export const dynamic = "force-dynamic";

interface ProductDetailPageProps {
  params: Promise<{ productID: string }>;
}

export default async function ProductDetailPage({ params }: ProductDetailPageProps) {
  const { productID } = await params;
  const productDetail = await getCatalogProductById(productID);

  if (!productDetail) {
    notFound();
  }

  const { item, vendor } = productDetail;

  return (
    <div className="space-y-8">
      <nav className="text-sm text-muted">
        <Link className="underline-offset-4 hover:underline" href="/search">
          Search
        </Link>
        <span className="px-2">/</span>
        <Link className="underline-offset-4 hover:underline" href={`/categories/${item.category_slug}`}>
          {item.category_slug}
        </Link>
      </nav>

      <section className="grid gap-8 lg:grid-cols-[1.4fr_1fr]">
        <article className="rounded-md border border-border bg-white p-6">
          <p className="text-xs uppercase tracking-[0.16em] text-muted">{item.category_slug}</p>
          <h1 className="mt-3 font-display text-4xl font-semibold leading-tight">{item.title}</h1>
          <p className="mt-4 text-base text-muted">{item.description}</p>

          <div className="mt-8 border-t border-border pt-5">
            <p className="font-display text-3xl font-semibold">{formatUSD(item.price_incl_tax_cents)}</p>
            <p className="mt-1 text-sm text-muted">Tax included • Ships globally</p>
          </div>

          <div className="mt-6 flex flex-wrap gap-2">
            {item.tags.map((tag) => (
              <span className="rounded-sm border border-border px-2 py-1 text-xs text-muted" key={tag}>
                {tag}
              </span>
            ))}
          </div>

          <form action={addCartItemAction} className="mt-6 flex items-center gap-3 border-t border-border pt-5">
            <input name="product_id" type="hidden" value={item.id} />
            <label className="sr-only" htmlFor="qty">
              Quantity
            </label>
            <input
              className="w-20 rounded-sm border border-border px-2 py-2 text-sm"
              defaultValue={1}
              id="qty"
              max={Math.max(1, item.stock_qty)}
              min={1}
              name="qty"
              type="number"
            />
            <button
              className="rounded-sm border border-ink bg-ink px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-black"
              type="submit"
            >
              Add to cart
            </button>
          </form>
        </article>

        <aside className="space-y-4 rounded-md border border-border bg-white p-5">
          <h2 className="font-display text-xl font-semibold">Sold by {vendor.displayName}</h2>
          <p className="text-sm text-muted">Rating {item.rating_average.toFixed(1)} • Stock {item.stock_qty}</p>
          <p className="rounded-sm border border-border bg-surface p-3 text-xs text-muted">
            This listing is visible only after moderation approval.
          </p>
        </aside>
      </section>
    </div>
  );
}
