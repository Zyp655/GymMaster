-- =====================================================================
-- 000009 – POS bán hàng & tồn kho theo chi nhánh
-- =====================================================================

CREATE TABLE product_categories (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    sort_order  INT          NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);
CREATE UNIQUE INDEX uq_product_categories_name ON product_categories (lower(name)) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_product_categories_updated_at BEFORE UPDATE ON product_categories FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE suppliers (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(150) NOT NULL,
    phone       VARCHAR(20)  NOT NULL DEFAULT '',
    email       VARCHAR(150) NOT NULL DEFAULT '',
    address     VARCHAR(255) NOT NULL DEFAULT '',
    note        VARCHAR(255) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);
CREATE TRIGGER trg_suppliers_updated_at BEFORE UPDATE ON suppliers FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE products (
    id            BIGSERIAL PRIMARY KEY,
    sku           VARCHAR(30)   NOT NULL,
    barcode       VARCHAR(50)   NOT NULL DEFAULT '',
    name          VARCHAR(150)  NOT NULL,
    category_id   BIGINT        REFERENCES product_categories (id),
    unit          VARCHAR(20)   NOT NULL DEFAULT 'cái',
    cost_price    NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (cost_price >= 0),   -- giá vốn bình quân
    sell_price    NUMERIC(14,2) NOT NULL CHECK (sell_price >= 0),
    member_price  NUMERIC(14,2) CHECK (member_price >= 0),                    -- giá cho hội viên (NULL = như giá bán)
    image_url     VARCHAR(500)  NOT NULL DEFAULT '',
    min_stock     INT           NOT NULL DEFAULT 0 CHECK (min_stock >= 0),    -- cảnh báo tồn thấp
    status        VARCHAR(20)   NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ,
    created_by    BIGINT,
    updated_by    BIGINT
);
CREATE UNIQUE INDEX uq_products_sku ON products (sku) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_barcode ON products (barcode) WHERE barcode <> '';
CREATE TRIGGER trg_products_updated_at BEFORE UPDATE ON products FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Tồn kho hiện tại. CHECK quantity >= 0 là chốt chặn cuối cùng chống bán âm kho
-- (service khóa dòng bằng SELECT ... FOR UPDATE trước khi trừ).
CREATE TABLE stock_levels (
    product_id  BIGINT      NOT NULL REFERENCES products (id),
    branch_id   BIGINT      NOT NULL REFERENCES branches (id),
    quantity    INT         NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (product_id, branch_id)
);

-- Thẻ kho: mọi biến động tồn đều ghi một dòng (quantity có dấu: + nhập, - xuất).
CREATE TABLE stock_transactions (
    id              BIGSERIAL PRIMARY KEY,
    product_id      BIGINT        NOT NULL REFERENCES products (id),
    branch_id       BIGINT        NOT NULL REFERENCES branches (id),
    type            VARCHAR(20)   NOT NULL CHECK (type IN ('IMPORT', 'SALE', 'SALE_RETURN', 'TRANSFER_IN', 'TRANSFER_OUT', 'ADJUST')),
    quantity        INT           NOT NULL CHECK (quantity <> 0),
    balance_after   INT           NOT NULL CHECK (balance_after >= 0),
    unit_cost       NUMERIC(14,2) NOT NULL DEFAULT 0,
    reference_type  VARCHAR(20)   NOT NULL DEFAULT '',   -- INVOICE | TRANSFER | ...
    reference_id    BIGINT,
    supplier_id     BIGINT        REFERENCES suppliers (id),
    -- Mã phiếu nhóm nhiều dòng của cùng một lần nhập/điều chuyển/kiểm kê
    voucher_no      VARCHAR(30)   NOT NULL DEFAULT '',
    note            VARCHAR(255)  NOT NULL DEFAULT '',
    created_by      BIGINT        REFERENCES users (id),
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX idx_stock_transactions_product ON stock_transactions (product_id, branch_id, created_at DESC);
CREATE INDEX idx_stock_transactions_voucher ON stock_transactions (voucher_no) WHERE voucher_no <> '';

INSERT INTO permissions (code, module, name) VALUES
    ('product.view',       'pos', 'Xem sản phẩm'),
    ('product.manage',     'pos', 'Thêm/sửa/xóa sản phẩm, danh mục, nhà cung cấp'),
    ('inventory.view',     'pos', 'Xem tồn kho, thẻ kho'),
    ('inventory.import',   'pos', 'Nhập hàng'),
    ('inventory.transfer', 'pos', 'Điều chuyển hàng giữa chi nhánh'),
    ('inventory.adjust',   'pos', 'Kiểm kê, điều chỉnh tồn'),
    ('pos.sell',           'pos', 'Bán hàng tại quầy');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.module = 'pos' WHERE r.code IN ('OWNER', 'MANAGER');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('product.view', 'inventory.view', 'pos.sell')
WHERE r.code = 'RECEPTIONIST';
