-- Sprint 29 & 15.1: Hồ sơ HLV công khai, cảm nhận hội viên, bài viết CMS, liên hệ, nhắn tin HLV-Hội viên, tùy chọn thông báo & lịch .ics

ALTER TABLE users ADD COLUMN IF NOT EXISTS calendar_token VARCHAR(80) UNIQUE;

CREATE TABLE IF NOT EXISTS trainer_public_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    slug VARCHAR(120) NOT NULL UNIQUE,
    headline VARCHAR(255) NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    specialties TEXT NOT NULL DEFAULT '',
    certifications TEXT NOT NULL DEFAULT '',
    experience_years INT NOT NULL DEFAULT 1,
    avatar_url VARCHAR(500) NOT NULL DEFAULT '',
    featured BOOLEAN NOT NULL DEFAULT FALSE,
    is_published BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS testimonials (
    id BIGSERIAL PRIMARY KEY,
    member_id BIGINT REFERENCES members(id) ON DELETE SET NULL,
    author_name VARCHAR(150) NOT NULL,
    author_role VARCHAR(120) NOT NULL DEFAULT '',
    avatar_url VARCHAR(500) NOT NULL DEFAULT '',
    rating SMALLINT NOT NULL DEFAULT 5 CHECK (rating BETWEEN 1 AND 5),
    content TEXT NOT NULL,
    before_after_note VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),
    approved_by BIGINT REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_testimonials_status ON testimonials(status, created_at DESC);

CREATE TABLE IF NOT EXISTS posts (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(180) NOT NULL UNIQUE,
    title VARCHAR(255) NOT NULL,
    excerpt TEXT NOT NULL DEFAULT '',
    cover_url VARCHAR(500) NOT NULL DEFAULT '',
    category VARCHAR(60) NOT NULL DEFAULT 'TAP_LUYEN',
    content TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PUBLISHED', 'ARCHIVED')),
    author_id BIGINT REFERENCES users(id),
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_posts_status_pub ON posts(status, published_at DESC);

CREATE TABLE IF NOT EXISTS contact_messages (
    id BIGSERIAL PRIMARY KEY,
    branch_id BIGINT REFERENCES branches(id) ON DELETE SET NULL,
    full_name VARCHAR(150) NOT NULL,
    phone VARCHAR(30) NOT NULL,
    email VARCHAR(150) NOT NULL DEFAULT '',
    subject VARCHAR(255) NOT NULL DEFAULT '',
    message TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'NEW' CHECK (status IN ('NEW', 'CONTACTED', 'CLOSED')),
    handled_by BIGINT REFERENCES users(id),
    handled_note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_contact_messages_status ON contact_messages(status, created_at DESC);

CREATE TABLE IF NOT EXISTS conversations (
    id BIGSERIAL PRIMARY KEY,
    pt_contract_id BIGINT NOT NULL UNIQUE REFERENCES pt_contracts(id) ON DELETE CASCADE,
    member_id BIGINT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    trainer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_message_at TIMESTAMPTZ,
    last_message_preview VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conversations_member ON conversations(member_id, last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversations_trainer ON conversations(trainer_id, last_message_at DESC);

CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sender_role VARCHAR(20) NOT NULL DEFAULT 'MEMBER',
    content TEXT NOT NULL,
    attachment_url VARCHAR(500) NOT NULL DEFAULT '',
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at ASC);

CREATE TABLE IF NOT EXISTS notification_preferences (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    channel VARCHAR(20) NOT NULL DEFAULT 'IN_APP' CHECK (channel IN ('IN_APP', 'EMAIL')),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, type, channel)
);

INSERT INTO permissions (code, module, name) VALUES
    ('cms.manage', 'cms', 'Quản lý nội dung Landing, Blog, Hồ sơ HLV & Cảm nhận'),
    ('equipment.report', 'equipment', 'Báo hỏng thiết bị phòng tập'),
    ('lead.convert', 'lead', 'Chuyển đổi khách tiềm năng thành hội viên'),
    ('attendance.view', 'payroll', 'Xem chấm công nhân viên'),
    ('payroll.lock', 'payroll', 'Khóa bảng lương tháng'),
    ('expense.approve', 'expense', 'Duyệt chi phí vận hành'),
    ('installment.manage', 'invoice', 'Quản lý kế hoạch trả góp'),
    ('report.profit_loss', 'report', 'Xem báo cáo Lãi / Lỗ (P&L)'),
    ('loyalty.adjust', 'loyalty', 'Điều chỉnh điểm thưởng hội viên'),
    ('nutrition.manage', 'engagement', 'Quản lý thực đơn & món ăn dinh dưỡng'),
    ('announcement.manage', 'engagement', 'Quản lý bảng tin chi nhánh'),
    ('survey.manage', 'engagement', 'Quản lý khảo sát NPS'),
    ('day_pass.sell', 'contract', 'Bán vé ngày & cấp vé khách mời'),
    ('availability.manage', 'leave', 'Cấu hình khung giờ rảnh HLV'),
    ('leave.request', 'leave', 'Gửi đơn xin nghỉ phép'),
    ('leave.approve', 'leave', 'Duyệt đơn nghỉ phép & phân công dạy thay'),
    ('report.cohort', 'kpi', 'Xem báo cáo Cohort Retention'),
    ('data_request.process', 'privacy', 'Xử lý yêu cầu xuất/xóa dữ liệu cá nhân')
ON CONFLICT (code) DO NOTHING;
