package realtime_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gymmaster-api/internal/pkg/realtime"
)

func TestHubSubscribePublish(t *testing.T) {
	h := realtime.NewHub()
	ch1, unsub1 := h.Subscribe(42)
	defer unsub1()
	ch2, unsub2 := h.Subscribe(99)
	defer unsub2()

	ev := realtime.Event{
		ID:        1,
		UserID:    42,
		Type:      "PAYMENT_SUCCESS",
		Title:     "Thanh toán thành công",
		Body:      "Hóa đơn HD001 đã được thanh toán",
		Link:      "/me/invoices",
		CreatedAt: time.Now(),
	}
	h.Publish(ev)

	select {
	case got := <-ch1:
		require.Equal(t, ev.Title, got.Title)
		require.Equal(t, uint(42), got.UserID)
	case <-time.After(time.Second):
		t.Fatal("expected event on ch1")
	}

	select {
	case <-ch2:
		t.Fatal("did not expect event on ch2")
	default:
	}

	unsub1()
	unsub1() // idempotent
}
