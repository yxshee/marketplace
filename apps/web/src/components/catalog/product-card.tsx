import Link from "next/link";
import type { CatalogProduct } from "@marketplace/shared/contracts/api";
import { formatUSD } from "@/lib/formatters";

interface ProductCardProps {
  product: CatalogProduct;
}

export function ProductCard({ product }: ProductCardProps) {
  return (
    <article className="group rounded-md border border-border bg-white p-4 transition-colors hover:border-black/25">
      <div className="space-y-2">
        <p className="text-xs uppercase tracking-[0.14em] text-muted">{product.category_slug}</p>
        <h3 className="font-display text-xl font-semibold leading-tight">
          <Link href={`/products/${product.id}`} className="outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70">
            {product.title}
          </Link>
        </h3>
        <p className="text-sm text-muted">{product.description}</p>
      </div>

      <div className="mt-5 flex items-center justify-between border-t border-border pt-4">
        <p className="font-display text-lg font-semibold text-ink">{formatUSD(product.price_incl_tax_cents)}</p>
        <p className="text-sm text-muted">{product.rating_average.toFixed(1)} rating</p>
      </div>
    </article>
  );
}
