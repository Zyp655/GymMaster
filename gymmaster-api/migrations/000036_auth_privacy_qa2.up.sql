-- QA vòng 2: bảo mật xác thực, quyền riêng tư, tìm kiếm không dấu, tách quyền chạy job.

-- Tìm kiếm toàn cục không phân biệt dấu tiếng Việt (unaccent là extension "trusted" từ PG13).
CREATE EXTENSION IF NOT EXISTS unaccent;

-- Chống dùng lại mã TOTP: lưu bước thời gian (unix/30) của mã TOTP hợp lệ gần nhất;
-- mã thuộc bước <= bước này bị từ chối.
ALTER TABLE user_mfa ADD COLUMN IF NOT EXISTS last_totp_step BIGINT NOT NULL DEFAULT 0;

-- Lưu bản trích xuất dữ liệu cá nhân để hội viên tải lại (GET /member/data-requests/{id}/download).
ALTER TABLE data_requests ADD COLUMN IF NOT EXISTS export_data JSONB;

-- Mỗi hội viên chỉ có tối đa một yêu cầu đang chờ cho mỗi loại (EXPORT/DELETE).
-- Dữ liệu cũ có thể đã trùng: chỉ tạo chỉ mục khi không còn bản trùng (service vẫn chặn 409).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM data_requests WHERE status = 'PENDING'
        GROUP BY member_id, type HAVING COUNT(*) > 1
    ) THEN
        CREATE UNIQUE INDEX IF NOT EXISTS uq_data_requests_pending
            ON data_requests (member_id, type) WHERE status = 'PENDING';
    END IF;
END $$;

-- job.view chỉ còn quyền XEM; chạy job/gửi email thử cần job.manage (cấp cho OWNER).
INSERT INTO permissions (code, module, name) VALUES
    ('job.manage', 'system', 'Chạy thủ công job định kỳ & gửi email thử')
ON CONFLICT (code) DO NOTHING;
UPDATE permissions SET name = 'Xem job định kỳ & lịch sử chạy' WHERE code = 'job.view';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.code = 'OWNER' AND p.code = 'job.manage'
ON CONFLICT DO NOTHING;

-- Xử lý yêu cầu dữ liệu cá nhân dùng quyền data_request.process: vai trò đang có privacy.manage
-- (OWNER, MANAGER) được cấp thêm để không mất chức năng.
INSERT INTO role_permissions (role_id, permission_id)
SELECT rp.role_id, p.id
FROM role_permissions rp
JOIN permissions pm ON pm.id = rp.permission_id AND pm.code = 'privacy.manage'
CROSS JOIN permissions p
WHERE p.code = 'data_request.process'
ON CONFLICT DO NOTHING;
