CREATE TABLE IF NOT EXISTS merch_settings (
    organization_id INTEGER PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    config TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS merch_products (
    id TEXT PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    short_description TEXT NOT NULL,
    description TEXT NOT NULL,
    category TEXT NOT NULL,
    price_cents INTEGER NOT NULL CHECK (price_cents > 0),
    promo_price_cents INTEGER CHECK (promo_price_cents > 0 AND promo_price_cents < price_cents),
    sku TEXT NOT NULL,
    stock INTEGER NOT NULL CHECK (stock >= 0),
    max_quantity INTEGER NOT NULL CHECK (max_quantity BETWEEN 1 AND 99),
    published INTEGER NOT NULL CHECK (published IN (0,1)),
    position INTEGER NOT NULL,
    featured INTEGER NOT NULL CHECK (featured IN (0,1)),
    on_demand INTEGER NOT NULL CHECK (on_demand IN (0,1)),
    sale_mode TEXT NOT NULL CHECK (sale_mode IN ('inherit','online','offline','both')),
    digital_asset_id TEXT NOT NULL,
    shipping_allowed INTEGER NOT NULL CHECK (shipping_allowed IN (0,1)),
    archived INTEGER NOT NULL DEFAULT 0 CHECK (archived IN (0,1)),
    version INTEGER NOT NULL DEFAULT 1,
    UNIQUE(organization_id, id)
);
CREATE INDEX IF NOT EXISTS merch_catalog_order ON merch_products(organization_id, archived, published, featured DESC, position, name);
CREATE TABLE IF NOT EXISTS merch_product_variants (
    id TEXT PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL,
    size TEXT NOT NULL,
    color TEXT NOT NULL,
    sku TEXT NOT NULL,
    price_cents INTEGER CHECK (price_cents > 0),
    stock INTEGER NOT NULL CHECK (stock >= 0),
    on_demand INTEGER NOT NULL CHECK (on_demand IN (0,1)),
    position INTEGER NOT NULL,
    active INTEGER NOT NULL CHECK (active IN (0,1)),
    UNIQUE(organization_id, product_id, id),
    FOREIGN KEY (organization_id, product_id) REFERENCES merch_products(organization_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS merch_variant_product ON merch_product_variants(organization_id, product_id, active, position);
CREATE TABLE IF NOT EXISTS merch_media (
    id TEXT PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    mime TEXT NOT NULL,
    content BLOB NOT NULL,
    created_at INTEGER NOT NULL,
    UNIQUE(organization_id, id)
);
CREATE TABLE IF NOT EXISTS merch_product_media (
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL,
    media_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    PRIMARY KEY(organization_id, product_id, media_id),
    FOREIGN KEY (organization_id, product_id) REFERENCES merch_products(organization_id, id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id, media_id) REFERENCES merch_media(organization_id, id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS merch_orders (
    id TEXT PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    request_id TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    code TEXT NOT NULL,
    fan_id INTEGER REFERENCES fan_profiles(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT NOT NULL,
    address TEXT NOT NULL,
    notes TEXT NOT NULL,
    mode TEXT NOT NULL CHECK (mode IN ('online','offline')),
    payment_status TEXT NOT NULL CHECK (payment_status IN ('PENDING_ONLINE','UNPAID_OFFLINE','PAID','PAID_OFFLINE','FAILED','EXPIRED','REFUNDED')),
    fulfillment_status TEXT NOT NULL CHECK (fulfillment_status IN ('PENDING','CONFIRMED','PREPARING','READY_FOR_PICKUP','COMPLETED','CANCELLED','EXPIRED','REVIEW_REQUIRED')),
    fulfillment TEXT NOT NULL CHECK (fulfillment IN ('pickup','shipping')),
    stock_status TEXT NOT NULL CHECK (stock_status IN ('HELD','DEFERRED','CONSUMED','RELEASED')),
    subtotal_cents INTEGER NOT NULL CHECK (subtotal_cents > 0),
    shipping_cents INTEGER NOT NULL CHECK (shipping_cents >= 0),
    total_cents INTEGER NOT NULL CHECK (total_cents = subtotal_cents + shipping_cents),
    currency TEXT NOT NULL,
    instructions TEXT NOT NULL,
    stripe_account TEXT NOT NULL DEFAULT '',
    stripe_session TEXT NOT NULL DEFAULT '',
    stripe_payment_intent TEXT NOT NULL DEFAULT '',
    stripe_url TEXT NOT NULL DEFAULT '',
    stripe_awaiting INTEGER NOT NULL DEFAULT 0 CHECK (stripe_awaiting IN (0,1)),
    stripe_checked_at INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    UNIQUE(organization_id, id),
    UNIQUE(organization_id, request_id),
    UNIQUE(organization_id, code)
);
CREATE INDEX IF NOT EXISTS merch_order_fan ON merch_orders(organization_id, fan_id, created_at DESC);
CREATE INDEX IF NOT EXISTS merch_order_org ON merch_orders(organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS merch_order_expiry ON merch_orders(expires_at) WHERE payment_status IN ('PENDING_ONLINE','UNPAID_OFFLINE') AND fulfillment_status NOT IN ('CANCELLED','EXPIRED');
CREATE UNIQUE INDEX IF NOT EXISTS merch_stripe_session ON merch_orders(stripe_account, stripe_session) WHERE stripe_session <> '';
CREATE UNIQUE INDEX IF NOT EXISTS merch_stripe_intent ON merch_orders(stripe_account, stripe_payment_intent) WHERE stripe_payment_intent <> '';
CREATE TABLE IF NOT EXISTS merch_order_items (
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    order_id TEXT NOT NULL,
    line_no INTEGER NOT NULL,
    product_id TEXT NOT NULL,
    variant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    variant_name TEXT NOT NULL,
    sku TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity BETWEEN 1 AND 99),
    unit_cents INTEGER NOT NULL CHECK (unit_cents > 0),
    tracked INTEGER NOT NULL CHECK (tracked IN (0,1)),
    PRIMARY KEY(organization_id, order_id, line_no),
    FOREIGN KEY (organization_id, order_id) REFERENCES merch_orders(organization_id, id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id, product_id) REFERENCES merch_products(organization_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS merch_item_inventory ON merch_order_items(organization_id, product_id, variant_id, tracked);
CREATE TABLE IF NOT EXISTS merch_inventory_movements (
    id INTEGER PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    order_id TEXT NOT NULL,
    product_id TEXT NOT NULL,
    variant_id TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('HELD','CONSUMED','RELEASED')),
    created_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id, order_id) REFERENCES merch_orders(organization_id, id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS merch_payment_records (
    id INTEGER PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    order_id TEXT NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('stripe','offline')),
    status TEXT NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    method TEXT NOT NULL,
    reference TEXT NOT NULL,
    actor TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id, order_id) REFERENCES merch_orders(organization_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS merch_payment_order ON merch_payment_records(organization_id, order_id, id);
CREATE TABLE IF NOT EXISTS merch_order_events (
    id INTEGER PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    order_id TEXT NOT NULL,
    type TEXT NOT NULL,
    actor TEXT NOT NULL,
    notes TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id, order_id) REFERENCES merch_orders(organization_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS merch_event_order ON merch_order_events(organization_id, order_id, id);
CREATE TABLE IF NOT EXISTS merch_stripe_events (
    event_id TEXT PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    order_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id, order_id) REFERENCES merch_orders(organization_id, id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS merch_analytics_events (
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event_id TEXT NOT NULL,
    type TEXT NOT NULL,
    product_id TEXT NOT NULL DEFAULT '',
    variant_id TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    PRIMARY KEY(organization_id, event_id)
);
CREATE INDEX IF NOT EXISTS merch_analytics_type ON merch_analytics_events(organization_id, type, product_id);
CREATE TABLE IF NOT EXISTS merch_rate_limits (
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    actor_hash TEXT NOT NULL,
    time_bucket INTEGER NOT NULL,
    count INTEGER NOT NULL,
    PRIMARY KEY(organization_id, action, actor_hash, time_bucket)
);
CREATE INDEX IF NOT EXISTS merch_rate_expiry ON merch_rate_limits(time_bucket);
