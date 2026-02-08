"use server";

import { revalidatePath } from "next/cache";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import {
  addCartItem,
  confirmCODPayment,
  createStripePaymentIntent,
  deleteCartItem,
  placeOrder,
  updateCartItem,
} from "@/lib/api-client";

const guestCookieName = "mkt_guest_token";

const readGuestToken = async (): Promise<string | undefined> => {
  return (await cookies()).get(guestCookieName)?.value;
};

const persistGuestToken = async (guestToken?: string): Promise<void> => {
  if (!guestToken) {
    return;
  }

  (await cookies()).set(guestCookieName, guestToken, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 60 * 60 * 24 * 30,
  });
};

const parsePositiveInt = (raw: FormDataEntryValue | null, fallback: number): number => {
  if (typeof raw !== "string") {
    return fallback;
  }
  const parsed = Number.parseInt(raw, 10);
  if (Number.isNaN(parsed) || parsed <= 0) {
    return fallback;
  }
  return parsed;
};

export async function addCartItemAction(formData: FormData): Promise<never> {
  const productID = String(formData.get("product_id") ?? "").trim();
  const qty = parsePositiveInt(formData.get("qty"), 1);
  if (!productID) {
    redirect("/search?error=missing-product");
  }

  try {
    const token = await readGuestToken();
    const response = await addCartItem({ product_id: productID, qty }, token);
    await persistGuestToken(response.guestToken);
    revalidatePath("/cart");
    revalidatePath("/checkout");
    revalidatePath(`/products/${productID}`);
  } catch {
    redirect(`/products/${productID}?error=cart-add-failed`);
  }

  redirect("/cart");
}

export async function updateCartItemQtyAction(formData: FormData): Promise<never> {
  const itemID = String(formData.get("item_id") ?? "").trim();
  const qty = parsePositiveInt(formData.get("qty"), 1);
  if (!itemID) {
    redirect("/cart?error=missing-item");
  }

  try {
    const token = await readGuestToken();
    const response = await updateCartItem(itemID, { qty }, token);
    await persistGuestToken(response.guestToken);
    revalidatePath("/cart");
    revalidatePath("/checkout");
  } catch {
    redirect("/cart?error=cart-update-failed");
  }

  redirect("/cart");
}

export async function deleteCartItemAction(formData: FormData): Promise<never> {
  const itemID = String(formData.get("item_id") ?? "").trim();
  if (!itemID) {
    redirect("/cart?error=missing-item");
  }

  try {
    const token = await readGuestToken();
    const response = await deleteCartItem(itemID, token);
    await persistGuestToken(response.guestToken);
    revalidatePath("/cart");
    revalidatePath("/checkout");
  } catch {
    redirect("/cart?error=cart-delete-failed");
  }

  redirect("/cart");
}

export async function placeOrderWithCODAction(formData: FormData): Promise<never> {
  const idempotencyKey = String(formData.get("idempotency_key") ?? "").trim();
  if (!idempotencyKey) {
    redirect("/checkout?error=missing-idempotency-key");
  }

  let orderID = "";
  let guestToken = await readGuestToken();
  try {
    const placeOrderResponse = await placeOrder({ idempotency_key: idempotencyKey }, guestToken);
    guestToken = placeOrderResponse.guestToken ?? guestToken;
    await persistGuestToken(guestToken);
    orderID = placeOrderResponse.payload.order.id;
  } catch {
    redirect("/checkout?error=place-order-failed");
  }

  let redirectURL = `/checkout/confirmation?orderId=${encodeURIComponent(orderID)}&paymentMethod=cod&paymentStatus=pending_collection&error=cod-confirmation-failed`;
  try {
    const codResponse = await confirmCODPayment(
      {
        order_id: orderID,
        idempotency_key: `cod_${idempotencyKey}`,
      },
      guestToken,
    );
    await persistGuestToken(codResponse.guestToken ?? guestToken);
    revalidatePath("/cart");
    revalidatePath("/checkout");
    redirectURL = `/checkout/confirmation?orderId=${encodeURIComponent(orderID)}&paymentMethod=cod&paymentProviderRef=${encodeURIComponent(codResponse.payload.provider_ref)}&paymentStatus=${encodeURIComponent(codResponse.payload.status)}`;
  } catch {
    // Keep fallback redirect URL when COD confirmation fails.
  }

  redirect(redirectURL);
}

export async function placeOrderWithStripeAction(formData: FormData): Promise<never> {
  const idempotencyKey = String(formData.get("idempotency_key") ?? "").trim();
  if (!idempotencyKey) {
    redirect("/checkout?error=missing-idempotency-key");
  }

  let orderID = "";
  let guestToken = await readGuestToken();
  try {
    const placeOrderResponse = await placeOrder({ idempotency_key: idempotencyKey }, guestToken);
    guestToken = placeOrderResponse.guestToken ?? guestToken;
    await persistGuestToken(guestToken);
    orderID = placeOrderResponse.payload.order.id;
  } catch {
    redirect("/checkout?error=place-order-failed");
  }

  let redirectURL = `/checkout/confirmation?orderId=${encodeURIComponent(orderID)}&paymentMethod=stripe&paymentStatus=pending&error=stripe-intent-failed`;
  try {
    const intentResponse = await createStripePaymentIntent(
      {
        order_id: orderID,
        idempotency_key: `pi_${idempotencyKey}`,
      },
      guestToken,
    );
    await persistGuestToken(intentResponse.guestToken ?? guestToken);
    revalidatePath("/cart");
    revalidatePath("/checkout");
    redirectURL = `/checkout/confirmation?orderId=${encodeURIComponent(orderID)}&paymentMethod=stripe&paymentProviderRef=${encodeURIComponent(intentResponse.payload.provider_ref)}&paymentStatus=${encodeURIComponent(intentResponse.payload.status)}`;
  } catch {
    // Keep fallback redirect URL when Stripe intent creation fails.
  }

  redirect(redirectURL);
}
