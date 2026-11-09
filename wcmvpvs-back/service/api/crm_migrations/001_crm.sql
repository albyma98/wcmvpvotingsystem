CREATE TABLE IF NOT EXISTS crm_companies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    import_key TEXT UNIQUE,
    abx_org_id INTEGER UNIQUE REFERENCES organizations(id) ON DELETE SET NULL,
    owner_id INTEGER REFERENCES admins(id) ON DELETE SET NULL,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    details TEXT NOT NULL CHECK (json_valid(details)),
    archived INTEGER NOT NULL DEFAULT 0 CHECK (archived IN (0,1)),
    revision INTEGER NOT NULL DEFAULT 1,
    updated_at TEXT NOT NULL
);

-- Stable keys cover imports, checklist items and generated renewals. The
-- composite foreign keys prevent relationships across CRM companies.
CREATE TABLE IF NOT EXISTS crm_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id INTEGER NOT NULL REFERENCES crm_companies(id),
    kind TEXT NOT NULL CHECK (kind IN ('contact','opportunity','activity','task','pilot','checklist','issue','goal','metric','contract','payment','history')),
    stable_key TEXT,
    contact_id INTEGER,
    opportunity_id INTEGER,
    contract_id INTEGER,
    owner_id INTEGER REFERENCES admins(id) ON DELETE SET NULL,
    details TEXT NOT NULL CHECK (json_valid(details)),
    archived INTEGER NOT NULL DEFAULT 0 CHECK (archived IN (0,1)),
    revision INTEGER NOT NULL DEFAULT 1,
    author_id INTEGER REFERENCES admins(id) ON DELETE SET NULL,
    author_name TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (company_id, id),
    UNIQUE (company_id, kind, stable_key),
    FOREIGN KEY (company_id, contact_id) REFERENCES crm_records(company_id, id),
    FOREIGN KEY (company_id, opportunity_id) REFERENCES crm_records(company_id, id),
    FOREIGN KEY (company_id, contract_id) REFERENCES crm_records(company_id, id)
);

CREATE INDEX IF NOT EXISTS crm_records_kind ON crm_records(kind, archived);
