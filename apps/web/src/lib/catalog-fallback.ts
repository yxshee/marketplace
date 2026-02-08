import type { CatalogCategory, CatalogProduct } from "@marketplace/shared/contracts/api";

export const fallbackCategories: CatalogCategory[] = [
  { slug: "stationery", name: "Stationery" },
  { slug: "prints", name: "Prints" },
  { slug: "home", name: "Home" },
];

export const fallbackProducts: CatalogProduct[] = [
  {
    id: "prd_seed_grid_notebook",
    vendor_id: "ven_seed_north",
    title: "Grid Notebook",
    description: "A minimal notebook with soft cover and grid pages.",
    category_slug: "stationery",
    tags: ["notebook", "paper", "grid"],
    price_incl_tax_cents: 2200,
    currency: "USD",
    stock_qty: 75,
    rating_average: 4.8,
    created_at: new Date().toISOString(),
  },
  {
    id: "prd_seed_weekly_planner",
    vendor_id: "ven_seed_north",
    title: "Desk Weekly Planner",
    description: "Weekly planner with clean blocks and bold date markers.",
    category_slug: "stationery",
    tags: ["planner", "desk"],
    price_incl_tax_cents: 1800,
    currency: "USD",
    stock_qty: 41,
    rating_average: 4.5,
    created_at: new Date(Date.now() - 1000 * 60 * 30).toISOString(),
  },
  {
    id: "prd_seed_monochrome_poster",
    vendor_id: "ven_seed_linepress",
    title: "Monochrome Poster Print",
    description: "Museum-grade paper print for modern workspaces.",
    category_slug: "prints",
    tags: ["poster", "print", "art"],
    price_incl_tax_cents: 4800,
    currency: "USD",
    stock_qty: 19,
    rating_average: 4.9,
    created_at: new Date(Date.now() - 1000 * 60 * 60).toISOString(),
  },
  {
    id: "prd_seed_ceramic_cup",
    vendor_id: "ven_seed_linepress",
    title: "Ceramic Coffee Cup",
    description: "Hand-glazed cup built for everyday use.",
    category_slug: "home",
    tags: ["home", "ceramic", "cup"],
    price_incl_tax_cents: 3200,
    currency: "USD",
    stock_qty: 33,
    rating_average: 4.4,
    created_at: new Date(Date.now() - 1000 * 60 * 90).toISOString(),
  },
];

export const fallbackVendorNameByID: Record<string, { id: string; slug: string; displayName: string }> = {
  ven_seed_north: {
    id: "ven_seed_north",
    slug: "north-studio",
    displayName: "North Studio",
  },
  ven_seed_linepress: {
    id: "ven_seed_linepress",
    slug: "line-press",
    displayName: "Line Press",
  },
};
