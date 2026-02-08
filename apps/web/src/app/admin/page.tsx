import { cookies } from "next/headers";
import {
  adminAuthLoginAction,
  adminModerationDecisionAction,
  adminAuthRegisterAction,
  adminLogoutAction,
  adminVendorVerificationAction,
} from "@/app/actions/admin";
import { getAdminModerationProducts, getAdminVendors } from "@/lib/api-client";
import { SurfaceCard } from "@/components/ui/surface-card";

const adminTokenCookieName = "mkt_admin_access_token";

interface AdminSurfacePageProps {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}

const first = (value: string | string[] | undefined): string | undefined => {
  if (Array.isArray(value)) {
    return value[0];
  }
  return value;
};

export default async function AdminSurfacePage({ searchParams }: AdminSurfacePageProps) {
  const params = await searchParams;
  const notice = first(params.notice);
  const error = first(params.error);
  const accessToken = (await cookies()).get(adminTokenCookieName)?.value;

  let pendingVendors = [] as Awaited<ReturnType<typeof getAdminVendors>>["payload"]["items"];
  let allVendors = [] as Awaited<ReturnType<typeof getAdminVendors>>["payload"]["items"];
  let moderationProducts = [] as Awaited<
    ReturnType<typeof getAdminModerationProducts>
  >["payload"]["items"];
  let loadError = false;

  if (accessToken) {
    try {
      const [pendingResult, allResult, moderationResult] = await Promise.all([
        getAdminVendors(accessToken, "pending"),
        getAdminVendors(accessToken),
        getAdminModerationProducts(accessToken, "pending_approval"),
      ]);
      pendingVendors = pendingResult.payload.items;
      allVendors = allResult.payload.items;
      moderationProducts = moderationResult.payload.items;
    } catch {
      loadError = true;
    }
  }

  return (
    <div className="space-y-6">
      <header className="space-y-2">
        <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted">Admin</p>
        <h1 className="font-display text-4xl font-semibold leading-tight">
          Vendor verification queue
        </h1>
        <p className="text-sm text-muted">
          Review vendor onboarding states and apply verification decisions.
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
            <h2 className="font-display text-xl font-semibold">Create admin account</h2>
            <p className="mt-1 text-sm text-muted">
              Use a bootstrap email configured for admin roles.
            </p>
            <form action={adminAuthRegisterAction} className="mt-4 space-y-3">
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
            <p className="mt-1 text-sm text-muted">Use an existing admin credential.</p>
            <form action={adminAuthLoginAction} className="mt-4 space-y-3">
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

      {accessToken ? (
        <div className="space-y-4">
          <SurfaceCard>
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h2 className="font-display text-xl font-semibold">Verification queue</h2>
                <p className="mt-1 text-sm text-muted">
                  Pending vendors {pendingVendors.length} · Total vendors {allVendors.length}
                </p>
              </div>
              <form action={adminLogoutAction}>
                <button
                  className="rounded-sm border border-border bg-surface px-3 py-2 text-xs font-medium text-muted"
                  type="submit"
                >
                  Sign out
                </button>
              </form>
            </div>
          </SurfaceCard>

          {loadError ? (
            <SurfaceCard>
              <p className="text-sm text-muted">
                Unable to load vendor verification queue right now. Retry in a moment.
              </p>
            </SurfaceCard>
          ) : (
            <>
              <SurfaceCard>
                <div className="space-y-3">
                  {allVendors.length === 0 ? (
                    <p className="text-sm text-muted">No vendors registered yet.</p>
                  ) : null}
                  {allVendors.map((vendor) => (
                    <article className="rounded-sm border border-border p-3" key={vendor.id}>
                      <div className="flex flex-wrap items-start justify-between gap-3">
                        <div>
                          <h3 className="font-medium text-ink">{vendor.display_name}</h3>
                          <p className="mt-1 text-xs text-muted">
                            {vendor.slug} · {vendor.id}
                          </p>
                          <p className="mt-1 text-sm text-muted">
                            State:{" "}
                            <span className="font-medium text-ink">
                              {vendor.verification_state}
                            </span>
                          </p>
                        </div>
                      </div>

                      <form
                        action={adminVendorVerificationAction}
                        className="mt-3 grid gap-2 sm:grid-cols-[1fr_2fr_auto]"
                      >
                        <input name="vendor_id" type="hidden" value={vendor.id} />
                        <label className="space-y-1 text-xs">
                          <span>Verification state</span>
                          <select
                            className="w-full rounded-sm border border-border px-2 py-1.5 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                            defaultValue={vendor.verification_state}
                            name="state"
                          >
                            <option value="pending">Pending</option>
                            <option value="verified">Verified</option>
                            <option value="rejected">Rejected</option>
                            <option value="suspended">Suspended</option>
                          </select>
                        </label>
                        <label className="space-y-1 text-xs">
                          <span>Reason</span>
                          <input
                            className="w-full rounded-sm border border-border px-2 py-1.5 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                            name="reason"
                            type="text"
                          />
                        </label>
                        <button
                          className="self-end rounded-sm border border-border bg-surface px-3 py-1.5 text-xs font-medium text-ink"
                          type="submit"
                        >
                          Apply
                        </button>
                      </form>
                    </article>
                  ))}
                </div>
              </SurfaceCard>
              <SurfaceCard>
                <h2 className="font-display text-xl font-semibold">Product moderation queue</h2>
                <p className="mt-1 text-sm text-muted">
                  Pending products {moderationProducts.length}. Approve or reject each submission.
                </p>

                <div className="mt-4 space-y-3">
                  {moderationProducts.length === 0 ? (
                    <p className="text-sm text-muted">No products waiting for moderation.</p>
                  ) : null}
                  {moderationProducts.map((product) => (
                    <article className="rounded-sm border border-border p-3" key={product.id}>
                      <div className="flex flex-wrap items-start justify-between gap-3">
                        <div>
                          <h3 className="font-medium text-ink">{product.title}</h3>
                          <p className="mt-1 text-xs text-muted">
                            {product.id} · Vendor {product.vendor_id}
                          </p>
                          <p className="mt-1 text-sm text-muted">
                            Status: <span className="font-medium text-ink">{product.status}</span>
                          </p>
                        </div>
                      </div>

                      <form
                        action={adminModerationDecisionAction}
                        className="mt-3 grid gap-2 sm:grid-cols-[1fr_2fr_auto]"
                      >
                        <input name="product_id" type="hidden" value={product.id} />
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
                          <span>Reason (required for reject)</span>
                          <input
                            className="w-full rounded-sm border border-border px-2 py-1.5 outline-none ring-offset-2 focus-visible:ring-2 focus-visible:ring-black/70"
                            name="reason"
                            type="text"
                          />
                        </label>
                        <button
                          className="self-end rounded-sm border border-border bg-surface px-3 py-1.5 text-xs font-medium text-ink"
                          type="submit"
                        >
                          Apply
                        </button>
                      </form>
                    </article>
                  ))}
                </div>
              </SurfaceCard>
            </>
          )}
        </div>
      ) : null}
    </div>
  );
}
