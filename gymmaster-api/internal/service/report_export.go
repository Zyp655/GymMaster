package service

import (
	"context"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	v1 "gymmaster-api/api/report/v1"
	"gymmaster-api/internal/pkg/rescode"
)

// Xuất báo cáo Excel: mỗi file có sheet "Tổng hợp" + sheet dữ liệu chi tiết.
// Tiêu đề in đậm, cố định dòng tiêu đề, bộ lọc, độ rộng cột, định dạng tiền VND và ngày dd/mm/yyyy.

const reportExportMaxRows = 50000

type reportColKind int

const (
	reportColText reportColKind = iota
	reportColInt
	reportColMoney
	reportColDate
	reportColDateTime
	reportColPct
)

type reportCol struct {
	Title string
	Width float64
	Kind  reportColKind
}

type reportXLSX struct {
	f      *excelize.File
	styles map[reportColKind]int
	title  int
	header int
	bold   int
	total  map[reportColKind]int
}

func newReportXLSX() (*reportXLSX, error) {
	f := excelize.NewFile()
	x := &reportXLSX{f: f, styles: map[reportColKind]int{}, total: map[reportColKind]int{}}
	border := []excelize.Border{
		{Type: "left", Color: "D9D9D9", Style: 1}, {Type: "right", Color: "D9D9D9", Style: 1},
		{Type: "top", Color: "D9D9D9", Style: 1}, {Type: "bottom", Color: "D9D9D9", Style: 1},
	}
	money := `#,##0" ₫"`
	intFmt := "#,##0"
	date := "dd/mm/yyyy"
	dt := "dd/mm/yyyy hh:mm"
	pct := `0.0"%"`
	defs := map[reportColKind]*excelize.Style{
		reportColText:     {Border: border},
		reportColInt:      {Border: border, CustomNumFmt: &intFmt},
		reportColMoney:    {Border: border, CustomNumFmt: &money},
		reportColDate:     {Border: border, CustomNumFmt: &date, Alignment: &excelize.Alignment{Horizontal: "center"}},
		reportColDateTime: {Border: border, CustomNumFmt: &dt, Alignment: &excelize.Alignment{Horizontal: "center"}},
		reportColPct:      {Border: border, CustomNumFmt: &pct},
	}
	var err error
	for k, st := range defs {
		if x.styles[k], err = f.NewStyle(st); err != nil {
			return nil, err
		}
		tst := *st
		tst.Font = &excelize.Font{Bold: true}
		tst.Fill = excelize.Fill{Type: "pattern", Color: []string{"FFF4E5"}, Pattern: 1}
		if x.total[k], err = f.NewStyle(&tst); err != nil {
			return nil, err
		}
	}
	if x.title, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14, Color: "C2410C"}}); err != nil {
		return nil, err
	}
	if x.bold, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}}); err != nil {
		return nil, err
	}
	x.header, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F97316"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    border,
	})
	return x, err
}

// sheet tạo sheet mới (sheet đầu tiên đổi tên từ "Sheet1"), ghi tiêu đề + dòng mô tả.
func (x *reportXLSX) sheet(name, title, subtitle string) error {
	if x.f.SheetCount == 1 && x.f.GetSheetName(0) == "Sheet1" {
		if err := x.f.SetSheetName("Sheet1", name); err != nil {
			return err
		}
	} else if _, err := x.f.NewSheet(name); err != nil {
		return err
	}
	_ = x.f.SetCellValue(name, "A1", title)
	_ = x.f.SetCellStyle(name, "A1", "A1", x.title)
	_ = x.f.SetCellValue(name, "A2", subtitle)
	return nil
}

func reportCell(col, row int) string {
	c, _ := excelize.CoordinatesToCellName(col, row)
	return c
}

// excelTime: excelize quy đổi theo đồng hồ UTC → truyền giờ VN dưới dạng "giờ tường" UTC.
func excelTime(t time.Time) time.Time {
	t = t.In(reportLoc)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
}

func (x *reportXLSX) value(sheet string, col, row int, kind reportColKind, v any, style int) {
	cell := reportCell(col, row)
	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			v = ""
		} else if kind == reportColDate {
			v = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		} else {
			v = excelTime(t)
		}
	case *time.Time:
		if t == nil {
			v = ""
		} else if kind == reportColDate {
			v = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		} else {
			v = excelTime(*t)
		}
	}
	_ = x.f.SetCellValue(sheet, cell, v)
	_ = x.f.SetCellStyle(sheet, cell, cell, style)
}

// table ghi bảng (tiêu đề + dữ liệu) bắt đầu từ dòng startRow; totals != nil → thêm dòng tổng in đậm.
// Trả dòng kế tiếp còn trống.
func (x *reportXLSX) table(sheet string, startRow int, cols []reportCol, rows [][]any, totals []any, freeze bool) int {
	for i, c := range cols {
		cell := reportCell(i+1, startRow)
		_ = x.f.SetCellValue(sheet, cell, c.Title)
		_ = x.f.SetCellStyle(sheet, cell, cell, x.header)
		colName, _ := excelize.ColumnNumberToName(i + 1)
		w := c.Width
		if w == 0 {
			w = 16
		}
		// Không thu hẹp cột đã rộng hơn (nhiều bảng trên cùng sheet).
		if cur, err := x.f.GetColWidth(sheet, colName); err != nil || cur < w {
			_ = x.f.SetColWidth(sheet, colName, colName, w)
		}
	}
	_ = x.f.SetRowHeight(sheet, startRow, 22)
	r := startRow + 1
	for _, row := range rows {
		for i, v := range row {
			x.value(sheet, i+1, r, cols[i].Kind, v, x.styles[cols[i].Kind])
		}
		r++
	}
	if totals != nil {
		for i, v := range totals {
			x.value(sheet, i+1, r, cols[i].Kind, v, x.total[cols[i].Kind])
		}
		r++
	}
	if freeze {
		_ = x.f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: startRow, TopLeftCell: reportCell(1, startRow+1), ActivePane: "bottomLeft"})
		if len(rows) > 0 {
			_ = x.f.AutoFilter(sheet, reportCell(1, startRow)+":"+reportCell(len(cols), startRow+len(rows)), nil)
		}
	}
	return r + 1
}

// section ghi tiêu đề nhỏ (in đậm) cho một bảng trong sheet tổng hợp.
func (x *reportXLSX) section(sheet string, row int, title string) int {
	cell := reportCell(1, row)
	_ = x.f.SetCellValue(sheet, cell, title)
	_ = x.f.SetCellStyle(sheet, cell, cell, x.bold)
	return row + 1
}

func (x *reportXLSX) bytes() ([]byte, error) {
	x.f.SetActiveSheet(0)
	buf, err := x.f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	_ = x.f.Close()
	return buf.Bytes(), nil
}

var reportExportNames = map[string]string{
	"revenue": "doanh-thu", "members": "hoi-vien", "checkins": "check-in", "subscriptions": "goi-tap", "invoices": "hoa-don",
}

func (s *reportService) Export(ctx context.Context, req *v1.ExportReq) ([]byte, string, error) {
	ids, r, err := s.prepare(ctx, req.Filter)
	if err != nil {
		return nil, "", err
	}
	slug, ok := reportExportNames[req.Type]
	if !ok {
		return nil, "", rescode.New(rescode.BadRequest, "Loại báo cáo không hợp lệ")
	}
	names, err := s.repo.BranchNames(ctx)
	if err != nil {
		return nil, "", err
	}
	branchLabel := "Tất cả chi nhánh"
	if req.BranchID > 0 {
		branchLabel = "Chi nhánh: " + names[req.BranchID]
	} else if ids != nil {
		branchLabel = "Các chi nhánh được phân quyền"
	}
	subtitle := fmt.Sprintf("Kỳ báo cáo: %s – %s · %s · Xuất lúc %s", r.From.Format("02/01/2006"), r.To.Format("02/01/2006"),
		branchLabel, s.now().In(reportLoc).Format("02/01/2006 15:04"))

	x, err := newReportXLSX()
	if err != nil {
		return nil, "", err
	}
	switch req.Type {
	case "revenue":
		err = s.exportRevenue(ctx, x, ids, r, subtitle)
	case "members":
		err = s.exportMembers(ctx, x, ids, r, subtitle, names)
	case "checkins":
		err = s.exportCheckins(ctx, x, ids, r, subtitle)
	case "subscriptions":
		err = s.exportSubscriptions(ctx, x, ids, r, subtitle)
	case "invoices":
		err = s.exportInvoices(ctx, x, ids, r, subtitle)
	}
	if err != nil {
		return nil, "", err
	}
	data, err := x.bytes()
	if err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("bao-cao-%s_%s_%s.xlsx", slug, r.From.Format("20060102"), r.To.Format("20060102"))
	return data, filename, nil
}

func reportLimitNote(n int) string {
	if n >= reportExportMaxRows {
		return fmt.Sprintf("Lưu ý: chỉ xuất %d dòng đầu tiên, hãy thu hẹp khoảng thời gian.", reportExportMaxRows)
	}
	return ""
}

func (s *reportService) exportRevenue(ctx context.Context, x *reportXLSX, ids []uint, r reportRange, subtitle string) error {
	groupBy := "day"
	if r.Days() > 62 {
		groupBy = "month"
	}
	res, err := s.revenue(ctx, ids, r, groupBy)
	if err != nil {
		return err
	}
	const sum = "Tổng hợp"
	if err := x.sheet(sum, "BÁO CÁO DOANH THU (THỰC THU)", subtitle); err != nil {
		return err
	}
	kv := []reportCol{{Title: "Chỉ tiêu", Width: 34}, {Title: "Giá trị", Width: 20, Kind: reportColMoney}}
	row := x.table(sum, 4, kv, [][]any{
		{"Tổng thu", res.Totals.Payments},
		{"Hoàn tiền", res.Totals.Refunds},
		{"Thực thu (thu − hoàn)", res.Totals.Net},
		{fmt.Sprintf("Thực thu kỳ trước (%s – %s)", reportDMY(res.Previous.From), reportDMY(res.Previous.To)), res.Previous.Net},
	}, nil, false)
	if res.Previous.ChangePct != nil {
		x.value(sum, 1, row-1, reportColText, "So với kỳ trước", x.styles[reportColText])
		x.value(sum, 2, row-1, reportColPct, *res.Previous.ChangePct, x.styles[reportColPct])
		row++
	}
	nv := []reportCol{{Title: "Hạng mục", Width: 34}, {Title: "Số tiền", Width: 20, Kind: reportColMoney}, {Title: "Tỷ trọng", Width: 12, Kind: reportColPct}}
	toRows := func(list []v1.NameValue) [][]any {
		out := make([][]any, 0, len(list))
		for _, it := range list {
			out = append(out, []any{it.Label, it.Value, it.Pct})
		}
		return out
	}
	row = x.section(sum, row, "Theo loại doanh thu")
	row = x.table(sum, row, nv, toRows(res.ByType), []any{"Tổng", res.Totals.Net, 100.0}, false)
	row = x.section(sum, row, "Theo phương thức thanh toán")
	row = x.table(sum, row, nv, toRows(res.ByMethod), []any{"Tổng", res.Totals.Net, 100.0}, false)
	row = x.section(sum, row, "Theo chi nhánh")
	br := make([][]any, 0, len(res.ByBranch))
	for _, b := range res.ByBranch {
		br = append(br, []any{b.BranchName, b.Amount, b.Pct})
	}
	x.table(sum, row, nv, br, []any{"Tổng", res.Totals.Net, 100.0}, false)

	const det = "Theo thời gian"
	title := "DOANH THU THEO NGÀY"
	if groupBy == "month" {
		title = "DOANH THU THEO THÁNG"
	}
	if err := x.sheet(det, title, subtitle); err != nil {
		return err
	}
	// Các cột theo loại là THỰC THU (đã trừ hoàn tiền) → đặt sau cột "Thực thu" và ghi rõ trong tiêu đề, tránh
	// đọc nhầm là phân rã của "Tổng thu" (gộp, chưa trừ hoàn). Σ các cột theo loại = cột "Thực thu".
	_ = x.f.SetCellValue(det, "A3", "Ghi chú: các cột \"Thực thu – …\" là thực thu theo loại (đã trừ hoàn tiền); cộng lại bằng cột \"Thực thu\".")
	cols := []reportCol{
		{Title: "Kỳ", Width: 14}, {Title: "Tổng thu", Width: 18, Kind: reportColMoney},
		{Title: "Hoàn tiền", Width: 16, Kind: reportColMoney}, {Title: "Thực thu", Width: 18, Kind: reportColMoney},
		{Title: "Thực thu – Gói tập", Width: 20, Kind: reportColMoney}, {Title: "Thực thu – PT", Width: 18, Kind: reportColMoney},
		{Title: "Thực thu – Sản phẩm", Width: 20, Kind: reportColMoney}, {Title: "Thực thu – Phí", Width: 18, Kind: reportColMoney},
		{Title: "Thực thu – Khác", Width: 18, Kind: reportColMoney},
	}
	var rows [][]any
	var t v1.RevenuePoint
	for _, p := range res.Series {
		label := reportDMY(p.Period)
		if groupBy == "month" {
			label = p.Label
		}
		rows = append(rows, []any{label, p.Payments, p.Refunds, p.Total, p.Membership, p.PT, p.Product, p.Fee, p.Other})
		t.Membership += p.Membership
		t.PT += p.PT
		t.Product += p.Product
		t.Fee += p.Fee
		t.Other += p.Other
		t.Payments += p.Payments
		t.Refunds += p.Refunds
		t.Total += p.Total
	}
	x.table(det, 4, cols, rows, []any{"Tổng", t.Payments, t.Refunds, t.Total, t.Membership, t.PT, t.Product, t.Fee, t.Other}, true)
	return nil
}

func reportDMY(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.Format("02/01/2006")
}

func (s *reportService) exportMembers(ctx context.Context, x *reportXLSX, ids []uint, r reportRange, subtitle string, names map[uint]string) error {
	sc := r.scope(ids)
	dims, err := s.repo.MemberDistribution(ctx, sc, true)
	if err != nil {
		return err
	}
	status, err := s.repo.MemberStatusCounts(ctx, ids)
	if err != nil {
		return err
	}
	list, err := s.repo.ExportMembers(ctx, sc, reportExportMaxRows)
	if err != nil {
		return err
	}
	var res v1.MembersRes
	reportBuildMemberDims(&res, dims, names)
	sm := map[string]int64{}
	for _, row := range status {
		sm[row.Key] += row.Count
	}
	res.Status, res.TotalMembers = reportNameCounts(sm, reportStatusOrder, reportStatusLabel)

	const sum = "Tổng hợp"
	if err := x.sheet(sum, "BÁO CÁO HỘI VIÊN", subtitle); err != nil {
		return err
	}
	_ = x.f.SetCellValue(sum, "A3", reportLimitNote(len(list)))
	cc := []reportCol{{Title: "Nhóm", Width: 30}, {Title: "Số hội viên", Width: 14, Kind: reportColInt}, {Title: "Tỷ lệ", Width: 12, Kind: reportColPct}}
	toRows := func(list []v1.NameCount) [][]any {
		out := make([][]any, 0, len(list))
		for _, it := range list {
			out = append(out, []any{it.Label, it.Count, it.Pct})
		}
		return out
	}
	row := x.section(sum, 4, fmt.Sprintf("Hội viên mới trong kỳ: %d", len(list)))
	row = x.section(sum, row+1, "Theo nguồn (hội viên mới)")
	row = x.table(sum, row, cc, toRows(res.BySource), nil, false)
	row = x.section(sum, row, "Theo giới tính (hội viên mới)")
	row = x.table(sum, row, cc, toRows(res.ByGender), nil, false)
	row = x.section(sum, row, "Theo độ tuổi (hội viên mới)")
	row = x.table(sum, row, cc, toRows(res.ByAge), nil, false)
	row = x.section(sum, row, "Theo chi nhánh (hội viên mới)")
	row = x.table(sum, row, cc, toRows(res.ByBranch), nil, false)
	row = x.section(sum, row, "Trạng thái hiện tại (toàn bộ hội viên)")
	x.table(sum, row, cc, toRows(res.Status), []any{"Tổng", res.TotalMembers, 100.0}, false)

	const det = "Danh sách"
	if err := x.sheet(det, "DANH SÁCH HỘI VIÊN MỚI", subtitle); err != nil {
		return err
	}
	cols := []reportCol{
		{Title: "Mã HV", Width: 12}, {Title: "Họ tên", Width: 26}, {Title: "Giới tính", Width: 10},
		{Title: "Ngày sinh", Width: 12, Kind: reportColDate}, {Title: "Điện thoại", Width: 14}, {Title: "Email", Width: 26},
		{Title: "Chi nhánh", Width: 20}, {Title: "Nguồn", Width: 14}, {Title: "Trạng thái", Width: 12},
		{Title: "Gói hiện tại", Width: 22}, {Title: "Hết hạn", Width: 12, Kind: reportColDate},
		{Title: "Ngày đăng ký", Width: 17, Kind: reportColDateTime},
	}
	rows := make([][]any, 0, len(list))
	for _, m := range list {
		rows = append(rows, []any{m.MemberCode, m.FullName, reportGenderLabel[m.Gender], m.Dob, m.Phone, m.Email, m.BranchName,
			reportSourceLabel[m.Source], m.Status, m.PlanName, m.EndDate, m.CreatedAt})
	}
	x.table(det, 4, cols, rows, nil, true)
	return nil
}

func (s *reportService) exportCheckins(ctx context.Context, x *reportXLSX, ids []uint, r reportRange, subtitle string) error {
	sc := r.scope(ids)
	heat, err := s.repo.CheckinHeatmap(ctx, sc)
	if err != nil {
		return err
	}
	totals, err := s.repo.CheckinTotals(ctx, sc)
	if err != nil {
		return err
	}
	list, err := s.repo.ExportCheckins(ctx, sc, reportExportMaxRows)
	if err != nil {
		return err
	}
	hm := reportBuildHeatmap(heat, r)

	const sum = "Tổng hợp"
	if err := x.sheet(sum, "BÁO CÁO CHECK-IN", subtitle); err != nil {
		return err
	}
	_ = x.f.SetCellValue(sum, "A3", reportLimitNote(len(list)))
	peak := ""
	if hm.Peak != nil {
		peak = fmt.Sprintf("%s (%d lượt)", hm.Peak.Label, hm.Peak.Count)
	}
	kv := []reportCol{{Title: "Chỉ tiêu", Width: 30}, {Title: "Giá trị", Width: 22}}
	row := x.table(sum, 4, kv, [][]any{
		{"Lượt vào hợp lệ", totals.Allowed},
		{"Lượt bị từ chối", totals.Denied},
		{"Số khách khác nhau (hội viên + vé ngày/tập thử)", totals.UniqueMembers},
		{"Trung bình lượt / ngày", hm.AvgPerDay},
		{"Khung giờ đông nhất", peak},
	}, nil, false)
	row = x.section(sum, row, "Heatmap lượt vào theo thứ × giờ")
	cols := []reportCol{{Title: "Thứ", Width: 30}}
	for h := 0; h < 24; h++ {
		cols = append(cols, reportCol{Title: fmt.Sprintf("%02dh", h), Width: 6, Kind: reportColInt})
	}
	cols = append(cols, reportCol{Title: "Tổng", Width: 10, Kind: reportColInt})
	var hrows [][]any
	for _, d := range hm.Matrix {
		line := []any{d.Label}
		for _, v := range d.Hours {
			line = append(line, v)
		}
		hrows = append(hrows, append(line, d.Total))
	}
	tot := []any{"Tổng"}
	for _, v := range hm.ByHour {
		tot = append(tot, v)
	}
	x.table(sum, row, cols, hrows, append(tot, hm.Total), false)

	const det = "Danh sách"
	if err := x.sheet(det, "NHẬT KÝ RA VÀO", subtitle); err != nil {
		return err
	}
	dcols := []reportCol{
		{Title: "Giờ vào", Width: 17, Kind: reportColDateTime}, {Title: "Giờ ra", Width: 17, Kind: reportColDateTime},
		{Title: "Mã HV", Width: 12}, {Title: "Họ tên", Width: 26}, {Title: "Chi nhánh", Width: 20},
		{Title: "Phương thức", Width: 12}, {Title: "Kết quả", Width: 12}, {Title: "Lý do từ chối", Width: 26}, {Title: "Nhân viên", Width: 22},
	}
	rows := make([][]any, 0, len(list))
	for _, c := range list {
		result := "Hợp lệ"
		if c.Result == "DENIED" {
			result = "Từ chối"
		}
		// Nhãn tiếng Việt dùng chung với màn hình check-in (đủ mọi lý do: vé ngày, tập thử, giờ thấp điểm…).
		deny := CheckinDenyMessage(c.DenyReason)
		if deny == "" {
			deny = c.DenyReason
		}
		rows = append(rows, []any{c.CheckedInAt, c.CheckedOutAt, c.MemberCode, c.MemberName, c.BranchName, c.Method, result, deny, c.StaffName})
	}
	x.table(det, 4, dcols, rows, nil, true)
	return nil
}

var reportSubStatusLabel = map[string]string{
	"PENDING": "Chờ thanh toán", "ACTIVE": "Đang hoạt động", "FROZEN": "Bảo lưu", "EXPIRED": "Hết hạn",
	"CANCELLED": "Đã hủy", "TRANSFERRED": "Đã chuyển nhượng",
}

func (s *reportService) exportSubscriptions(ctx context.Context, x *reportXLSX, ids []uint, r reportRange, subtitle string) error {
	groupBy := "day"
	if r.Days() > 62 {
		groupBy = "month"
	}
	res, err := s.subscriptions(ctx, ids, r, groupBy)
	if err != nil {
		return err
	}
	list, err := s.repo.ExportSubscriptions(ctx, r.scope(ids), reportExportMaxRows)
	if err != nil {
		return err
	}
	const sum = "Tổng hợp"
	if err := x.sheet(sum, "BÁO CÁO GÓI TẬP", subtitle); err != nil {
		return err
	}
	_ = x.f.SetCellValue(sum, "A3", reportLimitNote(len(list)))
	c := res.Counts
	kv := []reportCol{{Title: "Biến động", Width: 30}, {Title: "Số lượng", Width: 14, Kind: reportColInt}}
	row := x.table(sum, 4, kv, [][]any{
		{"Bán mới", c.New}, {"Gia hạn", c.Renewed}, {"Nâng cấp", c.Upgraded}, {"Chuyển nhượng", c.Transferred},
		{"Hủy", c.Cancelled}, {"Hết hạn", c.Expired}, {"Bắt đầu bảo lưu", c.Frozen},
	}, nil, false)
	row = x.section(sum, row, "Doanh số theo gói")
	pcols := []reportCol{
		{Title: "Mã gói", Width: 14}, {Title: "Tên gói", Width: 30}, {Title: "Số gói bán", Width: 12, Kind: reportColInt},
		{Title: "Bán mới", Width: 10, Kind: reportColInt}, {Title: "Gia hạn/nâng cấp/chuyển", Width: 14, Kind: reportColInt},
		{Title: "Giá trị bán", Width: 18, Kind: reportColMoney}, {Title: "Thực thu", Width: 18, Kind: reportColMoney},
		{Title: "Tỷ trọng", Width: 10, Kind: reportColPct},
	}
	var prow [][]any
	var tSold, tNew, tCont int64
	var tAmt, tCol float64
	for _, p := range res.ByPlan {
		prow = append(prow, []any{p.PlanCode, p.PlanName, p.Sold, p.NewSales, p.Continued, p.SoldAmount, p.Collected, p.Pct})
		tSold, tNew, tCont, tAmt, tCol = tSold+p.Sold, tNew+p.NewSales, tCont+p.Continued, tAmt+p.SoldAmount, tCol+p.Collected
	}
	x.table(sum, row, pcols, prow, []any{"Tổng", "", tSold, tNew, tCont, tAmt, tCol, 100.0}, false)

	const det = "Danh sách"
	if err := x.sheet(det, "DANH SÁCH ĐĂNG KÝ GÓI TẬP", subtitle); err != nil {
		return err
	}
	cols := []reportCol{
		{Title: "Mã ĐK", Width: 9, Kind: reportColInt}, {Title: "Mã HV", Width: 12}, {Title: "Hội viên", Width: 26},
		{Title: "Gói", Width: 26}, {Title: "Chi nhánh", Width: 20}, {Title: "Bắt đầu", Width: 12, Kind: reportColDate},
		{Title: "Kết thúc", Width: 12, Kind: reportColDate}, {Title: "Giá", Width: 16, Kind: reportColMoney},
		{Title: "Trạng thái", Width: 16}, {Title: "Người bán", Width: 22}, {Title: "Ngày tạo", Width: 17, Kind: reportColDateTime},
	}
	rows := make([][]any, 0, len(list))
	var total float64
	for _, it := range list {
		rows = append(rows, []any{it.ID, it.MemberCode, it.MemberName, it.PlanName, it.BranchName, it.StartDate, it.EndDate,
			it.Price, reportSubStatusLabel[it.Status], it.SoldBy, it.CreatedAt})
		total += it.Price
	}
	x.table(det, 4, cols, rows, []any{"", "", "Tổng", "", "", "", "", total, "", "", ""}, true)
	return nil
}

var reportInvoiceStatusLabel = map[string]string{
	"UNPAID": "Chưa thu", "PARTIAL": "Thu một phần", "PAID": "Đã thanh toán", "VOID": "Đã hủy", "REFUNDED": "Đã hoàn tiền",
}

var reportInvoiceSourceLabel = map[string]string{"COUNTER": "Quầy", "POS": "POS", "PORTAL": "Cổng hội viên"}

func (s *reportService) exportInvoices(ctx context.Context, x *reportXLSX, ids []uint, r reportRange, subtitle string) error {
	list, err := s.repo.ExportInvoices(ctx, r.scope(ids), reportExportMaxRows)
	if err != nil {
		return err
	}
	type agg struct {
		n                     int64
		total, paid, refunded float64
	}
	byStatus := map[string]*agg{}
	var all agg
	for _, it := range list {
		a, ok := byStatus[it.Status]
		if !ok {
			a = &agg{}
			byStatus[it.Status] = a
		}
		// Hóa đơn đã hủy (VOID) không phải doanh thu/công nợ → không cộng vào dòng "Tổng", chỉ hiện riêng theo trạng thái.
		targets := []*agg{a, &all}
		if it.Status == "VOID" {
			targets = []*agg{a}
		}
		for _, p := range targets {
			p.n++
			p.total += it.Total
			p.paid += it.Paid
			p.refunded += it.Refunded
		}
	}
	const sum = "Tổng hợp"
	if err := x.sheet(sum, "BÁO CÁO HÓA ĐƠN", subtitle); err != nil {
		return err
	}
	_ = x.f.SetCellValue(sum, "A3", reportLimitNote(len(list)))
	cols := []reportCol{
		{Title: "Trạng thái", Width: 20}, {Title: "Số hóa đơn", Width: 12, Kind: reportColInt},
		{Title: "Tổng tiền", Width: 18, Kind: reportColMoney}, {Title: "Đã thu", Width: 18, Kind: reportColMoney},
		{Title: "Đã hoàn", Width: 18, Kind: reportColMoney},
	}
	var srows [][]any
	for _, st := range []string{"PAID", "PARTIAL", "UNPAID", "REFUNDED", "VOID"} {
		if a, ok := byStatus[st]; ok {
			srows = append(srows, []any{reportInvoiceStatusLabel[st], a.n, a.total, a.paid, a.refunded})
		}
	}
	x.table(sum, 4, cols, srows, []any{"Tổng (không gồm HĐ đã hủy)", all.n, all.total, all.paid, all.refunded}, false)

	const det = "Danh sách"
	if err := x.sheet(det, "DANH SÁCH HÓA ĐƠN", subtitle); err != nil {
		return err
	}
	dcols := []reportCol{
		{Title: "Số HĐ", Width: 16}, {Title: "Ngày lập", Width: 17, Kind: reportColDateTime}, {Title: "Mã HV", Width: 12},
		{Title: "Khách hàng", Width: 26}, {Title: "Chi nhánh", Width: 20}, {Title: "Nguồn", Width: 14},
		{Title: "Tạm tính", Width: 16, Kind: reportColMoney}, {Title: "Giảm giá", Width: 14, Kind: reportColMoney},
		{Title: "Xóa nợ (hủy gói)", Width: 16, Kind: reportColMoney}, {Title: "Tổng tiền", Width: 16, Kind: reportColMoney}, {Title: "Đã thu", Width: 16, Kind: reportColMoney},
		{Title: "Đã hoàn", Width: 14, Kind: reportColMoney}, {Title: "Trạng thái", Width: 16}, {Title: "Người lập", Width: 22},
	}
	rows := make([][]any, 0, len(list))
	var sub, disc, wo float64
	for _, it := range list {
		// Xóa nợ khi hủy gói ghi riêng (total = subtotal − discount − write_off) – không lẫn vào giảm giá.
		writeOff := max(0, roundMoneyVND(it.Subtotal-it.Discount-it.Total))
		rows = append(rows, []any{it.InvoiceNo, it.IssuedAt, it.MemberCode, it.MemberName, it.BranchName,
			reportInvoiceSourceLabel[it.Source], it.Subtotal, it.Discount, writeOff, it.Total, it.Paid, it.Refunded,
			reportInvoiceStatusLabel[it.Status], it.IssuedBy})
		if it.Status == "VOID" {
			continue // dòng vẫn liệt kê nhưng không cộng vào tổng
		}
		sub += it.Subtotal
		disc += it.Discount
		wo += writeOff
	}
	x.table(det, 4, dcols, rows, []any{"Tổng (không gồm HĐ đã hủy)", "", "", "", "", "", sub, disc, wo, all.total, all.paid, all.refunded, "", ""}, true)
	return nil
}
