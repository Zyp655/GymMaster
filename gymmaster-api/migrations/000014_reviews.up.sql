-- =====================================================================
-- 000014 – Đánh giá huấn luyện viên & buổi tập / lớp học
-- =====================================================================

ALTER TABLE trainer_profiles
    ADD COLUMN IF NOT EXISTS rating_avg   NUMERIC(3,2) NOT NULL DEFAULT 0 CHECK (rating_avg BETWEEN 0 AND 5),
    ADD COLUMN IF NOT EXISTS rating_count INT          NOT NULL DEFAULT 0 CHECK (rating_count >= 0);

CREATE TABLE reviews (
    id           BIGSERIAL PRIMARY KEY,
    member_id    BIGINT       NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    target_type  VARCHAR(20)  NOT NULL CHECK (target_type IN ('PT_SESSION', 'CLASS_SCHEDULE')),
    target_id    BIGINT       NOT NULL,
    trainer_id   BIGINT       NOT NULL REFERENCES users (id),
    branch_id    BIGINT       REFERENCES branches (id),
    rating       INT          NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment      TEXT         NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_reviews_member_target UNIQUE (member_id, target_type, target_id)
);
CREATE INDEX idx_reviews_trainer ON reviews (trainer_id, created_at DESC);
CREATE INDEX idx_reviews_branch ON reviews (branch_id, created_at DESC);

INSERT INTO permissions (code, module, name) VALUES
    ('review.view', 'review', 'Xem đánh giá HLV & lớp học');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.module = 'review'
WHERE r.code IN ('OWNER', 'MANAGER', 'RECEPTIONIST', 'TRAINER');
