-- Sprint 23: Dinh dưỡng, bảng tin & khảo sát NPS (000022_engagement)

CREATE TABLE IF NOT EXISTS foods (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(150) NOT NULL UNIQUE,
    unit       VARCHAR(50)  NOT NULL DEFAULT '100g',
    calories   NUMERIC(7,2) NOT NULL DEFAULT 0,
    protein    NUMERIC(6,2) NOT NULL DEFAULT 0,
    carbs      NUMERIC(6,2) NOT NULL DEFAULT 0,
    fat        NUMERIC(6,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

INSERT INTO foods (name, unit, calories, protein, carbs, fat) VALUES
    ('Ức gà áp chảo (100g)',          '100g',  165, 31.0,  0.0,  3.6),
    ('Cơm trắng (1 bát vừa)',          '1 bát', 200,  4.0, 44.0,  0.5),
    ('Khoai lang luộc (100g)',         '100g',   86,  1.6, 20.1,  0.1),
    ('Trứng gà luộc (1 quả)',          '1 quả',  78,  6.3,  0.6,  5.3),
    ('Thịt bò thăn (100g)',            '100g',  250, 26.0,  0.0, 15.0),
    ('Cá hồi phi lê (100g)',           '100g',  208, 20.4,  0.0, 13.4),
    ('Phở bò tái (1 tô)',              '1 tô',  450, 24.0, 58.0, 12.0),
    ('Bún bò Huế (1 tô)',              '1 tô',  520, 26.0, 62.0, 16.0),
    ('Chuối chín (1 quả)',             '1 quả', 105,  1.3, 27.0,  0.4),
    ('Sữa chua Hy Lạp không đường',   '1 hộp', 100, 10.0,  6.0,  0.7),
    ('Whey Protein Isolate (1 muỗng)', '1 muỗng', 120, 25.0,  2.0,  1.0),
    ('Yến mạch cán dẹt (50g)',         '50g',   190,  6.5, 33.0,  3.5),
    ('Bông cải xanh luộc (100g)',      '100g',   35,  2.4,  7.2,  0.4),
    ('Đậu phụ trắng (100g)',           '100g',   76,  8.0,  1.9,  4.8)
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS meal_plans (
    id              BIGSERIAL    PRIMARY KEY,
    member_id       BIGINT       NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    trainer_id      BIGINT       REFERENCES users(id) ON DELETE SET NULL,
    name            VARCHAR(150) NOT NULL,
    goal            VARCHAR(255) NOT NULL DEFAULT '',
    target_calories INT          NOT NULL DEFAULT 2000,
    start_date      DATE,
    end_date        DATE,
    status          VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
                    CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS meal_plan_items (
    id         BIGSERIAL    PRIMARY KEY,
    plan_id    BIGINT       NOT NULL REFERENCES meal_plans(id) ON DELETE CASCADE,
    day_index  INT          NOT NULL DEFAULT 1 CHECK (day_index BETWEEN 1 AND 7),
    meal       VARCHAR(20)  NOT NULL CHECK (meal IN ('BREAKFAST', 'LUNCH', 'DINNER', 'SNACK')),
    food_id    BIGINT       NOT NULL REFERENCES foods(id) ON DELETE CASCADE,
    qty        NUMERIC(6,2) NOT NULL DEFAULT 1,
    note       VARCHAR(255) NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS food_logs (
    id         BIGSERIAL    PRIMARY KEY,
    member_id  BIGINT       NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    food_id    BIGINT       NOT NULL REFERENCES foods(id) ON DELETE CASCADE,
    qty        NUMERIC(6,2) NOT NULL DEFAULT 1,
    meal       VARCHAR(20)  NOT NULL CHECK (meal IN ('BREAKFAST', 'LUNCH', 'DINNER', 'SNACK')),
    eaten_at   DATE         NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_food_logs_member_date ON food_logs(member_id, eaten_at DESC);

CREATE TABLE IF NOT EXISTS announcements (
    id         BIGSERIAL    PRIMARY KEY,
    branch_id  BIGINT       REFERENCES branches(id) ON DELETE CASCADE,
    title      VARCHAR(200) NOT NULL,
    body       TEXT         NOT NULL,
    image_url  VARCHAR(500) NOT NULL DEFAULT '',
    publish_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expire_at  TIMESTAMPTZ,
    pinned     BOOLEAN      NOT NULL DEFAULT FALSE,
    created_by BIGINT       REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS surveys (
    id          BIGSERIAL    PRIMARY KEY,
    title       VARCHAR(200) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    type        VARCHAR(20)  NOT NULL DEFAULT 'NPS'
                CHECK (type IN ('NPS', 'CUSTOM')),
    start_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    end_at      TIMESTAMPTZ,
    status      VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
                CHECK (status IN ('ACTIVE', 'CLOSED')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS survey_responses (
    id         BIGSERIAL   PRIMARY KEY,
    survey_id  BIGINT      NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    member_id  BIGINT      NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    score      INT         NOT NULL CHECK (score BETWEEN 0 AND 10),
    comment    TEXT        NOT NULL DEFAULT '',
    answers    JSONB       NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (survey_id, member_id)
);

INSERT INTO permissions (code, module, name) VALUES
    ('engagement.view',   'engagement', 'Xem dinh dưỡng, bảng tin & khảo sát NPS'),
    ('engagement.manage', 'engagement', 'Quản lý thực đơn dinh dưỡng, bảng tin & khảo sát NPS')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code IN ('OWNER', 'MANAGER', 'TRAINER')
  AND p.code IN ('engagement.view', 'engagement.manage')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'RECEPTIONIST'
  AND p.code = 'engagement.view'
ON CONFLICT DO NOTHING;
