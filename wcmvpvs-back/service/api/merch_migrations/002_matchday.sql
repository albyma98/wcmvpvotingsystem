CREATE TABLE IF NOT EXISTS merch_match_coverage (
    event_id INTEGER PRIMARY KEY REFERENCES events(id) ON DELETE CASCADE,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    enabled INTEGER NOT NULL CHECK (enabled IN (0,1))
);
CREATE TABLE IF NOT EXISTS merch_match_activity (
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    analytics_id TEXT NOT NULL,
    fan_id INTEGER REFERENCES fan_profiles(id) ON DELETE SET NULL,
    device_key TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (organization_id, analytics_id),
    FOREIGN KEY (organization_id, analytics_id) REFERENCES merch_analytics_events(organization_id,event_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS merch_match_activity_event ON merch_match_activity(event_id, organization_id);
CREATE TABLE IF NOT EXISTS merch_match_orders (
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    order_id TEXT NOT NULL,
    device_key TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (organization_id, order_id),
    FOREIGN KEY (organization_id,order_id) REFERENCES merch_orders(organization_id,id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS merch_match_orders_event ON merch_match_orders(event_id, organization_id);
