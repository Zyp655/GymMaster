package seed

import (
	"context"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
)

func seedStage3And4Demo(ctx context.Context, tx *gorm.DB) error {
	now := time.Now().In(demoLoc())

	var branchRows []entity.Branch
	if err := tx.WithContext(ctx).Order("id ASC").Find(&branchRows).Error; err != nil || len(branchRows) == 0 {
		return err
	}
	branches := map[string]uint{
		"CG": branchRows[0].ID,
	}
	if len(branchRows) > 1 {
		branches["DD"] = branchRows[1].ID
	}

	var users []entity.User
	_ = tx.WithContext(ctx).Order("id ASC").Find(&users).Error
	staff := map[string]uint{}
	for _, u := range users {
		staff[strings.ToLower(u.Email)] = u.ID
	}
	if len(users) > 0 {
		if staff["admin@gymmaster.vn"] == 0 {
			staff["admin@gymmaster.vn"] = users[0].ID
		}
		if staff["quanly.cg@gymmaster.vn"] == 0 {
			staff["quanly.cg@gymmaster.vn"] = users[0].ID
		}
		if staff["letan.cg@gymmaster.vn"] == 0 {
			staff["letan.cg@gymmaster.vn"] = users[0].ID
		}
		if staff["hlv.nam@gymmaster.vn"] == 0 {
			staff["hlv.nam@gymmaster.vn"] = users[0].ID
		}
	}

	var members []entity.Member
	_ = tx.WithContext(ctx).Order("id ASC").Limit(10).Find(&members).Error
	var memberIDs []uint
	for _, m := range members {
		memberIDs = append(memberIDs, m.ID)
	}

	return seedDemoStage3And4(tx.WithContext(ctx), now, branches, staff, memberIDs, nil)
}

// seedDemoStage3And4 nạp dữ liệu mẫu cho Giai đoạn 3 & 4 (Sprint 18–27):
// Thiết bị Kiosk, Tủ đồ, Chấm công & Bảng lương, Chi phí & Ca thu ngân,
// Điểm thưởng Loyalty & Thử thách, Thực đơn & Bảng tin & Khảo sát NPS,
// Hợp đồng điện tử & Vé ngày, Lịch rảnh HLV & Ngày lễ, Chỉ tiêu KPI, Đồng ý dữ liệu cá nhân.
func seedDemoStage3And4(tx *gorm.DB, now time.Time, branches map[string]uint, staff map[string]uint, memberIDs []uint, planIDs map[string]uint) error {
	var existingDevices int64
	if err := tx.Model(&entity.AccessDevice{}).Count(&existingDevices).Error; err != nil {
		return err
	}
	if existingDevices > 0 {
		return nil
	}

	cgBranch := branches["CG"]
	ddBranch := branches["DD"]
	if cgBranch == 0 || len(memberIDs) == 0 {
		return nil
	}

	ownerID := staff["admin@gymmaster.vn"]
	managerCG := staff["quanly.cg@gymmaster.vn"]
	recepCG := staff["letan.cg@gymmaster.vn"]
	trainer1 := staff["hlv.nam@gymmaster.vn"]
	if trainer1 == 0 {
		trainer1 = ownerID
	}

	// 1. Sprint 18: Thiết bị kiểm soát ra vào & Kiosk (key mặc định cho demo: GMD-demo-kiosk-cg01)
	devices := []entity.AccessDevice{
		{
			BranchID:   cgBranch,
			Code:       "KIOSK-CG-01",
			Name:       "Kiosk Cảm Ứng Quầy Lễ Tân Cầu Giấy",
			Type:       entity.DeviceTypeKiosk,
			APIKeyHash: hash.SHA256("GMD-demo-kiosk-cg01"),
			LastSeenAt: &now,
			Status:     entity.DeviceStatusActive,
		},
		{
			BranchID:   cgBranch,
			Code:       "GATE-CG-01",
			Name:       "Cổng Xoay Tripod Lối Vào Chính",
			Type:       entity.DeviceTypeTurnstile,
			APIKeyHash: hash.SHA256("GMD-demo-gate-cg01"),
			LastSeenAt: &now,
			Status:     entity.DeviceStatusActive,
		},
	}
	if ddBranch > 0 {
		devices = append(devices, entity.AccessDevice{
			BranchID:   ddBranch,
			Code:       "KIOSK-DD-01",
			Name:       "Kiosk Tự Check-in Đống Đa",
			Type:       entity.DeviceTypeKiosk,
			APIKeyHash: hash.SHA256("GMD-demo-kiosk-dd01"),
			LastSeenAt: &now,
			Status:     entity.DeviceStatusActive,
		})
	}
	if err := tx.Create(&devices).Error; err != nil {
		return err
	}

	// 2. Sprint 19: Tủ đồ & Lượt thuê tủ
	lockers := []entity.Locker{
		{BranchID: cgBranch, Code: "A-01", Zone: "Khu Nam - Tầng 2", Size: entity.LockerSizeStandard, MonthlyFee: 150000, Status: entity.LockerStatusOccupied},
		{BranchID: cgBranch, Code: "A-02", Zone: "Khu Nam - Tầng 2", Size: entity.LockerSizeStandard, MonthlyFee: 150000, Status: entity.LockerStatusRented},
		{BranchID: cgBranch, Code: "A-03", Zone: "Khu Nam - Tầng 2", Size: entity.LockerSizeLarge, MonthlyFee: 220000, Status: entity.LockerStatusAvailable},
		{BranchID: cgBranch, Code: "A-04", Zone: "Khu Nam - Tầng 2", Size: entity.LockerSizeStandard, MonthlyFee: 150000, Status: entity.LockerStatusAvailable},
		{BranchID: cgBranch, Code: "B-01", Zone: "Khu Nữ - Tầng 2", Size: entity.LockerSizeStandard, MonthlyFee: 150000, Status: entity.LockerStatusAvailable},
		{BranchID: cgBranch, Code: "B-02", Zone: "Khu Nữ - Tầng 2", Size: entity.LockerSizeLarge, MonthlyFee: 220000, Status: entity.LockerStatusBroken, Note: "Hỏng khóa từ, chờ bảo trì"},
	}
	if err := tx.Create(&lockers).Error; err != nil {
		return err
	}
	endDaily := now.Add(4 * time.Hour)
	endMonthly := now.AddDate(0, 1, 0)
	rentals := []entity.LockerRental{
		{
			LockerID:  lockers[0].ID,
			MemberID:  memberIDs[0],
			Type:      entity.LockerRentalDaily,
			StartAt:   now.Add(-45 * time.Minute),
			EndAt:     &endDaily,
			Price:     0,
			Status:    entity.LockerRentalActive,
			Note:      "Gán tủ theo lượt khi check-in",
			CreatedBy: &recepCG,
		},
		{
			LockerID:  lockers[1].ID,
			MemberID:  memberIDs[minInt(1, len(memberIDs)-1)],
			Type:      entity.LockerRentalMonthly,
			StartAt:   now.AddDate(0, 0, -5),
			EndAt:     &endMonthly,
			Price:     150000,
			Status:    entity.LockerRentalActive,
			Note:      "Thuê tủ cố định 1 tháng",
			CreatedBy: &recepCG,
		},
	}
	if err := tx.Create(&rentals).Error; err != nil {
		return err
	}

	// 3. Sprint 20: Chấm công & Bảng lương mẫu
	checkInTime := now.Add(-6 * time.Hour)
	checkOutTime := now.Add(-1 * time.Hour)
	att := entity.StaffAttendance{
		StaffID:     recepCG,
		BranchID:    cgBranch,
		CheckInAt:   checkInTime,
		CheckOutAt:  &checkOutTime,
		LateMinutes: 0,
		Method:      entity.AttendanceMethodQR,
		Note:        "Ca sáng đúng giờ",
	}
	_ = tx.Create(&att).Error

	period := now.Format("2006-01")
	payroll := entity.Payroll{
		BranchID: cgBranch,
		Period:   period,
		Status:   entity.PayrollStatusDraft,
	}
	if err := tx.Create(&payroll).Error; err == nil {
		items := []entity.PayrollItem{
			{
				PayrollID:       payroll.ID,
				StaffID:         recepCG,
				BaseSalary:      9000000,
				WorkDays:        24,
				SalesCommission: 1200000,
				Bonus:           500000,
				Total:           10700000,
				Note:            "Lễ tân xuất sắc tháng",
			},
			{
				PayrollID:    payroll.ID,
				StaffID:      trainer1,
				BaseSalary:   8000000,
				WorkDays:     25,
				PTCommission: 6500000,
				Bonus:        1000000,
				Total:        15500000,
				Note:         "HLV đạt vượt KPI buổi dạy",
			},
		}
		_ = tx.Create(&items).Error
	}

	// 4. Sprint 21: Chi phí & Ca thu ngân
	var rentCat entity.ExpenseCategory
	if err := tx.Where("type = ?", entity.ExpenseTypeFixed).First(&rentCat).Error; err == nil {
		exp := entity.Expense{
			BranchID:    cgBranch,
			CategoryID:  rentCat.ID,
			Amount:      45000000,
			SpentAt:     now.AddDate(0, 0, -3),
			Description: "Thanh toán tiền thuê mặt bằng & điện nước tháng " + period,
			CreatedBy:   &managerCG,
			ApprovedBy:  &ownerID,
			Status:      entity.ExpenseStatusApproved,
		}
		_ = tx.Create(&exp).Error
	}

	cashSess := entity.CashSession{
		BranchID:     cgBranch,
		StaffID:      recepCG,
		OpenedAt:     now.Add(-5 * time.Hour),
		OpeningCash:  2000000,
		ExpectedCash: 2000000,
		Status:       entity.CashSessionOpen,
		Note:         "Ca thu ngân sáng tại quầy Cầu Giấy",
	}
	_ = tx.Create(&cashSess).Error

	// 5. Sprint 22: Điểm thưởng Loyalty, Huy hiệu & Thử thách
	_ = tx.Model(&entity.Member{}).Where("id = ?", memberIDs[0]).Update("loyalty_points", 350).Error
	ledgers := []entity.LoyaltyLedger{
		{MemberID: memberIDs[0], Points: 200, Reason: entity.LoyaltyReasonRenew, Note: "Thưởng đăng ký gói tập"},
		{MemberID: memberIDs[0], Points: 100, Reason: entity.LoyaltyReasonReferral, Note: "Giới thiệu bạn bè tập luyện"},
		{MemberID: memberIDs[0], Points: 50, Reason: entity.LoyaltyReasonCheckin, Note: "Chuỗi check-in chăm chỉ"},
	}
	_ = tx.Create(&ledgers).Error

	var firstBadge entity.Badge
	if err := tx.Order("id ASC").First(&firstBadge).Error; err == nil {
		_ = tx.Create(&entity.MemberBadge{
			MemberID: memberIDs[0],
			BadgeID:  firstBadge.ID,
			EarnedAt: now.AddDate(0, 0, -2),
		}).Error
	}

	challenge := entity.Challenge{
		BranchID:     &cgBranch,
		Name:         "Thử thách 12 buổi tập trong tháng",
		Description:  "Hoàn thành 12 lượt check-in trong tháng để nhận ngay 300 điểm thưởng Loyalty!",
		Metric:       entity.ChallengeMetricCheckins,
		Target:       12,
		StartDate:    now.AddDate(0, 0, -10),
		EndDate:      now.AddDate(0, 0, 20),
		RewardPoints: 300,
		Status:       "ACTIVE",
	}
	if err := tx.Create(&challenge).Error; err == nil {
		_ = tx.Create(&entity.ChallengeParticipant{
			ChallengeID: challenge.ID,
			MemberID:    memberIDs[0],
			Progress:    7,
			JoinedAt:    now.AddDate(0, 0, -8),
		}).Error
	}

	// 6. Sprint 23: Thực đơn dinh dưỡng, Bảng tin & Khảo sát NPS
	startDate := now.AddDate(0, 0, -7)
	endDate := now.AddDate(0, 1, 0)
	mealPlan := entity.MealPlan{
		MemberID:       memberIDs[0],
		TrainerID:      &trainer1,
		Name:           "Thực đơn Tăng cơ Giảm mỡ (Clean Bulk 2200 kcal)",
		Goal:           "Tăng cơ nạc, duy trì tỷ lệ mỡ dưới 15%",
		TargetCalories: 2200,
		StartDate:      &startDate,
		EndDate:        &endDate,
		Status:         "ACTIVE",
	}
	if err := tx.Create(&mealPlan).Error; err == nil {
		var foods []entity.Food
		if err := tx.Order("id ASC").Limit(4).Find(&foods).Error; err == nil && len(foods) >= 3 {
			mpItems := []entity.MealPlanItem{
				{PlanID: mealPlan.ID, DayIndex: 1, Meal: entity.MealBreakfast, FoodID: foods[0].ID, Qty: 1.5, Note: "Ăn trước tập 60 phút"},
				{PlanID: mealPlan.ID, DayIndex: 1, Meal: entity.MealLunch, FoodID: foods[1].ID, Qty: 1, Note: "Bữa chính sau tập"},
				{PlanID: mealPlan.ID, DayIndex: 1, Meal: entity.MealDinner, FoodID: foods[2].ID, Qty: 2, Note: "Ít tinh bột buổi tối"},
			}
			_ = tx.Create(&mpItems).Error
		}
	}

	announcements := []entity.Announcement{
		{
			BranchID:  &cgBranch,
			Title:     "Khai trương khu vực Functional & InBody mới tại Cầu Giấy",
			Body:      "Tất cả hội viên được đo chỉ số cơ thể InBody miễn phí định kỳ mỗi 2 tuần và trải nghiệm dàn tạ Rogue mới 100%.",
			PublishAt: now.Add(-24 * time.Hour),
			Pinned:    true,
			CreatedBy: &ownerID,
		},
		{
			BranchID:  nil,
			Title:     "Ra mắt chương trình Tích điểm Đổi quà & Thử thách Tháng 10",
			Body:      "Check-in mỗi ngày để tích điểm đổi voucher giảm giá gói tập và nhận huy hiệu Chiến binh tại mục Điểm thưởng trên Cổng hội viên.",
			PublishAt: now.Add(-12 * time.Hour),
			Pinned:    false,
			CreatedBy: &ownerID,
		},
	}
	_ = tx.Create(&announcements).Error

	survey := entity.Survey{
		Title:       "Khảo sát mức độ hài lòng hội viên Quý 4 (NPS)",
		Description: "Bạn sẵn sàng giới thiệu phòng tập GymMaster cho bạn bè/đồng nghiệp ở mức độ nào (0–10)?",
		Type:        entity.SurveyTypeNPS,
		StartAt:     now.AddDate(0, 0, -5),
		Status:      "ACTIVE",
	}
	if err := tx.Create(&survey).Error; err == nil && len(memberIDs) >= 2 {
		_ = tx.Create(&entity.SurveyResponse{
			SurveyID: survey.ID,
			MemberID: memberIDs[1],
			Score:    10,
			Comment:  "Phòng tập sạch sẽ, máy tập mới và HLV hướng dẫn rất tận tâm!",
			Answers:  datatypes.JSON(`{"cleanliness":5,"equipment":5}`),
		}).Error
	}

	// 7. Sprint 24 & 25 & 26 & 27: Vé ngày, Lịch rảnh HLV, Chỉ tiêu KPI & Đồng ý dữ liệu
	dayPass := entity.DayPass{
		BranchID:        cgBranch,
		BuyerName:       "Lê Hoàng Khách Mời",
		Phone:           "0977123123",
		SponsorMemberID: &memberIDs[0],
		Type:            entity.DayPassTypeGuest,
		ValidDate:       now,
		QRToken:         "DP-DEMO202601",
		Price:           0,
	}
	_ = tx.Create(&dayPass).Error

	for dow := 1; dow <= 6; dow++ {
		_ = tx.Create(&entity.TrainerAvailability{
			TrainerID: trainer1,
			BranchID:  cgBranch,
			DOW:       dow,
			StartTime: "07:00",
			EndTime:   "20:00",
		}).Error
	}

	targets := []entity.SalesTarget{
		{BranchID: cgBranch, Period: period, Metric: entity.KPIMetricRevenue, TargetValue: 150000000},
		{BranchID: cgBranch, Period: period, Metric: entity.KPIMetricNewMembers, TargetValue: 25},
		{BranchID: cgBranch, Period: period, Metric: entity.KPIMetricPTSales, TargetValue: 50000000},
	}
	_ = tx.Create(&targets).Error

	consents := []entity.Consent{
		{MemberID: memberIDs[0], Type: entity.ConsentTypeTerms, Granted: true, GrantedAt: now, IP: "127.0.0.1"},
		{MemberID: memberIDs[0], Type: entity.ConsentTypeMarketing, Granted: true, GrantedAt: now, IP: "127.0.0.1"},
	}
	_ = tx.Create(&consents).Error

	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
