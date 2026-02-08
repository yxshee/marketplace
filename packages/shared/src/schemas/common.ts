import { z } from "zod";

export const emailSchema = z.string().trim().email();

export const uuidSchema = z.string().uuid();

export const moneySchema = z.object({
  amountCents: z.number().int().nonnegative(),
  currency: z.literal("USD"),
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
