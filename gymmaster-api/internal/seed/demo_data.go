package seed

// Dữ liệu tĩnh cho bộ sinh dữ liệu demo: tên người Việt, địa chỉ, danh mục, thời khóa biểu lớp, ghi chú PT...

var (
	vnFamilyNames   = []string{"Nguyễn", "Trần", "Lê", "Phạm", "Hoàng", "Huỳnh", "Phan", "Vũ", "Võ", "Đặng", "Bùi", "Đỗ", "Hồ", "Ngô", "Dương", "Lý", "Đinh", "Trịnh", "Mai", "Đào", "Lương", "Tạ", "Cao", "Lâm"}
	vnFamilyWeights = []float64{38, 11, 9.5, 7, 5, 4, 4.5, 4, 3, 2, 2, 1.5, 1.3, 1.3, 1, 0.5, 0.6, 0.5, 0.5, 0.5, 0.4, 0.3, 0.4, 0.4}

	vnMaleMiddle   = []string{"Văn", "Đức", "Minh", "Quang", "Hữu", "Thành", "Ngọc", "Xuân", "Hoàng", "Trọng", "Công", "Đình", "Mạnh", "Gia", "Bảo", "Tuấn", "Anh", "Việt", "Duy", "Tiến"}
	vnFemaleMiddle = []string{"Thị", "Thu", "Ngọc", "Thanh", "Minh", "Phương", "Hồng", "Khánh", "Bảo", "Mai", "Thùy", "Hải", "Diệu", "Quỳnh", "Kim", "Lan", "Thảo", "Hà"}

	vnMaleGiven = []string{"Anh", "Bình", "Cường", "Dũng", "Duy", "Đạt", "Đức", "Giang", "Hải", "Hiếu", "Hoàng", "Hùng", "Huy", "Khánh", "Khoa",
		"Kiên", "Long", "Lâm", "Minh", "Nam", "Nghĩa", "Phong", "Phúc", "Quân", "Quang", "Sơn", "Tài", "Thắng", "Thành", "Tiến", "Toàn",
		"Trung", "Tú", "Tuấn", "Tùng", "Việt", "Vinh", "Vũ", "Hưng", "Thịnh", "Đông", "Hậu", "Khải", "Lộc", "Nhật", "Phát", "Thái", "Trí"}
	vnFemaleGiven = []string{"Anh", "Châu", "Chi", "Dung", "Duyên", "Giang", "Hà", "Hạnh", "Hằng", "Hiền", "Hoa", "Hương", "Huyền", "Lan", "Linh",
		"Loan", "Ly", "Mai", "My", "Nga", "Ngân", "Ngọc", "Nhung", "Oanh", "Phương", "Quyên", "Quỳnh", "Thảo", "Thơ", "Thu", "Thủy", "Trang",
		"Trinh", "Tú", "Uyên", "Vân", "Vy", "Yến", "Hồng", "Hoài", "Nhi", "Thư", "Diệp", "Khuê", "An"}

	// Đầu số di động hợp lệ (09x/03x/07x).
	vnPhonePrefixes = []string{"090", "091", "093", "094", "096", "097", "098", "032", "033", "034", "035", "036", "037", "038", "039", "070", "076", "077", "078", "079"}

	emailDomains       = []string{"gmail.com", "yahoo.com", "outlook.com", "icloud.com"}
	emailDomainWeights = []float64{80, 8, 7, 5}

	hanoiStreets = []string{"Trần Duy Hưng, Cầu Giấy", "Xuân Thủy, Cầu Giấy", "Hoàng Quốc Việt, Cầu Giấy", "Nguyễn Khánh Toàn, Cầu Giấy", "Trần Thái Tông, Cầu Giấy",
		"Láng Hạ, Đống Đa", "Chùa Bộc, Đống Đa", "Tây Sơn, Đống Đa", "Nguyễn Chí Thanh, Đống Đa", "Kim Mã, Ba Đình", "Đội Cấn, Ba Đình",
		"Lê Thanh Nghị, Hai Bà Trưng", "Bạch Mai, Hai Bà Trưng", "Minh Khai, Hai Bà Trưng", "Phố Huế, Hai Bà Trưng", "Trần Khát Chân, Hai Bà Trưng",
		"Lạc Long Quân, Tây Hồ", "Nguyễn Trãi, Thanh Xuân", "Khuất Duy Tiến, Thanh Xuân", "Phạm Văn Đồng, Bắc Từ Liêm", "Mỹ Đình, Nam Từ Liêm",
		"Giải Phóng, Hoàng Mai", "Tam Trinh, Hoàng Mai", "Nguyễn Văn Cừ, Long Biên"}
	hcmStreets = []string{"Lê Thánh Tôn, Quận 1", "Nguyễn Thị Minh Khai, Quận 1", "Hai Bà Trưng, Quận 1", "Điện Biên Phủ, Bình Thạnh", "Xô Viết Nghệ Tĩnh, Bình Thạnh",
		"Nguyễn Đình Chiểu, Quận 3", "Võ Văn Tần, Quận 3", "Cách Mạng Tháng Tám, Quận 3", "Nguyễn Trãi, Quận 5", "Phan Xích Long, Phú Nhuận",
		"Hoàng Văn Thụ, Tân Bình", "Cộng Hòa, Tân Bình", "Nguyễn Văn Linh, Quận 7", "Lê Văn Sỹ, Quận 3", "Trần Hưng Đạo, Quận 1",
		"Nguyễn Hữu Thọ, Quận 7", "Quang Trung, Gò Vấp", "Võ Văn Ngân, Thủ Đức"}
)

// ---------------------------------------------------------------------
// Danh mục bán
// ---------------------------------------------------------------------

type planDef struct {
	Code, Name, Desc, Type, Scope string
	Duration, Sessions            int
	Price, TransferFee            float64
	FreezeDays, FreezeTimes       int
	Classes                       bool
	Sort                          int
	GuestPasses                   int // số vé khách mời mỗi tháng
}

var demoPlanDefs = []planDef{
	{Code: "M1", Name: "Gói 1 tháng", Desc: "Tập không giới hạn 30 ngày tại chi nhánh đăng ký, gồm các lớp nhóm.", Type: "TIME", Scope: "SINGLE_BRANCH", Duration: 30, Price: 500000, TransferFee: 100000, FreezeDays: 7, FreezeTimes: 1, Classes: true, Sort: 1},
	{Code: "M3", Name: "Gói 3 tháng", Desc: "Tập không giới hạn 90 ngày tại mọi chi nhánh, gồm các lớp nhóm.", Type: "TIME", Scope: "ALL_BRANCHES", Duration: 90, Price: 1350000, TransferFee: 100000, FreezeDays: 15, FreezeTimes: 1, Classes: true, Sort: 2},
	{Code: "M6", Name: "Gói 6 tháng", Desc: "Tập phòng máy không giới hạn 180 ngày tại chi nhánh đăng ký (không gồm lớp nhóm).", Type: "TIME", Scope: "SINGLE_BRANCH", Duration: 180, Price: 2500000, TransferFee: 150000, FreezeDays: 30, FreezeTimes: 2, Classes: false, Sort: 3, GuestPasses: 1},
	{Code: "M12", Name: "Gói 12 tháng", Desc: "Tập không giới hạn 365 ngày tại mọi chi nhánh, gồm các lớp nhóm.", Type: "TIME", Scope: "ALL_BRANCHES", Duration: 365, Price: 4800000, TransferFee: 200000, FreezeDays: 60, FreezeTimes: 3, Classes: true, Sort: 4, GuestPasses: 2},
	{Code: "S12", Name: "Gói 12 buổi", Desc: "12 lượt tập phòng máy trong 60 ngày tại chi nhánh đăng ký.", Type: "SESSION", Scope: "SINGLE_BRANCH", Duration: 60, Sessions: 12, Price: 600000, FreezeDays: 0, FreezeTimes: 0, Classes: false, Sort: 5},
	{Code: "VIP12", Name: "VIP Toàn hệ thống 12 tháng", Desc: "Tập không giới hạn mọi chi nhánh 365 ngày, ưu tiên lớp nhóm, khăn & nước miễn phí.", Type: "TIME", Scope: "ALL_BRANCHES", Duration: 365, Price: 7900000, TransferFee: 300000, FreezeDays: 90, FreezeTimes: 3, Classes: true, Sort: 6, GuestPasses: 4},
}

type ptPkgDef struct {
	Code, Name, Desc            string
	Sessions, Minutes, Validity int
	Price                       float64
}

var demoPTPkgDefs = []ptPkgDef{
	{Code: "PT10", Name: "PT 10 buổi", Desc: "10 buổi huấn luyện 1 kèm 1, thời hạn 45 ngày.", Sessions: 10, Minutes: 60, Validity: 45, Price: 3500000},
	{Code: "PT20", Name: "PT 20 buổi", Desc: "20 buổi huấn luyện 1 kèm 1, thời hạn 90 ngày.", Sessions: 20, Minutes: 60, Validity: 90, Price: 6400000},
	{Code: "PT36", Name: "PT 36 buổi", Desc: "36 buổi 90 phút, giáo án chuyên sâu, thời hạn 150 ngày.", Sessions: 36, Minutes: 90, Validity: 150, Price: 10800000},
}

type promoDef struct {
	Code, Name, Type, AppliesTo string
	Value, MaxDiscount, MinAmt  float64
	Start, End                  string // YYYY-MM-DD (giờ VN, end tính hết ngày)
	Limit                       int    // 0 = không giới hạn
}

// Các khuyến mãi: 2 đã kết thúc, 1 đang hiệu lực, 1 sắp diễn ra (ngày tính tương đối trong demoPromoDefs()).
func demoPromoDefs(year int) []promoDef {
	y := func(s string) string { return fmtYear(year, s) }
	return []promoDef{
		{Code: "HE2026", Name: "Hè sôi động – giảm 20% gói tập", Type: "PERCENT", AppliesTo: "MEMBERSHIP", Value: 20, MaxDiscount: 1000000, Start: y("06-01"), End: y("08-31"), Limit: 0},
		{Code: "QUOCKHANH", Name: "Mừng Quốc khánh 2/9 – giảm 300.000đ gói từ 3 tháng", Type: "FIXED", AppliesTo: "MEMBERSHIP", Value: 300000, MinAmt: 1000000, Start: y("08-28"), End: y("09-07"), Limit: 80},
		{Code: "CHAODONG", Name: "Chào đông – giảm 10% gói tập", Type: "PERCENT", AppliesTo: "MEMBERSHIP", Value: 10, MaxDiscount: 500000, MinAmt: 500000, Start: y("10-01"), End: y("12-31"), Limit: 100},
		{Code: "TET2027", Name: "Tết 2027 – giảm 15%", Type: "PERCENT", AppliesTo: "ALL", Value: 15, Start: "2027-01-20", End: "2027-02-20", Limit: 0},
	}
}

func fmtYear(year int, md string) string { return itoa(year) + "-" + md }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// ---------------------------------------------------------------------
// POS – đúng bộ 12 sản phẩm của dữ liệu mẫu module POS.
// ---------------------------------------------------------------------

type categoryDef struct {
	Name string
	Sort int
}

var demoCategoryDefs = []categoryDef{{"Đồ uống", 1}, {"Thực phẩm bổ sung", 2}, {"Phụ kiện", 3}}

type supplierDef struct{ Name, Phone, Email, Address, Note string }

var demoSupplierDefs = []supplierDef{
	{"Công ty TNHH Dinh Dưỡng Thể Hình Sài Gòn", "02838123456", "sales@dinhduongthehinh.vn", "45 Nguyễn Thị Minh Khai, Quận 1, TP.HCM", "Thực phẩm bổ sung, phụ kiện tập"},
	{"Nhà phân phối Nước giải khát Hà Nội", "02437891234", "order@nuocgiaikhathn.vn", "120 Giải Phóng, Hai Bà Trưng, Hà Nội", "Nước suối, nước bù khoáng, nước tăng lực"},
}

type productDef struct {
	SKU, Barcode, Name, Unit string
	Category, Supplier       int     // chỉ số trong demoCategoryDefs / demoSupplierDefs
	Cost, Sell, Member       float64 // Member = 0 → NULL
	MinStock                 int
	Target                   [3]int // mức tồn mục tiêu khi nhập hàng (HN01, HN02, HCM01)
	Case                     int    // quy cách nhập (bội số)
	Weight                   float64
}

var demoProductDefs = []productDef{
	{"AQUA-500", "8934588012112", "Nước suối Aquafina 500ml", "chai", 0, 1, 5500, 10000, 9000, 24, [3]int{120, 96, 96}, 24, 25},
	{"REVIVE-500", "8934588063015", "Nước bù khoáng Revive chanh muối 500ml", "chai", 0, 1, 8500, 15000, 13000, 24, [3]int{96, 72, 72}, 24, 15},
	{"POCARI-500", "8997035600010", "Nước bù ion Pocari Sweat 500ml", "chai", 0, 1, 12500, 20000, 18000, 12, [3]int{48, 36, 48}, 24, 10},
	{"REDBULL-250", "8850228000017", "Nước tăng lực Red Bull 250ml", "lon", 0, 1, 9500, 15000, 0, 24, [3]int{48, 48, 48}, 24, 12},
	{"WHEY-ON-5LB", "0748927028669", "Whey ON Gold Standard 100% Whey 5lbs (Double Rich Chocolate)", "hộp", 1, 0, 1550000, 1950000, 1850000, 3, [3]int{6, 4, 5}, 1, 2},
	{"WHEY-ON-1SV", "0748927052701", "Whey ON Gold Standard gói 1 lần dùng 30g", "gói", 1, 0, 30000, 45000, 40000, 20, [3]int{60, 40, 50}, 10, 10},
	{"BCAA-XTEND", "0842595101012", "BCAA Scivation Xtend 30 lần dùng", "hộp", 1, 0, 560000, 750000, 700000, 3, [3]int{6, 5, 5}, 1, 3},
	{"PRE-C4", "0810390028153", "Pre-workout Cellucor C4 Original 30 lần dùng", "hộp", 1, 0, 500000, 690000, 650000, 2, [3]int{5, 4, 4}, 1, 3},
	{"BAR-QUEST", "0888849000012", "Thanh protein Quest Bar 60g", "thanh", 1, 0, 42000, 65000, 60000, 10, [3]int{36, 24, 30}, 12, 9},
	{"TOWEL-GM", "8936100000017", "Khăn tập GymMaster cotton 30x100cm", "cái", 2, 0, 28000, 60000, 50000, 10, [3]int{40, 30, 30}, 10, 5},
	{"GLOVE-PRO", "8936100000024", "Găng tay tập gym có quấn cổ tay (size M/L)", "đôi", 2, 0, 80000, 150000, 135000, 5, [3]int{15, 10, 12}, 5, 3},
	{"SHAKER-700", "8936100000031", "Bình lắc Shaker có lò xo trộn 700ml", "cái", 2, 0, 55000, 120000, 100000, 5, [3]int{20, 12, 15}, 5, 3},
}

// Cặp (SKU, chi nhánh) cố ý để tồn thấp ở cuối kỳ (demo cảnh báo tồn thấp).
var demoLowStockPairs = []struct {
	SKU    string
	Branch int
}{{"PRE-C4", 2}, {"BCAA-XTEND", 1}, {"POCARI-500", 0}, {"GLOVE-PRO", 2}}

// ---------------------------------------------------------------------
// Lớp nhóm
// ---------------------------------------------------------------------

type classTypeDef struct {
	Code, Name, Desc, Color, Level string
	Duration, Capacity             int
}

var demoClassTypeDefs = []classTypeDef{
	{"YOGA", "Yoga cơ bản", "Các tư thế yoga nền tảng, hơi thở và thư giãn, phù hợp người mới.", "#22C55E", "BEGINNER", 60, 20},
	{"ZUMBA", "Zumba Dance", "Nhảy theo nhạc Latin sôi động, đốt mỡ hiệu quả.", "#EC4899", "ALL", 60, 25},
	{"BODYPUMP", "Body Pump", "Tập tạ đòn theo nhạc, tăng sức bền cơ bắp toàn thân.", "#EF4444", "INTERMEDIATE", 60, 20},
	{"SPINNING", "Spinning", "Đạp xe trong nhà cường độ cao theo nhịp nhạc.", "#3B82F6", "ALL", 45, 20},
	{"BOXING", "Boxing Fitness", "Kỹ thuật đấm, né, kết hợp cardio – không đối kháng.", "#F97316", "INTERMEDIATE", 60, 16},
	{"PILATES", "Pilates Mat", "Tăng sức mạnh vùng lõi, cải thiện tư thế.", "#8B5CF6", "ALL", 60, 15},
	{"HIIT", "HIIT Cardio", "Bài tập cường độ cao ngắt quãng 45 phút.", "#EAB308", "ADVANCED", 45, 20},
	{"STRETCH", "Stretching & Recovery", "Giãn cơ, phục hồi sau tập, giảm đau mỏi.", "#14B8A6", "ALL", 45, 20},
}

type roomDef struct {
	Name, Desc string
	Capacity   int
}

var demoRoomDefs = [3][]roomDef{
	{{"Phòng Yoga", "Sàn gỗ, thảm & gạch yoga", 25}, {"Phòng Group X", "Phòng lớp nhóm có gương và âm thanh", 30}, {"Phòng Spinning", "20 xe đạp Spinning", 20}},
	{{"Phòng Yoga", "Sàn gỗ, thảm yoga", 20}, {"Phòng Group X", "Phòng lớp nhóm đa năng", 25}},
	{{"Phòng Yoga", "Sàn gỗ, view thành phố", 25}, {"Phòng Group X", "Phòng lớp nhóm lớn", 35}, {"Phòng Spinning", "24 xe đạp Spinning", 24}},
}

// Thời khóa biểu tuần: room = chỉ số phòng, trainer = chỉ số HLV trong chi nhánh.
type slotDef struct {
	Type     string
	Room     int
	Trainer  int
	Weekdays []int
	Time     string
}

var demoTimetable = [3][]slotDef{
	{ // HN01 – Cầu Giấy
		{"YOGA", 0, 1, []int{1, 3, 5}, "06:00"},
		{"PILATES", 0, 1, []int{2, 4}, "07:00"},
		{"YOGA", 0, 1, []int{2, 4}, "18:00"},
		{"STRETCH", 0, 1, []int{6}, "09:00"},
		{"YOGA", 0, 1, []int{7}, "08:00"},
		{"ZUMBA", 1, 2, []int{1, 3, 5}, "18:00"},
		{"ZUMBA", 1, 2, []int{6}, "08:30"},
		{"BODYPUMP", 1, 0, []int{2, 4}, "18:30"},
		{"BOXING", 1, 3, []int{1, 3}, "19:30"},
		{"HIIT", 1, 3, []int{2, 4, 6}, "06:30"},
		{"SPINNING", 2, 4, []int{1, 3, 5}, "17:30"},
		{"SPINNING", 2, 4, []int{2, 4, 6}, "06:15"},
	},
	{ // HN02 – Hai Bà Trưng
		{"YOGA", 0, 0, []int{1, 3, 5}, "06:00"},
		{"PILATES", 0, 0, []int{2, 4}, "18:00"},
		{"STRETCH", 0, 3, []int{6}, "09:00"},
		{"YOGA", 0, 0, []int{7}, "08:30"},
		{"ZUMBA", 1, 1, []int{1, 3, 5}, "18:00"},
		{"BODYPUMP", 1, 1, []int{2, 4}, "18:30"},
		{"BOXING", 1, 2, []int{1, 3, 5}, "19:30"},
		{"HIIT", 1, 2, []int{2, 4, 6}, "06:30"},
	},
	{ // HCM01 – Quận 1
		{"YOGA", 0, 0, []int{1, 3, 5}, "06:30"},
		{"YOGA", 0, 0, []int{2, 4}, "18:30"},
		{"STRETCH", 0, 0, []int{7}, "09:00"},
		{"PILATES", 0, 4, []int{1, 3, 5}, "18:00"},
		{"ZUMBA", 1, 1, []int{1, 3, 5}, "18:30"},
		{"BODYPUMP", 1, 1, []int{2, 4}, "18:00"},
		{"BOXING", 1, 3, []int{2, 4}, "19:30"},
		{"HIIT", 1, 2, []int{1, 3, 5}, "07:00"},
		{"SPINNING", 2, 2, []int{2, 4}, "18:00"},
		{"SPINNING", 2, 2, []int{6}, "08:00"},
	},
}

// ---------------------------------------------------------------------
// Nhân sự
// ---------------------------------------------------------------------

type staffDef struct {
	Email, Name, Gender, Position string
	Specialties                   []string
	Level                         string
	Rate                          float64
	Years                         int
	Bio, Certs                    string
}

// Mỗi chi nhánh: 1 quản lý, 2–3 lễ tân, 4–5 HLV (thứ tự HLV khớp chỉ số trong demoTimetable).
var demoStaffDefs = [3][]staffDef{
	{
		{Email: "quanly.cg@gymmaster.vn", Name: "Nguyễn Đức Thắng", Gender: "MALE", Position: "MANAGER"},
		{Email: "letan.cg@gymmaster.vn", Name: "Trần Thu Ngân", Gender: "FEMALE", Position: "RECEPTIONIST"},
		{Name: "Phạm Thị Hồng Nhung", Gender: "FEMALE", Position: "RECEPTIONIST"},
		{Name: "Đỗ Minh Tú", Gender: "MALE", Position: "RECEPTIONIST"},
		{Email: "hlv.cg@gymmaster.vn", Name: "Lê Hoàng Nam", Gender: "MALE", Position: "TRAINER", Specialties: []string{"Gym", "Giảm cân", "Body Pump"}, Level: "SENIOR", Rate: 15, Years: 7,
			Bio: "7 năm kinh nghiệm huấn luyện thể hình, chuyên giảm mỡ – tăng cơ cho người bận rộn.", Certs: "ACE Certified Personal Trainer; Chứng chỉ Dinh dưỡng thể thao ISSA"},
		{Name: "Vũ Thị Thanh Hương", Gender: "FEMALE", Position: "TRAINER", Specialties: []string{"Yoga", "Pilates", "Stretching"}, Level: "MASTER", Rate: 18, Years: 10,
			Bio: "Giảng viên Yoga 500 giờ, từng tu nghiệp tại Rishikesh (Ấn Độ).", Certs: "RYT-500 Yoga Alliance; Pilates Mat Level 2"},
		{Name: "Hoàng Thị Mai Anh", Gender: "FEMALE", Position: "TRAINER", Specialties: []string{"Zumba", "Aerobic", "Giảm cân"}, Level: "SENIOR", Rate: 14, Years: 5,
			Bio: "Huấn luyện viên Zumba năng động, chuyên các lớp đốt mỡ theo nhạc.", Certs: "Zumba Instructor Network (ZIN)"},
		{Name: "Bùi Quang Huy", Gender: "MALE", Position: "TRAINER", Specialties: []string{"Boxing", "HIIT", "Gym"}, Level: "SENIOR", Rate: 15, Years: 6,
			Bio: "Cựu võ sĩ quyền anh, chuyên Boxing Fitness và HIIT.", Certs: "Chứng chỉ HLV Quyền anh cấp quốc gia"},
		{Name: "Ngô Tiến Dũng", Gender: "MALE", Position: "TRAINER", Specialties: []string{"Spinning", "Gym", "Tăng cơ"}, Level: "JUNIOR", Rate: 10, Years: 2,
			Bio: "HLV trẻ, chuyên Spinning và các chương trình tăng cơ cơ bản.", Certs: "Spinning Instructor Certification"},
	},
	{
		{Name: "Trịnh Văn Hải", Gender: "MALE", Position: "MANAGER"},
		{Name: "Nguyễn Thị Phương Thảo", Gender: "FEMALE", Position: "RECEPTIONIST"},
		{Name: "Lê Thị Quỳnh", Gender: "FEMALE", Position: "RECEPTIONIST"},
		{Name: "Đặng Thùy Linh", Gender: "FEMALE", Position: "TRAINER", Specialties: []string{"Yoga", "Pilates"}, Level: "SENIOR", Rate: 15, Years: 6,
			Bio: "Chuyên Hatha Yoga và Pilates trị liệu cột sống.", Certs: "RYT-200 Yoga Alliance"},
		{Name: "Phan Thị Ngọc Hân", Gender: "FEMALE", Position: "TRAINER", Specialties: []string{"Zumba", "Body Pump"}, Level: "JUNIOR", Rate: 12, Years: 3,
			Bio: "Năng lượng, vui vẻ, chuyên lớp nhóm theo nhạc.", Certs: "Les Mills BODYPUMP Instructor"},
		{Name: "Trần Đức Mạnh", Gender: "MALE", Position: "TRAINER", Specialties: []string{"Boxing", "HIIT", "Gym"}, Level: "SENIOR", Rate: 16, Years: 8,
			Bio: "Chuyên sức mạnh – thể lực, từng huấn luyện đội tuyển trẻ.", Certs: "NSCA-CPT"},
		{Name: "Nguyễn Hữu Phúc", Gender: "MALE", Position: "TRAINER", Specialties: []string{"Gym", "Giảm cân", "Stretching"}, Level: "MASTER", Rate: 20, Years: 12,
			Bio: "12 năm kinh nghiệm, chuyên giảm cân an toàn và phục hồi chấn thương.", Certs: "NASM-CPT; NASM Corrective Exercise Specialist"},
	},
	{
		{Name: "Huỳnh Thanh Tùng", Gender: "MALE", Position: "MANAGER"},
		{Name: "Võ Ngọc Bảo Trân", Gender: "FEMALE", Position: "RECEPTIONIST"},
		{Name: "Lâm Gia Huy", Gender: "MALE", Position: "RECEPTIONIST"},
		{Name: "Trương Thị Kim Ngân", Gender: "FEMALE", Position: "RECEPTIONIST"},
		{Name: "Dương Thị Diệu Linh", Gender: "FEMALE", Position: "TRAINER", Specialties: []string{"Yoga", "Stretching"}, Level: "SENIOR", Rate: 15, Years: 6,
			Bio: "Yoga trị liệu, thiền và hơi thở.", Certs: "RYT-200 Yoga Alliance"},
		{Name: "Mai Quốc Khánh", Gender: "MALE", Position: "TRAINER", Specialties: []string{"Zumba", "Body Pump"}, Level: "JUNIOR", Rate: 11, Years: 3,
			Bio: "Chuyên lớp nhóm cường độ cao theo nhạc.", Certs: "Les Mills BODYPUMP; ZIN"},
		{Name: "Châu Minh Trí", Gender: "MALE", Position: "TRAINER", Specialties: []string{"Spinning", "HIIT"}, Level: "SENIOR", Rate: 14, Years: 5,
			Bio: "Vận động viên xe đạp nghiệp dư, chuyên Spinning và HIIT.", Certs: "Schwinn Cycling Instructor"},
		{Name: "Võ Thành Đạt", Gender: "MALE", Position: "TRAINER", Specialties: []string{"Boxing", "Gym", "Tăng cơ"}, Level: "SENIOR", Rate: 16, Years: 7,
			Bio: "HLV thể hình và Muay Thai, chuyên tăng cơ giảm mỡ.", Certs: "ISSA-CPT"},
		{Name: "Phạm Ngọc Hân", Gender: "FEMALE", Position: "TRAINER", Specialties: []string{"Pilates", "Giảm cân", "Gym"}, Level: "MASTER", Rate: 20, Years: 11,
			Bio: "Chuyên gia Pilates & giảm cân sau sinh.", Certs: "STOTT Pilates; ACE Health Coach"},
	},
}

// ---------------------------------------------------------------------
// Ghi chú PT, lý do...
// ---------------------------------------------------------------------

var ptWorkoutNotes = []string{
	"Khởi động 10 phút; Squat 4x10 (40kg); Leg press 3x12; Lunges 3x10/chân; Plank 3x45 giây",
	"Bench press 4x8 (35kg); Incline dumbbell press 3x10; Cable fly 3x12; Chống đẩy 2x15",
	"Deadlift 4x6 (60kg); Lat pulldown 4x10; Seated row 3x12; Face pull 3x15",
	"Cardio máy chạy 15 phút; Circuit toàn thân 4 vòng (burpee, kettlebell swing, mountain climber, jump squat)",
	"Shoulder press 4x10; Lateral raise 3x15; Bicep curl 3x12; Tricep pushdown 3x12",
	"HIIT 20 phút (40s tập/20s nghỉ); Core: crunch, leg raise, russian twist 3 vòng",
	"Romanian deadlift 4x10; Hip thrust 4x12; Bulgarian split squat 3x10; Calf raise 3x20",
	"Đánh giá thể lực: InBody, đo vòng, test chống đẩy & plank; lên giáo án 4 tuần",
	"Pull-up hỗ trợ 4x6; Dumbbell row 3x10; Back extension 3x15; Giãn cơ lưng 10 phút",
	"Kỹ thuật squat với gậy, mobility hông & cổ chân; Goblet squat 3x12",
	"Boxing pad work 6 hiệp x 2 phút; Nhảy dây 5 phút; Core 15 phút",
	"Chạy interval 25 phút; Step-up 3x12; Plank biến thể 3 vòng; Giãn cơ",
}

var ptTrainerNotes = []string{
	"Kỹ thuật squat tiến bộ, cần giữ lưng thẳng hơn ở rep cuối.",
	"Thể lực tốt, tăng 2,5kg mức tạ buổi sau.",
	"Học viên hơi mệt do ngủ ít, giảm cường độ 10%.",
	"Hoàn thành đủ giáo án, nhắc uống đủ 2,5 lít nước/ngày.",
	"Cân nặng giảm 0,8kg so với tuần trước, duy trì chế độ ăn.",
	"Vai trái hơi căng, bổ sung bài giãn cơ vai.",
	"Sức bền cải thiện rõ, có thể tăng thời gian cardio.",
	"Tư thế deadlift đã chuẩn, tuần sau tăng khối lượng.",
	"Nhắc ăn đủ đạm (1,6g/kg), hạn chế đồ ngọt buổi tối.",
	"Rất chăm chỉ, đạt mục tiêu plank 90 giây.",
}

var freezeReasons = []string{"Đi công tác nước ngoài", "Ốm, bác sĩ yêu cầu nghỉ ngơi", "Về quê có việc gia đình", "Chấn thương cổ chân, cần nghỉ", "Đi du lịch cùng gia đình", "Thi cuối kỳ, bận học"}

var cancelReasons = []string{"Chuyển công tác vào Nam", "Lý do sức khỏe, bác sĩ khuyên ngừng tập", "Chuyển nhà xa phòng tập", "Đi du học"}

var classCancelReasons = []string{"HLV nghỉ ốm đột xuất", "Bảo trì hệ thống âm thanh phòng tập", "HLV đi tập huấn"}

var memberNotes = []string{"", "", "", "", "", "Muốn giảm 5kg trong 3 tháng", "Đau lưng nhẹ, tránh bài nặng phần lưng", "Khách quen của chi nhánh", "Ưu tiên tập buổi tối", "Sinh viên", "Đi cùng bạn bè", "Thích lớp Yoga"}
