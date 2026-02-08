import Link from "next/link";
import { cookies } from "next/headers";
import { deleteCartItemAction, updateCartItemQtyAction } from "@/app/actions/cart";
import { getCart } from "@/lib/api-client";
import { formatUSD } from "@/lib/formatters";

interface CartPageProps {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}

const first = (value: string | string[] | undefined): string | undefined => {
  if (Array.isArray(value)) {
    return value[0];
  }
  return value;
};

export default async function CartPage({ searchParams }: CartPageProps) {
  const params = await searchParams;
  const error = first(params.error);

  const guestToken = (await cookies()).get("mkt_guest_token")?.value;
  const cartResult = await getCart(guestToken);
  const cart = cartResult.payload;

  return (
    <div className="space-y-8">
      <header className="space-y-2">
        <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted">Cart</p>
        <h1 className="font-display text-4xl font-semibold leading-tight">Review your order before checkout</h1>
        <p className="text-sm text-muted">Each vendor will become a separate shipment at checkout.</p>
      </header>

      {error ? <p className="rounded-sm border border-border bg-surface px-3 py-2 text-sm text-muted">{error}</p> : null}

      {cart.items.length === 0 ? (
        <section className="space-y-4 rounded-md border border-border bg-white p-8 text-center">
          <h2 className="font-display text-2xl font-semibold">Your cart is empty</h2>
          <p className="text-sm text-muted">Add products from the catalog to start checkout.</p>
          <div>
            <Link className="text-sm underline-offset-4 hover:underline" href="/search">
              Browse products
            </Link>
          </div>
        </section>
      ) : (
        <section className="space-y-4">
          <div className="space-y-3">
            {cart.items.map((item) => (
              <article className="rounded-md border border-border bg-white p-4" key={item.id}>
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <p className="font-display text-xl font-semibold">{item.title}</p>
                    <p className="text-sm text-muted">{formatUSD(item.unit_price_cents)} each</p>
                  </div>

                  <div className="flex items-center gap-2">
                    <form action={updateCartItemQtyAction} className="flex items-center gap-2">
                      <input name="item_id" type="hidden" value={item.id} />
                      <label className="sr-only" htmlFor={`qty-${item.id}`}>
                        Quantity for {item.title}
                      </label>
                      <input
                        className="w-20 rounded-sm border border-border px-2 py-1 text-sm"
                        defaultValue={item.qty}
                        id={`qty-${item.id}`}
                        max={Math.max(1, item.available_stock)}
                        min={1}
                        name="qty"
                        type="number"
                      />
                      <button
                        className="rounded-sm border border-border px-3 py-1 text-xs font-medium text-muted hover:border-black/20 hover:text-ink"
                        type="submit"
                      >
                        Update
                      </button>
                    </form>

                    <form action={deleteCartItemAction}>
                      <input name="item_id" type="hidden" value={item.id} />
                      <button
                        className="rounded-sm border border-border px-3 py-1 text-xs font-medium text-muted hover:border-black/20 hover:text-ink"
                        type="submit"
                      >
                        Remove
                      </button>
                    </form>
                  </div>
                </div>

                <p className="mt-3 text-sm text-muted">Line total: {formatUSD(item.line_total_cents)}</p>
              </article>
            ))}
          </div>

          <aside className="rounded-md border border-border bg-white p-5">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted">Items ({cart.item_count})</span>
              <span className="font-medium text-ink">{formatUSD(cart.subtotal_cents)}</span>
            </div>
            <div className="mt-3 border-t border-border pt-3 text-sm text-muted">
              Shipping is calculated per vendor shipment at checkout.
            </div>
            <div className="mt-4 flex items-center justify-between">
              <Link className="text-sm underline-offset-4 hover:underline" href="/search">
                Continue shopping
              </Link>
              <Link
                className="rounded-sm border border-ink bg-ink px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-black"
                href="/checkout"
              >
                Continue to checkout
              </Link>
            </div>
          </aside>
        </section>
      )}
    </div>
  );
}
