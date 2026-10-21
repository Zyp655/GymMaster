package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/i18n/gi18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gvalid"

	"gymmaster-api/internal/pkg/dberr"
	"gymmaster-api/internal/pkg/fieldlabel"
	"gymmaster-api/internal/pkg/rescode"
)

const (
	msgServerError   = "Hệ thống đang gặp sự cố, vui lòng thử lại sau"
	msgBadFormat     = "Dữ liệu gửi lên không đúng định dạng"
	msgInvalidInput  = "Dữ liệu không hợp lệ"
	i18nRulePrefix   = "gf.gvalid.rule."
	fieldPlaceholder = "{field} "
)

// classifyError quyết định mã nghiệp vụ + thông điệp trả cho client từ lỗi của handler:
//
//   - Lỗi validate (gvalid, mã 51 của GoFrame) → 40000 + thông điệp đầu tiên, tên trường đổi sang nhãn tiếng Việt.
//   - Lỗi đọc/chuyển kiểu request (mã 53/54/66 của GoFrame, panic reflect/gconv khi bind body) → 40000.
//   - Lỗi chưa gắn mã nghiệp vụ / mã nội bộ (500) nhưng thực chất do dữ liệu đầu vào
//     (PgError 23505/23503/22001..., pgx encode tràn số) → 4xx theo bảng của package dberr.
//   - Lỗi nghiệp vụ đã có mã (rescode) → giữ nguyên mã và thông điệp.
//   - Còn lại → 500, thông điệp chung (không lộ chi tiết), log đầy đủ stack.
//
// Lỗi gốc luôn được log phía server (Warning cho lỗi đã dịch sang 4xx, Error cho 500).
func classifyError(r *ghttp.Request, err error) (gcode.Code, string) {
	code := gerror.Code(err)
	switch code {
	case gcode.CodeValidationFailed:
		return rescode.BadRequest, validationMessage(r, err)
	case gcode.CodeInvalidParameter, gcode.CodeMissingParameter, gcode.CodeInvalidRequest:
		g.Log().Warningf(r.Context(), "request không đúng định dạng: %v", err)
		return rescode.BadRequest, msgBadFormat
	}

	if isInternalCode(code) {
		if t, ok := dberr.Translate(err); ok {
			g.Log().Warningf(r.Context(), "lỗi DB do dữ liệu đầu vào → %d %q: %+v", t.Code.Code(), t.Message, err)
			return t.Code, t.Message
		}
		if isBindingError(err) {
			g.Log().Warningf(r.Context(), "không bind được request → 400: %+v", err)
			return rescode.BadRequest, msgBadFormat
		}
		if code == gcode.CodeNil {
			code = gcode.CodeInternalError
		}
	}

	if httpStatus(code) >= http.StatusInternalServerError {
		g.Log().Errorf(r.Context(), "%+v", err)
		return code, msgServerError
	}
	return code, err.Error()
}

// isInternalCode: các mã mà lỗi gốc có thể là lỗi DB/bind chưa được service dịch.
func isInternalCode(code gcode.Code) bool {
	switch code {
	case gcode.CodeNil, gcode.CodeInternalError, gcode.CodeInternalPanic, gcode.CodeDbOperationError,
		gcode.CodeUnknown, gcode.CodeOperationFailed, rescode.Internal:
		return true
	}
	return false
}

// bindingMarkers: dấu hiệu lỗi chuyển kiểu khi GoFrame bind JSON/query vào struct Req
// (vd. gửi số/chuỗi/bool cho trường kiểu struct hoặc mảng struct → panic reflect.Value.Convert
// sang kiểu DTO "v1.Xxx"; json.Number chỉ xuất hiện khi giải mã body JSON).
var bindingMarkers = []string{
	"cannot be converted to type v1.",
	"cannot be converted to type []v1.",
	"cannot be converted to type *v1.",
	"value of type json.Number cannot be converted",
	"json: cannot unmarshal",
}

// isBindingError: lỗi/panic khi bind request vào DTO. Panic có stack đi qua internal/service
// hoặc internal/repository (bug trong nghiệp vụ) vẫn giữ 500 để lộ ra lỗi thật.
func isBindingError(err error) bool {
	msg := err.Error()
	matched := false
	for _, m := range bindingMarkers {
		if strings.Contains(msg, m) {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}
	stack := gerror.Stack(err)
	if stack == "" {
		stack = fmt.Sprintf("%+v", err)
	}
	return !strings.Contains(stack, "internal/service/") && !strings.Contains(stack, "internal/repository/")
}

// validationMessage lấy lỗi validate đầu tiên; nếu thông điệp sinh từ mẫu mặc định
// (resource/i18n/vi/validation.toml, luôn bắt đầu bằng "{field} ") thì thay tên field Go
// (vd. "BranchID") bằng nhãn tiếng Việt ("Chi nhánh"). Thông điệp tùy chỉnh trong tag `v:"...#msg"` giữ nguyên.
func validationMessage(r *ghttp.Request, err error) string {
	var verr gvalid.Error
	if !errors.As(err, &verr) {
		msg := err.Error()
		if i := strings.Index(msg, "; "); i > 0 {
			msg = msg[:i]
		}
		return msg
	}
	key, _ := verr.FirstItem()
	rule, ferr := verr.FirstRule()
	if ferr == nil {
		return msgInvalidInput
	}
	msg := ferr.Error()
	if key == "" || !strings.HasPrefix(msg, key+" ") {
		return msg
	}
	if i := strings.IndexByte(rule, ':'); i >= 0 {
		rule = rule[:i]
	}
	tpl := gi18n.Instance().GetContent(r.Context(), i18nRulePrefix+rule)
	if tpl == "" {
		tpl = gi18n.Instance().GetContent(r.Context(), i18nRulePrefix+"__default__")
	}
	if !strings.HasPrefix(tpl, fieldPlaceholder) {
		return msg
	}
	static := tpl[len(fieldPlaceholder):]
	if i := strings.IndexByte(static, '{'); i >= 0 {
		static = static[:i]
	}
	rest := msg[len(key)+1:]
	if !strings.HasPrefix(rest, static) {
		return msg // thông điệp tùy chỉnh trùng hợp bắt đầu bằng tên field
	}
	label, jsonName := fieldLabel(r, key)
	// Trường mã định danh (id, *_id) vi phạm min/between: "Chi nhánh không hợp lệ" tự nhiên hơn "Chi nhánh phải ≥ 1".
	if (rule == "min" || rule == "between" || rule == "integer") && (jsonName == "id" || strings.HasSuffix(jsonName, "_id")) {
		return fieldlabel.Capitalize(label) + " không hợp lệ"
	}
	return fieldlabel.Capitalize(label) + " " + rest
}

// fieldLabel: tên field Go → tên JSON (tra trong struct Req của handler) → nhãn tiếng Việt.
// Trả kèm tên JSON (snake_case) để nơi gọi biết loại trường.
func fieldLabel(r *ghttp.Request, key string) (label, jsonName string) {
	if h := r.GetServeHandler(); h != nil && h.Handler != nil && h.Handler.Info.Type != nil {
		if t := h.Handler.Info.Type; t.Kind() == reflect.Func && t.NumIn() == 2 {
			jsonName = jsonNameOf(t.In(1), key)
		}
	}
	if jsonName != "" {
		if l, ok := fieldlabel.Label(jsonName); ok {
			return l, jsonName
		}
	}
	if jsonName == "" {
		jsonName = fieldlabel.ToSnake(key)
	}
	if l, ok := fieldlabel.Label(key); ok {
		return l, jsonName
	}
	return "trường \"" + jsonName + "\"", jsonName
}

var jsonNameCache sync.Map // key: reflect.Type + "|" + field → json name

// jsonNameOf tìm (đệ quy qua struct lồng, embedded, slice/map/pointer) field Go tên `field`
// hoặc có tag p/param trùng `field`, trả tên trong tag json.
func jsonNameOf(t reflect.Type, field string) string {
	ck := t.String() + "|" + field
	if v, ok := jsonNameCache.Load(ck); ok {
		return v.(string)
	}
	name := findJSONName(t, field, map[reflect.Type]bool{})
	jsonNameCache.Store(ck, name)
	return name
}

func findJSONName(t reflect.Type, field string, seen map[reflect.Type]bool) string {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Map {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || seen[t] {
		return ""
	}
	seen[t] = true
	var nested []reflect.Type
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		alias := f.Tag.Get("p")
		if alias == "" {
			alias = f.Tag.Get("param")
		}
		if f.Name == field || (alias != "" && alias == field) {
			if j := strings.Split(f.Tag.Get("json"), ",")[0]; j != "" && j != "-" {
				return j
			}
			if alias != "" {
				return alias
			}
			return fieldlabel.ToSnake(f.Name)
		}
		nested = append(nested, f.Type)
	}
	for _, nt := range nested {
		if j := findJSONName(nt, field, seen); j != "" {
			return j
		}
	}
	return ""
}
