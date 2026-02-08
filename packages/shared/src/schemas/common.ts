import { z } from "zod";

export const emailSchema = z.string().trim().email();

export const uuidSchema = z.string().uuid();

export const moneySchema = z.object({
  amountCents: z.number().int().nonnegative(),
  currency: z.literal("USD"),
});
