export const roleList = ["buyer", "vendor_owner", "super_admin", "support", "finance", "catalog_moderator"] as const;

export type PrincipalRole = (typeof roleList)[number];

export interface ApiErrorResponse {
  error: {
    code: string;
    message: string;
  };
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

export interface Order {
  id: string;
  buyer_user_id?: string;
  guest_token?: string;
  status: string;
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
