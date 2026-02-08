"use server";

import { revalidatePath } from "next/cache";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import {
  createAdminPromotion,
  deleteAdminPromotion,
  loginAuthUser,
  registerAuthUser,
  updateAdminPromotion,
  updateAdminOrderStatus,
  updateAdminModerationProduct,
  updateAdminVendorVerification,
} from "@/lib/api-client";

const adminTokenCookieName = "mkt_admin_access_token";

const readAdminToken = async (): Promise<string | undefined> => {
  return (await cookies()).get(adminTokenCookieName)?.value;
};

const requireAdminToken = async (): Promise<string> => {
  const token = await readAdminToken();
  if (!token) {
    redirect("/admin?error=admin-auth-required");
  }
  return token;
};

const persistAdminToken = async (token: string): Promise<void> => {
  (await cookies()).set(adminTokenCookieName, token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 60 * 60 * 24 * 30,
  });
};

const normalizeDateTimeField = (value: FormDataEntryValue | null): string | undefined => {
  const raw = String(value ?? "").trim();
  if (!raw) {
    return undefined;
  }
  const parsed = new Date(raw);
  if (Number.isNaN(parsed.getTime())) {
    return undefined;
  }
  return parsed.toISOString();
};

const parseRuleJSONField = (value: FormDataEntryValue | null): Record<string, unknown> | null => {
  const raw = String(value ?? "").trim();
  if (!raw) {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return null;
    }
    return parsed as Record<string, unknown>;
  } catch {
    return null;
  }
};

export async function adminAuthRegisterAction(formData: FormData): Promise<never> {
  const email = String(formData.get("email") ?? "").trim();
  const password = String(formData.get("password") ?? "");

  try {
    const response = await registerAuthUser({ email, password });
    await persistAdminToken(response.payload.access_token);
  } catch {
    redirect("/admin?error=admin-register-failed");
  }

  redirect("/admin?notice=admin-account-created");
}

export async function adminAuthLoginAction(formData: FormData): Promise<never> {
  const email = String(formData.get("email") ?? "").trim();
  const password = String(formData.get("password") ?? "");

  try {
    const response = await loginAuthUser({ email, password });
    await persistAdminToken(response.payload.access_token);
  } catch {
    redirect("/admin?error=admin-login-failed");
  }

  redirect("/admin?notice=admin-login-success");
}

export async function adminVendorVerificationAction(formData: FormData): Promise<never> {
  const accessToken = await requireAdminToken();
  const vendorID = String(formData.get("vendor_id") ?? "").trim();
  const state = String(formData.get("state") ?? "").trim() as
    | "pending"
    | "verified"
    | "rejected"
    | "suspended";
  const reason = String(formData.get("reason") ?? "").trim();

  if (!vendorID) {
    redirect("/admin?error=admin-vendor-missing");
  }

  try {
    await updateAdminVendorVerification(
      vendorID,
      { state, reason: reason || undefined },
      accessToken,
    );
    revalidatePath("/admin");
  } catch {
    redirect("/admin?error=admin-vendor-verification-failed");
  }

  redirect("/admin?notice=admin-vendor-updated");
}

export async function adminModerationDecisionAction(formData: FormData): Promise<never> {
  const accessToken = await requireAdminToken();
  const productID = String(formData.get("product_id") ?? "").trim();
  const decision = String(formData.get("decision") ?? "").trim() as "approve" | "reject";
  const reason = String(formData.get("reason") ?? "").trim();

  if (!productID) {
    redirect("/admin?error=admin-product-missing");
  }

  try {
    await updateAdminModerationProduct(
      productID,
      {
        decision,
        reason: reason || undefined,
      },
      accessToken,
    );
    revalidatePath("/admin");
  } catch {
    redirect("/admin?error=admin-moderation-decision-failed");
  }

  redirect("/admin?notice=admin-moderation-updated");
}

export async function adminOrderStatusAction(formData: FormData): Promise<never> {
  const accessToken = await requireAdminToken();
  const orderID = String(formData.get("order_id") ?? "").trim();
  const status = String(formData.get("status") ?? "").trim() as
    | "pending_payment"
    | "payment_failed"
    | "cod_confirmed"
    | "paid";

  if (!orderID) {
    redirect("/admin?error=admin-order-missing");
  }

  try {
    await updateAdminOrderStatus(orderID, { status }, accessToken);
    revalidatePath("/admin");
  } catch {
    redirect("/admin?error=admin-order-status-update-failed");
  }

  redirect("/admin?notice=admin-order-updated");
}

export async function adminPromotionCreateAction(formData: FormData): Promise<never> {
  const accessToken = await requireAdminToken();
  const name = String(formData.get("name") ?? "").trim();
  const ruleJSON = parseRuleJSONField(formData.get("rule_json"));
  const startsAt = normalizeDateTimeField(formData.get("starts_at"));
  const endsAt = normalizeDateTimeField(formData.get("ends_at"));
  const stackable = String(formData.get("stackable") ?? "").trim() === "on";
  const active = String(formData.get("active") ?? "").trim() === "on";

  if (!name || !ruleJSON) {
    redirect("/admin?error=admin-promotion-invalid");
  }

  try {
    await createAdminPromotion(
      {
        name,
        rule_json: ruleJSON,
        starts_at: startsAt,
        ends_at: endsAt,
        stackable,
        active,
      },
      accessToken,
    );
    revalidatePath("/admin");
  } catch {
    redirect("/admin?error=admin-promotion-create-failed");
  }

  redirect("/admin?notice=admin-promotion-created");
}

export async function adminPromotionUpdateAction(formData: FormData): Promise<never> {
  const accessToken = await requireAdminToken();
  const promotionID = String(formData.get("promotion_id") ?? "").trim();
  const name = String(formData.get("name") ?? "").trim();
  const ruleJSON = parseRuleJSONField(formData.get("rule_json"));
  const startsAt = normalizeDateTimeField(formData.get("starts_at"));
  const endsAt = normalizeDateTimeField(formData.get("ends_at"));
  const stackable = String(formData.get("stackable") ?? "").trim() === "on";
  const active = String(formData.get("active") ?? "").trim() === "on";

  if (!promotionID || !name || !ruleJSON) {
    redirect("/admin?error=admin-promotion-invalid");
  }

  try {
    await updateAdminPromotion(
      promotionID,
      {
        name,
        rule_json: ruleJSON,
        starts_at: startsAt,
        ends_at: endsAt,
        stackable,
        active,
      },
      accessToken,
    );
    revalidatePath("/admin");
  } catch {
    redirect("/admin?error=admin-promotion-update-failed");
  }

  redirect("/admin?notice=admin-promotion-updated");
}

export async function adminPromotionDeleteAction(formData: FormData): Promise<never> {
  const accessToken = await requireAdminToken();
  const promotionID = String(formData.get("promotion_id") ?? "").trim();

  if (!promotionID) {
    redirect("/admin?error=admin-promotion-invalid");
  }

  try {
    await deleteAdminPromotion(promotionID, accessToken);
    revalidatePath("/admin");
  } catch {
    redirect("/admin?error=admin-promotion-delete-failed");
  }

  redirect("/admin?notice=admin-promotion-deleted");
}

export async function adminLogoutAction(): Promise<never> {
  (await cookies()).delete(adminTokenCookieName);
  redirect("/admin?notice=admin-logged-out");
}
