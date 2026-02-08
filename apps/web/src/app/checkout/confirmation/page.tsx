import Link from "next/link";
import { cookies } from "next/headers";
import { getOrderByID } from "@/lib/api-client";
import { formatUSD } from "@/lib/formatters";

interface ConfirmationPageProps {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}

const first = (value: string | string[] | undefined): string | undefined => {
  if (Array.isArray(value)) {
    return value[0];
  }
  return value;
};

export default async function CheckoutConfirmationPage({ searchParams }: ConfirmationPageProps) {
  const params = await searchParams;
  const orderID = first(params.orderId);
  const paymentMethod = first(params.paymentMethod);
  const paymentStatusParam = first(params.paymentStatus);
  const paymentProviderRef = first(params.paymentProviderRef);
  const flowError = first(params.error);

  if (!orderID) {
    return (
      <div className="space-y-4">
        <h1 className="font-display text-3xl font-semibold">Order not found</h1>
        <Link className="text-sm underline-offset-4 hover:underline" href="/cart">
          Back to cart
        </Link>
      </div>
    );
  }

  const guestToken = (await cookies()).get("mkt_guest_token")?.value;

  let order = null as Awaited<ReturnType<typeof getOrderByID>>["payload"]["order"] | null;
  try {
    order = (await getOrderByID(orderID, guestToken)).payload.order;
  } catch {
    order = null;
  }

  if (!order) {
    return (
      <div className="space-y-4">
        <h1 className="font-display text-3xl font-semibold">Order unavailable</h1>
        <p className="text-sm text-muted">The order could not be loaded for this session.</p>
        <Link className="text-sm underline-offset-4 hover:underline" href="/search">
          Continue shopping
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <header className="space-y-2">
        <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted">Order confirmation</p>
        <h1 className="font-display text-4xl font-semibold leading-tight">Order placed successfully</h1>
        <p className="text-sm text-muted">Order ID: {order.id}</p>
        {flowError ? <p className="text-sm text-muted">Payment note: {flowError}</p> : null}
      </header>

      <section className="space-y-3">
        {order.shipments.map((shipment) => (
          <article className="rounded-md border border-border bg-white p-4" key={shipment.id}>
            <div className="flex items-center justify-between">
              <h2 className="font-display text-xl font-semibold">Shipment {shipment.id}</h2>
              <p className="text-sm text-muted">Vendor {shipment.vendor_id}</p>
            </div>
            <div className="mt-2 text-sm text-muted">
              <p>Status: {shipment.status}</p>
              <p>Items: {shipment.item_count}</p>
            </div>
            <div className="mt-3 border-t border-border pt-3 text-sm">
              <div className="flex items-center justify-between">
                <span className="text-muted">Shipment subtotal</span>
                <span>{formatUSD(shipment.subtotal_cents)}</span>
              </div>
              <div className="mt-1 flex items-center justify-between">
                <span className="text-muted">Shipping</span>
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
          <span>{formatUSD(order.subtotal_cents)}</span>
        </div>
        <div className="mt-1 flex items-center justify-between text-sm">
          <span className="text-muted">Shipping</span>
          <span>{formatUSD(order.shipping_cents)}</span>
        </div>
        <div className="mt-3 border-t border-border pt-3">
          <div className="flex items-center justify-between font-display text-2xl font-semibold">
            <span>Total</span>
            <span>{formatUSD(order.total_cents)}</span>
          </div>
          <p className="mt-1 text-xs text-muted">
            Payment method: {paymentMethod ?? "unassigned"}. Status: {paymentStatusParam ?? order.status}.{" "}
            {paymentProviderRef ? `Reference: ${paymentProviderRef}.` : ""}
          </p>
        </div>
      </aside>

      <footer className="flex items-center justify-between">
        <Link className="text-sm underline-offset-4 hover:underline" href="/search">
          Continue shopping
        </Link>
        <Link className="text-sm underline-offset-4 hover:underline" href="/cart">
          View cart
        </Link>
      </footer>
    </div>
  );
}
