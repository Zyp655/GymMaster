package middleware

import (
	"mime"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"gymmaster-api/internal/pkg/rescode"
)

// DefaultHandlerResponse là envelope chung của mọi response JSON.
type DefaultHandlerResponse struct {
	Code    int    `json:"code"    dc:"0 = thành công; lỗi nghiệp vụ 5 chữ số, HTTP status = code/100"`
	Message string `json:"message" dc:"Thông điệp cho người dùng"`
	Data    any    `json:"data"    dc:"Dữ liệu trả về"`
}

var streamContentTypes = []string{"text/event-stream", "application/octet-stream", "multipart/x-mixed-replace"}

func (m *middleware) Response() ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		r.Middleware.Next()

		// Handler đã tự ghi response (file, stream...) thì bỏ qua.
		if r.Response.BufferLength() > 0 || r.Response.BytesWritten() > 0 {
			return
		}
		mediaType, _, _ := mime.ParseMediaType(r.Response.Header().Get("Content-Type"))
		for _, ct := range streamContentTypes {
			if mediaType == ct {
				return
			}
		}

		var (
			err  = r.GetError()
			res  = r.GetHandlerResponse()
			code = gerror.Code(err)
			msg  string
		)

		switch {
		case err != nil:
			if code == gcode.CodeNil {
				code = gcode.CodeInternalError
			}
			msg = userMessage(err, code)
			if isServerError(code) {
				g.Log().Errorf(r.Context(), "%+v", err)
			}
			res = nil
		case r.Response.Status == http.StatusNotFound:
			code, msg = rescode.NotFound, "Không tìm thấy đường dẫn"
		default:
			code, msg = gcode.CodeOK, "OK"
		}

		r.Response.ClearBuffer()
		r.Response.WriteHeader(httpStatus(code))
		r.Response.WriteJson(DefaultHandlerResponse{Code: code.Code(), Message: msg, Data: res})
	}
}

// httpStatus: mã nghiệp vụ 5 chữ số → code/100; mã nội bộ của GoFrame → status tương ứng.
func httpStatus(code gcode.Code) int {
	c := code.Code()
	switch {
	case c == gcode.CodeOK.Code():
		return http.StatusOK
	case c >= 10000 && c <= 59999:
		return c / 100
	case c == gcode.CodeValidationFailed.Code(), c == gcode.CodeMissingParameter.Code(), c == gcode.CodeInvalidParameter.Code():
		return http.StatusBadRequest
	case c == gcode.CodeNotFound.Code():
		return http.StatusNotFound
	case c == gcode.CodeNotAuthorized.Code():
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

func isServerError(code gcode.Code) bool {
	return httpStatus(code) >= http.StatusInternalServerError
}

// userMessage trả thông điệp an toàn cho client: lỗi validate chỉ lấy lỗi đầu tiên,
// lỗi hệ thống không lộ chi tiết nội bộ.
func userMessage(err error, code gcode.Code) string {
	if isServerError(code) {
		return "Hệ thống đang gặp sự cố, vui lòng thử lại sau"
	}
	msg := err.Error()
	if code == gcode.CodeValidationFailed {
		if i := strings.Index(msg, "; "); i > 0 {
			msg = msg[:i]
		}
	}
	return msg
}
