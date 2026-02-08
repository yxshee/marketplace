"use server";

import { revalidatePath } from "next/cache";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import {
  loginAuthUser,
  registerAuthUser,
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

export async function adminLogoutAction(): Promise<never> {
  (await cookies()).delete(adminTokenCookieName);
  redirect("/admin?notice=admin-logged-out");
}
