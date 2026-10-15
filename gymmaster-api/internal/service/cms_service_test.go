package service

import (
	"strings"
	"testing"
	"time"

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
