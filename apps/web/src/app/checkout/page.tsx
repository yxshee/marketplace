import Link from "next/link";
import { cookies } from "next/headers";
import { placeOrderAction, placeOrderWithStripeAction } from "@/app/actions/cart";
import { getCheckoutQuote } from "@/lib/api-client";
import { formatUSD } from "@/lib/formatters";

interface CheckoutPageProps {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}

const first = (value: string | string[] | undefined): string | undefined => {
  if (Array.isArray(value)) {
    return value[0];
  }
  return value;
};

export default async function CheckoutPage({ searchParams }: CheckoutPageProps) {
  const params = await searchParams;
  const error = first(params.error);

  const guestToken = (await cookies()).get("mkt_guest_token")?.value;

  let quote = null as Awaited<ReturnType<typeof getCheckoutQuote>>["payload"] | null;
  try {
    quote = (await getCheckoutQuote(guestToken)).payload;
  } catch {
    quote = null;
  }

  if (!quote || quote.shipments.length === 0) {
    return (
      <div className="space-y-6">
        <header className="space-y-2">
          <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted">Checkout</p>
          <h1 className="font-display text-4xl font-semibold leading-tight">Checkout is ready when your cart has items</h1>
          <p className="text-sm text-muted">Add items from one or more vendors, then return here.</p>
        </header>
        <Link className="text-sm underline-offset-4 hover:underline" href="/cart">
          Back to cart
        </Link>
      </div>
    );
  }

  const idempotencyKey = `chk_${Date.now()}_${Math.random().toString(36).slice(2, 10)}`;

  return (
    <div className="space-y-8">
      <header className="space-y-2">
        <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted">Checkout</p>
        <h1 className="font-display text-4xl font-semibold leading-tight">Confirm multi-vendor order</h1>
        <p className="text-sm text-muted">One order, {quote.shipment_count} vendor shipments, independent fulfillment timelines.</p>
      </header>

      {error ? <p className="rounded-sm border border-border bg-surface px-3 py-2 text-sm text-muted">{error}</p> : null}

      <section className="space-y-3">
        {quote.shipments.map((shipment) => (
          <article className="rounded-md border border-border bg-white p-4" key={shipment.vendor_id}>
            <div className="flex items-center justify-between">
              <h2 className="font-display text-xl font-semibold">Vendor shipment {shipment.vendor_id}</h2>
              <p className="text-sm text-muted">{shipment.item_count} items</p>
            </div>

            <ul className="mt-3 space-y-2 text-sm">
              {shipment.items.map((item) => (
                <li className="flex items-center justify-between" key={item.id}>
                  <span>
                    {item.title} × {item.qty}
                  </span>
                  <span>{formatUSD(item.line_total_cents)}</span>
                </li>
              ))}
            </ul>

            <div className="mt-3 border-t border-border pt-3 text-sm text-muted">
              <div className="flex items-center justify-between">
                <span>Shipment subtotal</span>
                <span>{formatUSD(shipment.subtotal_cents)}</span>
              </div>
              <div className="mt-1 flex items-center justify-between">
                <span>Shipping</span>
                <span>{formatUSD(shipment.shipping_fee_cents)}</span>
              </div>
              <div className="mt-1 flex items-center justify-between font-medium text-ink">
                <span>Shipment total</span>
                <span>{formatUSD(shipment.total_cents)}</span>
              </div>
            </div>
          </article>
        ))}
      </section>

      <aside className="rounded-md border border-border bg-white p-5">
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted">Subtotal</span>
          <span>{formatUSD(quote.subtotal_cents)}</span>
        </div>
        <div className="mt-1 flex items-center justify-between text-sm">
          <span className="text-muted">Shipping</span>
          <span>{formatUSD(quote.shipping_cents)}</span>
        </div>
        <div className="mt-3 border-t border-border pt-3">
          <div className="flex items-center justify-between font-display text-2xl font-semibold">
            <span>Total</span>
            <span>{formatUSD(quote.total_cents)}</span>
          </div>
          <p className="mt-1 text-xs text-muted">Taxes are already included in displayed prices.</p>
        </div>

        <form action={placeOrderWithStripeAction} className="mt-4">
          <input name="idempotency_key" type="hidden" value={idempotencyKey} />
          <button
            className="w-full rounded-sm border border-ink bg-ink px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-black"
            type="submit"
          >
            Pay online with Stripe
          </button>
        </form>
        <form action={placeOrderAction} className="mt-2">
          <input name="idempotency_key" type="hidden" value={`${idempotencyKey}_cod`} />
          <button
            className="w-full rounded-sm border border-border bg-surface px-4 py-2 text-sm font-medium text-ink transition-colors hover:bg-white"
            type="submit"
          >
            Place order with cash on delivery
          </button>
        </form>
        <p className="mt-2 text-xs text-muted">
          Stripe flow creates a payment intent and awaits secure confirmation; COD keeps payment pending.
        </p>

        <div className="mt-3 text-right">
          <Link className="text-sm underline-offset-4 hover:underline" href="/cart">
            Back to cart
          </Link>
        </div>
      </aside>
    </div>
  );
}
