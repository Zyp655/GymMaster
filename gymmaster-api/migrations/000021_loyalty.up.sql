-- Sprint 22: Điểm thưởng, huy hiệu, thử thách, giới thiệu bạn bè (000021_loyalty)

ALTER TABLE members
    ADD COLUMN IF NOT EXISTS loyalty_points INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS referral_code VARCHAR(30) UNIQUE,
    ADD COLUMN IF NOT EXISTS referred_by BIGINT REFERENCES members(id) ON DELETE SET NULL;

UPDATE members
SET referral_code = 'REF-' || member_code
WHERE referral_code IS NULL AND member_code IS NOT NULL AND member_code <> '';

CREATE TABLE IF NOT EXISTS loyalty_ledgers (
    id         BIGSERIAL    PRIMARY KEY,
    member_id  BIGINT       NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    points     INT          NOT NULL,
    reason     VARCHAR(30)  NOT NULL
               CHECK (reason IN ('CHECKIN', 'RENEW', 'REFERRAL', 'REVIEW', 'REDEEM', 'CHALLENGE', 'ADJUST')),
    ref_type   VARCHAR(30)  NOT NULL DEFAULT '',
    ref_id     BIGINT,
    note       VARCHAR(255) NOT NULL DEFAULT '',
    created_by BIGINT       REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_loyalty_ledgers_member_date ON loyalty_ledgers(member_id, created_at DESC);

CREATE TABLE IF NOT EXISTS loyalty_rewards (
    id          BIGSERIAL     PRIMARY KEY,
    name        VARCHAR(150)  NOT NULL,
    description VARCHAR(500)  NOT NULL DEFAULT '',
    cost_points INT           NOT NULL CHECK (cost_points > 0),
    reward_type VARCHAR(20)   NOT NULL CHECK (reward_type IN ('VOUCHER', 'PRODUCT')),
    ref_id      BIGINT,
    discount_pct NUMERIC(5,2) NOT NULL DEFAULT 10,
    stock       INT           NOT NULL DEFAULT 100,
    status      VARCHAR(20)   NOT NULL DEFAULT 'ACTIVE'
                CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS badges (
    id          BIGSERIAL    PRIMARY KEY,
    code        VARCHAR(50)  NOT NULL UNIQUE,
    name        VARCHAR(150) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    icon        VARCHAR(100) NOT NULL DEFAULT 'trophy',
    rule        JSONB        NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS member_badges (
    member_id BIGINT      NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    badge_id  BIGINT      NOT NULL REFERENCES badges(id) ON DELETE CASCADE,
    earned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (member_id, badge_id)
);

CREATE TABLE IF NOT EXISTS challenges (
    id            BIGSERIAL    PRIMARY KEY,
    branch_id     BIGINT       REFERENCES branches(id) ON DELETE CASCADE,
    name          VARCHAR(150) NOT NULL,
    description   VARCHAR(500) NOT NULL DEFAULT '',
    metric        VARCHAR(30)  NOT NULL CHECK (metric IN ('CHECKINS', 'CLASSES', 'PT_SESSIONS')),
    target        INT          NOT NULL CHECK (target > 0),
    start_date    DATE         NOT NULL,
    end_date      DATE         NOT NULL,
    reward_points INT          NOT NULL DEFAULT 100,
    status        VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
                  CHECK (status IN ('ACTIVE', 'ENDED', 'DRAFT')),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS challenge_participants (
    challenge_id BIGINT      NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    member_id    BIGINT      NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    progress     INT         NOT NULL DEFAULT 0,
    completed_at TIMESTAMPTZ,
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (challenge_id, member_id)
);

INSERT INTO badges (code, name, description, icon, rule) VALUES
    ('STREAK_7',     'Chiến binh 7 ngày',   'Hoàn thành 7 lượt check-in tại phòng tập',       'fire',   '{"metric":"CHECKINS","target":7}'),
    ('STREAK_30',    'Bền bỉ 30 buổi',      'Đạt cột mốc 30 buổi tập luyện',                  'trophy', '{"metric":"CHECKINS","target":30}'),
    ('CENTURY_100',  'Câu lạc bộ 100 buổi', 'Chinh phục 100 buổi tập tại GymMaster',          'crown',  '{"metric":"CHECKINS","target":100}'),
    ('CLASS_LOVER',  'Ngôi sao lớp nhóm',   'Tham gia ít nhất 10 buổi lớp học nhóm',          'star',   '{"metric":"CLASSES","target":10}')
ON CONFLICT (code) DO NOTHING;

INSERT INTO loyalty_rewards (name, description, cost_points, reward_type, discount_pct, stock, status) VALUES
    ('Voucher giảm 10% gia hạn gói tập', 'Sinh mã ưu đãi giảm 10% dùng 1 lần khi mua/gia hạn gói tập', 200, 'VOUCHER', 10, 500, 'ACTIVE'),
    ('Voucher giảm 20% gói tập bất kỳ',  'Sinh mã ưu đãi giảm 20% dùng 1 lần khi mua/gia hạn gói tập', 450, 'VOUCHER', 20, 200, 'ACTIVE'),
    ('Nước khoáng thể thao miễn phí',    'Nhận 1 chai nước khoáng tại quầy lễ tân',                     50,  'PRODUCT', 0,  300, 'ACTIVE')
ON CONFLICT DO NOTHING;

INSERT INTO permissions (code, module, name) VALUES
    ('loyalty.view',   'loyalty', 'Xem điểm thưởng, huy hiệu & thử thách'),
    ('loyalty.manage', 'loyalty', 'Quản lý quà đổi điểm, huy hiệu, thử thách & điều chỉnh điểm')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code IN ('OWNER', 'MANAGER')
  AND p.code IN ('loyalty.view', 'loyalty.manage')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'RECEPTIONIST'
  AND p.code = 'loyalty.view'
ON CONFLICT DO NOTHING;
