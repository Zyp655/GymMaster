package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/setting/v1"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
)

type (
	// SettingAdminService phục vụ màn hình cấu hình: đọc toàn bộ và sửa từng mục (có validate kiểu, ghi audit).
	// Các module nghiệp vụ đọc cấu hình qua SettingService, không qua service này.
	SettingAdminService interface {
		List(ctx context.Context) ([]*v1.Setting, error)
		Update(ctx context.Context, key string, value any) (*v1.Setting, error)
	}

	settingAdminService struct {
		settings SettingService
		audit    AuditService
	}
)

func NewSettingAdminService(settings SettingService, audit AuditService) SettingAdminService {
	return &settingAdminService{settings: settings, audit: audit}
}

// settingRule mô tả kiểu dữ liệu hợp lệ của một khóa cấu hình.
type settingRule struct {
	isSlice  bool
	min, max int
	maxItems int // chỉ dùng cho mảng
}

var settingRules = map[string]settingRule{
	SettingCheckinDuplicateWindowMinutes: {min: 1, max: 1440},
	SettingCheckinQRTTLSeconds:           {min: 15, max: 600},
	SettingBookingAdvanceDays:            {min: 1, max: 60},
	SettingBookingCancelBeforeMinutes:    {min: 1, max: 10080},
	SettingBookingNoShowLimit:            {min: 1, max: 100},
	SettingBookingBlockDays:              {min: 1, max: 365},
	SettingSubscriptionRemindBeforeDays:  {isSlice: true, min: 1, max: 90, maxItems: 10},
	SettingPTCancelBeforeMinutes:         {min: 1, max: 10080},
}

func (s *settingAdminService) List(ctx context.Context) ([]*v1.Setting, error) {
	rows, err := s.settings.All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Setting, 0, len(rows))
	for _, r := range rows {
		out = append(out, &v1.Setting{Key: r.Key, Value: json.RawMessage(r.Value), Description: r.Description, UpdatedAt: r.UpdatedAt})
	}
	return out, nil
}

func (s *settingAdminService) Update(ctx context.Context, key string, value any) (*v1.Setting, error) {
	before, err := s.find(ctx, key)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Giá trị cấu hình không hợp lệ")
	}
	normalized, err := ValidateSettingValue(key, raw)
	if err != nil {
		return nil, err
	}
	if err := s.settings.Set(ctx, key, normalized, reqctx.UserID(ctx)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy mục cấu hình")
		}
		return nil, err
	}
	after := &v1.Setting{Key: key, Value: normalized, Description: before.Description, UpdatedAt: time.Now()}
	auditCatalog(ctx, s.audit, AuditUpdate, "setting", 0,
		map[string]any{"key": key, "value": before.Value}, map[string]any{"key": key, "value": normalized})
	return after, nil
}

func (s *settingAdminService) find(ctx context.Context, key string) (*v1.Setting, error) {
	all, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, st := range all {
		if st.Key == key {
			return st, nil
		}
	}
	return nil, rescode.New(rescode.NotFound, "Không tìm thấy mục cấu hình")
}

// ValidateSettingValue kiểm tra giá trị JSON theo kiểu của khóa và trả giá trị đã chuẩn hóa:
//   - khóa dạng số: số nguyên trong khoảng cho phép;
//   - subscription.remind_before_days: mảng số nguyên dương, không trùng, sắp xếp giảm dần.
func ValidateSettingValue(key string, raw json.RawMessage) (json.RawMessage, error) {
	rule, ok := settingRules[key]
	if !ok {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy mục cấu hình")
	}
	if !rule.isSlice {
		var f float64
		if err := json.Unmarshal(raw, &f); err != nil || f != math.Trunc(f) {
			return nil, rescode.New(rescode.BadRequest, "Giá trị phải là số nguyên")
		}
		if f < float64(rule.min) || f > float64(rule.max) {
			return nil, rescode.New(rescode.BadRequest, fmt.Sprintf("Giá trị phải từ %d đến %d", rule.min, rule.max))
		}
		return json.RawMessage(fmt.Sprintf("%d", int(f))), nil
	}

	var floats []float64
	if err := json.Unmarshal(raw, &floats); err != nil {
		return nil, rescode.New(rescode.BadRequest, "Giá trị phải là danh sách số nguyên")
	}
	arr := make([]int, 0, len(floats))
	for _, f := range floats {
		if f != math.Trunc(f) {
			return nil, rescode.New(rescode.BadRequest, "Giá trị phải là danh sách số nguyên")
		}
		if f < float64(rule.min) || f > float64(rule.max) {
			return nil, rescode.New(rescode.BadRequest, fmt.Sprintf("Mỗi giá trị phải từ %d đến %d", rule.min, rule.max))
		}
		arr = append(arr, int(f))
	}
	if len(arr) == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập ít nhất một giá trị")
	}
	if len(arr) > rule.maxItems {
		return nil, rescode.New(rescode.BadRequest, fmt.Sprintf("Tối đa %d giá trị", rule.maxItems))
	}
	slices.Sort(arr)
	arr = slices.Compact(arr)
	slices.Reverse(arr)
	out, _ := json.Marshal(arr)
	return out, nil
}
