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
