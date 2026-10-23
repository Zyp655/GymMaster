-- QA vòng 5 (R-B).

-- 1) Vé mời (GUEST): một SĐT chỉ có MỘT vé mời cho mỗi ngày – chặn race hai người mời khác nhau tạo đồng thời.
-- Chuẩn hóa SĐT cũ về dạng 0xxxxxxxxx giống NormalizeVNPhone (bỏ khoảng trắng . - ( ), +84/84 → 0).
UPDATE day_passes d
SET phone = n.p
FROM (
    SELECT id,
           CASE
               WHEN s LIKE '+84%' THEN '0' || substr(s, 4)
               WHEN s LIKE '84%' AND length(s) >= 11 THEN '0' || substr(s, 3)
               ELSE s
           END AS p
    FROM (SELECT id, regexp_replace(phone, '[ .()-]', '', 'g') AS s FROM day_passes WHERE type = 'GUEST') x
) n
WHERE d.id = n.id AND d.phone <> n.p AND n.p ~ '^(0(3|5|7|8|9)[0-9]{8}|02[0-9]{9})$';

-- Gộp vé trùng (SĐT, ngày) CHƯA dùng: giữ vé đã dùng (nếu có) hoặc vé tạo trước; vé đã dùng được giữ làm lịch sử.
DELETE FROM day_passes d
WHERE d.type = 'GUEST' AND d.used_at IS NULL
  AND EXISTS (SELECT 1 FROM day_passes e
              WHERE e.type = 'GUEST' AND e.id <> d.id AND e.phone = d.phone AND e.valid_date = d.valid_date
                AND (e.used_at IS NOT NULL OR e.id < d.id));

-- Hai lệnh tạo đồng thời đều tạo vé chưa dùng → index này cho đúng một lệnh thành công (lệnh kia 409).
CREATE UNIQUE INDEX IF NOT EXISTS uq_day_passes_guest_phone_date
    ON day_passes (phone, valid_date) WHERE type = 'GUEST' AND used_at IS NULL;

-- 2) user_mfa.created_at bị ghi đè giá trị rỗng (0001-01-01) khi thiết lập lại MFA → sửa dữ liệu cũ.
UPDATE user_mfa
SET created_at = CASE WHEN updated_at >= '2000-01-01' THEN updated_at ELSE now() END
WHERE created_at < '2000-01-01';

-- 3) Hội thoại PT: thời điểm HLV hiện tại tham gia (sau khi đổi HLV) – HLV mới không thấy tin cũ gửi HLV trước là "chưa đọc".
-- NULL = HLV tham gia từ đầu.
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS trainer_joined_at TIMESTAMPTZ;

-- Dữ liệu cũ: lấy lần đổi HLV gần nhất của hợp đồng từ nhật ký (CHANGE_TRAINER / pt_contract).
UPDATE conversations cv
SET trainer_joined_at = a.at
FROM (SELECT entity_id, max(created_at) AS at FROM audit_logs
      WHERE entity = 'pt_contract' AND action = 'CHANGE_TRAINER' GROUP BY entity_id) a
WHERE a.entity_id = cv.pt_contract_id AND cv.trainer_joined_at IS NULL;
