-- Sprint 25: Lịch rảnh HLV, đơn nghỉ & HLV thay thế, lịch ngày lễ (000024_availability_holidays)

CREATE TABLE IF NOT EXISTS trainer_availabilities (
    id             BIGSERIAL   PRIMARY KEY,
    trainer_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    branch_id      BIGINT      NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    dow            INT         NOT NULL CHECK (dow BETWEEN 0 AND 6), -- 0=CN, 1=Thứ 2 ... 6=Thứ 7
    start_time     VARCHAR(5)  NOT NULL, -- HH:MM
    end_time       VARCHAR(5)  NOT NULL, -- HH:MM
    effective_from DATE,
    effective_to   DATE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trainer_availabilities_trainer ON trainer_availabilities(trainer_id, branch_id, dow);

CREATE TABLE IF NOT EXISTS staff_leave_requests (
    id          BIGSERIAL    PRIMARY KEY,
    staff_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    branch_id   BIGINT       REFERENCES branches(id) ON DELETE SET NULL,
    from_at     TIMESTAMPTZ  NOT NULL,
    to_at       TIMESTAMPTZ  NOT NULL,
    reason      VARCHAR(500) NOT NULL DEFAULT '',
    status      VARCHAR(20)  NOT NULL DEFAULT 'PENDING'
                CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),
    approved_by BIGINT       REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    review_note VARCHAR(255) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_staff_leave_requests_staff ON staff_leave_requests(staff_id, from_at DESC);

ALTER TABLE class_schedules
    ADD COLUMN IF NOT EXISTS substitute_trainer_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE pt_sessions
    ADD COLUMN IF NOT EXISTS original_trainer_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS branch_holidays (
    id          BIGSERIAL    PRIMARY KEY,
    branch_id   BIGINT       REFERENCES branches(id) ON DELETE CASCADE, -- NULL = toàn hệ thống
    date        DATE         NOT NULL,
    name        VARCHAR(150) NOT NULL,
    closed      BOOLEAN      NOT NULL DEFAULT TRUE,
    open_time   VARCHAR(5),
    close_time  VARCHAR(5),
    extend_subs BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_branch_holidays_date ON branch_holidays(date, branch_id);

INSERT INTO permissions (code, module, name) VALUES
    ('leave.manage',   'leave',   'Quản lý lịch rảnh HLV, duyệt đơn nghỉ phép & phân công dạy thay'),
    ('holiday.manage', 'holiday', 'Quản lý lịch nghỉ lễ của phòng tập')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code IN ('OWNER', 'MANAGER')
  AND p.code IN ('leave.manage', 'holiday.manage')
ON CONFLICT DO NOTHING;
