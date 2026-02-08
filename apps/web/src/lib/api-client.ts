import type {
  CatalogCategoriesResponse,
  CatalogCategory,
  CatalogListResponse,
  CatalogProductDetailResponse,
  CatalogSearchParams,
} from "@marketplace/shared/contracts/api";
import { catalogSearchSchema } from "@marketplace/shared/schemas/common";
import { fallbackCategories, fallbackProducts, fallbackVendorNameByID } from "@/lib/catalog-fallback";

const API_BASE_URL = process.env.MARKETPLACE_API_BASE_URL ?? "http://localhost:8080/api/v1";

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

const fetchJSON = async <T>(path: string): Promise<T> => {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    cache: "no-store",
  });

  if (!response.ok) {
    throw new Error(`api request failed: ${response.status}`);
  }

  return (await response.json()) as T;
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

export const getCatalogProducts = async (params: CatalogSearchParams = {}): Promise<CatalogListResponse> => {
  const normalized = normalizeSearchParams(params);
  try {
    return await fetchJSON<CatalogListResponse>(`/catalog/products${toQueryString(normalized)}`);
  } catch {
    return fallbackSearch(normalized);
  }
};

export const getCatalogCategories = async (): Promise<CatalogCategory[]> => {
  try {
    const response = await fetchJSON<CatalogCategoriesResponse>("/catalog/categories");
    return response.items;
  } catch {
    return fallbackCategories;
  }
};

export const getCatalogProductById = async (productID: string): Promise<CatalogProductDetailResponse | null> => {
  try {
    return await fetchJSON<CatalogProductDetailResponse>(`/catalog/products/${productID}`);
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
