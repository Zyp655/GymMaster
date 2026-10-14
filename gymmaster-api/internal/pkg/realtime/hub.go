// Package realtime cung cấp hub in-process đẩy thông báo thời gian thực qua Server-Sent Events (SSE).
package realtime

import (
	"sync"
	"time"
)

// Event là sự kiện đẩy xuống client qua SSE.
type Event struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Link      string    `json:"link"`
	CreatedAt time.Time `json:"created_at"`
}

// Hub quản lý các kết nối SSE theo userID.
type Hub struct {
	mu      sync.RWMutex
	clients map[uint]map[chan Event]struct{}
}

var defaultHub = NewHub()

// Default trả về hub mặc định toàn tiến trình.
func Default() *Hub {
	return defaultHub
}

// NewHub khởi tạo một Hub mới.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[uint]map[chan Event]struct{}),
	}
}

// Subscribe đăng ký nhận sự kiện cho userID. Trả về channel nhận Event và hàm hủy đăng ký.
func (h *Hub) Subscribe(userID uint) (<-chan Event, func()) {
	ch := make(chan Event, 16)
	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[chan Event]struct{})
	}
	h.clients[userID][ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			if subs, ok := h.clients[userID]; ok {
				delete(subs, ch)
				if len(subs) == 0 {
					delete(h.clients, userID)
				}
			}
			h.mu.Unlock()
			close(ch)
		})
	}
	return ch, unsubscribe
}

// Publish đẩy sự kiện tới mọi kết nối đang mở của ev.UserID (không chặn nếu buffer đầy).
func (h *Hub) Publish(ev Event) {
	h.mu.RLock()
	subs := h.clients[ev.UserID]
	targets := make([]chan Event, 0, len(subs))
	for ch := range subs {
		targets = append(targets, ch)
	}
	h.mu.RUnlock()

	for _, ch := range targets {
		select {
		case ch <- ev:
		default:
		}
	}
}
