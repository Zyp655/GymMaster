// Package reqctx đọc thông tin người dùng hiện tại từ context.
// Service chỉ phụ thuộc context.Context, không phụ thuộc ghttp.Request.
package reqctx

import (
	"context"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/token"
)

// Auth trả payload của người dùng đã đăng nhập (nil nếu request public).
func Auth(ctx context.Context) *token.Payload {
	p, _ := ctx.Value(consts.CtxAuthKey).(*token.Payload)
	return p
}

// UserID trả id người dùng hiện tại, nil nếu chưa đăng nhập (dùng cho CreatedBy/UpdatedBy).
func UserID(ctx context.Context) *uint {
	if p := Auth(ctx); p != nil {
		id := p.UserID
		return &id
	}
	return nil
}

// WithAuth gắn payload vào context (dùng trong test).
func WithAuth(ctx context.Context, p *token.Payload) context.Context {
	return context.WithValue(ctx, consts.CtxAuthKey, p)
}
