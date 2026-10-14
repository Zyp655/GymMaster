package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

// Khóa cấu hình nghiệp vụ (bảng settings, migration 000004).
const (
	SettingCheckinDuplicateWindowMinutes = "checkin.duplicate_window_minutes"
	SettingCheckinQRTTLSeconds           = "checkin.qr_ttl_seconds"
	SettingBookingAdvanceDays            = "booking.advance_days"
	SettingBookingCancelBeforeMinutes    = "booking.cancel_before_minutes"
	SettingBookingNoShowLimit            = "booking.no_show_limit"
	SettingBookingBlockDays              = "booking.block_days"
	SettingSubscriptionRemindBeforeDays  = "subscription.remind_before_days"
	SettingPTCancelBeforeMinutes         = "pt.cancel_before_minutes"
)

// SettingService đọc cấu hình nghiệp vụ, có cache ngắn hạn trong bộ nhớ.
// Mọi module đọc cấu hình qua đây (không hard-code con số trong nghiệp vụ).
type SettingService interface {
	Int(ctx context.Context, key string, def int) int
	IntSlice(ctx context.Context, key string, def []int) []int
	All(ctx context.Context) ([]entity.Setting, error)
	Set(ctx context.Context, key string, value json.RawMessage, updatedBy *uint) error
}

type settingService struct {
	db    *gorm.DB
	ttl   time.Duration
	mu    sync.RWMutex
	cache map[string]json.RawMessage
	at    time.Time
}

func NewSettingService(db *gorm.DB) SettingService {
	return &settingService{db: db, ttl: 30 * time.Second}
}

func (s *settingService) raw(ctx context.Context, key string) (json.RawMessage, bool) {
	s.mu.RLock()
	fresh := s.cache != nil && time.Since(s.at) < s.ttl
	if fresh {
		v, ok := s.cache[key]
		s.mu.RUnlock()
		return v, ok
	}
	s.mu.RUnlock()

	rows, err := s.All(ctx)
	if err != nil {
		return nil, false
	}
	m := make(map[string]json.RawMessage, len(rows))
	for _, r := range rows {
		m[r.Key] = json.RawMessage(r.Value)
	}
	s.mu.Lock()
	s.cache, s.at = m, time.Now()
	s.mu.Unlock()
	v, ok := m[key]
	return v, ok
}

func (s *settingService) Int(ctx context.Context, key string, def int) int {
	var v int
	if raw, ok := s.raw(ctx, key); ok && json.Unmarshal(raw, &v) == nil {
		return v
	}
	return def
}

func (s *settingService) IntSlice(ctx context.Context, key string, def []int) []int {
	var v []int
	if raw, ok := s.raw(ctx, key); ok && json.Unmarshal(raw, &v) == nil {
		return v
	}
	return def
}

func (s *settingService) All(ctx context.Context) ([]entity.Setting, error) {
	var rows []entity.Setting
	err := postgres.Conn(ctx, s.db).Order("key").Find(&rows).Error
	return rows, err
}

func (s *settingService) Set(ctx context.Context, key string, value json.RawMessage, updatedBy *uint) error {
	if !json.Valid(value) {
		return errors.New("giá trị cấu hình không phải JSON hợp lệ")
	}
	res := postgres.Conn(ctx, s.db).Model(&entity.Setting{}).Where("key = ?", key).
		Updates(map[string]any{"value": value, "updated_by": updatedBy, "updated_at": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	s.mu.Lock()
	s.cache = nil
	s.mu.Unlock()
	return nil
}
