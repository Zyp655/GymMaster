-- =====================================================================
-- 000002 – Nhân sự, huấn luyện viên, phòng tập, ca làm
-- =====================================================================

-- Hồ sơ nhân viên (1-1 với users). Chi nhánh làm việc thể hiện qua user_roles.branch_id.
CREATE TABLE staff_profiles (
    user_id          BIGINT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    staff_code       VARCHAR(20)   NOT NULL,
    position         VARCHAR(30)   NOT NULL CHECK (position IN ('MANAGER', 'RECEPTIONIST', 'TRAINER', 'OTHER')),
    gender           VARCHAR(10)   NOT NULL DEFAULT 'OTHER' CHECK (gender IN ('MALE', 'FEMALE', 'OTHER')),
    dob              DATE,
    id_card          VARCHAR(20)   NOT NULL DEFAULT '',
    address          VARCHAR(255)  NOT NULL DEFAULT '',
    hire_date        DATE          NOT NULL DEFAULT CURRENT_DATE,
    base_salary      NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (base_salary >= 0),
    -- % hoa hồng trên doanh thu PT/buổi dạy (0-100)
    commission_rate  NUMERIC(5,2)  NOT NULL DEFAULT 0 CHECK (commission_rate BETWEEN 0 AND 100),
    note             TEXT          NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE SEQUENCE staff_code_seq;
ALTER TABLE staff_profiles ALTER COLUMN staff_code SET DEFAULT ('NV' || lpad(nextval('staff_code_seq')::text, 4, '0'));
CREATE UNIQUE INDEX uq_staff_code ON staff_profiles (staff_code);
CREATE TRIGGER trg_staff_profiles_updated_at BEFORE UPDATE ON staff_profiles FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Hồ sơ huấn luyện viên (chỉ cho nhân viên position = TRAINER).
CREATE TABLE trainer_profiles (
    user_id         BIGINT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    level           VARCHAR(20)  NOT NULL DEFAULT 'JUNIOR' CHECK (level IN ('JUNIOR', 'SENIOR', 'MASTER')),
    specialties     TEXT[]       NOT NULL DEFAULT '{}',
    bio             TEXT         NOT NULL DEFAULT '',
    certifications  TEXT         NOT NULL DEFAULT '',
    years_experience INT         NOT NULL DEFAULT 0 CHECK (years_experience >= 0),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_trainer_profiles_updated_at BEFORE UPDATE ON trainer_profiles FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Phòng tập (phòng lớp nhóm: yoga, spinning, ...).
CREATE TABLE rooms (
    id          BIGSERIAL PRIMARY KEY,
    branch_id   BIGINT       NOT NULL REFERENCES branches (id),
    name        VARCHAR(100) NOT NULL,
    capacity    INT          NOT NULL CHECK (capacity > 0),
    description VARCHAR(255) NOT NULL DEFAULT '',
    status      VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,
    created_by  BIGINT,
    updated_by  BIGINT
);
CREATE UNIQUE INDEX uq_rooms_branch_name ON rooms (branch_id, lower(name)) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_rooms_updated_at BEFORE UPDATE ON rooms FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Ca làm việc. EXCLUDE: một nhân viên không thể có 2 ca chồng giờ nhau.
CREATE TABLE shifts (
    id          BIGSERIAL PRIMARY KEY,
    branch_id   BIGINT       NOT NULL REFERENCES branches (id),
    staff_id    BIGINT       NOT NULL REFERENCES users (id),
    start_at    TIMESTAMPTZ  NOT NULL,
    end_at      TIMESTAMPTZ  NOT NULL,
    note        VARCHAR(255) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by  BIGINT,
    updated_by  BIGINT,
    CONSTRAINT chk_shifts_time CHECK (end_at > start_at),
    CONSTRAINT ex_shifts_staff_overlap EXCLUDE USING gist (staff_id WITH =, tstzrange(start_at, end_at) WITH &&)
);
CREATE INDEX idx_shifts_branch_time ON shifts (branch_id, start_at);
CREATE TRIGGER trg_shifts_updated_at BEFORE UPDATE ON shifts FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO permissions (code, module, name) VALUES
    ('staff.view',   'staff', 'Xem nhân viên & HLV'),
    ('staff.create', 'staff', 'Thêm nhân viên'),
    ('staff.update', 'staff', 'Sửa nhân viên, khóa/mở tài khoản, gán vai trò'),
    ('staff.delete', 'staff', 'Xóa nhân viên'),
    ('room.view',    'room',  'Xem phòng tập'),
    ('room.manage',  'room',  'Thêm/sửa/xóa phòng tập'),
    ('shift.view',   'shift', 'Xem lịch ca làm'),
    ('shift.manage', 'shift', 'Xếp/sửa/xóa ca làm');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.module IN ('staff', 'room', 'shift') WHERE r.code = 'OWNER';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p
  ON p.code IN ('staff.view', 'staff.create', 'staff.update', 'room.view', 'room.manage', 'shift.view', 'shift.manage')
WHERE r.code = 'MANAGER';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('staff.view', 'room.view', 'shift.view')
WHERE r.code IN ('RECEPTIONIST', 'TRAINER');
