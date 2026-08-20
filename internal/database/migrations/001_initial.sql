-- 001_initial.sql
CREATE TABLE IF NOT EXISTS categories (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS products (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    barcode        TEXT UNIQUE,
    sku            TEXT UNIQUE,
    name           TEXT NOT NULL,
    category_id    INTEGER REFERENCES categories(id),
    purchase_price INTEGER NOT NULL DEFAULT 0,
    selling_price  INTEGER NOT NULL DEFAULT 0,
    stock          INTEGER NOT NULL DEFAULT 0,
    minimum_stock  INTEGER NOT NULL DEFAULT 0,
    unit           TEXT NOT NULL DEFAULT 'pcs',
    created_at     TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at     TEXT NOT NULL DEFAULT (datetime('now')),
    deleted_at     TEXT
);

CREATE TABLE IF NOT EXISTS transactions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    invoice_number TEXT NOT NULL UNIQUE,
    subtotal       INTEGER NOT NULL,
    discount       INTEGER NOT NULL DEFAULT 0,
    total          INTEGER NOT NULL,
    paid           INTEGER NOT NULL,
    change         INTEGER NOT NULL,
    payment_method TEXT NOT NULL,
    cashier        TEXT NOT NULL DEFAULT '',
    created_at     TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS transaction_items (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    transaction_id INTEGER NOT NULL REFERENCES transactions(id),
    product_id     INTEGER REFERENCES products(id),
    product_name   TEXT NOT NULL,
    quantity       INTEGER NOT NULL,
    price          INTEGER NOT NULL,
    subtotal       INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS stock_movements (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id   INTEGER NOT NULL REFERENCES products(id),
    type         TEXT NOT NULL,
    quantity     INTEGER NOT NULL,
    reference_id INTEGER,
    note         TEXT,
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS settings (
    key        TEXT PRIMARY KEY,
    value      TEXT,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT OR IGNORE INTO settings (key, value) VALUES
    ('store_name', 'TOKO SEMBAKO'),
    ('store_address', ''),
    ('store_phone', ''),
    ('currency', 'IDR'),
    ('receipt_footer', 'Terima kasih');

CREATE INDEX IF NOT EXISTS idx_products_name    ON products(name);
CREATE INDEX IF NOT EXISTS idx_products_barcode ON products(barcode);
CREATE INDEX IF NOT EXISTS idx_products_sku     ON products(sku);
CREATE INDEX IF NOT EXISTS idx_items_transaction ON transaction_items(transaction_id);
CREATE INDEX IF NOT EXISTS idx_movements_product ON stock_movements(product_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created ON transactions(created_at);