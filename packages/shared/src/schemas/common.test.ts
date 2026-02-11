import { describe, expect, it } from "vitest";

import {
  cartItemMutationSchema,
  catalogSearchSchema,
  checkoutPlaceOrderSchema,
  codConfirmPaymentSchema,
  paymentSettingsUpdateSchema,
  stripeCreateIntentSchema,
  vendorProductUpdateSchema,
} from "./common";

describe("shared request schemas", () => {
  it("validates checkout place-order payload requires idempotency key", () => {
    expect(checkoutPlaceOrderSchema.safeParse({}).success).toBe(false);
    expect(checkoutPlaceOrderSchema.safeParse({ idempotency_key: "idem_12345678" }).success).toBe(
      true,
    );
  });

  it("validates payment mutation payloads", () => {
    expect(
      stripeCreateIntentSchema.safeParse({
        order_id: "ord_123",
        idempotency_key: "pi_12345678",
      }).success,
    ).toBe(true);

    expect(
      codConfirmPaymentSchema.safeParse({
        order_id: "ord_123",
        idempotency_key: "cod_12345678",
      }).success,
    ).toBe(true);

    expect(
      stripeCreateIntentSchema.safeParse({
        order_id: "ord_123",
        idempotency_key: "x",
      }).success,
    ).toBe(false);
  });

  it("enforces positive cart quantity", () => {
    expect(cartItemMutationSchema.safeParse({ product_id: "prd_1", qty: 1 }).success).toBe(true);
    expect(cartItemMutationSchema.safeParse({ product_id: "prd_1", qty: 0 }).success).toBe(false);
  });

  it("coerces and validates catalog query bounds", () => {
    const parsed = catalogSearchSchema.parse({
      limit: "20",
      offset: "0",
      price_min: "100",
      min_rating: "4.5",
    });

    expect(parsed.limit).toBe(20);
    expect(parsed.offset).toBe(0);
    expect(parsed.price_min).toBe(100);
    expect(parsed.min_rating).toBe(4.5);

    expect(catalogSearchSchema.safeParse({ limit: 0 }).success).toBe(false);
    expect(catalogSearchSchema.safeParse({ min_rating: 6 }).success).toBe(false);
  });

  it("requires at least one mutable field for patch payloads", () => {
    expect(paymentSettingsUpdateSchema.safeParse({}).success).toBe(false);
    expect(paymentSettingsUpdateSchema.safeParse({ stripe_enabled: false }).success).toBe(true);

    expect(vendorProductUpdateSchema.safeParse({}).success).toBe(false);
    expect(vendorProductUpdateSchema.safeParse({ title: "Updated product title" }).success).toBe(
      true,
    );
  });
});
