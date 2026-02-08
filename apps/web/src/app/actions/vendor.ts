"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { loginAuthUser, registerAuthUser, registerVendorProfile } from "@/lib/api-client";

const vendorTokenCookieName = "mkt_vendor_access_token";

const readVendorToken = async (): Promise<string | undefined> => {
  return (await cookies()).get(vendorTokenCookieName)?.value;
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

export async function vendorLogoutAction(): Promise<never> {
  (await cookies()).delete(vendorTokenCookieName);
  redirect("/vendor?notice=vendor-logged-out");
}
