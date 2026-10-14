package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	v1 "gymmaster-api/api/member/v1"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
)

// Giới hạn số dòng dữ liệu của một lần nhập (tránh file quá lớn giữ transaction lâu).
const memberImportMaxRows = 2000

// Cột của file mẫu – thứ tự cố định.
var memberImportHeaders = []string{
	"Họ tên*", "SĐT*", "Email", "Giới tính (Nam/Nữ/Khác)", "Ngày sinh (dd/mm/yyyy)", "CCCD", "Địa chỉ", "Nguồn",
}

const memberImportSheet = "Hội viên"

var memberIDCardRegex = regexp.MustCompile(`^[0-9]{9,12}$`)

// memberImportRow là một dòng đã parse hợp lệ (chưa kiểm tra trùng với DB).
type memberImportRow struct {
	FullName string
	Phone    string
	Email    string
	Gender   string
	Dob      *time.Time
	IDCard   string
	Address  string
	Source   string
}

// parseMemberImportRow đọc và validate một dòng Excel (cells theo thứ tự memberImportHeaders).
// Trả lỗi là câu tiếng Việt để hiển thị cho người dùng.
func parseMemberImportRow(cells []string, now time.Time) (*memberImportRow, error) {
	cell := func(i int) string {
		if i < len(cells) {
			return strings.TrimSpace(cells[i])
		}
		return ""
	}
	row := &memberImportRow{
		FullName: cell(0),
		Email:    strings.ToLower(cell(2)),
		IDCard:   cell(5),
		Address:  cell(6),
	}
	if row.FullName == "" {
		return nil, fmt.Errorf("Thiếu họ tên")
	}
	if len([]rune(row.FullName)) > 150 {
		return nil, fmt.Errorf("Họ tên tối đa 150 ký tự")
	}

	rawPhone := cell(1)
	if rawPhone == "" {
		return nil, fmt.Errorf("Thiếu số điện thoại")
	}
	// Ô kiểu số trong Excel làm mất số 0 đầu (912345678) → bổ sung lại.
	if len(rawPhone) == 9 && strings.ContainsRune("35789", rune(rawPhone[0])) {
		if _, err := strconv.Atoi(rawPhone); err == nil {
			rawPhone = "0" + rawPhone
		}
	}
	phone, ok := NormalizeVNPhone(rawPhone)
	if !ok {
		return nil, fmt.Errorf("Số điện thoại không hợp lệ: %s", cell(1))
	}
	row.Phone = phone

	if row.Email != "" {
		if addr, err := mail.ParseAddress(row.Email); err != nil || addr.Address != row.Email || len(row.Email) > 150 {
			return nil, fmt.Errorf("Email không hợp lệ: %s", cell(2))
		}
	}

	gender, ok := memberImportGender(cell(3))
	if !ok {
		return nil, fmt.Errorf("Giới tính không hợp lệ: %s (Nam/Nữ/Khác)", cell(3))
	}
	row.Gender = gender

	if raw := cell(4); raw != "" {
		d, ok := memberImportDate(raw)
		if !ok {
			return nil, fmt.Errorf("Ngày sinh không hợp lệ: %s (dd/mm/yyyy)", raw)
		}
		dob, err := memberValidateDob(d, now)
		if err != nil {
			return nil, err
		}
		row.Dob = dob
	}

	if row.IDCard != "" && !memberIDCardRegex.MatchString(row.IDCard) {
		return nil, fmt.Errorf("Số CCCD/CMND gồm 9-12 chữ số")
	}
	if len([]rune(row.Address)) > 255 {
		return nil, fmt.Errorf("Địa chỉ tối đa 255 ký tự")
	}

	source, ok := memberImportSource(cell(7))
	if !ok {
		return nil, fmt.Errorf("Nguồn không hợp lệ: %s", cell(7))
	}
	row.Source = source
	return row, nil
}

func memberImportGender(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "khác", "khac", "other":
		return entity.MemberGenderOther, true
	case "nam", "male", "m":
		return entity.MemberGenderMale, true
	case "nữ", "nu", "female", "f":
		return entity.MemberGenderFemale, true
	}
	return "", false
}

func memberImportSource(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "vãng lai", "vang lai", "tự đến", "walk_in", "walk-in":
		return entity.MemberSourceWalkIn, true
	case "giới thiệu", "gioi thieu", "referral":
		return entity.MemberSourceReferral, true
	case "facebook", "fb":
		return entity.MemberSourceFacebook, true
	case "website", "web":
		return entity.MemberSourceWebsite, true
	case "khác", "khac", "other":
		return entity.MemberSourceOther, true
	}
	return "", false
}

// memberImportDate nhận dd/mm/yyyy (khuyến nghị), yyyy-mm-dd hoặc số serial ngày của Excel.
func memberImportDate(s string) (time.Time, bool) {
	for _, layout := range []string{"02/01/2006", "2/1/2006", "02-01-2006", "2-1-2006", "2006-01-02"} {
		if d, err := time.Parse(layout, s); err == nil {
			return d, true
		}
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 0 && f < 2958466 {
		if d, err := excelize.ExcelDateToTime(f, false); err == nil {
			return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC), true
		}
	}
	return time.Time{}, false
}

// readMemberImportRows đọc sheet đầu tiên, trả các dòng dữ liệu (bỏ dòng tiêu đề) kèm số dòng Excel.
func readMemberImportRows(r io.Reader) (rows [][]string, err error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "File không đúng định dạng Excel (.xlsx)")
	}
	defer func() { _ = f.Close() }()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, rescode.New(rescode.BadRequest, "File Excel không có dữ liệu")
	}
	// RawCellValue: ô ngày trả số serial thay vì chuỗi đã định dạng theo locale.
	all, err := f.GetRows(sheets[0], excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Không đọc được file Excel")
	}
	if len(all) <= 1 {
		return [][]string{}, nil
	}
	return all[1:], nil
}

func memberImportBlankRow(cells []string) bool {
	for _, c := range cells {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func (s *memberService) Import(ctx context.Context, branchID uint, file io.Reader) (*v1.ImportRes, error) {
	if _, err := s.checkBranch(ctx, branchID); err != nil {
		return nil, err
	}
	rows, err := readMemberImportRows(file)
	if err != nil {
		return nil, err
	}

	res := &v1.ImportRes{Errors: []v1.ImportError{}}
	type candidate struct {
		excelRow int
		row      *memberImportRow
	}
	var (
		candidates []candidate
		seen       = map[string]int{} // phone -> dòng Excel đầu tiên
		dataRows   = 0
		now        = s.now()
	)
	for i, cells := range rows {
		excelRow := i + 2 // dòng 1 là tiêu đề
		if memberImportBlankRow(cells) {
			continue
		}
		dataRows++
		if dataRows > memberImportMaxRows {
			return nil, rescode.New(rescode.BadRequest, fmt.Sprintf("Mỗi lần chỉ nhập tối đa %d dòng", memberImportMaxRows))
		}
		row, err := parseMemberImportRow(cells, now)
		if err != nil {
			res.Errors = append(res.Errors, v1.ImportError{Row: excelRow, Message: err.Error()})
			continue
		}
		if first, ok := seen[row.Phone]; ok {
			res.Errors = append(res.Errors, v1.ImportError{Row: excelRow, Message: fmt.Sprintf("SĐT %s trùng với dòng %d", row.Phone, first)})
			continue
		}
		seen[row.Phone] = excelRow
		candidates = append(candidates, candidate{excelRow: excelRow, row: row})
	}
	if dataRows == 0 {
		return nil, rescode.New(rescode.BadRequest, "File Excel không có dòng dữ liệu nào")
	}

	phones := make([]string, 0, len(candidates))
	for _, c := range candidates {
		phones = append(phones, c.row.Phone)
	}
	existing, err := s.memberRepo.ExistingPhones(ctx, phones)
	if err != nil {
		return nil, err
	}

	actor := reqctx.UserID(ctx)
	var toCreate []*entity.Member
	for _, c := range candidates {
		if existing[c.row.Phone] {
			res.Errors = append(res.Errors, v1.ImportError{Row: c.excelRow, Message: fmt.Sprintf("SĐT %s đã tồn tại trong hệ thống", c.row.Phone)})
			continue
		}
		toCreate = append(toCreate, &entity.Member{
			BaseEntity:   entity.BaseEntity{CreatedBy: actor, UpdatedBy: actor},
			FullName:     c.row.FullName,
			Gender:       c.row.Gender,
			Dob:          c.row.Dob,
			Phone:        c.row.Phone,
			Email:        c.row.Email,
			IDCard:       c.row.IDCard,
			Address:      c.row.Address,
			HomeBranchID: branchID,
			Source:       c.row.Source,
			Status:       entity.MemberStatusActive,
		})
	}

	if len(toCreate) > 0 {
		err = s.tx.WithTx(ctx, func(ctx context.Context) error {
			for _, m := range toCreate {
				if err := s.memberRepo.Create(ctx, m); err != nil {
					return mapDuplicate(err, "Có SĐT vừa được thêm bởi người khác, vui lòng nhập lại")
				}
			}
			s.audit.Log(ctx, AuditImport, auditEntityMember, 0, nil, map[string]any{
				"branch_id": branchID, "created": len(toCreate), "skipped": len(res.Errors),
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	memberImportSortErrors(res.Errors)
	res.Created = len(toCreate)
	res.Skipped = len(res.Errors)
	return res, nil
}

func memberImportSortErrors(errs []v1.ImportError) {
	// Sắp theo số dòng (insertion sort – danh sách nhỏ).
	for i := 1; i < len(errs); i++ {
		for j := i; j > 0 && errs[j].Row < errs[j-1].Row; j-- {
			errs[j], errs[j-1] = errs[j-1], errs[j]
		}
	}
}

// ImportTemplate sinh file Excel mẫu: sheet dữ liệu (tiêu đề + dropdown) và sheet hướng dẫn.
func (s *memberService) ImportTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SetSheetName("Sheet1", memberImportSheet); err != nil {
		return nil, err
	}
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F97316"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}
	// Định dạng Text (@) cho SĐT, ngày sinh, CCCD để Excel không làm mất số 0 đầu / tự đổi ngày.
	textStyle, err := f.NewStyle(&excelize.Style{NumFmt: 49})
	if err != nil {
		return nil, err
	}
	for i, h := range memberImportHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(memberImportSheet, cell, h); err != nil {
			return nil, err
		}
	}
	_ = f.SetCellStyle(memberImportSheet, "A1", "H1", headerStyle)
	widths := map[string]float64{"A": 28, "B": 16, "C": 28, "D": 22, "E": 22, "F": 16, "G": 36, "H": 16}
	for col, w := range widths {
		_ = f.SetColWidth(memberImportSheet, col, col, w)
	}
	for _, col := range []string{"B", "E", "F"} {
		_ = f.SetCellStyle(memberImportSheet, col+"2", col+"2001", textStyle)
	}
	_ = f.SetPanes(memberImportSheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	gender := excelize.NewDataValidation(true)
	gender.Sqref = "D2:D2001"
	_ = gender.SetDropList([]string{"Nam", "Nữ", "Khác"})
	_ = f.AddDataValidation(memberImportSheet, gender)
	source := excelize.NewDataValidation(true)
	source.Sqref = "H2:H2001"
	_ = source.SetDropList([]string{"Vãng lai", "Giới thiệu", "Facebook", "Website", "Khác"})
	_ = f.AddDataValidation(memberImportSheet, source)

	guide := "Hướng dẫn"
	if _, err := f.NewSheet(guide); err != nil {
		return nil, err
	}
	lines := []string{
		"HƯỚNG DẪN NHẬP HỘI VIÊN",
		"1. Nhập dữ liệu ở sheet \"Hội viên\", bắt đầu từ dòng 2. Không đổi thứ tự cột, không xóa dòng tiêu đề.",
		"2. Cột có dấu * là bắt buộc: Họ tên, SĐT.",
		"3. SĐT: số Việt Nam 10 chữ số (VD 0912345678). SĐT đã có trong hệ thống hoặc trùng trong file sẽ bị bỏ qua.",
		"4. Giới tính: Nam / Nữ / Khác (để trống = Khác).",
		"5. Ngày sinh: dd/mm/yyyy (VD 15/08/1995), không được ở tương lai.",
		"6. CCCD: 9-12 chữ số (tùy chọn).",
		"7. Nguồn: Vãng lai / Giới thiệu / Facebook / Website / Khác (để trống = Vãng lai).",
		fmt.Sprintf("8. Mỗi lần nhập tối đa %d dòng. Hội viên được gán vào chi nhánh chọn khi tải file lên.", memberImportMaxRows),
	}
	for i, l := range lines {
		_ = f.SetCellValue(guide, fmt.Sprintf("A%d", i+1), l)
	}
	_ = f.SetColWidth(guide, "A", "A", 110)
	f.SetActiveSheet(0)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
