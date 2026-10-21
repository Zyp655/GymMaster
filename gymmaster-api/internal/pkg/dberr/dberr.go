// Package dberr dịch lỗi PostgreSQL/pgx/GORM (do dữ liệu đầu vào gây ra) thành lỗi nghiệp vụ 4xx
// có thông điệp tiếng Việt, để middleware Response không trả 500 cho các trường hợp như trùng khóa,
// khóa ngoại không tồn tại, chuỗi quá dài, số vượt giới hạn kiểu cột...
//
// Bảng ánh xạ (SQLSTATE → mã nghiệp vụ / HTTP):
//
//	23505 unique_violation              → 40900 / 409 "Dữ liệu đã tồn tại (trùng <trường>)"
//	23P01 exclusion_violation           → 40900 / 409 "Dữ liệu bị trùng lặp hoặc xung đột với dữ liệu khác"
//	23503 foreign_key_violation (ghi)   → 40000 / 400 "Dữ liệu tham chiếu không tồn tại (<trường>)"
//	23503 foreign_key_violation (xóa)   → 40900 / 409 "Không thể xóa vì dữ liệu đang được sử dụng"
//	23514 check_violation               → 40000 / 400 "Giá trị không hợp lệ"
//	23502 not_null_violation            → 40000 / 400 "Thiếu giá trị bắt buộc (<trường>)"
//	22001 string_data_right_truncation  → 40000 / 400 "Dữ liệu quá dài"
//	22003 numeric_value_out_of_range    → 40000 / 400 "Giá trị số vượt giới hạn"
//	22P02 invalid_text_representation   → 40000 / 400 "Dữ liệu không đúng định dạng"
//	22007/22008 datetime                → 40000 / 400 "Ngày giờ không hợp lệ"
//	22021/22P05 ký tự không hợp lệ      → 40000 / 400 "Dữ liệu chứa ký tự không hợp lệ"
//	22xxx khác (trừ 22012 chia cho 0)   → 40000 / 400 "Dữ liệu không hợp lệ"
//	40001/40P01/55P03 tranh chấp khóa   → 40900 / 409 "Dữ liệu đang được cập nhật đồng thời, vui lòng thử lại"
//	pgx "unable to encode ... greater than maximum / less than minimum" → 40000 / 400 "Giá trị số vượt giới hạn"
//	gorm.ErrRecordNotFound              → 40400 / 404 "Không tìm thấy dữ liệu"
//
// Lỗi được nhận diện xuyên qua mọi lớp bọc (fmt.Errorf %w, gerror.Wrap, errors.Join) nhờ errors.As/Is.
package dberr

import (
	"errors"
	"regexp"
	"strings"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"gymmaster-api/internal/pkg/fieldlabel"
	"gymmaster-api/internal/pkg/rescode"
)

// Translated là kết quả dịch: mã nghiệp vụ + thông điệp hiển thị cho người dùng.
type Translated struct {
	Code    gcode.Code
	Message string
}

const (
	MsgDuplicate       = "Dữ liệu đã tồn tại"
	MsgExclusion       = "Dữ liệu bị trùng lặp hoặc xung đột với dữ liệu khác"
	MsgFKMissing       = "Dữ liệu tham chiếu không tồn tại"
	MsgFKInUse         = "Không thể xóa vì dữ liệu đang được sử dụng"
	MsgCheck           = "Giá trị không hợp lệ"
	MsgNotNull         = "Thiếu giá trị bắt buộc"
	MsgTooLong         = "Dữ liệu quá dài"
	MsgNumericRange    = "Giá trị số vượt giới hạn"
	MsgInvalidText     = "Dữ liệu không đúng định dạng"
	MsgInvalidDatetime = "Ngày giờ không hợp lệ"
	MsgInvalidChar     = "Dữ liệu chứa ký tự không hợp lệ"
	MsgInvalidData     = "Dữ liệu không hợp lệ"
	MsgConcurrent      = "Dữ liệu đang được cập nhật đồng thời, vui lòng thử lại"
	MsgNotFound        = "Không tìm thấy dữ liệu"
)

// Translate trả (kết quả, true) nếu err là lỗi DB do dữ liệu đầu vào; (zero, false) nếu không nhận ra
// (lỗi hệ thống thật sự – giữ 500).
func Translate(err error) (Translated, bool) {
	if err == nil {
		return Translated{}, false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return translatePg(pgErr)
	}
	switch {
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return Translated{rescode.Conflict, MsgDuplicate}, true
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		return Translated{rescode.BadRequest, MsgFKMissing}, true
	case errors.Is(err, gorm.ErrCheckConstraintViolated):
		return Translated{rescode.BadRequest, MsgCheck}, true
	case errors.Is(err, gorm.ErrRecordNotFound):
		return Translated{rescode.NotFound, MsgNotFound}, true
	}
	if IsEncodeOverflow(err) {
		return Translated{rescode.BadRequest, MsgNumericRange}, true
	}
	return Translated{}, false
}

// IsEncodeOverflow nhận diện lỗi pgx khi mã hóa tham số vượt miền giá trị của kiểu cột
// (vd. uint64 0xffffffffffffffff vào int8, 2147483648 vào int4). pgx không có kiểu lỗi riêng nên so chuỗi.
func IsEncodeOverflow(err error) bool {
	msg := err.Error()
	if !strings.Contains(msg, "unable to encode") && !strings.Contains(msg, "failed to encode args") {
		return false
	}
	return strings.Contains(msg, "greater than maximum value") ||
		strings.Contains(msg, "less than minimum value") ||
		strings.Contains(msg, "out of range") ||
		strings.Contains(msg, "overflow")
}

func translatePg(e *pgconn.PgError) (Translated, bool) {
	switch e.Code {
	case "23505":
		return Translated{rescode.Conflict, withFields(MsgDuplicate, "trùng ", keyColumns(e))}, true
	case "23P01":
		return Translated{rescode.Conflict, MsgExclusion}, true
	case "23503":
		if strings.Contains(e.Detail, "still referenced") || strings.HasPrefix(e.Message, "update or delete on table") {
			return Translated{rescode.Conflict, MsgFKInUse}, true
		}
		return Translated{rescode.BadRequest, withFields(MsgFKMissing, "", keyColumns(e))}, true
	case "23514":
		return Translated{rescode.BadRequest, MsgCheck}, true
	case "23502":
		var cols []string
		if e.ColumnName != "" {
			cols = []string{e.ColumnName}
		}
		return Translated{rescode.BadRequest, withFields(MsgNotNull, "", cols)}, true
	case "22001":
		return Translated{rescode.BadRequest, MsgTooLong}, true
	case "22003":
		return Translated{rescode.BadRequest, MsgNumericRange}, true
	case "22P02":
		return Translated{rescode.BadRequest, MsgInvalidText}, true
	case "22007", "22008":
		return Translated{rescode.BadRequest, MsgInvalidDatetime}, true
	case "22021", "22P05":
		return Translated{rescode.BadRequest, MsgInvalidChar}, true
	case "22012": // division_by_zero: lỗi logic phía server, không phải do người dùng.
		return Translated{}, false
	case "40001", "40P01", "55P03":
		return Translated{rescode.Conflict, MsgConcurrent}, true
	}
	if strings.HasPrefix(e.Code, "22") {
		return Translated{rescode.BadRequest, MsgInvalidData}, true
	}
	return Translated{}, false
}

// keyColRe lấy danh sách cột trong Detail dạng `Key (branch_id, lower(code::text))=(...)`.
var (
	identRe   = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
	skipIdent = map[string]bool{"lower": true, "upper": true, "coalesce": true, "text": true, "varchar": true, "trim": true, "date": true}
	// Cột phạm vi (scope) thường đi kèm khóa duy nhất tổng hợp – không hữu ích khi hiển thị nếu còn cột khác.
	scopeCols = map[string]bool{"branch_id": true, "deleted_at": true, "tenant_id": true}
)

func keyColumns(e *pgconn.PgError) []string {
	var cols []string
	if i := strings.Index(e.Detail, "Key ("); i >= 0 {
		rest := e.Detail[i+len("Key ("):]
		if j := strings.Index(rest, ")=("); j > 0 {
			for _, part := range strings.Split(rest[:j], ",") {
				for _, id := range identRe.FindAllString(part, -1) {
					if !skipIdent[strings.ToLower(id)] {
						cols = append(cols, id)
						break
					}
				}
			}
		}
	}
	if len(cols) == 0 && e.ConstraintName != "" && e.TableName != "" {
		// Quy ước tên mặc định của Postgres: <bảng>_<cột>_key / <bảng>_<cột>_fkey.
		name := strings.TrimPrefix(e.ConstraintName, e.TableName+"_")
		for _, suf := range []string{"_fkey", "_key", "_idx"} {
			name = strings.TrimSuffix(name, suf)
		}
		if name != e.ConstraintName {
			if _, ok := fieldlabel.Label(name); ok {
				cols = []string{name}
			}
		}
	}
	if len(cols) > 1 {
		filtered := cols[:0:0]
		for _, c := range cols {
			if !scopeCols[c] {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) > 0 {
			cols = filtered
		}
	}
	return cols
}

// withFields ghép nhãn trường vào thông điệp: "Dữ liệu đã tồn tại (trùng mã)".
// Chỉ dùng nhãn tiếng Việt có trong từ điển; không lộ tên cột lạ.
func withFields(base, prefix string, cols []string) string {
	var ls []string
	for _, c := range cols {
		if l, ok := fieldlabel.Label(c); ok {
			ls = append(ls, l)
		}
	}
	if len(ls) == 0 {
		return base
	}
	return base + " (" + prefix + strings.Join(ls, ", ") + ")"
}
