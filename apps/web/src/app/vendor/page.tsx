import { cookies } from "next/headers";
import {
  vendorAuthLoginAction,
  vendorAuthRegisterAction,
  vendorCreateCouponAction,
  vendorCreateProductAction,
  vendorDeleteCouponAction,
  vendorDeleteProductAction,
  vendorLogoutAction,
  vendorOnboardingAction,
  vendorRefundDecisionAction,
  vendorSubmitModerationAction,
  vendorToggleCouponAction,
  vendorUpdateShipmentStatusAction,
  vendorUpdateProductPricingAction,
} from "@/app/actions/vendor";
import {
  getVendorAnalyticsCoupons,
  getVendorAnalyticsOverview,
  getVendorAnalyticsTopProducts,
  getVendorCoupons,
  getVendorProducts,
  getVendorRefundRequests,
  getVendorShipments,
} from "@/lib/api-client";
import { formatUSD } from "@/lib/formatters";
import { SurfaceCard } from "@/components/ui/surface-card";

const API_BASE_URL = process.env.MARKETPLACE_API_BASE_URL ?? "http://localhost:8080/api/v1";
const vendorTokenCookieName = "mkt_vendor_access_token";

interface VendorSurfacePageProps {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}

interface VendorStatusPayload {
  id: string;
  slug: string;
  display_name: string;
  verification_state: "pending" | "verified" | "rejected" | "suspended";
  created_at: string;
}

const first = (value: string | string[] | undefined): string | undefined => {
  if (Array.isArray(value)) {
    return value[0];
  }
  return value;
};

const verificationCopy: Record<VendorStatusPayload["verification_state"], string> = {
  pending:
    "Your vendor profile is awaiting admin verification. You can prepare products and coupons while waiting.",
  verified:
    "Your vendor profile is verified. Products can be submitted to moderation and listed after approval.",
  rejected:
    "Verification was rejected. Review your vendor details and contact support for resubmission.",
  suspended: "This vendor profile is currently suspended. Contact platform support for resolution.",
};

async function fetchVendorStatus(accessToken: string): Promise<{
  vendor: VendorStatusPayload | null;
  statusCode: number;
}> {
  const response = await fetch(`${API_BASE_URL}/vendor/verification-status`, {
    method: "GET",
    cache: "no-store",
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });

  if (response.status === 404) {
    return { vendor: null, statusCode: 404 };
  }

  if (!response.ok) {
    return { vendor: null, statusCode: response.status };
  }

  return {
    vendor: (await response.json()) as VendorStatusPayload,
    statusCode: response.status,
  };
}

export default async function VendorSurfacePage({ searchParams }: VendorSurfacePageProps) {
  const params = await searchParams;
  const notice = first(params.notice);
  const error = first(params.error);

  const accessToken = (await cookies()).get(vendorTokenCookieName)?.value;
  const statusResult = accessToken
    ? await fetchVendorStatus(accessToken)
    : { vendor: null, statusCode: 0 };
  const vendor = statusResult.vendor;
  const canSubmitForModeration = vendor?.verification_state === "verified";

  let products = [] as Awaited<ReturnType<typeof getVendorProducts>>["payload"]["items"];
  let coupons = [] as Awaited<ReturnType<typeof getVendorCoupons>>["payload"]["items"];
  let shipments = [] as Awaited<ReturnType<typeof getVendorShipments>>["payload"]["items"];
  let refundRequests = [] as Awaited<
    ReturnType<typeof getVendorRefundRequests>
  >["payload"]["items"];
  let analyticsOverview = {
    currency: "USD",
    revenue_cents: 0,
    order_count: 0,
    paid_order_count: 0,
    shipment_count: 0,
    conversion_funnel: {
      orders_total: 0,
      orders_paid: 0,
      shipments_total: 0,
      shipments_shipped: 0,
      shipments_delivered: 0,
    },
    refund_stats: {
      requests_total: 0,
      pending_total: 0,
      approved_total: 0,
      rejected_total: 0,
      approval_rate_bps: 0,
      order_refund_rate_bps: 0,
    },
  } as Awaited<ReturnType<typeof getVendorAnalyticsOverview>>["payload"];
  let analyticsTopProducts = [] as Awaited<
    ReturnType<typeof getVendorAnalyticsTopProducts>
  >["payload"]["items"];
  let analyticsCoupons = [] as Awaited<
    ReturnType<typeof getVendorAnalyticsCoupons>
  >["payload"]["items"];
  let managementLoadError = false;

  if (accessToken && vendor) {
    try {
      const [
        productResult,
        couponResult,
        shipmentResult,
        refundResult,
        overviewResult,
        topProductsResult,
        analyticsCouponsResult,
      ] = await Promise.all([
        getVendorProducts(accessToken),
        getVendorCoupons(accessToken),
        getVendorShipments(accessToken),
        getVendorRefundRequests(accessToken),
        getVendorAnalyticsOverview(accessToken),
        getVendorAnalyticsTopProducts(accessToken),
        getVendorAnalyticsCoupons(accessToken),
      ]);

      products = productResult.payload.items;
      coupons = couponResult.payload.items;
      shipments = shipmentResult.payload.items;
      refundRequests = refundResult.payload.items;
      analyticsOverview = overviewResult.payload;
      analyticsTopProducts = topProductsResult.payload.items;
      analyticsCoupons = analyticsCouponsResult.payload.items;
    } catch {
      managementLoadError = true;
    }
  }

  return (
    <div className="space-y-6">
      <header className="space-y-2">
        <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted">
          Vendor workspace
        </p>
        <h1 className="font-display text-4xl font-semibold leading-tight">
          Manage products and coupons
        </h1>
        <p className="text-sm text-muted">
          Sign in, create your vendor profile, and maintain catalog + discount operations.
        </p>
      </header>

      {notice ? (
        <p className="rounded-sm border border-border bg-surface px-3 py-2 text-sm text-muted">
          {notice}
        </p>
      ) : null}
      {error ? (
        <p className="rounded-sm border border-border bg-surface px-3 py-2 text-sm text-muted">
          {error}
        </p>
      ) : null}

      {!accessToken ? (
        <div className="grid gap-4 md:grid-cols-2">
          <SurfaceCard>
            <h2 className="font-display text-xl font-semibold">Create vendor account</h2>
            <p className="mt-1 text-sm text-muted">
              Creates a platform user and signs you in for onboarding.
            </p>
            <form action={vendorAuthRegisterAction} className="mt-4 space-y-3">
              <label className="block space-y-1 text-sm">
                <span>Email</span>
                <input
                  className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                  name="email"
                  required
                  type="email"
                />
              </label>
              <label className="block space-y-1 text-sm">
                <span>Password</span>
                <input
                  className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                  minLength={8}
                  name="password"
                  required
                  type="password"
                />
              </label>
              <button
                className="rounded-sm border border-ink bg-ink px-4 py-2 text-sm font-medium text-white"
                type="submit"
              >
                Create account
              </button>
            </form>
          </SurfaceCard>

          <SurfaceCard>
            <h2 className="font-display text-xl font-semibold">Sign in</h2>
            <p className="mt-1 text-sm text-muted">Use your existing vendor owner credentials.</p>
            <form action={vendorAuthLoginAction} className="mt-4 space-y-3">
              <label className="block space-y-1 text-sm">
                <span>Email</span>
                <input
                  className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                  name="email"
                  required
                  type="email"
                />
              </label>
              <label className="block space-y-1 text-sm">
                <span>Password</span>
                <input
                  className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                  minLength={8}
                  name="password"
                  required
                  type="password"
                />
              </label>
              <button
                className="rounded-sm border border-border bg-surface px-4 py-2 text-sm font-medium text-ink"
                type="submit"
              >
                Sign in
              </button>
            </form>
          </SurfaceCard>
        </div>
      ) : null}

      {accessToken && !vendor && statusResult.statusCode === 404 ? (
        <SurfaceCard>
          <h2 className="font-display text-xl font-semibold">Create vendor profile</h2>
          <p className="mt-1 text-sm text-muted">
            One owner per vendor profile. Slug must use lowercase letters, numbers, and hyphens.
          </p>
          <form action={vendorOnboardingAction} className="mt-4 space-y-3">
            <label className="block space-y-1 text-sm">
              <span>Vendor slug</span>
              <input
                className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                name="slug"
                pattern="[a-z0-9-]+"
                required
                type="text"
              />
            </label>
            <label className="block space-y-1 text-sm">
              <span>Display name</span>
              <input
                className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                name="display_name"
                required
                type="text"
              />
            </label>
            <button
              className="rounded-sm border border-ink bg-ink px-4 py-2 text-sm font-medium text-white"
              type="submit"
            >
              Submit onboarding
            </button>
          </form>
          <form action={vendorLogoutAction} className="mt-3">
            <button
              className="rounded-sm border border-border bg-surface px-3 py-2 text-xs font-medium text-muted"
              type="submit"
            >
              Sign out
            </button>
          </form>
        </SurfaceCard>
      ) : null}

      {accessToken && vendor ? (
        <div className="space-y-4">
          <SurfaceCard>
            <h2 className="font-display text-xl font-semibold">{vendor.display_name}</h2>
            <p className="mt-1 text-sm text-muted">Slug: {vendor.slug}</p>
            <p className="mt-3 text-sm">
              Verification status: <span className="font-medium">{vendor.verification_state}</span>
            </p>
            <p className="mt-1 text-sm text-muted">{verificationCopy[vendor.verification_state]}</p>
            <p className="mt-2 text-xs text-muted">Vendor ID: {vendor.id}</p>
            <p className="text-xs text-muted">
              Created: {new Date(vendor.created_at).toLocaleString()}
            </p>
            <form action={vendorLogoutAction} className="mt-4">
              <button
                className="rounded-sm border border-border bg-surface px-3 py-2 text-xs font-medium text-muted"
                type="submit"
              >
                Sign out
              </button>
            </form>
          </SurfaceCard>

          {managementLoadError ? (
            <SurfaceCard>
              <p className="text-sm text-muted">
                Unable to load vendor products/coupons/shipments/refunds right now. Retry in a
                moment.
              </p>
            </SurfaceCard>
          ) : (
            <div className="space-y-4">
              <SurfaceCard>
                <h3 className="font-display text-lg font-semibold">Analytics</h3>
                <p className="mt-1 text-sm text-muted">
                  Revenue, fulfillment funnel, refund rates, top products, and coupon impact.
                </p>

                <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                  <div className="rounded-sm border border-border bg-surface px-3 py-2">
                    <p className="text-xs uppercase tracking-[0.12em] text-muted">Revenue</p>
                    <p className="mt-1 font-display text-xl font-semibold">
                      {formatUSD(analyticsOverview.revenue_cents)}
                    </p>
                  </div>
                  <div className="rounded-sm border border-border bg-surface px-3 py-2">
                    <p className="text-xs uppercase tracking-[0.12em] text-muted">Orders</p>
                    <p className="mt-1 font-display text-xl font-semibold">
                      {analyticsOverview.order_count}
                    </p>
                    <p className="text-xs text-muted">
                      Paid/confirmed {analyticsOverview.paid_order_count}
                    </p>
                  </div>
                  <div className="rounded-sm border border-border bg-surface px-3 py-2">
                    <p className="text-xs uppercase tracking-[0.12em] text-muted">Shipments</p>
                    <p className="mt-1 font-display text-xl font-semibold">
                      {analyticsOverview.shipment_count}
                    </p>
                    <p className="text-xs text-muted">
                      Delivered {analyticsOverview.conversion_funnel.shipments_delivered}
                    </p>
                  </div>
                  <div className="rounded-sm border border-border bg-surface px-3 py-2">
                    <p className="text-xs uppercase tracking-[0.12em] text-muted">Refund rate</p>
                    <p className="mt-1 font-display text-xl font-semibold">
                      {(analyticsOverview.refund_stats.order_refund_rate_bps / 100).toFixed(2)}%
                    </p>
                    <p className="text-xs text-muted">
                      Requests {analyticsOverview.refund_stats.requests_total}
                    </p>
                  </div>
                </div>

                <div className="mt-4 grid gap-4 lg:grid-cols-2">
                  <div className="rounded-sm border border-border p-3">
                    <p className="text-xs uppercase tracking-[0.12em] text-muted">Top products</p>
                    <div className="mt-2 space-y-2">
                      {analyticsTopProducts.length === 0 ? (
                        <p className="text-sm text-muted">No product sales data yet.</p>
                      ) : null}
                      {analyticsTopProducts.slice(0, 5).map((item) => (
                        <div
                          className="flex items-center justify-between gap-3 border-b border-border pb-2 last:border-b-0 last:pb-0"
                          key={item.product_id}
                        >
                          <div>
                            <p className="text-sm font-medium text-ink">{item.title}</p>
                            <p className="text-xs text-muted">
                              Orders {item.order_count} · Units {item.units_sold}
                            </p>
                          </div>
                          <p className="text-sm">{formatUSD(item.revenue_cents)}</p>
                        </div>
                      ))}
                    </div>
                  </div>

                  <div className="rounded-sm border border-border p-3">
                    <p className="text-xs uppercase tracking-[0.12em] text-muted">
                      Coupon performance
                    </p>
                    <div className="mt-2 space-y-2">
                      {analyticsCoupons.length === 0 ? (
                        <p className="text-sm text-muted">No coupons created yet.</p>
                      ) : null}
                      {analyticsCoupons.slice(0, 5).map((couponMetric) => (
                        <div
                          className="flex items-center justify-between gap-3 border-b border-border pb-2 last:border-b-0 last:pb-0"
                          key={couponMetric.coupon_id}
                        >
                          <div>
                            <p className="text-sm font-medium text-ink">{couponMetric.code}</p>
                            <p className="text-xs text-muted">
                              Usage {couponMetric.usage_count} ·{" "}
                              {couponMetric.active ? "Active" : "Inactive"}
                            </p>
                          </div>
                          <p className="text-xs text-muted">
                            Rev {formatUSD(couponMetric.attributed_revenue_cents)}
                          </p>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </SurfaceCard>

              <div className="grid gap-4 lg:grid-cols-2">
                <SurfaceCard>
                  <h3 className="font-display text-lg font-semibold">Products</h3>
                  <p className="mt-1 text-sm text-muted">
                    Create drafts, update pricing and stock, then submit for moderation.
                  </p>
                  <form action={vendorCreateProductAction} className="mt-4 space-y-3">
                    <label className="block space-y-1 text-sm">
                      <span>Title</span>
                      <input
                        className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                        name="title"
                        required
                        type="text"
                      />
                    </label>
                    <label className="block space-y-1 text-sm">
                      <span>Description</span>
                      <textarea
                        className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                        name="description"
                        required
                        rows={3}
                      />
                    </label>
                    <div className="grid gap-3 sm:grid-cols-2">
                      <label className="block space-y-1 text-sm">
                        <span>Category slug</span>
                        <input
                          className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                          defaultValue="general"
                          name="category_slug"
                          required
                          type="text"
                        />
                      </label>
                      <label className="block space-y-1 text-sm">
                        <span>Tags (comma separated)</span>
                        <input
                          className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                          name="tags"
                          type="text"
                        />
                      </label>
                    </div>
                    <div className="grid gap-3 sm:grid-cols-3">
                      <label className="block space-y-1 text-sm">
                        <span>Price (cents)</span>
                        <input
                          className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                          min={1}
                          name="price_incl_tax_cents"
                          required
                          type="number"
                        />
                      </label>
                      <label className="block space-y-1 text-sm">
                        <span>Currency</span>
                        <input
                          className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                          defaultValue="USD"
                          name="currency"
                          required
                          type="text"
                        />
                      </label>
                      <label className="block space-y-1 text-sm">
                        <span>Stock qty</span>
                        <input
                          className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                          defaultValue={0}
                          min={0}
                          name="stock_qty"
                          required
                          type="number"
                        />
                      </label>
                    </div>
                    <button
                      className="rounded-sm border border-ink bg-ink px-4 py-2 text-sm font-medium text-white"
                      type="submit"
                    >
                      Create product draft
                    </button>
                  </form>

                  <div className="mt-5 space-y-3">
                    {products.length === 0 ? (
                      <p className="text-sm text-muted">No products yet.</p>
                    ) : null}
                    {products.map((product) => (
                      <article className="rounded-sm border border-border p-3" key={product.id}>
                        <div className="flex items-center justify-between gap-2">
                          <h4 className="font-medium text-ink">{product.title}</h4>
                          <span className="text-xs uppercase tracking-[0.12em] text-muted">
                            {product.status}
                          </span>
                        </div>
                        <p className="mt-1 text-xs text-muted">{product.category_slug}</p>
                        <p className="mt-2 text-sm text-muted">
                          Price {formatUSD(product.price_incl_tax_cents)} · Stock{" "}
                          {product.stock_qty}
                        </p>
                        <form
                          action={vendorUpdateProductPricingAction}
                          className="mt-3 grid gap-2 sm:grid-cols-[1fr_1fr_auto]"
                        >
                          <input name="product_id" type="hidden" value={product.id} />
                          <label className="space-y-1 text-xs">
                            <span>Price (cents)</span>
                            <input
                              className="w-full rounded-sm border border-border px-2 py-1.5 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                              defaultValue={product.price_incl_tax_cents}
                              min={1}
                              name="price_incl_tax_cents"
                              required
                              type="number"
                            />
                          </label>
                          <label className="space-y-1 text-xs">
                            <span>Stock qty</span>
                            <input
                              className="w-full rounded-sm border border-border px-2 py-1.5 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                              defaultValue={product.stock_qty}
                              min={0}
                              name="stock_qty"
                              required
                              type="number"
                            />
                          </label>
                          <button
                            className="self-end rounded-sm border border-border bg-surface px-3 py-1.5 text-xs font-medium text-ink"
                            type="submit"
                          >
                            Update
                          </button>
                        </form>
                        <div className="mt-2 flex flex-wrap gap-2">
                          {canSubmitForModeration &&
                          (product.status === "draft" || product.status === "rejected") ? (
                            <form action={vendorSubmitModerationAction}>
                              <input name="product_id" type="hidden" value={product.id} />
                              <button
                                className="rounded-sm border border-border bg-surface px-3 py-1.5 text-xs font-medium text-ink"
                                type="submit"
                              >
                                Submit moderation
                              </button>
                            </form>
                          ) : null}
                          <form action={vendorDeleteProductAction}>
                            <input name="product_id" type="hidden" value={product.id} />
                            <button
                              className="rounded-sm border border-border bg-surface px-3 py-1.5 text-xs font-medium text-muted"
                              type="submit"
                            >
                              Delete
                            </button>
                          </form>
                        </div>
                      </article>
                    ))}
                  </div>
                </SurfaceCard>

                <SurfaceCard>
                  <h3 className="font-display text-lg font-semibold">Coupons</h3>
                  <p className="mt-1 text-sm text-muted">
                    Create and manage vendor-scoped discount codes.
                  </p>
                  <form
                    action={vendorCreateCouponAction}
                    className="mt-4 grid gap-3 sm:grid-cols-2"
                  >
                    <label className="block space-y-1 text-sm">
                      <span>Code</span>
                      <input
                        className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                        name="code"
                        required
                        type="text"
                      />
                    </label>
                    <label className="block space-y-1 text-sm">
                      <span>Type</span>
                      <select
                        className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                        defaultValue="percent"
                        name="discount_type"
                      >
                        <option value="percent">Percent</option>
                        <option value="amount_cents">Amount (cents)</option>
                      </select>
                    </label>
                    <label className="block space-y-1 text-sm sm:col-span-2">
                      <span>Value</span>
                      <input
                        className="w-full rounded-sm border border-border px-3 py-2 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                        min={1}
                        name="discount_value"
                        required
                        type="number"
                      />
                    </label>
                    <button
                      className="rounded-sm border border-ink bg-ink px-4 py-2 text-sm font-medium text-white sm:col-span-2"
                      type="submit"
                    >
                      Create coupon
                    </button>
                  </form>

                  <div className="mt-5 space-y-3">
                    {coupons.length === 0 ? (
                      <p className="text-sm text-muted">No coupons yet.</p>
                    ) : null}
                    {coupons.map((coupon) => (
                      <article className="rounded-sm border border-border p-3" key={coupon.id}>
                        <div className="flex items-center justify-between gap-2">
                          <h4 className="font-medium text-ink">{coupon.code}</h4>
                          <span className="text-xs uppercase tracking-[0.12em] text-muted">
                            {coupon.active ? "active" : "inactive"}
                          </span>
                        </div>
                        <p className="mt-1 text-sm text-muted">
                          {coupon.discount_type === "percent"
                            ? `${coupon.discount_value}% off`
                            : `${formatUSD(coupon.discount_value)} off`}
                        </p>
                        <div className="mt-3 flex gap-2">
                          <form action={vendorToggleCouponAction}>
                            <input name="coupon_id" type="hidden" value={coupon.id} />
                            <input
                              name="active"
                              type="hidden"
                              value={coupon.active ? "false" : "true"}
                            />
                            <button
                              className="rounded-sm border border-border bg-surface px-3 py-1.5 text-xs font-medium text-ink"
                              type="submit"
                            >
                              {coupon.active ? "Deactivate" : "Activate"}
                            </button>
                          </form>
                          <form action={vendorDeleteCouponAction}>
                            <input name="coupon_id" type="hidden" value={coupon.id} />
                            <button
                              className="rounded-sm border border-border bg-surface px-3 py-1.5 text-xs font-medium text-muted"
                              type="submit"
                            >
                              Delete
                            </button>
                          </form>
                        </div>
                      </article>
                    ))}
                  </div>
                </SurfaceCard>
              </div>

              <SurfaceCard>
                <h3 className="font-display text-lg font-semibold">Shipments</h3>
                <p className="mt-1 text-sm text-muted">
                  Track shipment timelines and update fulfillment status for your vendor orders.
                </p>

                <div className="mt-4 space-y-3">
                  {shipments.length === 0 ? (
                    <p className="text-sm text-muted">No shipment orders yet.</p>
                  ) : null}
                  {shipments.map((shipment) => (
                    <article className="rounded-sm border border-border p-3" key={shipment.id}>
                      <div className="flex items-center justify-between gap-2">
                        <h4 className="font-medium text-ink">{shipment.id}</h4>
                        <span className="text-xs uppercase tracking-[0.12em] text-muted">
                          {shipment.status}
                        </span>
                      </div>
                      <p className="mt-1 text-xs text-muted">Order {shipment.order_id}</p>
                      <p className="mt-2 text-sm text-muted">
                        Items {shipment.item_count} · Total {formatUSD(shipment.total_cents)}
                      </p>
                      <ul className="mt-2 space-y-1 text-xs text-muted">
                        {shipment.items.map((item) => (
                          <li key={item.id}>
                            {item.title} × {item.qty}
                          </li>
                        ))}
                      </ul>
                      <form
                        action={vendorUpdateShipmentStatusAction}
                        className="mt-3 grid gap-2 sm:grid-cols-[1fr_auto]"
                      >
                        <input name="shipment_id" type="hidden" value={shipment.id} />
                        <label className="space-y-1 text-xs">
                          <span>Status</span>
                          <select
                            className="w-full rounded-sm border border-border px-2 py-1.5 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                            defaultValue={shipment.status}
                            name="status"
                          >
                            <option value="pending">Pending</option>
                            <option value="packed">Packed</option>
                            <option value="shipped">Shipped</option>
                            <option value="delivered">Delivered</option>
                            <option value="cancelled">Cancelled</option>
                          </select>
                        </label>
                        <button
                          className="self-end rounded-sm border border-border bg-surface px-3 py-1.5 text-xs font-medium text-ink"
                          type="submit"
                        >
                          Update status
                        </button>
                      </form>
                      <p className="mt-2 text-xs text-muted">
                        Timeline events: {shipment.timeline.length}
                      </p>
                    </article>
                  ))}
                </div>
              </SurfaceCard>

              <SurfaceCard>
                <h3 className="font-display text-lg font-semibold">Refund requests</h3>
                <p className="mt-1 text-sm text-muted">
                  Review buyer refund requests tied to your shipments and record final decisions.
                </p>

                <div className="mt-4 space-y-3">
                  {refundRequests.length === 0 ? (
                    <p className="text-sm text-muted">No refund requests yet.</p>
                  ) : null}
                  {refundRequests.map((refundRequest) => (
                    <article className="rounded-sm border border-border p-3" key={refundRequest.id}>
                      <div className="flex items-center justify-between gap-2">
                        <h4 className="font-medium text-ink">{refundRequest.id}</h4>
                        <span className="text-xs uppercase tracking-[0.12em] text-muted">
                          {refundRequest.status}
                        </span>
                      </div>
                      <p className="mt-1 text-xs text-muted">
                        Order {refundRequest.order_id} · Shipment {refundRequest.shipment_id}
                      </p>
                      <p className="mt-2 text-sm text-muted">
                        Requested {formatUSD(refundRequest.requested_amount_cents)} · Reason{" "}
                        {refundRequest.reason}
                      </p>
                      {refundRequest.status === "pending" ? (
                        <form
                          action={vendorRefundDecisionAction}
                          className="mt-3 grid gap-2 sm:grid-cols-[1fr_1fr_auto]"
                        >
                          <input name="refund_request_id" type="hidden" value={refundRequest.id} />
                          <label className="space-y-1 text-xs">
                            <span>Decision</span>
                            <select
                              className="w-full rounded-sm border border-border px-2 py-1.5 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                              defaultValue="approve"
                              name="decision"
                            >
                              <option value="approve">Approve</option>
                              <option value="reject">Reject</option>
                            </select>
                          </label>
                          <label className="space-y-1 text-xs">
                            <span>Decision note</span>
                            <input
                              className="w-full rounded-sm border border-border px-2 py-1.5 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                              name="decision_reason"
                              type="text"
                            />
                          </label>
                          <button
                            className="self-end rounded-sm border border-border bg-surface px-3 py-1.5 text-xs font-medium text-ink"
                            type="submit"
                          >
                            Apply decision
                          </button>
                        </form>
                      ) : (
                        <p className="mt-2 text-xs text-muted">
                          Outcome {refundRequest.outcome}
                          {refundRequest.decision_reason
                            ? ` · Note ${refundRequest.decision_reason}`
                            : ""}
                        </p>
                      )}
                    </article>
                  ))}
                </div>
              </SurfaceCard>
            </div>
          )}
        </div>
      ) : null}

      {accessToken &&
      !vendor &&
      statusResult.statusCode !== 0 &&
      statusResult.statusCode !== 404 ? (
        <SurfaceCard>
          <p className="text-sm text-muted">
            Unable to load vendor status right now (HTTP {statusResult.statusCode}). Sign out and
            sign in again.
          </p>
          <form action={vendorLogoutAction} className="mt-3">
            <button
              className="rounded-sm border border-border bg-surface px-3 py-2 text-xs font-medium text-muted"
              type="submit"
            >
              Sign out
            </button>
          </form>
        </SurfaceCard>
      ) : null}
    </div>
  );
}
