"use server";

import { revalidatePath } from "next/cache";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { addCartItem, createStripePaymentIntent, deleteCartItem, placeOrder, updateCartItem } from "@/lib/api-client";

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

export async function placeOrderAction(formData: FormData): Promise<never> {
  const idempotencyKey = String(formData.get("idempotency_key") ?? "").trim();
  if (!idempotencyKey) {
    redirect("/checkout?error=missing-idempotency-key");
  }

  let orderID = "";
  try {
    const token = await readGuestToken();
    const response = await placeOrder({ idempotency_key: idempotencyKey }, token);
    await persistGuestToken(response.guestToken);
    revalidatePath("/cart");
    revalidatePath("/checkout");
    orderID = response.payload.order.id;
  } catch {
    redirect("/checkout?error=place-order-failed");
  }

  redirect(`/checkout/confirmation?orderId=${encodeURIComponent(orderID)}`);
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
    redirect(
      `/checkout/confirmation?orderId=${encodeURIComponent(orderID)}&paymentProviderRef=${encodeURIComponent(intentResponse.payload.provider_ref)}&paymentStatus=${encodeURIComponent(intentResponse.payload.status)}`,
    );
  } catch {
    redirect(`/checkout/confirmation?orderId=${encodeURIComponent(orderID)}&paymentStatus=pending&error=stripe-intent-failed`);
  }
}
