package notification

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/net/ghttp"

	"gymmaster-api/internal/config"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/middleware"
	"gymmaster-api/internal/pkg/realtime"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
)

// StreamController phục vụ kết nối SSE (/v1/notifications/stream).
type StreamController struct {
	tokenMaker token.Maker
	hub        *realtime.Hub
}

func NewStreamController(tokenMaker token.Maker) *StreamController {
	return &StreamController{
		tokenMaker: tokenMaker,
		hub:        realtime.Default(),
	}
}

// StreamHandler trả ghttp.HandlerFunc cho route GET /notifications/stream.
func StreamHandler(tokenMaker token.Maker) ghttp.HandlerFunc {
	return NewStreamController(tokenMaker).Stream
}

// NewStreamHandler là bí danh của StreamHandler.
func NewStreamHandler(tokenMaker token.Maker) ghttp.HandlerFunc {
	return StreamHandler(tokenMaker)
}

// Stream là handler SSE cấp package (dùng cấu hình JWT mặc định hoặc context đã xác thực).
func Stream(r *ghttp.Request) {
	NewStreamController(nil).Stream(r)
}

func (c *ControllerV1) Stream(r *ghttp.Request) {
	NewStreamController(nil).Stream(r)
}

func (c *MemberControllerV1) Stream(r *ghttp.Request) {
	NewStreamController(nil).Stream(r)
}

// Chu kỳ của luồng SSE (biến để test rút ngắn).
var (
	streamPingInterval        = 25 * time.Second
	streamStatusCheckInterval = 45 * time.Second
)

// resolveUserID xác thực người mở luồng; trả kèm payload để biết thời điểm access token hết hạn.
func (s *StreamController) resolveUserID(r *ghttp.Request) (uint, *token.Payload, bool) {
	if auth := reqctx.Auth(r.Context()); auth != nil && auth.UserID > 0 {
		return auth.UserID, auth, true
	}

	rawToken := ""
	if tok, ok := middleware.BearerToken(r.GetHeader(consts.HeaderAuthorization)); ok {
		rawToken = tok
	}
	if rawToken == "" {
		rawToken = strings.TrimSpace(r.GetQuery("token").String())
	}
	if rawToken == "" {
		return 0, nil, false
	}

	maker := s.tokenMaker
	if maker == nil {
		cfg := config.GetConfig()
		if m, err := token.NewJWTMaker(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.AccessTTL); err == nil {
			maker = m
		}
	}
	if maker == nil {
		return 0, nil, false
	}

	payload, err := maker.VerifyToken(rawToken)
	if err != nil || payload == nil || payload.UserID == 0 {
		return 0, nil, false
	}
	// Route SSE không đi qua middleware Authentication nên tự chặn tài khoản đã khóa/xóa.
	if middleware.CheckUserActive(r.Context(), payload.UserID) != nil {
		return 0, nil, false
	}
	return payload.UserID, payload, true
}

// Stream thiết lập luồng Server-Sent Events đẩy thông báo thời gian thực cho người dùng.
func (s *StreamController) Stream(r *ghttp.Request) {
	userID, payload, ok := s.resolveUserID(r)
	if !ok {
		r.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
		r.Response.WriteStatus(http.StatusUnauthorized, `{"code":40100,"message":"Bạn chưa đăng nhập","data":null}`)
		return
	}

	hdr := r.Response.Header()
	hdr.Set("Content-Type", "text/event-stream")
	hdr.Set("Cache-Control", "no-cache")
	hdr.Set("Connection", "keep-alive")
	hdr.Set("X-Accel-Buffering", "no")

	hub := s.hub
	if hub == nil {
		hub = realtime.Default()
	}
	ch, unsubscribe := hub.Subscribe(userID)
	defer unsubscribe()

	w := r.Response.RawWriter()
	flusher, _ := w.(http.Flusher)

	_, _ = fmt.Fprint(w, ": connected\n\n")
	if flusher != nil {
		flusher.Flush()
	}

	ticker := time.NewTicker(streamPingInterval)
	defer ticker.Stop()
	// Kết nối SSE sống lâu hơn access token: định kỳ kiểm tra lại tài khoản (bị khóa/xóa → đóng ngay) và đóng
	// luồng khi access token hết hạn (client làm mới token rồi mở lại luồng).
	statusTicker := time.NewTicker(streamStatusCheckInterval)
	defer statusTicker.Stop()
	var expired <-chan time.Time
	if payload != nil && payload.ExpiresAt != nil {
		expiry := time.NewTimer(time.Until(payload.ExpiresAt.Time))
		defer expiry.Stop()
		expired = expiry.C
	}
	closeWith := func(event, data string) {
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		if flusher != nil {
			flusher.Flush()
		}
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-expired:
			closeWith("auth_expired", `{"code":40101,"message":"Phiên đăng nhập đã hết hạn"}`)
			return
		case <-statusTicker.C:
			// Chỉ đóng khi tài khoản thật sự bị khóa/xóa (401); lỗi DB tạm thời thì giữ kết nối.
			if err := middleware.CheckUserActive(r.Context(), userID); err != nil && gerror.Code(err).Code() == rescode.Unauthorized.Code() {
				closeWith("auth_revoked", `{"code":40100,"message":"Tài khoản đã bị khóa hoặc không còn tồn tại"}`)
				return
			}
		case ev, open := <-ch:
			if !open {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: notification\ndata: %s\n\n", data); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}
