-- =====================================================================
-- 000001 – Định danh, phân quyền (RBAC) và chi nhánh
-- =====================================================================

-- btree_gist dùng cho EXCLUDE constraint chống trùng lịch ở các migration sau.
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Tự cập nhật updated_at khi UPDATE (phòng trường hợp cập nhật ngoài GORM).
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------
-- Chi nhánh
-- ---------------------------------------------------------------------
CREATE TABLE branches (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(20)  NOT NULL,
    name        VARCHAR(150) NOT NULL,
    address     VARCHAR(255) NOT NULL DEFAULT '',
    phone       VARCHAR(20)  NOT NULL DEFAULT '',
    email       VARCHAR(150) NOT NULL DEFAULT '',
    open_time   VARCHAR(5)   NOT NULL DEFAULT '05:00' CHECK (open_time  ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    close_time  VARCHAR(5)   NOT NULL DEFAULT '22:00' CHECK (close_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    status      VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,
    created_by  BIGINT,
    updated_by  BIGINT,
    CONSTRAINT chk_branches_hours CHECK (close_time > open_time)
);
-- Mã chỉ cần duy nhất trong các bản ghi chưa bị xóa mềm.
CREATE UNIQUE INDEX uq_branches_code ON branches (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_branches_deleted_at ON branches (deleted_at);
CREATE TRIGGER trg_branches_updated_at BEFORE UPDATE ON branches FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------
-- Người dùng (nhân viên + hội viên có tài khoản)
-- ---------------------------------------------------------------------
CREATE TABLE users (
    id             BIGSERIAL PRIMARY KEY,
    email          VARCHAR(150) NOT NULL,
    phone          VARCHAR(20)  NOT NULL DEFAULT '',
    password_hash  VARCHAR(100) NOT NULL,
    full_name      VARCHAR(150) NOT NULL,
    avatar_url     VARCHAR(500) NOT NULL DEFAULT '',
    status         VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'LOCKED')),
    last_login_at  TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    created_by     BIGINT,
    updated_by     BIGINT
);
CREATE UNIQUE INDEX uq_users_email ON users (lower(email)) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------
-- RBAC: roles – permissions – role_permissions – user_roles
-- ---------------------------------------------------------------------
CREATE TABLE roles (
    id           BIGSERIAL PRIMARY KEY,
    code         VARCHAR(50)  NOT NULL,
    name         VARCHAR(100) NOT NULL,
    description  VARCHAR(255) NOT NULL DEFAULT '',
    is_system    BOOLEAN      NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,
    created_by   BIGINT,
    updated_by   BIGINT
);
CREATE UNIQUE INDEX uq_roles_code ON roles (code) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_roles_updated_at BEFORE UPDATE ON roles FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE permissions (
    id      BIGSERIAL PRIMARY KEY,
    code    VARCHAR(100) NOT NULL UNIQUE,
    module  VARCHAR(50)  NOT NULL,
    name    VARCHAR(150) NOT NULL
);

CREATE TABLE role_permissions (
    role_id        BIGINT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    permission_id  BIGINT NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- branch_id NULL = role áp dụng trên toàn hệ thống; có giá trị = chỉ ở chi nhánh đó.
CREATE TABLE user_roles (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id     BIGINT NOT NULL REFERENCES roles (id),
    branch_id   BIGINT REFERENCES branches (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_user_roles ON user_roles (user_id, role_id, COALESCE(branch_id, 0));
CREATE INDEX idx_user_roles_branch ON user_roles (branch_id);

-- ---------------------------------------------------------------------
-- Refresh token (lưu SHA-256, xoay vòng mỗi lần refresh)
-- ---------------------------------------------------------------------
CREATE TABLE refresh_tokens (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT       NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  CHAR(64)     NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ  NOT NULL,
    revoked_at  TIMESTAMPTZ,
    user_agent  VARCHAR(255) NOT NULL DEFAULT '',
    ip          VARCHAR(64)  NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens (user_id) WHERE revoked_at IS NULL;

-- ---------------------------------------------------------------------
-- Dữ liệu danh mục: permission + role hệ thống
-- ---------------------------------------------------------------------
INSERT INTO permissions (code, module, name) VALUES
    ('branch.view',   'branch', 'Xem chi nhánh'),
    ('branch.create', 'branch', 'Thêm chi nhánh'),
    ('branch.update', 'branch', 'Sửa chi nhánh'),
    ('branch.delete', 'branch', 'Xóa chi nhánh'),
    ('role.view',     'role',   'Xem vai trò & quyền'),
    ('role.create',   'role',   'Thêm vai trò'),
    ('role.update',   'role',   'Sửa vai trò & phân quyền'),
    ('role.delete',   'role',   'Xóa vai trò');

INSERT INTO roles (code, name, description, is_system) VALUES
    ('OWNER',        'Chủ trung tâm', 'Toàn quyền trên hệ thống',                          true),
    ('MANAGER',      'Quản lý',       'Quản lý vận hành một chi nhánh',                    true),
    ('RECEPTIONIST', 'Lễ tân',        'Tiếp nhận hội viên, bán gói, thu tiền, check-in',   true),
    ('TRAINER',      'Huấn luyện viên','Dạy lớp, huấn luyện cá nhân (PT)',                 true),
    ('MEMBER',       'Hội viên',      'Tài khoản cổng hội viên',                           true);

-- OWNER: tất cả quyền. MANAGER: xem chi nhánh, xem vai trò.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p WHERE r.code = 'OWNER';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('branch.view', 'role.view')
WHERE r.code = 'MANAGER';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('branch.view')
WHERE r.code = 'RECEPTIONIST';
