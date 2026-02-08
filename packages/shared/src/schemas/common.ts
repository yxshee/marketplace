import { z } from "zod";

export const emailSchema = z.string().trim().email();

export const uuidSchema = z.string().uuid();

export const moneySchema = z.object({
  amountCents: z.number().int().nonnegative(),
  currency: z.literal("USD"),
});

export const authCredentialsSchema = z.object({
  email: emailSchema,
  password: z.string().min(8).max(128),
});

export const catalogSortSchema = z.enum([
  "relevance",
  "newest",
  "price_low_high",
  "price_high_low",
  "rating",
]);

export const catalogSearchSchema = z.object({
  q: z.string().trim().optional(),
  category: z.string().trim().optional(),
  vendor: z.string().trim().optional(),
  price_min: z.coerce.number().int().min(0).optional(),
  price_max: z.coerce.number().int().min(0).optional(),
  min_rating: z.coerce.number().min(0).max(5).optional(),
  sort: catalogSortSchema.optional(),
  limit: z.coerce.number().int().min(1).max(100).optional(),
  offset: z.coerce.number().int().min(0).optional(),
});

export const cartItemMutationSchema = z.object({
  product_id: z.string().trim().min(1),
  qty: z.coerce.number().int().min(1).max(999),
});

export const cartItemQtySchema = z.object({
  qty: z.coerce.number().int().min(1).max(999),
});

export const checkoutPlaceOrderSchema = z.object({
  idempotency_key: z.string().trim().min(8).max(128),
});

export const stripeCreateIntentSchema = z.object({
  order_id: z.string().trim().min(1),
  idempotency_key: z.string().trim().min(8).max(128),
});

export const codConfirmPaymentSchema = z.object({
  order_id: z.string().trim().min(1),
  idempotency_key: z.string().trim().min(8).max(128),
});

export const vendorRegisterSchema = z.object({
  slug: z
    .string()
    .trim()
    .min(3)
    .max(48)
    .regex(/^[a-z0-9-]+$/),
  display_name: z.string().trim().min(2).max(80),
});

export const paymentSettingsUpdateSchema = z
  .object({
    stripe_enabled: z.boolean().optional(),
    cod_enabled: z.boolean().optional(),
  })
  .refine((value) => value.stripe_enabled !== undefined || value.cod_enabled !== undefined, {
    message: "at least one settings field is required",
  });

export const vendorProductCreateSchema = z.object({
  title: z.string().trim().min(2).max(120),
  description: z.string().trim().min(2).max(4000),
  category_slug: z.string().trim().min(2).max(48).default("general"),
  tags: z.array(z.string().trim().min(1).max(32)).max(20).default([]),
  price_incl_tax_cents: z.number().int().positive(),
  currency: z.string().trim().length(3),
  stock_qty: z.number().int().min(0).default(0),
});

export const vendorProductUpdateSchema = z
  .object({
    title: z.string().trim().min(2).max(120).optional(),
    description: z.string().trim().min(2).max(4000).optional(),
    category_slug: z.string().trim().min(2).max(48).optional(),
    tags: z.array(z.string().trim().min(1).max(32)).max(20).optional(),
    price_incl_tax_cents: z.number().int().positive().optional(),
    currency: z.string().trim().length(3).optional(),
    stock_qty: z.number().int().min(0).optional(),
  })
  .refine(
    (value) =>
      value.title !== undefined ||
      value.description !== undefined ||
      value.category_slug !== undefined ||
      value.tags !== undefined ||
      value.price_incl_tax_cents !== undefined ||
      value.currency !== undefined ||
      value.stock_qty !== undefined,
    {
      message: "at least one field is required",
    },
  );

export const vendorCouponCreateSchema = z.object({
  code: z.string().trim().min(3).max(32).regex(/^[A-Za-z0-9_-]+$/),
  discount_type: z.enum(["percent", "amount_cents"]),
  discount_value: z.number().int().positive(),
  starts_at: z.string().datetime().optional(),
  ends_at: z.string().datetime().optional(),
  usage_limit: z.number().int().positive().optional(),
  active: z.boolean().optional(),
});

export const vendorCouponUpdateSchema = z
  .object({
    code: z.string().trim().min(3).max(32).regex(/^[A-Za-z0-9_-]+$/).optional(),
    discount_type: z.enum(["percent", "amount_cents"]).optional(),
    discount_value: z.number().int().positive().optional(),
    active: z.boolean().optional(),
  })
  .refine(
    (value) =>
      value.code !== undefined ||
      value.discount_type !== undefined ||
      value.discount_value !== undefined ||
      value.active !== undefined,
    {
      message: "at least one field is required",
    },
  );

export const vendorShipmentStatusSchema = z.enum(["pending", "packed", "shipped", "delivered", "cancelled"]);

export const vendorShipmentStatusUpdateSchema = z.object({
  status: vendorShipmentStatusSchema,
});
