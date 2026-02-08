import type {
  AuthResponse,
  CODPaymentResponse,
  CartResponse,
  CatalogCategoriesResponse,
  CatalogCategory,
  CatalogListResponse,
  CatalogProductDetailResponse,
  CatalogSearchParams,
  CheckoutQuoteResponse,
  OrderResponse,
  PaymentSettingsResponse,
  StripeIntentResponse,
  VendorProfile,
} from "@marketplace/shared/contracts/api";
import {
  authCredentialsSchema,
  cartItemMutationSchema,
  cartItemQtySchema,
  catalogSearchSchema,
  codConfirmPaymentSchema,
  checkoutPlaceOrderSchema,
  paymentSettingsUpdateSchema,
  stripeCreateIntentSchema,
  vendorRegisterSchema,
} from "@marketplace/shared/schemas/common";
import { fallbackCategories, fallbackProducts, fallbackVendorNameByID } from "@/lib/catalog-fallback";

const API_BASE_URL = process.env.MARKETPLACE_API_BASE_URL ?? "http://localhost:8080/api/v1";
const guestTokenHeader = "X-Guest-Token";

interface RequestOptions {
  method?: "GET" | "POST" | "PATCH" | "DELETE";
  body?: unknown;
  guestToken?: string;
  accessToken?: string;
}

export interface ApiCallResult<T> {
  payload: T;
  guestToken?: string;
}

const toQueryString = (params: CatalogSearchParams): string => {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") {
      return;
    }
    query.set(key, String(value));
  });
  const serialized = query.toString();
  return serialized.length > 0 ? `?${serialized}` : "";
};

const normalizeSearchParams = (params: CatalogSearchParams): CatalogSearchParams => {
  const parsed = catalogSearchSchema.safeParse(params);
  if (!parsed.success) {
    return {};
  }
  return parsed.data;
};

const fetchJSON = async <T>(path: string, options: RequestOptions = {}): Promise<ApiCallResult<T>> => {
  const headers = new Headers({
    "Content-Type": "application/json",
  });
  if (options.guestToken) {
    headers.set(guestTokenHeader, options.guestToken);
  }
  if (options.accessToken) {
    headers.set("Authorization", `Bearer ${options.accessToken}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: options.method ?? "GET",
    cache: "no-store",
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  });

  if (!response.ok) {
    throw new Error(`api request failed: ${response.status}`);
  }

  const payload = (await response.json()) as T;
  const headerGuestToken = response.headers.get(guestTokenHeader) ?? undefined;
  const bodyGuestToken =
    typeof payload === "object" && payload !== null && "guest_token" in payload
      ? (payload as { guest_token?: string }).guest_token
      : undefined;

  return {
    payload,
    guestToken: headerGuestToken ?? bodyGuestToken,
  };
};

const fallbackSearch = (params: CatalogSearchParams): CatalogListResponse => {
  const query = params.q?.toLowerCase().trim();
  const category = params.category?.toLowerCase().trim();
  const vendor = params.vendor?.trim();

  const filtered = fallbackProducts.filter((product) => {
    if (category && product.category_slug !== category) {
      return false;
    }
    if (vendor && product.vendor_id !== vendor) {
      return false;
    }
    if (params.price_min !== undefined && product.price_incl_tax_cents < params.price_min) {
      return false;
    }
    if (params.price_max !== undefined && product.price_incl_tax_cents > params.price_max) {
      return false;
    }
    if (params.min_rating !== undefined && product.rating_average < params.min_rating) {
      return false;
    }

    if (!query) {
      return true;
    }

    const haystack = [product.title, product.description, ...product.tags].join(" ").toLowerCase();
    return haystack.includes(query);
  });

  const sorted = [...filtered].sort((left, right) => {
    switch (params.sort) {
      case "price_low_high":
        return left.price_incl_tax_cents - right.price_incl_tax_cents;
      case "price_high_low":
        return right.price_incl_tax_cents - left.price_incl_tax_cents;
      case "rating":
        return right.rating_average - left.rating_average;
      case "newest":
      case "relevance":
      default:
        return new Date(right.created_at).getTime() - new Date(left.created_at).getTime();
    }
  });

  const limit = params.limit ?? 20;
  const offset = params.offset ?? 0;

  return {
    items: sorted.slice(offset, offset + limit),
    total: sorted.length,
    limit,
    offset,
  };
};

const emptyCart = (guestToken?: string): CartResponse => ({
  id: "",
  currency: "USD",
  item_count: 0,
  subtotal_cents: 0,
  items: [],
  updated_at: new Date(0).toISOString(),
  guest_token: guestToken,
});

export const getCatalogProducts = async (params: CatalogSearchParams = {}): Promise<CatalogListResponse> => {
  const normalized = normalizeSearchParams(params);
  try {
    return (await fetchJSON<CatalogListResponse>(`/catalog/products${toQueryString(normalized)}`)).payload;
  } catch {
    return fallbackSearch(normalized);
  }
};

export const getCatalogCategories = async (): Promise<CatalogCategory[]> => {
  try {
    const response = await fetchJSON<CatalogCategoriesResponse>("/catalog/categories");
    return response.payload.items;
  } catch {
    return fallbackCategories;
  }
};

export const getCatalogProductById = async (productID: string): Promise<CatalogProductDetailResponse | null> => {
  try {
    return (await fetchJSON<CatalogProductDetailResponse>(`/catalog/products/${productID}`)).payload;
  } catch {
    const product = fallbackProducts.find((item) => item.id === productID);
    if (!product) {
      return null;
    }
    return {
      item: product,
      vendor:
        fallbackVendorNameByID[product.vendor_id] ?? {
          id: product.vendor_id,
          slug: product.vendor_id,
          displayName: "Independent Vendor",
        },
    };
  }
};

export const getCart = async (guestToken?: string): Promise<ApiCallResult<CartResponse>> => {
  try {
    const response = await fetchJSON<CartResponse>("/cart", { guestToken });
    return {
      payload: response.payload,
      guestToken: response.guestToken,
    };
  } catch {
    return {
      payload: emptyCart(guestToken),
      guestToken,
    };
  }
};

export const addCartItem = async (
  input: { product_id: string; qty: number },
  guestToken?: string,
): Promise<ApiCallResult<CartResponse>> => {
  const parsed = cartItemMutationSchema.parse(input);
  return fetchJSON<CartResponse>("/cart/items", {
    method: "POST",
    body: parsed,
    guestToken,
  });
};

export const updateCartItem = async (
  itemID: string,
  input: { qty: number },
  guestToken?: string,
): Promise<ApiCallResult<CartResponse>> => {
  const parsed = cartItemQtySchema.parse(input);
  return fetchJSON<CartResponse>(`/cart/items/${itemID}`, {
    method: "PATCH",
    body: parsed,
    guestToken,
  });
};

export const deleteCartItem = async (itemID: string, guestToken?: string): Promise<ApiCallResult<CartResponse>> => {
  return fetchJSON<CartResponse>(`/cart/items/${itemID}`, {
    method: "DELETE",
    guestToken,
  });
};

export const getCheckoutQuote = async (guestToken?: string): Promise<ApiCallResult<CheckoutQuoteResponse>> => {
  return fetchJSON<CheckoutQuoteResponse>("/checkout/quote", {
    method: "POST",
    body: {},
    guestToken,
  });
};

export const placeOrder = async (
  input: { idempotency_key: string },
  guestToken?: string,
): Promise<ApiCallResult<OrderResponse>> => {
  const parsed = checkoutPlaceOrderSchema.parse(input);
  return fetchJSON<OrderResponse>("/checkout/place-order", {
    method: "POST",
    body: parsed,
    guestToken,
  });
};

export const getOrderByID = async (orderID: string, guestToken?: string): Promise<ApiCallResult<OrderResponse>> => {
  return fetchJSON<OrderResponse>(`/orders/${orderID}`, {
    guestToken,
  });
};

export const createStripePaymentIntent = async (
  input: { order_id: string; idempotency_key: string },
  guestToken?: string,
): Promise<ApiCallResult<StripeIntentResponse>> => {
  const parsed = stripeCreateIntentSchema.parse(input);
  return fetchJSON<StripeIntentResponse>("/payments/stripe/intent", {
    method: "POST",
    body: parsed,
    guestToken,
  });
};

export const confirmCODPayment = async (
  input: { order_id: string; idempotency_key: string },
  guestToken?: string,
): Promise<ApiCallResult<CODPaymentResponse>> => {
  const parsed = codConfirmPaymentSchema.parse(input);
  return fetchJSON<CODPaymentResponse>("/payments/cod/confirm", {
    method: "POST",
    body: parsed,
    guestToken,
  });
};

export const registerAuthUser = async (input: {
  email: string;
  password: string;
}): Promise<ApiCallResult<AuthResponse>> => {
  const parsed = authCredentialsSchema.parse(input);
  return fetchJSON<AuthResponse>("/auth/register", {
    method: "POST",
    body: parsed,
  });
};

export const loginAuthUser = async (input: {
  email: string;
  password: string;
}): Promise<ApiCallResult<AuthResponse>> => {
  const parsed = authCredentialsSchema.parse(input);
  return fetchJSON<AuthResponse>("/auth/login", {
    method: "POST",
    body: parsed,
  });
};

export const registerVendorProfile = async (
  input: { slug: string; display_name: string },
  accessToken: string,
): Promise<ApiCallResult<VendorProfile>> => {
  const parsed = vendorRegisterSchema.parse(input);
  return fetchJSON<VendorProfile>("/vendors/register", {
    method: "POST",
    body: parsed,
    accessToken,
  });
};

export const getVendorVerificationStatus = async (accessToken: string): Promise<ApiCallResult<VendorProfile>> => {
  return fetchJSON<VendorProfile>("/vendor/verification-status", {
    accessToken,
  });
};

export const getAdminPaymentSettings = async (
  accessToken: string,
): Promise<ApiCallResult<PaymentSettingsResponse>> => {
  return fetchJSON<PaymentSettingsResponse>("/admin/settings/payments", {
    accessToken,
  });
};

export const updateAdminPaymentSettings = async (
  input: { stripe_enabled?: boolean; cod_enabled?: boolean },
  accessToken: string,
): Promise<ApiCallResult<PaymentSettingsResponse>> => {
  const parsed = paymentSettingsUpdateSchema.parse(input);
  return fetchJSON<PaymentSettingsResponse>("/admin/settings/payments", {
    method: "PATCH",
    body: parsed,
    accessToken,
  });
};
