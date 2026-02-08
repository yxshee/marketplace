import type {
  AdminAuditLogListResponse,
  AdminAnalyticsRevenueResponse,
  AdminAnalyticsVendorsResponse,
  AdminDashboardOverviewResponse,
  AdminModerationProductListResponse,
  AdminOrderListResponse,
  AdminPromotion,
  AdminPromotionListResponse,
  AdminVendorListResponse,
  AuthResponse,
  BuyerRefundRequestCreateResponse,
  CODPaymentResponse,
  CartResponse,
  CatalogCategoriesResponse,
  CatalogCategory,
  CatalogListResponse,
  CatalogProductDetailResponse,
  CatalogSearchParams,
  CheckoutQuoteResponse,
  OrderResponse,
  OrderStatus,
  PaymentSettingsResponse,
  StripeIntentResponse,
  VendorRefundDecision,
  VendorRefundRequestListResponse,
  VendorCoupon,
  VendorCouponListResponse,
  VendorAnalyticsCouponsResponse,
  VendorAnalyticsOverviewResponse,
  VendorAnalyticsTopProductsResponse,
  VendorProduct,
  VendorProductListResponse,
  VendorProfile,
  VendorShipment,
  VendorShipmentListResponse,
  VendorShipmentStatus,
} from "@marketplace/shared/contracts/api";
import {
  adminAuditLogQuerySchema,
  adminRevenueAnalyticsQuerySchema,
  adminPromotionCreateSchema,
  adminPromotionUpdateSchema,
  adminOrderStatusUpdateSchema,
  authCredentialsSchema,
  cartItemMutationSchema,
  cartItemQtySchema,
  catalogSearchSchema,
  codConfirmPaymentSchema,
  checkoutPlaceOrderSchema,
  paymentSettingsUpdateSchema,
  refundRequestCreateSchema,
  stripeCreateIntentSchema,
  vendorCouponCreateSchema,
  vendorRefundDecisionUpdateSchema,
  vendorCouponUpdateSchema,
  vendorProductCreateSchema,
  vendorProductUpdateSchema,
  vendorRegisterSchema,
  vendorShipmentStatusUpdateSchema,
} from "@marketplace/shared/schemas/common";
import {
  fallbackCategories,
  fallbackProducts,
  fallbackVendorNameByID,
} from "@/lib/catalog-fallback";

const API_BASE_URL = process.env.MARKETPLACE_API_BASE_URL ?? "http://localhost:8080/api/v1";
const guestTokenHeader = "X-Guest-Token";

interface RequestOptions {
  method?: "GET" | "POST" | "PATCH" | "DELETE";
  body?: unknown;
  guestToken?: string;
  accessToken?: string;
}

export interface PaginationQueryParams {
  limit?: number;
  offset?: number;
}

export interface ApiCallResult<T> {
  payload: T;
  guestToken?: string;
}

const toQueryStringFromRecord = <T extends object>(params: T): string => {
  const query = new URLSearchParams();
  Object.entries(params as Record<string, unknown>).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") {
      return;
    }
    query.set(key, String(value));
  });
  const serialized = query.toString();
  return serialized.length > 0 ? `?${serialized}` : "";
};

const toQueryString = (params: CatalogSearchParams): string => {
  return toQueryStringFromRecord(params);
};

const normalizePagination = (params: PaginationQueryParams = {}): PaginationQueryParams => {
  const normalized: PaginationQueryParams = {};
  if (typeof params.limit === "number" && Number.isFinite(params.limit)) {
    normalized.limit = Math.min(200, Math.max(1, Math.trunc(params.limit)));
  }
  if (typeof params.offset === "number" && Number.isFinite(params.offset)) {
    normalized.offset = Math.max(0, Math.trunc(params.offset));
  }
  return normalized;
};

const normalizeSearchParams = (params: CatalogSearchParams): CatalogSearchParams => {
  const parsed = catalogSearchSchema.safeParse(params);
  if (!parsed.success) {
    return {};
  }
  return parsed.data;
};

const fetchJSON = async <T>(
  path: string,
  options: RequestOptions = {},
): Promise<ApiCallResult<T>> => {
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

  const headerGuestToken = response.headers.get(guestTokenHeader) ?? undefined;
  if (response.status === 204) {
    return {
      payload: {} as T,
      guestToken: headerGuestToken,
    };
  }

  const rawBody = await response.text();
  const payload = rawBody.length > 0 ? (JSON.parse(rawBody) as T) : ({} as T);
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

export const getCatalogProducts = async (
  params: CatalogSearchParams = {},
): Promise<CatalogListResponse> => {
  const normalized = normalizeSearchParams(params);
  try {
    return (await fetchJSON<CatalogListResponse>(`/catalog/products${toQueryString(normalized)}`))
      .payload;
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

export const getCatalogProductById = async (
  productID: string,
): Promise<CatalogProductDetailResponse | null> => {
  try {
    return (await fetchJSON<CatalogProductDetailResponse>(`/catalog/products/${productID}`))
      .payload;
  } catch {
    const product = fallbackProducts.find((item) => item.id === productID);
    if (!product) {
      return null;
    }
    return {
      item: product,
      vendor: fallbackVendorNameByID[product.vendor_id] ?? {
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

export const deleteCartItem = async (
  itemID: string,
  guestToken?: string,
): Promise<ApiCallResult<CartResponse>> => {
  return fetchJSON<CartResponse>(`/cart/items/${itemID}`, {
    method: "DELETE",
    guestToken,
  });
};

export const getCheckoutQuote = async (
  guestToken?: string,
): Promise<ApiCallResult<CheckoutQuoteResponse>> => {
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

export const getOrderByID = async (
  orderID: string,
  guestToken?: string,
): Promise<ApiCallResult<OrderResponse>> => {
  return fetchJSON<OrderResponse>(`/orders/${orderID}`, {
    guestToken,
  });
};

export const createOrderRefundRequest = async (
  orderID: string,
  input: {
    shipment_id: string;
    reason: string;
    requested_amount_cents?: number;
  },
  guestToken?: string,
  accessToken?: string,
): Promise<ApiCallResult<BuyerRefundRequestCreateResponse>> => {
  const parsed = refundRequestCreateSchema.parse(input);
  return fetchJSON<BuyerRefundRequestCreateResponse>(`/orders/${orderID}/refund-requests`, {
    method: "POST",
    body: parsed,
    guestToken,
    accessToken,
  });
};

export const getBuyerPaymentSettings = async (
  guestToken?: string,
): Promise<ApiCallResult<PaymentSettingsResponse>> => {
  return fetchJSON<PaymentSettingsResponse>("/payments/settings", {
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

export const getVendorVerificationStatus = async (
  accessToken: string,
): Promise<ApiCallResult<VendorProfile>> => {
  return fetchJSON<VendorProfile>("/vendor/verification-status", {
    accessToken,
  });
};

export interface AdminAuditLogQueryParams {
  actor_type?: "admin" | "vendor" | "buyer";
  actor_id?: string;
  action?: string;
  target_type?: string;
  target_id?: string;
  limit?: number;
  offset?: number;
}

export interface AdminRevenueAnalyticsQueryParams {
  days?: number;
}

const normalizeAdminAuditLogQuery = (
  params: AdminAuditLogQueryParams,
): AdminAuditLogQueryParams => {
  const parsed = adminAuditLogQuerySchema.safeParse(params);
  if (!parsed.success) {
    return {};
  }
  return parsed.data;
};

const normalizeAdminRevenueAnalyticsQuery = (
  params: AdminRevenueAnalyticsQueryParams,
): AdminRevenueAnalyticsQueryParams => {
  const parsed = adminRevenueAnalyticsQuerySchema.safeParse(params);
  if (!parsed.success) {
    return {};
  }
  return parsed.data;
};

export const getAdminAuditLogs = async (
  accessToken: string,
  params: AdminAuditLogQueryParams = {},
): Promise<ApiCallResult<AdminAuditLogListResponse>> => {
  const normalized = normalizeAdminAuditLogQuery(params);
  const suffix = toQueryStringFromRecord(normalized);
  return fetchJSON<AdminAuditLogListResponse>(`/admin/audit-logs${suffix}`, {
    accessToken,
  });
};

export const getAdminDashboardOverview = async (
  accessToken: string,
): Promise<ApiCallResult<AdminDashboardOverviewResponse>> => {
  return fetchJSON<AdminDashboardOverviewResponse>("/admin/dashboard/overview", {
    accessToken,
  });
};

export const getAdminAnalyticsRevenue = async (
  accessToken: string,
  params: AdminRevenueAnalyticsQueryParams = {},
): Promise<ApiCallResult<AdminAnalyticsRevenueResponse>> => {
  const normalized = normalizeAdminRevenueAnalyticsQuery(params);
  const suffix = toQueryStringFromRecord(normalized);
  return fetchJSON<AdminAnalyticsRevenueResponse>(`/admin/analytics/revenue${suffix}`, {
    accessToken,
  });
};

export const getAdminAnalyticsVendors = async (
  accessToken: string,
  params: PaginationQueryParams = {},
): Promise<ApiCallResult<AdminAnalyticsVendorsResponse>> => {
  const suffix = toQueryStringFromRecord(normalizePagination(params));
  return fetchJSON<AdminAnalyticsVendorsResponse>(`/admin/analytics/vendors${suffix}`, {
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

export const getAdminVendors = async (
  accessToken: string,
  verificationState?: "pending" | "verified" | "rejected" | "suspended",
  params: PaginationQueryParams = {},
): Promise<ApiCallResult<AdminVendorListResponse>> => {
  const suffix = toQueryStringFromRecord({
    verification_state: verificationState,
    ...normalizePagination(params),
  });
  return fetchJSON<AdminVendorListResponse>(`/admin/vendors${suffix}`, {
    accessToken,
  });
};

export const getAdminPromotions = async (
  accessToken: string,
  params: PaginationQueryParams = {},
): Promise<ApiCallResult<AdminPromotionListResponse>> => {
  const suffix = toQueryStringFromRecord(normalizePagination(params));
  return fetchJSON<AdminPromotionListResponse>(`/admin/promotions${suffix}`, {
    accessToken,
  });
};

export const createAdminPromotion = async (
  input: {
    name: string;
    rule_json: Record<string, unknown>;
    starts_at?: string;
    ends_at?: string;
    stackable?: boolean;
    active?: boolean;
  },
  accessToken: string,
): Promise<ApiCallResult<AdminPromotion>> => {
  const parsed = adminPromotionCreateSchema.parse(input);
  return fetchJSON<AdminPromotion>("/admin/promotions", {
    method: "POST",
    body: parsed,
    accessToken,
  });
};

export const updateAdminPromotion = async (
  promotionID: string,
  input: {
    name?: string;
    rule_json?: Record<string, unknown>;
    starts_at?: string;
    ends_at?: string;
    stackable?: boolean;
    active?: boolean;
  },
  accessToken: string,
): Promise<ApiCallResult<AdminPromotion>> => {
  const parsed = adminPromotionUpdateSchema.parse(input);
  return fetchJSON<AdminPromotion>(`/admin/promotions/${promotionID}`, {
    method: "PATCH",
    body: parsed,
    accessToken,
  });
};

export const deleteAdminPromotion = async (
  promotionID: string,
  accessToken: string,
): Promise<void> => {
  await fetchJSON<object>(`/admin/promotions/${promotionID}`, {
    method: "DELETE",
    accessToken,
  });
};

export const updateAdminVendorVerification = async (
  vendorID: string,
  input: { state: "pending" | "verified" | "rejected" | "suspended"; reason?: string },
  accessToken: string,
): Promise<ApiCallResult<VendorProfile>> => {
  return fetchJSON<VendorProfile>(`/admin/vendors/${vendorID}/verification`, {
    method: "PATCH",
    body: input,
    accessToken,
  });
};

export const getAdminModerationProducts = async (
  accessToken: string,
  status: "draft" | "pending_approval" | "approved" | "rejected" = "pending_approval",
  params: PaginationQueryParams = {},
): Promise<ApiCallResult<AdminModerationProductListResponse>> => {
  const suffix = toQueryStringFromRecord({
    status,
    ...normalizePagination(params),
  });
  return fetchJSON<AdminModerationProductListResponse>(
    `/admin/moderation/products${suffix}`,
    {
      accessToken,
    },
  );
};

export const getAdminOrders = async (
  accessToken: string,
  status?: OrderStatus,
  params: PaginationQueryParams = {},
): Promise<ApiCallResult<AdminOrderListResponse>> => {
  const suffix = toQueryStringFromRecord({
    status,
    ...normalizePagination(params),
  });
  return fetchJSON<AdminOrderListResponse>(`/admin/orders${suffix}`, {
    accessToken,
  });
};

export const getAdminOrderByID = async (
  orderID: string,
  accessToken: string,
): Promise<ApiCallResult<OrderResponse>> => {
  return fetchJSON<OrderResponse>(`/admin/orders/${orderID}`, {
    accessToken,
  });
};

export const updateAdminOrderStatus = async (
  orderID: string,
  input: { status: OrderStatus },
  accessToken: string,
): Promise<ApiCallResult<OrderResponse>> => {
  const parsed = adminOrderStatusUpdateSchema.parse(input);
  return fetchJSON<OrderResponse>(`/admin/orders/${orderID}/status`, {
    method: "PATCH",
    body: parsed,
    accessToken,
  });
};

export const updateAdminModerationProduct = async (
  productID: string,
  input: { decision: "approve" | "reject"; reason?: string },
  accessToken: string,
): Promise<ApiCallResult<VendorProduct>> => {
  return fetchJSON<VendorProduct>(`/admin/moderation/products/${productID}`, {
    method: "PATCH",
    body: input,
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

export const getVendorProducts = async (
  accessToken: string,
  params: PaginationQueryParams = {},
): Promise<ApiCallResult<VendorProductListResponse>> => {
  const suffix = toQueryStringFromRecord(normalizePagination(params));
  return fetchJSON<VendorProductListResponse>(`/vendor/products${suffix}`, {
    accessToken,
  });
};

export const createVendorProduct = async (
  input: {
    title: string;
    description: string;
    category_slug?: string;
    tags?: string[];
    price_incl_tax_cents: number;
    currency: string;
    stock_qty?: number;
  },
  accessToken: string,
): Promise<ApiCallResult<VendorProduct>> => {
  const parsed = vendorProductCreateSchema.parse(input);
  return fetchJSON<VendorProduct>("/vendor/products", {
    method: "POST",
    body: parsed,
    accessToken,
  });
};

export const updateVendorProduct = async (
  productID: string,
  input: {
    title?: string;
    description?: string;
    category_slug?: string;
    tags?: string[];
    price_incl_tax_cents?: number;
    currency?: string;
    stock_qty?: number;
  },
  accessToken: string,
): Promise<ApiCallResult<VendorProduct>> => {
  const parsed = vendorProductUpdateSchema.parse(input);
  return fetchJSON<VendorProduct>(`/vendor/products/${productID}`, {
    method: "PATCH",
    body: parsed,
    accessToken,
  });
};

export const deleteVendorProduct = async (
  productID: string,
  accessToken: string,
): Promise<void> => {
  await fetchJSON<object>(`/vendor/products/${productID}`, {
    method: "DELETE",
    accessToken,
  });
};

export const submitVendorProductModeration = async (
  productID: string,
  accessToken: string,
): Promise<ApiCallResult<VendorProduct>> => {
  return fetchJSON<VendorProduct>(`/vendor/products/${productID}/submit-moderation`, {
    method: "POST",
    body: {},
    accessToken,
  });
};

export const getVendorCoupons = async (
  accessToken: string,
): Promise<ApiCallResult<VendorCouponListResponse>> => {
  return fetchJSON<VendorCouponListResponse>("/vendor/coupons", {
    accessToken,
  });
};

export const createVendorCoupon = async (
  input: {
    code: string;
    discount_type: "percent" | "amount_cents";
    discount_value: number;
    starts_at?: string;
    ends_at?: string;
    usage_limit?: number;
    active?: boolean;
  },
  accessToken: string,
): Promise<ApiCallResult<VendorCoupon>> => {
  const parsed = vendorCouponCreateSchema.parse(input);
  return fetchJSON<VendorCoupon>("/vendor/coupons", {
    method: "POST",
    body: parsed,
    accessToken,
  });
};

export const updateVendorCoupon = async (
  couponID: string,
  input: {
    code?: string;
    discount_type?: "percent" | "amount_cents";
    discount_value?: number;
    active?: boolean;
  },
  accessToken: string,
): Promise<ApiCallResult<VendorCoupon>> => {
  const parsed = vendorCouponUpdateSchema.parse(input);
  return fetchJSON<VendorCoupon>(`/vendor/coupons/${couponID}`, {
    method: "PATCH",
    body: parsed,
    accessToken,
  });
};

export const deleteVendorCoupon = async (couponID: string, accessToken: string): Promise<void> => {
  await fetchJSON<object>(`/vendor/coupons/${couponID}`, {
    method: "DELETE",
    accessToken,
  });
};

export const getVendorShipments = async (
  accessToken: string,
  params: PaginationQueryParams = {},
): Promise<ApiCallResult<VendorShipmentListResponse>> => {
  const suffix = toQueryStringFromRecord(normalizePagination(params));
  return fetchJSON<VendorShipmentListResponse>(`/vendor/shipments${suffix}`, {
    accessToken,
  });
};

export const getVendorShipmentByID = async (
  shipmentID: string,
  accessToken: string,
): Promise<ApiCallResult<VendorShipment>> => {
  return fetchJSON<VendorShipment>(`/vendor/shipments/${shipmentID}`, {
    accessToken,
  });
};

export const updateVendorShipmentStatus = async (
  shipmentID: string,
  input: { status: VendorShipmentStatus },
  accessToken: string,
): Promise<ApiCallResult<VendorShipment>> => {
  const parsed = vendorShipmentStatusUpdateSchema.parse(input);
  return fetchJSON<VendorShipment>(`/vendor/shipments/${shipmentID}/status`, {
    method: "PATCH",
    body: parsed,
    accessToken,
  });
};

export const getVendorRefundRequests = async (
  accessToken: string,
  status?: "pending" | "approved" | "rejected",
  params: PaginationQueryParams = {},
): Promise<ApiCallResult<VendorRefundRequestListResponse>> => {
  const suffix = toQueryStringFromRecord({
    status,
    ...normalizePagination(params),
  });
  return fetchJSON<VendorRefundRequestListResponse>(`/vendor/refund-requests${suffix}`, {
    accessToken,
  });
};

export const getVendorAnalyticsOverview = async (
  accessToken: string,
): Promise<ApiCallResult<VendorAnalyticsOverviewResponse>> => {
  return fetchJSON<VendorAnalyticsOverviewResponse>("/vendor/analytics/overview", {
    accessToken,
  });
};

export const getVendorAnalyticsTopProducts = async (
  accessToken: string,
): Promise<ApiCallResult<VendorAnalyticsTopProductsResponse>> => {
  return fetchJSON<VendorAnalyticsTopProductsResponse>("/vendor/analytics/top-products", {
    accessToken,
  });
};

export const getVendorAnalyticsCoupons = async (
  accessToken: string,
): Promise<ApiCallResult<VendorAnalyticsCouponsResponse>> => {
  return fetchJSON<VendorAnalyticsCouponsResponse>("/vendor/analytics/coupons", {
    accessToken,
  });
};

export const decideVendorRefundRequest = async (
  refundRequestID: string,
  input: {
    decision: VendorRefundDecision;
    decision_reason?: string;
  },
  accessToken: string,
): Promise<ApiCallResult<VendorRefundRequestListResponse["items"][number]>> => {
  const parsed = vendorRefundDecisionUpdateSchema.parse(input);
  return fetchJSON<VendorRefundRequestListResponse["items"][number]>(
    `/vendor/refund-requests/${refundRequestID}/decision`,
    {
      method: "PATCH",
      body: parsed,
      accessToken,
    },
  );
};
