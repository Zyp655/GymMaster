-- QA2 (web): nhật ký ăn uống lưu số khẩu phần chính xác hơn để món tính theo gam
-- (VD: 12,5 g món "100g" = 0,125 khẩu phần) không bị làm tròn về 2 chữ số thập phân.
ALTER TABLE food_logs ALTER COLUMN qty TYPE NUMERIC(10,4);
