package service

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestWriteICSEvent_ValidRFC5545Format(t *testing.T) {
	var b strings.Builder
	start := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	writeICSEvent(&b, "pt-101@gymmaster.vn", "Buổi tập PT 1-1", "Tập thân trên", start, end)

	ics := b.String()
	assert.Contains(t, ics, "BEGIN:VEVENT\r\n")
	assert.Contains(t, ics, "UID:pt-101@gymmaster.vn\r\n")
	assert.Contains(t, ics, "DTSTART:20261015T090000Z\r\n")
	assert.Contains(t, ics, "DTEND:20261015T100000Z\r\n")
	assert.Contains(t, ics, "SUMMARY:Buổi tập PT 1-1\r\n")
	assert.Contains(t, ics, "END:VEVENT\r\n")
}

func TestTruncateRunes_VietnameseUTF8(t *testing.T) {
	msg := strings.Repeat("Chào HLV, hôm nay tôi muốn đổi lịch tập sang buổi chiều được không ạ? ", 5)
	out := truncateRunes(msg, 120)
	assert.True(t, utf8.ValidString(out), "kết quả cắt phải là UTF-8 hợp lệ")
	assert.Equal(t, 123, utf8.RuneCountInString(out))
	assert.True(t, strings.HasSuffix(out, "..."))
	assert.Equal(t, "Xin chào", truncateRunes("Xin chào", 120))
	// Đúng 120 ký tự (nhiều byte) → giữ nguyên, không thêm "...".
	exact := strings.Repeat("ệ", 120)
	assert.Equal(t, exact, truncateRunes(exact, 120))
}
