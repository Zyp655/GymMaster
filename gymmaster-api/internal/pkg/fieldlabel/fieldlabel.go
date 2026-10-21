// Package fieldlabel ánh xạ tên trường kỹ thuật (tên field Go như "BranchID", tên JSON "branch_id"
// hoặc tên cột DB) sang nhãn tiếng Việt để hiển thị trong thông báo lỗi cho người dùng
// (lỗi validate, lỗi trùng dữ liệu/khóa ngoại của DB).
package fieldlabel

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// labels: tên snake_case (JSON tag/cột DB) → nhãn tiếng Việt (chữ thường, ghép vào câu).
var labels = map[string]string{
	// Định danh & tham chiếu
	"id":                    "mã định danh",
	"ids":                   "danh sách mã định danh",
	"branch_id":             "chi nhánh",
	"home_branch_id":        "chi nhánh chính",
	"from_branch_id":        "chi nhánh chuyển đi",
	"to_branch_id":          "chi nhánh nhận",
	"member_id":             "hội viên",
	"member_ids":            "danh sách hội viên",
	"to_member_id":          "hội viên nhận",
	"from_member_id":        "hội viên chuyển",
	"user_id":               "người dùng",
	"staff_id":              "nhân viên",
	"trainer_id":            "huấn luyện viên",
	"new_trainer_id":        "huấn luyện viên mới",
	"assigned_to":           "người phụ trách",
	"plan_id":               "gói tập",
	"new_plan_id":           "gói mới",
	"interest_plan_id":      "gói quan tâm",
	"pt_package_id":         "gói PT",
	"contract_id":           "hợp đồng",
	"pt_contract_id":        "hợp đồng PT",
	"subscription_id":       "gói đăng ký",
	"invoice_id":            "hóa đơn",
	"payment_id":            "thanh toán",
	"product_id":            "sản phẩm",
	"category_id":           "danh mục",
	"supplier_id":           "nhà cung cấp",
	"room_id":               "phòng",
	"class_type_id":         "loại lớp",
	"class_schedule_id":     "lịch lớp",
	"schedule_id":           "lịch",
	"booking_id":            "lượt đặt chỗ",
	"locker_id":             "tủ đồ",
	"equipment_id":          "thiết bị",
	"maintenance_log_id":    "phiếu bảo trì",
	"day_pass_id":           "vé ngày",
	"trial_pass_id":         "vé tập thử",
	"day_id":                "ngày tập",
	"plan_item_id":          "mục thực đơn",
	"conversation_id":       "cuộc trò chuyện",
	"permission_id":         "quyền",
	"online_txn_id":         "giao dịch trực tuyến",
	"previous_id":           "bản ghi trước",
	"source_id":             "nguồn",
	"sponsor_member_id":     "hội viên bảo lãnh",
	"original_trainer_id":   "huấn luyện viên ban đầu",
	"substitute_trainer_id": "huấn luyện viên thay thế",
	"sender_user_id":        "người gửi",
	"actor_id":              "người thực hiện",
	"author_id":             "tác giả",
	"created_by":            "người tạo",
	"approved_by":           "người duyệt",
	"reviewed_by":           "người duyệt",
	"requested_by":          "người yêu cầu",
	"handled_by":            "người xử lý",
	"issued_by":             "người phát hành",
	"locked_by":             "người khóa",
	"measured_by":           "người đo",
	"paid_by":               "người thanh toán",
	"received_by":           "người nhận",
	"refunded_by":           "người hoàn tiền",
	"reported_by":           "người báo cáo",
	"sold_by":               "người bán",
	"voided_by":             "người hủy",
	"referred_by":           "người giới thiệu",
	"referral_code":         "mã giới thiệu",
	"member_code":           "mã hội viên",
	"staff_code":            "mã nhân viên",
	"invoice_no":            "số hóa đơn",
	"txn_ref":               "mã giao dịch",
	"calendar_token":        "mã lịch",
	"token_hash":            "mã xác thực",
	"device_id":             "thiết bị",
	"exercise_id":           "bài tập",
	"food_id":               "món ăn",
	"reward_id":             "phần thưởng",
	"promotion_id":          "khuyến mãi",
	"role_id":               "vai trò",
	"shift_id":              "ca làm",
	"session_id":            "buổi tập",
	"lead_id":               "khách hàng tiềm năng",
	"target_id":             "đối tượng",
	"ref_id":                "đối tượng tham chiếu",
	"template_id":           "mẫu",
	"badge_id":              "huy hiệu",
	"challenge_id":          "thử thách",
	"survey_id":             "khảo sát",
	"post_id":               "bài viết",
	"workout_plan_id":       "giáo án",
	"meal_plan_id":          "thực đơn",
	"payroll_id":            "bảng lương",
	"cash_session_id":       "ca thu ngân",

	// Thông tin chung
	"name":             "tên",
	"full_name":        "họ tên",
	"code":             "mã",
	"title":            "tiêu đề",
	"slug":             "đường dẫn (slug)",
	"status":           "trạng thái",
	"type":             "loại",
	"kind":             "loại",
	"category":         "danh mục",
	"description":      "mô tả",
	"content":          "nội dung",
	"body":             "nội dung",
	"body_html":        "nội dung",
	"message":          "nội dung tin nhắn",
	"comment":          "bình luận",
	"note":             "ghi chú",
	"notes":            "ghi chú",
	"reason":           "lý do",
	"review_note":      "ghi chú duyệt",
	"trainer_note":     "ghi chú của HLV",
	"workout_note":     "ghi chú buổi tập",
	"source":           "nguồn",
	"source_type":      "loại nguồn",
	"channel":          "kênh",
	"scope":            "phạm vi",
	"access_scope":     "phạm vi truy cập",
	"mode":             "chế độ",
	"action":           "thao tác",
	"result":           "kết quả",
	"key":              "khóa",
	"value":            "giá trị",
	"version":          "phiên bản",
	"color":            "màu",
	"position":         "chức vụ",
	"level":            "cấp độ",
	"goal":             "mục tiêu",
	"q":                "từ khóa tìm kiếm",
	"keyword":          "từ khóa tìm kiếm",
	"page":             "trang",
	"size":             "kích thước trang",
	"page_size":        "kích thước trang",
	"limit":            "giới hạn",
	"sort_order":       "thứ tự",
	"group_by":         "kiểu nhóm",
	"metric":           "chỉ số",
	"population":       "nhóm đối tượng",
	"preferences":      "tùy chọn",
	"rows":             "dữ liệu",
	"items":            "danh sách mục",
	"roles":            "vai trò",
	"role_code":        "mã vai trò",
	"permissions":      "quyền",
	"equipment":        "thiết bị",
	"file":             "tệp",
	"image_url":        "ảnh",
	"avatar_url":       "ảnh đại diện",
	"cover_url":        "ảnh bìa",
	"video_url":        "video",
	"url":              "đường dẫn",
	"author_name":      "tên tác giả",
	"bio":              "giới thiệu",
	"certifications":   "chứng chỉ",
	"years_experience": "số năm kinh nghiệm",
	"experience_years": "số năm kinh nghiệm",

	// Liên hệ & cá nhân
	"phone":                   "số điện thoại",
	"email":                   "email",
	"to":                      "người nhận",
	"address":                 "địa chỉ",
	"gender":                  "giới tính",
	"dob":                     "ngày sinh",
	"birthday":                "ngày sinh",
	"id_card":                 "số CCCD/CMND",
	"emergency_contact_name":  "tên người liên hệ khẩn cấp",
	"emergency_contact_phone": "số điện thoại liên hệ khẩn cấp",
	"buyer_name":              "tên người mua",
	"password":                "mật khẩu",
	"old_password":            "mật khẩu hiện tại",
	"new_password":            "mật khẩu mới",
	"otp":                     "mã OTP",
	"token":                   "mã xác thực",
	"refresh_token":           "mã làm mới phiên",
	"qr_token":                "mã QR",
	"signature_data":          "chữ ký",
	"hire_date":               "ngày vào làm",

	// Tiền & số lượng
	"amount":                 "số tiền",
	"price":                  "giá",
	"sell_price":             "giá bán",
	"member_price":           "giá hội viên",
	"cost_price":             "giá vốn",
	"unit_cost":              "đơn giá nhập",
	"cost":                   "chi phí",
	"monthly_fee":            "phí hằng tháng",
	"transfer_fee":           "phí chuyển nhượng",
	"base_salary":            "lương cơ bản",
	"commission_rate":        "tỷ lệ hoa hồng",
	"discount_pct":           "phần trăm giảm giá",
	"min_amount":             "giá trị đơn tối thiểu",
	"opening_cash":           "tiền mặt đầu ca",
	"counted_cash":           "tiền mặt đếm thực tế",
	"payment_method":         "phương thức thanh toán",
	"refund_method":          "phương thức hoàn tiền",
	"method":                 "phương thức",
	"payment":                "thông tin thanh toán",
	"payments":               "thông tin thanh toán",
	"refund":                 "thông tin hoàn tiền",
	"installments":           "số kỳ trả góp",
	"promotion_code":         "mã khuyến mãi",
	"voucher_no":             "số chứng từ",
	"reference":              "mã tham chiếu",
	"quantity":               "số lượng",
	"qty":                    "số lượng",
	"actual_quantity":        "số lượng thực tế",
	"stock":                  "tồn kho",
	"min_stock":              "tồn kho tối thiểu",
	"sku":                    "mã SKU",
	"barcode":                "mã vạch",
	"unit":                   "đơn vị",
	"points":                 "điểm",
	"cost_points":            "số điểm đổi",
	"reward_points":          "điểm thưởng",
	"reward_type":            "loại phần thưởng",
	"usage_limit":            "giới hạn lượt dùng",
	"applies_to":             "phạm vi áp dụng",
	"capacity":               "sức chứa",
	"default_capacity":       "sức chứa mặc định",
	"session_count":          "số buổi",
	"guest_passes_per_month": "số lượt khách mỗi tháng",
	"max_freeze_times":       "số lần bảo lưu tối đa",
	"max_freeze_days":        "số ngày bảo lưu tối đa",
	"rating":                 "số sao",
	"score":                  "điểm",
	"target":                 "mục tiêu",
	"target_value":           "giá trị mục tiêu",
	"target_type":            "loại đối tượng",
	"target_calories":        "lượng calo mục tiêu",

	// Thời gian
	"date":                      "ngày",
	"start_date":                "ngày bắt đầu",
	"end_date":                  "ngày kết thúc",
	"from":                      "từ ngày",
	"to_date":                   "đến ngày",
	"from_date":                 "từ ngày",
	"start_at":                  "thời gian bắt đầu",
	"end_at":                    "thời gian kết thúc",
	"from_at":                   "thời gian bắt đầu",
	"to_at":                     "thời gian kết thúc",
	"start_time":                "giờ bắt đầu",
	"end_time":                  "giờ kết thúc",
	"open_time":                 "giờ mở cửa",
	"close_time":                "giờ đóng cửa",
	"spent_at":                  "ngày chi",
	"period":                    "kỳ",
	"month":                     "tháng",
	"year":                      "năm",
	"dow":                       "thứ trong tuần",
	"weekdays":                  "các ngày trong tuần",
	"day_index":                 "thứ tự ngày",
	"duration_min":              "thời lượng (phút)",
	"default_duration_min":      "thời lượng mặc định (phút)",
	"duration_days":             "số ngày hiệu lực",
	"validity_days":             "số ngày hiệu lực",
	"validity":                  "thời hạn",
	"session_minutes":           "thời lượng buổi (phút)",
	"window_days":               "số ngày",
	"expiring_in_days":          "số ngày sắp hết hạn",
	"maintenance_interval_days": "chu kỳ bảo trì (ngày)",
	"from_week_start":           "tuần nguồn",
	"to_week_start":             "tuần đích",
	"schedule":                  "lịch",
	"slots":                     "khung giờ",
	"access_hours":              "khung giờ ra vào",
	"days":                      "danh sách ngày tập",

	// Tập luyện & dinh dưỡng
	"sets":                "số hiệp",
	"reps":                "số lần",
	"rest_sec":            "thời gian nghỉ (giây)",
	"weight":              "mức tạ",
	"weight_kg":           "cân nặng (kg)",
	"height_cm":           "chiều cao (cm)",
	"body_fat_pct":        "tỷ lệ mỡ",
	"muscle_mass_kg":      "khối lượng cơ (kg)",
	"waist_cm":            "vòng eo (cm)",
	"hip_cm":              "vòng hông (cm)",
	"chest_cm":            "vòng ngực (cm)",
	"muscle_group":        "nhóm cơ",
	"meal":                "bữa ăn",
	"calories":            "lượng calo",
	"protein":             "chất đạm",
	"carbs":               "tinh bột",
	"fat":                 "chất béo",
	"trainer_profile":     "hồ sơ HLV",
	"subscription_status": "trạng thái gói",
}

// Label trả nhãn tiếng Việt của một tên trường (chấp nhận tên Go dạng CamelCase hoặc snake_case).
// ok=false nếu không có trong từ điển.
func Label(name string) (string, bool) {
	if name == "" {
		return "", false
	}
	if l, ok := labels[name]; ok {
		return l, true
	}
	l, ok := labels[ToSnake(name)]
	return l, ok
}

// LabelOr trả nhãn tiếng Việt, hoặc fallback nếu không tìm thấy.
func LabelOr(name, fallback string) string {
	if l, ok := Label(name); ok {
		return l
	}
	return fallback
}

// Capitalize viết hoa chữ cái đầu (dùng khi nhãn đứng đầu câu).
func Capitalize(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + s[size:]
}

// ToSnake đổi tên Go CamelCase sang snake_case, giữ nguyên các cụm viết tắt:
// "BranchID" → "branch_id", "IDCard" → "id_card", "BodyHTML" → "body_html".
func ToSnake(s string) string {
	if strings.ContainsRune(s, '_') || strings.ToLower(s) == s {
		return strings.ToLower(s)
	}
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
					b.WriteByte('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
