export const roleList = [
  "buyer",
  "vendor_owner",
  "super_admin",
  "support",
  "finance",
  "catalog_moderator",
] as const;

export type PrincipalRole = (typeof roleList)[number];

export interface ApiErrorResponse {
  error: {
    code: string;
    message: string;
  };
}

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  user: {
    id: string;
    email: string;
    role: string;
    vendor_id: string | null;
  };
}

export interface VendorProfile {
  id: string;
  owner_user_id: string;
  slug: string;
  display_name: string;
  verification_state: "pending" | "verified" | "rejected" | "suspended";
  commission_override_bps: number | null;
  created_at: string;
  updated_at: string;
}

export interface AdminVendorListResponse {
  items: VendorProfile[];
  total: number;
  limit: number;
  offset: number;
}

export interface AdminPromotion {
  id: string;
  name: string;
  rule_json: Record<string, unknown>;
  starts_at?: string;
  ends_at?: string;
  stackable: boolean;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface AdminPromotionListResponse {
  items: AdminPromotion[];
  total: number;
  limit: number;
  offset: number;
}

export interface AuditLogEntry {
  id: string;
  actor_type: "admin" | "vendor" | "buyer";
  actor_id: string;
  actor_role?: string;
  action: string;
  target_type: string;
  target_id: string;
  before_json?: Record<string, unknown>;
  after_json?: Record<string, unknown>;
  metadata_json?: Record<string, unknown>;
  created_at: string;
}

export interface AdminAuditLogListResponse {
  items: AuditLogEntry[];
  total: number;
}

export interface AdminDashboardOrderVolumes {
  total: number;
  pending_payment: number;
  cod_confirmed: number;
  paid: number;
  payment_failed: number;
}

export interface AdminDashboardVendorMetrics {
  total_vendors: number;
  pending_verification: number;
  verified: number;
  rejected: number;
  suspended: number;
  active_with_sales: number;
  vendors_with_refund_risk: number;
}

export interface AdminDashboardModerationQueueMetrics {
  pending_products: number;
}

export interface AdminDashboardDisputeMetrics {
  refund_requests_total: number;
  pending_total: number;
  approved_total: number;
  rejected_total: number;
}

export interface AdminDashboardOverviewResponse {
  currency: string;
  platform_revenue_cents: number;
  commission_earned_cents: number;
  order_volumes: AdminDashboardOrderVolumes;
  vendor_metrics: AdminDashboardVendorMetrics;
  moderation_queue: AdminDashboardModerationQueueMetrics;
  disputes: AdminDashboardDisputeMetrics;
  generated_at: string;
}

export interface AdminAnalyticsRevenueSummary {
  settled_orders_total: number;
  gross_revenue_cents: number;
  commission_earned_cents: number;
  average_order_value_cents: number;
}

export interface AdminAnalyticsRevenuePoint {
  date: string;
  order_count: number;
  gross_revenue_cents: number;
  commission_earned_cents: number;
}

export interface AdminAnalyticsRevenueResponse {
  currency: string;
  window_days: number;
  summary: AdminAnalyticsRevenueSummary;
  points: AdminAnalyticsRevenuePoint[];
}

export interface AdminVendorAnalyticsItem {
  vendor_id: string;
  slug: string;
  display_name: string;
  verification_state: "pending" | "verified" | "rejected" | "suspended";
  commission_bps: number;
  order_count: number;
  settled_order_count: number;
  gross_revenue_cents: number;
  commission_earned_cents: number;
  shipment_count: number;
  pending_shipment_count: number;
  shipped_shipment_count: number;
  delivered_shipment_count: number;
  cancelled_shipment_count: number;
  refund_requests_total: number;
  refund_pending_total: number;
  refund_approved_total: number;
  refund_rejected_total: number;
  refund_approval_rate_bps: number;
  settled_order_refund_rate_bps: number;
}

export interface AdminAnalyticsVendorsResponse {
  items: AdminVendorAnalyticsItem[];
  total: number;
  limit: number;
  offset: number;
}

export interface VendorProduct {
  id: string;
  vendor_id: string;
  owner_user_id: string;
  title: string;
  description: string;
  category_slug: string;
  tags: string[];
  price_incl_tax_cents: number;
  currency: string;
  stock_qty: number;
  rating_average: number;
  status: "draft" | "pending_approval" | "approved" | "rejected";
  moderation_reason?: string;
  created_at: string;
  updated_at: string;
}

export interface VendorProductListResponse {
  items: VendorProduct[];
  total: number;
  limit: number;
  offset: number;
}

export interface AdminModerationProductListResponse {
  items: VendorProduct[];
  total: number;
  limit: number;
  offset: number;
}

export interface VendorCoupon {
  id: string;
  vendor_id: string;
  code: string;
  discount_type: "percent" | "amount_cents";
  discount_value: number;
  starts_at?: string;
  ends_at?: string;
  usage_limit?: number;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface VendorCouponListResponse {
  items: VendorCoupon[];
  total: number;
}

export interface HealthResponse {
  status: "ok";
  service: string;
  timestamp: string;
}

export interface Money {
  amountCents: number;
  currency: "USD";
}

export interface CatalogCategory {
  slug: string;
  name: string;
}

export interface CatalogProduct {
  id: string;
  vendor_id: string;
  title: string;
  description: string;
  category_slug: string;
  tags: string[];
  price_incl_tax_cents: number;
  currency: string;
  stock_qty: number;
  rating_average: number;
  created_at: string;
}

export interface CatalogListResponse {
  items: CatalogProduct[];
  total: number;
  limit: number;
  offset: number;
}

export interface CatalogCategoriesResponse {
  items: CatalogCategory[];
}

export interface CatalogProductDetailResponse {
  item: CatalogProduct;
  vendor: {
    id: string;
    slug: string;
    displayName: string;
  };
}

export interface CatalogSearchParams {
  q?: string;
  category?: string;
  vendor?: string;
  price_min?: number;
  price_max?: number;
  min_rating?: number;
  sort?: "relevance" | "newest" | "price_low_high" | "price_high_low" | "rating";
  limit?: number;
  offset?: number;
}

export interface CartItem {
  id: string;
  product_id: string;
  vendor_id: string;
  title: string;
  qty: number;
  unit_price_cents: number;
  line_total_cents: number;
  currency: string;
  available_stock: number;
  last_updated_unix: number;
}

export interface CartResponse {
  id: string;
  currency: string;
  item_count: number;
  subtotal_cents: number;
  items: CartItem[];
  updated_at: string;
  guest_token?: string;
}

export interface QuoteShipment {
  vendor_id: string;
  item_count: number;
  subtotal_cents: number;
  shipping_fee_cents: number;
  total_cents: number;
  items: CartItem[];
}

export interface CheckoutQuoteResponse {
  currency: string;
  item_count: number;
  shipment_count: number;
  subtotal_cents: number;
  shipping_cents: number;
  total_cents: number;
  shipments: QuoteShipment[];
  guest_token?: string;
}

export interface OrderShipment {
  id: string;
  vendor_id: string;
  status: string;
  item_count: number;
  subtotal_cents: number;
  shipping_fee_cents: number;
  total_cents: number;
}

export interface OrderItem {
  id: string;
  shipment_id: string;
  product_id: string;
  vendor_id: string;
  title: string;
  qty: number;
  unit_price_cents: number;
  line_total_cents: number;
  currency: string;
}

export type VendorShipmentStatus = "pending" | "packed" | "shipped" | "delivered" | "cancelled";

export interface VendorShipmentStatusEvent {
  shipment_id: string;
  vendor_id: string;
  status: VendorShipmentStatus;
  actor_user_id?: string;
  at: string;
}

export interface VendorShipment {
  id: string;
  order_id: string;
  order_status: string;
  vendor_id: string;
  status: VendorShipmentStatus;
  item_count: number;
  subtotal_cents: number;
  shipping_fee_cents: number;
  total_cents: number;
  currency: string;
  items: OrderItem[];
  created_at: string;
  updated_at: string;
  shipped_at?: string;
  delivered_at?: string;
  timeline: VendorShipmentStatusEvent[];
}

export interface VendorShipmentListResponse {
  items: VendorShipment[];
  total: number;
  limit: number;
  offset: number;
}

export interface VendorAnalyticsConversionFunnel {
  orders_total: number;
  orders_paid: number;
  shipments_total: number;
  shipments_shipped: number;
  shipments_delivered: number;
}

export interface VendorAnalyticsRefundStats {
  requests_total: number;
  pending_total: number;
  approved_total: number;
  rejected_total: number;
  approval_rate_bps: number;
  order_refund_rate_bps: number;
}

export interface VendorAnalyticsOverviewResponse {
  currency: string;
  revenue_cents: number;
  order_count: number;
  paid_order_count: number;
  shipment_count: number;
  conversion_funnel: VendorAnalyticsConversionFunnel;
  refund_stats: VendorAnalyticsRefundStats;
}

export interface VendorAnalyticsTopProduct {
  product_id: string;
  title: string;
  order_count: number;
  units_sold: number;
  revenue_cents: number;
}

export interface VendorAnalyticsTopProductsResponse {
  items: VendorAnalyticsTopProduct[];
  total: number;
}

export interface VendorAnalyticsCouponPerformance {
  coupon_id: string;
  code: string;
  active: boolean;
  discount_type: "percent" | "amount_cents";
  discount_value: number;
  usage_count: number;
  discounts_granted_cents: number;
  attributed_revenue_cents: number;
  conversion_rate_bps: number;
  created_at: string;
  updated_at: string;
}

export interface VendorAnalyticsCouponsResponse {
  items: VendorAnalyticsCouponPerformance[];
  total: number;
}

export type RefundRequestStatus = "pending" | "approved" | "rejected";
export type VendorRefundDecision = "approve" | "reject";

export interface RefundRequest {
  id: string;
  order_id: string;
  shipment_id: string;
  vendor_id: string;
  buyer_user_id?: string;
  guest_token?: string;
  reason: string;
  requested_amount_cents: number;
  currency: string;
  status: RefundRequestStatus;
  outcome: RefundRequestStatus;
  decision?: VendorRefundDecision;
  decision_reason?: string;
  decided_by_user_id?: string;
  decided_at?: string;
  created_at: string;
  updated_at: string;
}

export interface BuyerRefundRequestCreateResponse {
  refund_request: RefundRequest;
  guest_token?: string;
}

export interface VendorRefundRequestListResponse {
  items: RefundRequest[];
  total: number;
  limit: number;
  offset: number;
}

export interface Order {
  id: string;
  buyer_user_id?: string;
  guest_token?: string;
  status: OrderStatus;
  currency: string;
  item_count: number;
  shipment_count: number;
  subtotal_cents: number;
  shipping_cents: number;
  discount_cents: number;
  tax_cents: number;
  total_cents: number;
  idempotency_key: string;
  shipments: OrderShipment[];
  items: OrderItem[];
  created_at: string;
}

export interface OrderResponse {
  order: Order;
  guest_token?: string;
}

export type OrderStatus = "pending_payment" | "cod_confirmed" | "paid" | "payment_failed";

export interface AdminOrderListResponse {
  items: Order[];
  total: number;
  limit: number;
  offset: number;
}

export interface StripeIntentResponse {
  id: string;
  order_id: string;
  method: "stripe";
  status: "pending" | "succeeded" | "failed";
  provider: "stripe";
  provider_ref: string;
  client_secret: string;
  amount_cents: number;
  currency: string;
  created_at: string;
  updated_at: string;
  guest_token?: string;
}

export interface CODPaymentResponse {
  id: string;
  order_id: string;
  method: "cod";
  status: "pending_collection";
  provider: "cod";
  provider_ref: string;
  amount_cents: number;
  currency: string;
  created_at: string;
  updated_at: string;
  guest_token?: string;
}

export interface StripeWebhookResponse {
  event_id: string;
  processed: boolean;
  duplicate: boolean;
  payment_id?: string;
  order_id?: string;
  payment_status?: "pending" | "succeeded" | "failed";
}

export interface PaymentSettingsResponse {
  stripe_enabled: boolean;
  cod_enabled: boolean;
  updated_at: string;
}
