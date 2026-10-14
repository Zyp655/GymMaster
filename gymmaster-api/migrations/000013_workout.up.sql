-- =====================================================================
-- 000013 – Thư viện bài tập, giáo án tập luyện & nhật ký tập của hội viên
-- =====================================================================

CREATE TABLE exercises (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(150) NOT NULL,
    muscle_group  VARCHAR(50)  NOT NULL
                  CHECK (muscle_group IN ('CHEST', 'BACK', 'SHOULDERS', 'LEGS', 'ARMS', 'CORE', 'CARDIO', 'FULL_BODY')),
    equipment     VARCHAR(100) NOT NULL DEFAULT '',
    video_url     VARCHAR(500) NOT NULL DEFAULT '',
    description   TEXT         NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ,
    created_by    BIGINT,
    updated_by    BIGINT
);
CREATE UNIQUE INDEX uq_exercises_name ON exercises (lower(name)) WHERE deleted_at IS NULL;
CREATE INDEX idx_exercises_muscle_group ON exercises (muscle_group) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_exercises_updated_at BEFORE UPDATE ON exercises FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Giáo án tập luyện (member_id NULL = giáo án mẫu dùng chung để clone).
CREATE TABLE workout_plans (
    id              BIGSERIAL PRIMARY KEY,
    member_id       BIGINT       REFERENCES members (id) ON DELETE CASCADE,
    trainer_id      BIGINT       NOT NULL REFERENCES users (id),
    pt_contract_id  BIGINT       REFERENCES pt_contracts (id),
    name            VARCHAR(150) NOT NULL,
    goal            VARCHAR(255) NOT NULL DEFAULT '',
    start_date      DATE,
    end_date        DATE,
    status          VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
                    CHECK (status IN ('DRAFT', 'ACTIVE', 'COMPLETED', 'ARCHIVED')),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,
    created_by      BIGINT,
    updated_by      BIGINT,
    CONSTRAINT chk_workout_plans_dates CHECK (start_date IS NULL OR end_date IS NULL OR end_date >= start_date)
);
CREATE INDEX idx_workout_plans_member ON workout_plans (member_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_workout_plans_trainer ON workout_plans (trainer_id) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_workout_plans_updated_at BEFORE UPDATE ON workout_plans FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE workout_plan_days (
    id         BIGSERIAL PRIMARY KEY,
    plan_id    BIGINT       NOT NULL REFERENCES workout_plans (id) ON DELETE CASCADE,
    day_index  INT          NOT NULL CHECK (day_index BETWEEN 1 AND 14),
    title      VARCHAR(150) NOT NULL
);
CREATE INDEX idx_workout_plan_days_plan ON workout_plan_days (plan_id, day_index);

CREATE TABLE workout_plan_items (
    id           BIGSERIAL PRIMARY KEY,
    day_id       BIGINT        NOT NULL REFERENCES workout_plan_days (id) ON DELETE CASCADE,
    exercise_id  BIGINT        NOT NULL REFERENCES exercises (id),
    sets         INT           NOT NULL DEFAULT 3 CHECK (sets > 0),
    reps         INT           NOT NULL DEFAULT 10 CHECK (reps > 0),
    weight       NUMERIC(6,2)  NOT NULL DEFAULT 0 CHECK (weight >= 0),
    rest_sec     INT           NOT NULL DEFAULT 60 CHECK (rest_sec >= 0),
    note         VARCHAR(255)  NOT NULL DEFAULT '',
    sort         INT           NOT NULL DEFAULT 0
);
CREATE INDEX idx_workout_plan_items_day ON workout_plan_items (day_id, sort);

-- Nhật ký buổi tập do hội viên tự ghi nhận (theo dõi tiến bộ mức tạ).
CREATE TABLE workout_logs (
    id            BIGSERIAL PRIMARY KEY,
    member_id     BIGINT        NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    plan_item_id  BIGINT        REFERENCES workout_plan_items (id) ON DELETE SET NULL,
    exercise_id   BIGINT        NOT NULL REFERENCES exercises (id),
    performed_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    sets          INT           NOT NULL CHECK (sets > 0),
    reps          INT           NOT NULL CHECK (reps > 0),
    weight        NUMERIC(6,2)  NOT NULL DEFAULT 0 CHECK (weight >= 0),
    note          VARCHAR(255)  NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX idx_workout_logs_member_exercise ON workout_logs (member_id, exercise_id, performed_at DESC);
CREATE INDEX idx_workout_logs_member_time ON workout_logs (member_id, performed_at DESC);

INSERT INTO permissions (code, module, name) VALUES
    ('workout.view',   'workout', 'Xem thư viện bài tập & giáo án'),
    ('workout.manage', 'workout', 'Quản lý bài tập & soạn giáo án');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.module = 'workout'
WHERE r.code IN ('OWNER', 'MANAGER', 'TRAINER');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code = 'workout.view'
WHERE r.code = 'RECEPTIONIST';

-- Seed 40 bài tập cơ bản trong thư viện.
INSERT INTO exercises (name, muscle_group, equipment, description) VALUES
    ('Barbell Bench Press (Đẩy ngực thanh đòn phẳng)', 'CHEST', 'Thanh đòn, Ghế phẳng', 'Bài tập cơ ngực chính với thanh đòn trên ghế phẳng.'),
    ('Incline Dumbbell Press (Đẩy ngực trên tạ đơn)', 'CHEST', 'Tạ đơn, Ghế dốc lên', 'Tập trung phát triển phần cơ ngực trên.'),
    ('Decline Barbell Press (Đẩy ngực dưới thanh đòn)', 'CHEST', 'Thanh đòn, Ghế dốc xuống', 'Tác động mạnh vào cơ ngực dưới.'),
    ('Cable Chest Fly (Ép ngực dây cáp)', 'CHEST', 'Máy kéo cáp', 'Cô lập cơ ngực trong suốt biên độ chuyển động.'),
    ('Push-Up (Hít đất cơ bản)', 'CHEST', 'Trọng lượng cơ thể', 'Bài tập cơ ngực, vai trước và tay sau với trọng lượng cơ thể.'),
    ('Chest Dips (Xà kép tập ngực)', 'CHEST', 'Khung xà kép', 'Hơi đổ người về trước để tác động tối đa vào ngực dưới.'),
    ('Conventional Deadlift (Kéo lưng đòn tạ)', 'BACK', 'Thanh đòn', 'Bài tập toàn diện cho chuỗi cơ sau, lưng dưới và đùi sau.'),
    ('Pull-Up (Kéo xà đơn rộng tay)', 'BACK', 'Xà đơn', 'Phát triển độ rộng cơ xô (Latissimus Dorsi).'),
    ('Lat Pulldown (Kéo cáp xô trước ngực)', 'BACK', 'Máy kéo xô', 'Bài tập cơ xô phù hợp cho mọi trình độ.'),
    ('Barbell Bent-Over Row (Gập người chèo tạ đòn)', 'BACK', 'Thanh đòn', 'Tăng độ dày cơ lưng giữa và cơ xô.'),
    ('Seated Cable Row (Ngồi kéo cáp lưng giữa)', 'BACK', 'Máy kéo cáp', 'Giữ lưng thẳng, ép bả vai khi kéo tay cầm về sát bụng.'),
    ('Single-Arm Dumbbell Row (Chèo tạ đơn một tay)', 'BACK', 'Tạ đơn, Ghế phẳng', 'Cải thiện sự cân bằng hai bên cơ lưng.'),
    ('Face Pull (Kéo dây thừng về phía mặt)', 'SHOULDERS', 'Máy kéo cáp, Dây thừng', 'Tăng cường cơ vai sau và nhóm cơ xoay vai.'),
    ('Overhead Barbell Press (Đẩy vai thanh đòn đứng)', 'SHOULDERS', 'Thanh đòn', 'Bài tập sức mạnh nền tảng cho cơ vai trước và vai giữa.'),
    ('Seated Dumbbell Shoulder Press (Ngồi đẩy vai tạ đơn)', 'SHOULDERS', 'Tạ đơn, Ghế tựa', 'Phát triển toàn diện cơ vai với biên độ tự nhiên.'),
    ('Dumbbell Lateral Raise (Dang tạ đơn vai giữa)', 'SHOULDERS', 'Tạ đơn', 'Tạo độ rộng cho bờ vai (cơ Delta giữa).'),
    ('Rear Delt Fly (Dang tạ đơn vai sau)', 'SHOULDERS', 'Tạ đơn hoặc Máy Pec Deck', 'Cô lập cơ vai sau, cải thiện tư thế lưng trên.'),
    ('Arnold Press (Đẩy vai xoay cổ tay)', 'SHOULDERS', 'Tạ đơn', 'Kích hoạt cả 3 đầu cơ vai.'),
    ('Barbell Back Squat (Gánh tạ đòn sau cổ)', 'LEGS', 'Thanh đòn, Khung Squat', 'Bài tập vua cho cơ đùi trước, cơ mông và sức mạnh thân dưới.'),
    ('Front Squat (Gánh tạ đòn trước ngực)', 'LEGS', 'Thanh đòn, Khung Squat', 'Tập trung mạnh vào cơ đùi trước và giữ lưng thẳng.'),
    ('Romanian Deadlift - RDL (Deadlift chân hơi chùng)', 'LEGS', 'Thanh đòn hoặc Tạ đơn', 'Tác động sâu vào cơ đùi sau (Hamstrings) và cơ mông.'),
    ('Leg Press (Đạp đùi trên máy)', 'LEGS', 'Máy Leg Press', 'Tập đùi trước và mông với mức tạ nặng an toàn cho lưng.'),
    ('Bulgarian Split Squat (Chùng chân sau lên ghế)', 'LEGS', 'Tạ đơn, Ghế phẳng', 'Phát triển cơ đùi, cơ mông và khả năng giữ thăng bằng.'),
    ('Lying Leg Curl (Nằm cuốn đùi sau trên máy)', 'LEGS', 'Máy Leg Curl', 'Cô lập nhóm cơ đùi sau.'),
    ('Leg Extension (Ngồi đá đùi trước trên máy)', 'LEGS', 'Máy Leg Extension', 'Cô lập cơ tứ đầu đùi (Quads).'),
    ('Barbell Hip Thrust (Đẩy hông với thanh đòn)', 'LEGS', 'Thanh đòn, Ghế phẳng', 'Bài tập hiệu quả nhất cho cơ mông (Glutes).'),
    ('Standing Calf Raise (Đứng nhón bắp chân)', 'LEGS', 'Máy tập bắp chân hoặc Tạ đơn', 'Phát triển cơ bắp chân (Gastrocnemius).'),
    ('Barbell Bicep Curl (Cuốn tạ đòn tập tay trước)', 'ARMS', 'Thanh đòn hoặc Thanh EZ', 'Tăng kích thước cơ nhị đầu (Biceps).'),
    ('Incline Dumbbell Curl (Cuốn tạ đơn trên ghế dốc)', 'ARMS', 'Tạ đơn, Ghế dốc', 'Kéo giãn tối đa đầu dài cơ tay trước.'),
    ('Hammer Curl (Cuốn tạ đơn kiểu búa)', 'ARMS', 'Tạ đơn', 'Tác động vào cơ Brachialis và cẳng tay.'),
    ('Tricep Rope Pushdown (Kéo cáp dây thừng tay sau)', 'ARMS', 'Máy kéo cáp, Dây thừng', 'Cô lập cơ tam đầu (Triceps).'),
    ('Skull Crusher (Nằm duỗi tay sau với thanh EZ)', 'ARMS', 'Thanh EZ, Ghế phẳng', 'Phát triển đầu dài và đầu giữa cơ tay sau.'),
    ('Overhead Dumbbell Tricep Extension (Đứng duỗi tạ đơn qua đầu)', 'ARMS', 'Tạ đơn', 'Kéo giãn và phát triển đầu dài cơ tay sau.'),
    ('Plank (Giữ thân người thẳng trên khuỷu tay)', 'CORE', 'Thảm tập', 'Tăng sức bền nhóm cơ lõi (Core) và ổn định cột sống.'),
    ('Hanging Leg Raise (Treo xà nâng chân)', 'CORE', 'Xà đơn', 'Tác động mạnh vào cơ bụng dưới và cơ gập hông.'),
    ('Cable Woodchopper (Xoay người kéo cáp chéo)', 'CORE', 'Máy kéo cáp', 'Phát triển cơ liên sườn (Obliques).'),
    ('Ab Wheel Rollout (Con lăn tập bụng)', 'CORE', 'Con lăn bụng', 'Bài tập nâng cao cho toàn bộ vùng cơ bụng.'),
    ('Treadmill HIIT Interval (Chạy biến tốc trên máy)', 'CARDIO', 'Máy chạy bộ', 'Đốt mỡ và tăng sức bền tim mạch hiệu quả.'),
    ('Rowing Machine Ergometer (Chèo thuyền kháng lực)', 'CARDIO', 'Máy chèo thuyền Concept2', 'Cardio kết hợp vận động 85% nhóm cơ toàn thân.'),
    ('Kettlebell Swing (Vung tạ bình vôi)', 'FULL_BODY', 'Tạ Kettlebell', 'Phát triển sức mạnh bùng nổ hông và sức bền toàn thân.');
