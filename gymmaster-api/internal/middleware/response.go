package middleware

import (
	"mime"
	"net/http"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/net/ghttp"

	"gymmaster-api/internal/pkg/rescode"
)

// DefaultHandlerResponse là envelope chung của mọi response JSON.
type DefaultHandlerResponse struct {
	Code    int    `json:"code"    dc:"0 = thành công; lỗi nghiệp vụ 5 chữ số (HTTP status suy theo bảng trong package rescode)"`
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
			code gcode.Code
			msg  string
		)

		switch {
		case err != nil:
			code, msg = classifyError(r, err)
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

// httpStatus: mã nghiệp vụ 5 chữ số → bảng rescode.HTTPStatus (status chuẩn, không còn 420/430/440...);
// mã nội bộ của GoFrame → status tương ứng.
func httpStatus(code gcode.Code) int {
	c := code.Code()
	if c == gcode.CodeOK.Code() {
		return http.StatusOK
	}
	if st, ok := rescode.HTTPStatus(c); ok {
		return st
	}
	switch c {
	case gcode.CodeValidationFailed.Code(), gcode.CodeMissingParameter.Code(), gcode.CodeInvalidParameter.Code(),
		gcode.CodeInvalidRequest.Code():
		return http.StatusBadRequest
	case gcode.CodeNotFound.Code():
		return http.StatusNotFound
	case gcode.CodeNotAuthorized.Code():
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
