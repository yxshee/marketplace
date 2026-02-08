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
