"use server";

import { revalidatePath } from "next/cache";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import {
  createVendorCoupon,
  createVendorProduct,
  deleteVendorCoupon,
  deleteVendorProduct,
  loginAuthUser,
  registerAuthUser,
  registerVendorProfile,
  submitVendorProductModeration,
  updateVendorCoupon,
  updateVendorProduct,
} from "@/lib/api-client";

const vendorTokenCookieName = "mkt_vendor_access_token";

const readVendorToken = async (): Promise<string | undefined> => {
  return (await cookies()).get(vendorTokenCookieName)?.value;
};

const requireVendorToken = async (): Promise<string> => {
  const token = await readVendorToken();
  if (!token) {
    redirect("/vendor?error=vendor-auth-required");
  }
  return token;
};

const persistVendorToken = async (token: string): Promise<void> => {
  (await cookies()).set(vendorTokenCookieName, token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 60 * 60 * 24 * 30,
  });
};

export async function vendorAuthRegisterAction(formData: FormData): Promise<never> {
  const email = String(formData.get("email") ?? "").trim();
  const password = String(formData.get("password") ?? "");

  try {
    const response = await registerAuthUser({ email, password });
    await persistVendorToken(response.payload.access_token);
  } catch {
    redirect("/vendor?error=vendor-register-failed");
  }

  redirect("/vendor?notice=vendor-account-created");
}

export async function vendorAuthLoginAction(formData: FormData): Promise<never> {
  const email = String(formData.get("email") ?? "").trim();
  const password = String(formData.get("password") ?? "");

  try {
    const response = await loginAuthUser({ email, password });
    await persistVendorToken(response.payload.access_token);
  } catch {
    redirect("/vendor?error=vendor-login-failed");
  }

  redirect("/vendor?notice=vendor-login-success");
}

export async function vendorOnboardingAction(formData: FormData): Promise<never> {
  const slug = String(formData.get("slug") ?? "").trim();
  const displayName = String(formData.get("display_name") ?? "").trim();

  const accessToken = await readVendorToken();
  if (!accessToken) {
    redirect("/vendor?error=vendor-auth-required");
  }

  try {
    await registerVendorProfile(
      {
        slug,
        display_name: displayName,
      },
      accessToken,
    );
  } catch {
    redirect("/vendor?error=vendor-onboarding-failed");
  }

  redirect("/vendor?notice=vendor-onboarding-submitted");
}

export async function vendorCreateProductAction(formData: FormData): Promise<never> {
  const accessToken = await requireVendorToken();
  const title = String(formData.get("title") ?? "").trim();
  const description = String(formData.get("description") ?? "").trim();
  const categorySlug = String(formData.get("category_slug") ?? "general").trim();
  const tagsRaw = String(formData.get("tags") ?? "").trim();
  const currency = String(formData.get("currency") ?? "USD").trim().toUpperCase();
  const priceCents = Number.parseInt(String(formData.get("price_incl_tax_cents") ?? "0"), 10);
  const stockQty = Number.parseInt(String(formData.get("stock_qty") ?? "0"), 10);

  const tags = tagsRaw.length === 0 ? [] : tagsRaw.split(",").map((value) => value.trim());

  try {
    await createVendorProduct(
      {
        title,
        description,
        category_slug: categorySlug,
        tags,
        price_incl_tax_cents: Number.isNaN(priceCents) ? 0 : priceCents,
        currency,
        stock_qty: Number.isNaN(stockQty) ? 0 : stockQty,
      },
      accessToken,
    );
    revalidatePath("/vendor");
  } catch {
    redirect("/vendor?error=vendor-product-create-failed");
  }

  redirect("/vendor?notice=vendor-product-created");
}

export async function vendorUpdateProductPricingAction(formData: FormData): Promise<never> {
  const accessToken = await requireVendorToken();
  const productID = String(formData.get("product_id") ?? "").trim();
  const priceCents = Number.parseInt(String(formData.get("price_incl_tax_cents") ?? "0"), 10);
  const stockQty = Number.parseInt(String(formData.get("stock_qty") ?? "0"), 10);

  if (!productID) {
    redirect("/vendor?error=vendor-product-missing");
  }

  try {
    await updateVendorProduct(
      productID,
      {
        price_incl_tax_cents: Number.isNaN(priceCents) ? 0 : priceCents,
        stock_qty: Number.isNaN(stockQty) ? 0 : stockQty,
      },
      accessToken,
    );
    revalidatePath("/vendor");
  } catch {
    redirect("/vendor?error=vendor-product-update-failed");
  }

  redirect("/vendor?notice=vendor-product-updated");
}

export async function vendorDeleteProductAction(formData: FormData): Promise<never> {
  const accessToken = await requireVendorToken();
  const productID = String(formData.get("product_id") ?? "").trim();

  if (!productID) {
    redirect("/vendor?error=vendor-product-missing");
  }

  try {
    await deleteVendorProduct(productID, accessToken);
    revalidatePath("/vendor");
  } catch {
    redirect("/vendor?error=vendor-product-delete-failed");
  }

  redirect("/vendor?notice=vendor-product-deleted");
}

export async function vendorSubmitModerationAction(formData: FormData): Promise<never> {
  const accessToken = await requireVendorToken();
  const productID = String(formData.get("product_id") ?? "").trim();

  if (!productID) {
    redirect("/vendor?error=vendor-product-missing");
  }

  try {
    await submitVendorProductModeration(productID, accessToken);
    revalidatePath("/vendor");
  } catch {
    redirect("/vendor?error=vendor-moderation-submit-failed");
  }

  redirect("/vendor?notice=vendor-moderation-submitted");
}

export async function vendorCreateCouponAction(formData: FormData): Promise<never> {
  const accessToken = await requireVendorToken();
  const code = String(formData.get("code") ?? "").trim();
  const discountType = String(formData.get("discount_type") ?? "").trim() as "percent" | "amount_cents";
  const discountValue = Number.parseInt(String(formData.get("discount_value") ?? "0"), 10);

  try {
    await createVendorCoupon(
      {
        code,
        discount_type: discountType,
        discount_value: Number.isNaN(discountValue) ? 0 : discountValue,
        active: true,
      },
      accessToken,
    );
    revalidatePath("/vendor");
  } catch {
    redirect("/vendor?error=vendor-coupon-create-failed");
  }

  redirect("/vendor?notice=vendor-coupon-created");
}

export async function vendorToggleCouponAction(formData: FormData): Promise<never> {
  const accessToken = await requireVendorToken();
  const couponID = String(formData.get("coupon_id") ?? "").trim();
  const active = String(formData.get("active") ?? "").trim() === "true";

  if (!couponID) {
    redirect("/vendor?error=vendor-coupon-missing");
  }

  try {
    await updateVendorCoupon(couponID, { active }, accessToken);
    revalidatePath("/vendor");
  } catch {
    redirect("/vendor?error=vendor-coupon-update-failed");
  }

  redirect("/vendor?notice=vendor-coupon-updated");
}

export async function vendorDeleteCouponAction(formData: FormData): Promise<never> {
  const accessToken = await requireVendorToken();
  const couponID = String(formData.get("coupon_id") ?? "").trim();

  if (!couponID) {
    redirect("/vendor?error=vendor-coupon-missing");
  }

  try {
    await deleteVendorCoupon(couponID, accessToken);
    revalidatePath("/vendor");
  } catch {
    redirect("/vendor?error=vendor-coupon-delete-failed");
  }

  redirect("/vendor?notice=vendor-coupon-deleted");
}

export async function vendorLogoutAction(): Promise<never> {
  (await cookies()).delete(vendorTokenCookieName);
  redirect("/vendor?notice=vendor-logged-out");
}
