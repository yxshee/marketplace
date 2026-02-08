# API Endpoints (v1)

Base path: `/api/v1`

## Public + Auth
- `GET /health`
- `GET /catalog/categories`
- `GET /catalog/products`
- `GET /catalog/products/{productID}`
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`
- `GET /auth/me`
- `POST /vendors/register`
- `GET /vendor/profile`
- `GET /vendor/verification-status`

## Buyer
- `GET /cart`
- `POST /cart/items`
- `PATCH /cart/items/{itemID}`
- `DELETE /cart/items/{itemID}`
- `POST /checkout/quote`
- `POST /checkout/place-order`
- `GET /payments/settings`
- `POST /payments/stripe/intent`
- `POST /payments/cod/confirm`
- `GET /orders/{orderID}`
- `POST /orders/{orderID}/refund-requests`
- `GET /invoices/{orderID}/download`

## Vendor
- `GET /vendor/products`
- `POST /vendor/products`
- `PATCH /vendor/products/{productID}`
- `DELETE /vendor/products/{productID}`
- `POST /vendor/products/{productID}/submit-moderation`
- `GET /vendor/coupons`
- `POST /vendor/coupons`
- `PATCH /vendor/coupons/{couponID}`
- `DELETE /vendor/coupons/{couponID}`
- `GET /vendor/shipments`
- `GET /vendor/shipments/{shipmentID}`
- `PATCH /vendor/shipments/{shipmentID}/status`
- `GET /vendor/refund-requests`
- `PATCH /vendor/refund-requests/{refundRequestID}/decision`
- `GET /vendor/analytics/overview`
- `GET /vendor/analytics/top-products`
- `GET /vendor/analytics/coupons`

## Admin
- `GET /admin/vendors`
- `PATCH /admin/vendors/{vendorID}/verification`
- `PATCH /admin/vendors/{vendorID}/commission`
- `GET /admin/moderation/products`
- `PATCH /admin/moderation/products/{productID}`
- `GET /admin/orders`
- `GET /admin/orders/{orderID}`
- `PATCH /admin/orders/{orderID}/status`
- `GET /admin/promotions`
- `POST /admin/promotions`
- `PATCH /admin/promotions/{promotionID}`
- `DELETE /admin/promotions/{promotionID}`
- `GET /admin/audit-logs`
- `GET /admin/dashboard/overview`
- `GET /admin/analytics/revenue`
- `GET /admin/analytics/vendors`
- `GET /admin/settings/payments`
- `PATCH /admin/settings/payments`

## Webhooks
- `POST /webhooks/stripe`

## Cross-cutting behavior
- Protected endpoints require bearer auth.
- RBAC is validated server-side.
- Pagination uses `limit` + `offset` with bounded values.
- Mutating payment/checkout operations require idempotency keys.
