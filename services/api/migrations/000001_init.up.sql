CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE user_type AS ENUM ('buyer', 'vendor_owner', 'admin');
CREATE TYPE user_status AS ENUM ('active', 'suspended', 'deleted');
CREATE TYPE vendor_verification_state AS ENUM ('pending', 'verified', 'rejected', 'suspended');
CREATE TYPE product_status AS ENUM ('draft', 'pending_approval', 'approved', 'rejected', 'archived');
CREATE TYPE shipment_status AS ENUM ('pending', 'processing', 'packed', 'shipped', 'in_transit', 'delivered', 'returned', 'cancelled');
CREATE TYPE payment_method AS ENUM ('stripe', 'cod');
CREATE TYPE payment_status AS ENUM ('pending', 'authorized', 'succeeded', 'failed', 'cancelled', 'refunded', 'partially_refunded');
CREATE TYPE refund_status AS ENUM ('requested', 'approved', 'rejected', 'processing', 'succeeded', 'failed');
CREATE TYPE return_status AS ENUM ('requested', 'approved', 'rejected', 'received', 'refunded');
CREATE TYPE review_status AS ENUM ('published', 'hidden', 'flagged');
CREATE TYPE wallet_entry_type AS ENUM ('credit', 'debit', 'adjustment');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    user_type user_type NOT NULL DEFAULT 'buyer',
    status user_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX users_email_unique_idx ON users ((lower(email)));

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    ip INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE admin_roles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('super_admin', 'support', 'finance', 'catalog_moderator')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vendors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE RESTRICT,
    slug TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    verification_state vendor_verification_state NOT NULL DEFAULT 'pending',
    commission_override_bps INTEGER CHECK (commission_override_bps BETWEEN 0 AND 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX vendors_verification_state_idx ON vendors (verification_state);

CREATE TABLE vendor_verification_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    actor_admin_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    from_state vendor_verification_state NOT NULL,
    to_state vendor_verification_state NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX vendor_verification_reviews_vendor_id_idx ON vendor_verification_reviews (vendor_id, created_at DESC);

CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    parent_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE RESTRICT,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    price_incl_tax_cents BIGINT NOT NULL CHECK (price_incl_tax_cents >= 0),
    currency CHAR(3) NOT NULL,
    stock_qty INTEGER NOT NULL DEFAULT 0 CHECK (stock_qty >= 0),
    status product_status NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX products_vendor_status_idx ON products (vendor_id, status);
CREATE INDEX products_category_idx ON products (category_id);
CREATE INDEX products_created_at_idx ON products (created_at DESC);

CREATE TABLE product_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    storage_key TEXT NOT NULL,
    url TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, sort_order)
);

CREATE TABLE product_moderation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    state product_status NOT NULL CHECK (state IN ('pending_approval', 'approved', 'rejected')),
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX product_moderation_product_id_idx ON product_moderation (product_id, created_at DESC);

CREATE TABLE carts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    guest_token TEXT,
    currency CHAR(3) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (buyer_user_id IS NOT NULL OR guest_token IS NOT NULL)
);
CREATE UNIQUE INDEX carts_guest_token_unique_idx ON carts (guest_token) WHERE guest_token IS NOT NULL;
CREATE UNIQUE INDEX carts_buyer_user_id_unique_idx ON carts (buyer_user_id) WHERE buyer_user_id IS NOT NULL;

CREATE TABLE cart_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cart_id UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE RESTRICT,
    qty INTEGER NOT NULL CHECK (qty > 0),
    unit_price_snapshot_cents BIGINT NOT NULL CHECK (unit_price_snapshot_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (cart_id, product_id)
);
CREATE INDEX cart_items_vendor_id_idx ON cart_items (vendor_id);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    guest_email TEXT,
    status TEXT NOT NULL,
    subtotal_cents BIGINT NOT NULL CHECK (subtotal_cents >= 0),
    discount_cents BIGINT NOT NULL DEFAULT 0 CHECK (discount_cents >= 0),
    tax_cents BIGINT NOT NULL DEFAULT 0 CHECK (tax_cents >= 0),
    shipping_cents BIGINT NOT NULL DEFAULT 0 CHECK (shipping_cents >= 0),
    total_cents BIGINT NOT NULL CHECK (total_cents >= 0),
    currency CHAR(3) NOT NULL,
    idempotency_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (idempotency_key)
);
CREATE INDEX orders_buyer_user_id_idx ON orders (buyer_user_id, created_at DESC);

CREATE TABLE shipments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE RESTRICT,
    status shipment_status NOT NULL DEFAULT 'pending',
    shipping_fee_cents BIGINT NOT NULL DEFAULT 0 CHECK (shipping_fee_cents >= 0),
    tracking_ref TEXT,
    shipped_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (order_id, vendor_id)
);
CREATE INDEX shipments_vendor_id_idx ON shipments (vendor_id, status, created_at DESC);

CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    shipment_id UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE RESTRICT,
    qty INTEGER NOT NULL CHECK (qty > 0),
    unit_price_cents BIGINT NOT NULL CHECK (unit_price_cents >= 0),
    tax_cents BIGINT NOT NULL DEFAULT 0 CHECK (tax_cents >= 0),
    discount_cents BIGINT NOT NULL DEFAULT 0 CHECK (discount_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX order_items_order_id_idx ON order_items (order_id);
CREATE INDEX order_items_shipment_id_idx ON order_items (shipment_id);

CREATE TABLE shipment_status_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shipment_id UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    status shipment_status NOT NULL,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX shipment_status_events_shipment_id_idx ON shipment_status_events (shipment_id, at DESC);

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    method payment_method NOT NULL,
    status payment_status NOT NULL,
    provider TEXT NOT NULL,
    provider_ref TEXT,
    amount_cents BIGINT NOT NULL CHECK (amount_cents >= 0),
    currency CHAR(3) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX payments_order_id_idx ON payments (order_id);
CREATE INDEX payments_provider_ref_idx ON payments (provider, provider_ref);

CREATE TABLE payment_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    payload_json JSONB NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider_event_id)
);
CREATE INDEX payment_events_payment_id_idx ON payment_events (payment_id, processed_at DESC);

CREATE TABLE vendor_coupons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    discount_type TEXT NOT NULL CHECK (discount_type IN ('percentage', 'fixed')),
    discount_value BIGINT NOT NULL CHECK (discount_value >= 0),
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    usage_limit INTEGER,
    used_count INTEGER NOT NULL DEFAULT 0 CHECK (used_count >= 0),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (vendor_id, code)
);
CREATE INDEX vendor_coupons_active_idx ON vendor_coupons (vendor_id, active, starts_at, ends_at);

CREATE TABLE admin_promotions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    rule_json JSONB NOT NULL,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    stackable BOOLEAN NOT NULL DEFAULT FALSE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX admin_promotions_active_idx ON admin_promotions (active, starts_at, ends_at);

CREATE TABLE applied_discounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    shipment_id UUID REFERENCES shipments(id) ON DELETE CASCADE,
    source_type TEXT NOT NULL CHECK (source_type IN ('vendor_coupon', 'admin_promotion')),
    source_id UUID NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX applied_discounts_order_id_idx ON applied_discounts (order_id);

CREATE TABLE refund_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    shipment_id UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    buyer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT NOT NULL,
    status refund_status NOT NULL DEFAULT 'requested',
    requested_amount_cents BIGINT NOT NULL CHECK (requested_amount_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX refund_requests_shipment_id_idx ON refund_requests (shipment_id, status);

CREATE TABLE refunds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    refund_request_id UUID NOT NULL UNIQUE REFERENCES refund_requests(id) ON DELETE CASCADE,
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE RESTRICT,
    decision TEXT NOT NULL CHECK (decision IN ('approved', 'rejected')),
    amount_cents BIGINT NOT NULL CHECK (amount_cents >= 0),
    status refund_status NOT NULL,
    provider_ref TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE return_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_item_id UUID NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    buyer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT NOT NULL,
    status return_status NOT NULL DEFAULT 'requested',
    window_expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX return_requests_order_item_id_idx ON return_requests (order_item_id);

CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    buyer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_item_id UUID NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    rating SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    body TEXT NOT NULL,
    status review_status NOT NULL DEFAULT 'published',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (order_item_id, buyer_id)
);
CREATE INDEX reviews_product_id_idx ON reviews (product_id, status, created_at DESC);

CREATE TABLE wallet_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_type wallet_entry_type NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    currency CHAR(3) NOT NULL,
    ref_type TEXT NOT NULL,
    ref_id UUID,
    balance_after_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX wallet_ledger_buyer_id_idx ON wallet_ledger (buyer_id, created_at DESC);

CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    invoice_number TEXT NOT NULL UNIQUE,
    issued_at TIMESTAMPTZ NOT NULL,
    pdf_storage_key TEXT NOT NULL,
    total_cents BIGINT NOT NULL CHECK (total_cents >= 0),
    currency CHAR(3) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE invoice_sequences (
    sequence_date DATE PRIMARY KEY,
    last_value BIGINT NOT NULL DEFAULT 0 CHECK (last_value >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_type TEXT NOT NULL CHECK (actor_type IN ('admin', 'vendor', 'system')),
    actor_id UUID,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id UUID,
    before_json JSONB,
    after_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX audit_logs_actor_idx ON audit_logs (actor_type, actor_id, created_at DESC);
CREATE INDEX audit_logs_target_idx ON audit_logs (target_type, target_id, created_at DESC);

CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_name TEXT NOT NULL,
    actor_id UUID,
    session_id UUID,
    entity_type TEXT,
    entity_id UUID,
    props_json JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX events_name_created_at_idx ON events (event_name, created_at DESC);
CREATE INDEX events_actor_created_at_idx ON events (actor_id, created_at DESC);
