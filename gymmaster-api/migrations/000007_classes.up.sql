-- =====================================================================
-- 000007 – Lớp nhóm: loại lớp, lịch lớp, đặt chỗ, waitlist
-- =====================================================================

CREATE TABLE class_types (
    id                    BIGSERIAL PRIMARY KEY,
    code                  VARCHAR(30)  NOT NULL,
    name                  VARCHAR(100) NOT NULL,
    description           TEXT         NOT NULL DEFAULT '',
    color                 VARCHAR(7)   NOT NULL DEFAULT '#F97316' CHECK (color ~ '^#[0-9A-Fa-f]{6}$'),
    default_duration_min  INT          NOT NULL DEFAULT 60 CHECK (default_duration_min BETWEEN 15 AND 240),
    default_capacity      INT          NOT NULL DEFAULT 20 CHECK (default_capacity > 0),
    level                 VARCHAR(20)  NOT NULL DEFAULT 'ALL' CHECK (level IN ('ALL', 'BEGINNER', 'INTERMEDIATE', 'ADVANCED')),
    status                VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at            TIMESTAMPTZ,
    created_by            BIGINT,
    updated_by            BIGINT
);
CREATE UNIQUE INDEX uq_class_types_code ON class_types (code) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_class_types_updated_at BEFORE UPDATE ON class_types FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Một buổi lớp cụ thể. series_id gom các buổi được tạo cùng lúc theo lịch lặp tuần.
-- EXCLUDE: không trùng phòng và không trùng HLV trong cùng khoảng thời gian (trừ buổi đã hủy).
CREATE TABLE class_schedules (
    id             BIGSERIAL PRIMARY KEY,
    class_type_id  BIGINT       NOT NULL REFERENCES class_types (id),
    branch_id      BIGINT       NOT NULL REFERENCES branches (id),
    room_id        BIGINT       NOT NULL REFERENCES rooms (id),
    trainer_id     BIGINT       NOT NULL REFERENCES users (id),
    start_at       TIMESTAMPTZ  NOT NULL,
    end_at         TIMESTAMPTZ  NOT NULL,
    capacity       INT          NOT NULL CHECK (capacity > 0),
    booked_count   INT          NOT NULL DEFAULT 0 CHECK (booked_count >= 0),
    status         VARCHAR(20)  NOT NULL DEFAULT 'SCHEDULED' CHECK (status IN ('SCHEDULED', 'CANCELLED', 'COMPLETED')),
    series_id      UUID,
    cancel_reason  VARCHAR(255) NOT NULL DEFAULT '',
    note           VARCHAR(255) NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by     BIGINT,
    updated_by     BIGINT,
    CONSTRAINT chk_class_schedules_time CHECK (end_at > start_at),
    CONSTRAINT chk_class_schedules_booked CHECK (booked_count <= capacity),
    CONSTRAINT ex_class_room_overlap EXCLUDE USING gist (room_id WITH =, tstzrange(start_at, end_at) WITH &&)
        WHERE (status <> 'CANCELLED'),
    CONSTRAINT ex_class_trainer_overlap EXCLUDE USING gist (trainer_id WITH =, tstzrange(start_at, end_at) WITH &&)
        WHERE (status <> 'CANCELLED')
);
CREATE INDEX idx_class_schedules_branch_time ON class_schedules (branch_id, start_at);
CREATE INDEX idx_class_schedules_trainer_time ON class_schedules (trainer_id, start_at);
CREATE INDEX idx_class_schedules_series ON class_schedules (series_id);
CREATE TRIGGER trg_class_schedules_updated_at BEFORE UPDATE ON class_schedules FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Đặt chỗ. booked_count trên class_schedules được cập nhật trong cùng transaction
-- sau khi SELECT ... FOR UPDATE dòng lịch lớp (chống vượt sức chứa khi đặt đồng thời).
CREATE TABLE class_bookings (
    id                 BIGSERIAL PRIMARY KEY,
    schedule_id        BIGINT       NOT NULL REFERENCES class_schedules (id),
    member_id          BIGINT       NOT NULL REFERENCES members (id),
    subscription_id    BIGINT       REFERENCES subscriptions (id),
    status             VARCHAR(20)  NOT NULL CHECK (status IN ('BOOKED', 'WAITLIST', 'CANCELLED', 'ATTENDED', 'NO_SHOW')),
    waitlist_position  INT,
    booked_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    cancelled_at       TIMESTAMPTZ,
    attended_at        TIMESTAMPTZ,
    source             VARCHAR(10)  NOT NULL DEFAULT 'COUNTER' CHECK (source IN ('COUNTER', 'PORTAL')),
    created_by         BIGINT       REFERENCES users (id),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_class_bookings_active ON class_bookings (schedule_id, member_id)
    WHERE status IN ('BOOKED', 'WAITLIST', 'ATTENDED');
CREATE INDEX idx_class_bookings_member ON class_bookings (member_id, booked_at DESC);
CREATE INDEX idx_class_bookings_schedule ON class_bookings (schedule_id, status);
CREATE TRIGGER trg_class_bookings_updated_at BEFORE UPDATE ON class_bookings FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO permissions (code, module, name) VALUES
    ('class.view',     'class', 'Xem loại lớp, lịch lớp'),
    ('class.manage',   'class', 'Thêm/sửa/hủy loại lớp, lịch lớp'),
    ('booking.manage', 'class', 'Đặt chỗ hộ hội viên, điểm danh');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.module = 'class' WHERE r.code IN ('OWNER', 'MANAGER');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('class.view', 'booking.manage')
WHERE r.code IN ('RECEPTIONIST', 'TRAINER');
