package seed

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
	"gymmaster-api/internal/pkg/htmlsafe"
)

// seedStage3And4Demo nạp dữ liệu mẫu Giai đoạn 3–5 (Sprint 18–30) cho cả 3 chi nhánh:
// thiết bị ra vào, tủ đồ, chấm công & bảng lương, chi phí & ca thu ngân, chỉ tiêu KPI, ngày lễ,
// lịch rảnh & nghỉ phép HLV, điểm thưởng/huy hiệu/thử thách, dinh dưỡng, bảng tin, khảo sát NPS,
// vé khách mời, hợp đồng điện tử, đồng ý dữ liệu, hồ sơ HLV công khai, CMS và tin nhắn PT.
// Mọi số liệu được suy ra từ dữ liệu lõi đã seed (ca làm, check-in, hóa đơn, hợp đồng PT…) để các
// báo cáo khớp nhau. Chạy lặp lại không tạo trùng (đánh dấu bằng bảng access_devices).
func seedStage3And4Demo(ctx context.Context, tx *gorm.DB) error {
	var existingDevices int64
	if err := tx.WithContext(ctx).Model(&entity.AccessDevice{}).Count(&existingDevices).Error; err != nil {
		return err
	}
	if existingDevices > 0 {
		return nil
	}

	loc := demoLoc()
	now := time.Now().In(loc)
	g := &richGen{
		ctx:   ctx,
		tx:    tx.WithContext(ctx),
		r:     demoRand{rand.New(rand.NewSource(20261022))}, // #nosec G404
		loc:   loc,
		now:   now,
		today: dayOf(now, loc),
	}
	if err := g.load(); err != nil {
		return err
	}
	if len(g.branches) == 0 || len(g.members) == 0 {
		return nil
	}

	steps := []func() error{
		g.seedDevices,
		g.seedOnlinePayments,
		g.seedLockers,
		g.seedLeaves,
		g.seedAttendance,
		g.seedPayrolls,
		g.seedExpenses,
		g.seedCashSessions,
		g.seedSalesTargets,
		g.seedHolidays,
		g.seedAvailabilities,
		g.seedLeads,
		g.seedReviews,
		g.seedEquipment,
		g.seedWorkoutPlans,
		g.seedReferrals,
		g.seedBadges,
		g.seedChallenges,
		g.seedLoyalty,
		g.seedMealPlans,
		g.seedAnnouncements,
		g.seedSurveys,
		g.seedGuestPasses,
		g.seedContracts,
		g.seedConsents,
		g.seedTrainerProfiles,
		g.seedTestimonials,
		g.seedPosts,
		g.seedContactMessages,
		g.seedConversations,
		g.seedNotifications,
		g.seedAuditLogs,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

type richGen struct {
	ctx   context.Context
	tx    *gorm.DB
	r     demoRand
	loc   *time.Location
	now   time.Time
	today time.Time

	ownerID    uint
	branches   []*rsBranch
	branchByID map[uint]*rsBranch
	staff      []*rsStaff
	members    []*rsMember
	memberByID map[uint]*rsMember
	showcase   *rsMember

	leaveDays map[uint]map[string]bool // nhân viên → ngày nghỉ phép đã duyệt (không chấm công)
	ledger    *bulkRows                // sổ điểm gom lại, ghi một lần ở seedLoyalty
	points    map[uint]int             // số dư điểm tạm tính theo hội viên
}

type rsBranch struct {
	ID        uint
	Code      string
	Name      string
	OpenTime  string
	CloseTime string

	short    string
	manager  *rsStaff
	receps   []*rsStaff
	trainers []*rsStaff
}

type rsStaff struct {
	ID         uint
	FullName   string
	Email      string
	Position   string
	BaseSalary float64
	BranchID   uint
}

type rsMember struct {
	ID           uint
	MemberCode   string
	FullName     string
	Gender       string
	Phone        string
	Email        string
	HomeBranchID uint
	UserID       *uint
	CreatedAt    time.Time
	Source       string
	ReferralCode *string
	SubID        *uint
	SubCreated   *time.Time
	PlanName     *string
	GuestPasses  *int
}

func (m *rsMember) active() bool { return m.SubID != nil }

func (g *richGen) load() error {
	var owner entity.User
	if err := g.tx.Joins("JOIN user_roles ur ON ur.user_id = users.id JOIN roles r ON r.id = ur.role_id").
		Where("r.code = ?", consts.RoleOwner).Order("users.id ASC").First(&owner).Error; err != nil {
		return err
	}
	g.ownerID = owner.ID

	if err := g.tx.Raw(`SELECT id, code, name, open_time, close_time FROM branches
		WHERE code IN ('HN01','HN02','HCM01') AND deleted_at IS NULL ORDER BY id`).Scan(&g.branches).Error; err != nil {
		return err
	}
	g.branchByID = map[uint]*rsBranch{}
	for _, b := range g.branches {
		b.short = strings.TrimPrefix(b.Name, "GymMaster ")
		g.branchByID[b.ID] = b
	}

	if err := g.tx.Raw(`SELECT DISTINCT ON (u.id) u.id, u.full_name, u.email, sp.position, sp.base_salary, ur.branch_id
		FROM users u
		JOIN staff_profiles sp ON sp.user_id = u.id
		JOIN user_roles ur ON ur.user_id = u.id AND ur.branch_id IS NOT NULL
		WHERE u.deleted_at IS NULL AND u.status = 'ACTIVE'
		ORDER BY u.id`).Scan(&g.staff).Error; err != nil {
		return err
	}
	for _, s := range g.staff {
		b := g.branchByID[s.BranchID]
		if b == nil {
			continue
		}
		switch s.Position {
		case "MANAGER":
			b.manager = s
		case "RECEPTIONIST":
			b.receps = append(b.receps, s)
		case "TRAINER":
			b.trainers = append(b.trainers, s)
		}
	}
	for _, b := range g.branches {
		if b.manager == nil || len(b.receps) == 0 || len(b.trainers) == 0 {
			return fmt.Errorf("seed: chi nhánh %s thiếu quản lý/lễ tân/HLV", b.Code)
		}
	}

	if err := g.tx.Raw(`SELECT m.id, m.member_code, m.full_name, m.gender, m.phone, m.email, m.home_branch_id, m.user_id,
			m.created_at, m.source, m.referral_code,
			s.id AS sub_id, s.created_at AS sub_created, p.name AS plan_name, p.guest_passes_per_month AS guest_passes
		FROM members m
		LEFT JOIN LATERAL (
			SELECT id, plan_id, created_at FROM subscriptions
			WHERE member_id = m.id AND status = 'ACTIVE' ORDER BY end_date DESC LIMIT 1
		) s ON true
		LEFT JOIN membership_plans p ON p.id = s.plan_id
		WHERE m.deleted_at IS NULL AND m.status = 'ACTIVE'
		ORDER BY m.id`).Scan(&g.members).Error; err != nil {
		return err
	}
	g.memberByID = map[uint]*rsMember{}
	for _, m := range g.members {
		g.memberByID[m.ID] = m
		if strings.EqualFold(m.Email, "hoivien@gymmaster.vn") {
			g.showcase = m
		}
	}
	if g.showcase == nil && len(g.members) > 0 {
		g.showcase = g.members[0]
	}

	g.leaveDays = map[uint]map[string]bool{}
	g.ledger = newBulk("loyalty_ledgers", "member_id", "points", "reason", "ref_type", "ref_id", "note", "created_by", "created_at")
	g.points = map[uint]int{}
	return nil
}

// ---------------------------------------------------------------------
// Tiện ích
// ---------------------------------------------------------------------

func (g *richGen) day(offset int) time.Time { return addDays(g.today, offset) }

func (g *richGen) monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, g.loc)
}

func (g *richGen) at(d time.Time, hh, mm int) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), hh, mm, 0, 0, g.loc)
}

func (g *richGen) capNow(t time.Time) time.Time {
	if t.After(g.now) {
		return g.now.Add(-time.Duration(g.r.between(5, 90)) * time.Minute)
	}
	return t
}

func (g *richGen) activeMembers(branchID uint, gender string) []*rsMember {
	var out []*rsMember
	for _, m := range g.members {
		if !m.active() || (branchID != 0 && m.HomeBranchID != branchID) || (gender != "" && m.Gender != gender) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func (g *richGen) pickMember(list []*rsMember) *rsMember { return list[g.r.Intn(len(list))] }

func (g *richGen) shuffled(list []*rsMember) []*rsMember {
	out := append([]*rsMember(nil), list...)
	g.r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func (g *richGen) addLedger(memberID uint, pts int, reason, refType string, refID *uint, note string, by *uint, at time.Time) {
	g.ledger.add(memberID, pts, reason, refType, nUint(refID), note, nUint(by), at)
	g.points[memberID] += pts
}

func slugVN(s string) string {
	parts := strings.Fields(s)
	for i, p := range parts {
		parts[i] = asciiFold(p)
	}
	return strings.Join(parts, "-")
}

func truncRunes(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:n-1]) + "…"
}

// ---------------------------------------------------------------------
// Sprint 18 – Thiết bị kiểm soát ra vào & Kiosk
// ---------------------------------------------------------------------

func (g *richGen) seedDevices() error {
	suffix := map[string]string{"HN01": "cg01", "HN02": "hbt01", "HCM01": "q1-01"}
	var devices []entity.AccessDevice
	for _, b := range g.branches {
		sfx := suffix[b.Code]
		seen := g.now.Add(-time.Duration(g.r.between(1, 4)) * time.Minute)
		devices = append(devices,
			entity.AccessDevice{BranchID: b.ID, Code: "KIOSK-" + b.Code + "-01", Name: "Kiosk tự check-in quầy lễ tân " + b.short,
				Type: entity.DeviceTypeKiosk, APIKeyHash: hash.SHA256("GMD-demo-kiosk-" + sfx), LastSeenAt: &seen, Status: entity.DeviceStatusActive},
			entity.AccessDevice{BranchID: b.ID, Code: "GATE-" + b.Code + "-01", Name: "Cổng xoay Tripod lối vào chính " + b.short,
				Type: entity.DeviceTypeTurnstile, APIKeyHash: hash.SHA256("GMD-demo-gate-" + sfx), LastSeenAt: &seen, Status: entity.DeviceStatusActive},
		)
		door := entity.AccessDevice{BranchID: b.ID, Code: "DOOR-" + b.Code + "-YOGA", Name: "Khóa từ cửa phòng Yoga " + b.short,
			Type: entity.DeviceTypeDoor, APIKeyHash: hash.SHA256("GMD-demo-door-" + sfx), LastSeenAt: &seen, Status: entity.DeviceStatusActive}
		if b.Code == "HCM01" {
			off := g.now.Add(-52 * time.Hour)
			door.LastSeenAt = &off
			door.Status = entity.DeviceStatusOffline
		}
		devices = append(devices, door)
	}
	for i := range devices {
		devices[i].CreatedAt = g.now.AddDate(0, -3, 0)
		devices[i].UpdatedAt = g.now
		devices[i].CreatedBy = &g.ownerID
	}
	return g.tx.Create(&devices).Error
}

// ---------------------------------------------------------------------
// Sprint 19 – Tủ đồ & lượt thuê
// ---------------------------------------------------------------------

func (g *richGen) seedLockers() error {
	zones := []struct{ prefix, zone, gender string }{
		{"A", "Khu Nam – Tầng 2", "MALE"},
		{"B", "Khu Nữ – Tầng 2", "FEMALE"},
	}
	var rentals []entity.LockerRental
	holding := map[uint]bool{} // hội viên đang giữ một tủ ACTIVE
	for bi, b := range g.branches {
		var lockers []entity.Locker
		for _, z := range zones {
			for i := 1; i <= 16; i++ {
				l := entity.Locker{BranchID: b.ID, Code: fmt.Sprintf("%s-%02d", z.prefix, i), Zone: z.zone,
					Size: entity.LockerSizeStandard, MonthlyFee: 150000, Status: entity.LockerStatusAvailable}
				switch {
				case i%4 == 0:
					l.Size, l.MonthlyFee = entity.LockerSizeLarge, 220000
				case i%5 == 0:
					l.Size, l.MonthlyFee = entity.LockerSizeSmall, 100000
				}
				if (z.prefix == "B" && i == 11) || (z.prefix == "A" && i == 7+bi) {
					l.Status = entity.LockerStatusBroken
					l.Note = "Hỏng khóa từ, đã báo bộ phận bảo trì"
				}
				lockers = append(lockers, l)
			}
		}
		if err := g.tx.Create(&lockers).Error; err != nil {
			return err
		}

		recep := b.receps[0]
		for li := range lockers {
			l := &lockers[li]
			if l.Status == entity.LockerStatusBroken {
				continue
			}
			gender := "MALE"
			if strings.HasPrefix(l.Code, "B") {
				gender = "FEMALE"
			}
			pool := g.activeMembers(b.ID, gender)
			if len(pool) == 0 {
				continue
			}

			// Tủ thuê tháng cố định cho hội viên mẫu.
			if g.showcase != nil && !holding[g.showcase.ID] && g.showcase.HomeBranchID == b.ID && g.showcase.Gender == gender {
				start := g.at(g.day(-12), 18, 20)
				end := start.AddDate(0, 1, 0)
				rentals = append(rentals, entity.LockerRental{LockerID: l.ID, MemberID: g.showcase.ID, Type: entity.LockerRentalMonthly,
					StartAt: start, EndAt: &end, Price: l.MonthlyFee, Status: entity.LockerRentalActive,
					Note: "Thuê tủ cố định 1 tháng – thu tại quầy", CreatedBy: &recep.ID, CreatedAt: start, UpdatedAt: start})
				holding[g.showcase.ID] = true
				l.Status = entity.LockerStatusRented
				continue
			}

			// Lịch sử thuê ~4 tháng: xen kẽ thuê tháng và thuê theo lượt.
			cursor := g.day(-120)
			for {
				start := g.at(addDays(cursor, g.r.between(0, 18)), g.r.between(6, 20), g.r.between(0, 59))
				if !start.Before(g.now) {
					break
				}
				m := g.pickMember(pool)
				if g.r.chance(0.65) {
					months := 1
					if g.r.chance(0.2) {
						months = 3
					}
					end := start.AddDate(0, months, 0)
					rt := entity.LockerRental{LockerID: l.ID, MemberID: m.ID, Type: entity.LockerRentalMonthly, StartAt: start, EndAt: &end,
						Price: l.MonthlyFee * float64(months), Note: fmt.Sprintf("Thuê tủ %d tháng – thu tại quầy", months),
						CreatedBy: &g.r.pickStaff(b.receps).ID, CreatedAt: start, UpdatedAt: start}
					if end.After(g.now) {
						if holding[m.ID] || !g.r.chance(0.55) {
							break // ~55% tủ đang được thuê tháng, phần còn lại để trống
						}
						rt.Status = entity.LockerRentalActive
						holding[m.ID] = true
						l.Status = entity.LockerStatusRented
						rentals = append(rentals, rt)
						break
					}
					rt.Status = entity.LockerRentalEnded
					if g.r.chance(0.25) {
						rt.Status = entity.LockerRentalExpired
						rt.Note = "Hết hạn thuê, hội viên không gia hạn"
					}
					rt.UpdatedAt = end
					rentals = append(rentals, rt)
					cursor = end
					continue
				}
				// Thuê theo lượt trong ngày (miễn phí kèm check-in) – chỉ ghi lịch sử đã trả tủ.
				end := start.Add(time.Duration(g.r.between(60, 150)) * time.Minute)
				if end.After(g.now) {
					break
				}
				rentals = append(rentals, entity.LockerRental{LockerID: l.ID, MemberID: m.ID, Type: entity.LockerRentalDaily, StartAt: start, EndAt: &end,
					Status: entity.LockerRentalEnded, Note: "Tủ theo lượt khi check-in", CreatedBy: &g.r.pickStaff(b.receps).ID, CreatedAt: start, UpdatedAt: end})
				cursor = dayOf(end, g.loc)
			}

			// Một số tủ đang có hội viên dùng theo lượt ngay lúc này.
			if l.Status == entity.LockerStatusAvailable && g.r.chance(0.3) {
				m := g.pickMember(pool)
				if !holding[m.ID] {
					start := g.now.Add(-time.Duration(g.r.between(15, 80)) * time.Minute)
					end := start.Add(4 * time.Hour)
					rentals = append(rentals, entity.LockerRental{LockerID: l.ID, MemberID: m.ID, Type: entity.LockerRentalDaily, StartAt: start, EndAt: &end,
						Status: entity.LockerRentalActive, Note: "Tủ theo lượt khi check-in", CreatedBy: &recep.ID, CreatedAt: start, UpdatedAt: start})
					holding[m.ID] = true
					l.Status = entity.LockerStatusOccupied
				}
			}
		}
		for _, l := range lockers {
			if l.Status == entity.LockerStatusRented || l.Status == entity.LockerStatusOccupied {
				if err := g.tx.Model(&entity.Locker{}).Where("id = ?", l.ID).Update("status", l.Status).Error; err != nil {
					return err
				}
			}
		}
	}
	return g.tx.CreateInBatches(&rentals, 500).Error
}

func (r demoRand) pickStaff(list []*rsStaff) *rsStaff { return list[r.Intn(len(list))] }

// ---------------------------------------------------------------------
// Sprint 25 – Nghỉ phép (tạo trước chấm công để bỏ qua ngày nghỉ đã duyệt)
// ---------------------------------------------------------------------

func (g *richGen) seedLeaves() error {
	reasons := []string{
		"Về quê dự đám cưới người thân",
		"Khám sức khỏe định kỳ tại bệnh viện",
		"Con ốm, cần ở nhà chăm sóc",
		"Giải quyết thủ tục giấy tờ cá nhân",
		"Tham gia khóa đào tạo chứng chỉ HLV cuối tuần",
		"Nghỉ phép năm",
	}
	var leaves []entity.StaffLeaveRequest
	add := func(s *rsStaff, b *rsBranch, from time.Time, days int, status string, note string) {
		to := g.at(addDays(from, days-1), 23, 59)
		created := g.capNow(from.AddDate(0, 0, -g.r.between(3, 10)))
		lv := entity.StaffLeaveRequest{StaffID: s.ID, BranchID: &b.ID, FromAt: g.at(from, 0, 0), ToAt: to,
			Reason: reasons[g.r.Intn(len(reasons))], Status: status, ReviewNote: note, CreatedAt: created, UpdatedAt: created}
		if status != entity.LeaveStatusPending {
			approved := g.capNow(created.Add(time.Duration(g.r.between(2, 30)) * time.Hour))
			lv.ApprovedBy = &b.manager.ID
			lv.ApprovedAt = &approved
			lv.UpdatedAt = approved
		}
		if status == entity.LeaveStatusApproved && from.Before(g.today) {
			if g.leaveDays[s.ID] == nil {
				g.leaveDays[s.ID] = map[string]bool{}
			}
			for d := 0; d < days; d++ {
				g.leaveDays[s.ID][ymd(addDays(from, d))] = true
			}
		}
		leaves = append(leaves, lv)
	}
	for _, b := range g.branches {
		// Đã duyệt trong quá khứ (chỉ lễ tân/quản lý để không đụng lịch dạy đã có).
		add(b.receps[0], b, g.day(-g.r.between(40, 70)), 2, entity.LeaveStatusApproved, "Đồng ý, đã sắp xếp người trực thay")
		add(b.receps[len(b.receps)-1], b, g.day(-g.r.between(10, 30)), 1, entity.LeaveStatusApproved, "Đồng ý")
		add(b.manager, b, g.day(-g.r.between(75, 95)), 1, entity.LeaveStatusApproved, "Chủ phòng tập đã duyệt")
		// Bị từ chối.
		add(b.trainers[0], b, g.day(-g.r.between(20, 50)), 2, entity.LeaveStatusRejected, "Trùng lịch lớp nhóm đông học viên, đề nghị đổi ngày khác")
		// Sắp tới: chờ duyệt / đã duyệt.
		add(b.trainers[len(b.trainers)-1], b, g.day(g.r.between(9, 16)), 2, entity.LeaveStatusPending, "")
		add(b.receps[0], b, g.day(g.r.between(4, 8)), 1, entity.LeaveStatusPending, "")
		add(b.receps[len(b.receps)-1], b, g.day(g.r.between(18, 25)), 3, entity.LeaveStatusApproved, "Đồng ý, nhớ bàn giao sổ ca")
	}
	return g.tx.Create(&leaves).Error
}

// ---------------------------------------------------------------------
// Sprint 20 – Chấm công theo ca làm thực tế
// ---------------------------------------------------------------------

func (g *richGen) seedAttendance() error {
	var shifts []struct {
		ID       uint
		BranchID uint
		StaffID  uint
		StartAt  time.Time
		EndAt    time.Time
	}
	if err := g.tx.Raw(`SELECT id, branch_id, staff_id, start_at, end_at FROM shifts WHERE start_at < ? ORDER BY start_at, id`, g.now).
		Scan(&shifts).Error; err != nil {
		return err
	}
	lateNotes := []string{"Kẹt xe giờ cao điểm", "Mưa lớn, đường ngập", "Xe hỏng giữa đường", "Đưa con đi học"}
	bulk := newBulk("staff_attendances", "staff_id", "branch_id", "shift_id", "check_in_at", "check_out_at",
		"late_minutes", "early_minutes", "method", "note", "created_at", "updated_at")
	for _, sh := range shifts {
		start, end := sh.StartAt.In(g.loc), sh.EndAt.In(g.loc)
		if g.leaveDays[sh.StaffID][ymd(start)] {
			continue
		}
		ongoing := end.After(g.now)
		if !ongoing && g.r.chance(0.03) {
			continue // vắng không phép
		}
		late, early := 0, 0
		in := start.Add(-time.Duration(g.r.between(0, 12)) * time.Minute)
		note := ""
		switch p := g.r.Float64(); {
		case p < 0.80:
		case p < 0.96:
			late = g.r.between(1, 15)
		default:
			late = g.r.between(16, 45)
			note = lateNotes[g.r.Intn(len(lateNotes))]
		}
		if late > 0 {
			in = start.Add(time.Duration(late) * time.Minute)
		}
		if in.After(g.now) {
			continue
		}
		var out any
		if !ongoing {
			o := end.Add(time.Duration(g.r.between(0, 25)) * time.Minute)
			if g.r.chance(0.05) {
				early = g.r.between(10, 40)
				o = end.Add(-time.Duration(early) * time.Minute)
				note = "Xin về sớm việc gia đình (đã báo quản lý)"
			}
			out = o
		}
		method := entity.AttendanceMethodQR
		if g.r.chance(0.1) {
			method = entity.AttendanceMethodManual
			if note == "" {
				note = "Quên quét QR – quản lý xác nhận"
			}
		}
		bulk.add(sh.StaffID, sh.BranchID, sh.ID, in, out, late, early, method, note, in, in)
	}
	return bulk.flush(g.ctx, g.tx)
}

// ---------------------------------------------------------------------
// Sprint 20 – Bảng lương: cùng công thức với payrollService.Generate
// (lương cơ bản + hoa hồng PT + 2% doanh số gói bán ra) + thưởng/phạt điều chỉnh.
// ---------------------------------------------------------------------

func (g *richGen) seedPayrolls() error {
	var first time.Time
	if err := g.tx.Raw(`SELECT MIN(check_in_at) FROM staff_attendances`).Scan(&first).Error; err != nil || first.IsZero() {
		return err
	}
	cur := g.monthStart(g.today)
	period := g.monthStart(first.In(g.loc)).AddDate(0, 1, 0) // bỏ tháng đầu không trọn vẹn
	for ; !period.After(cur); period = period.AddDate(0, 1, 0) {
		next := period.AddDate(0, 1, 0)
		code := period.Format("2006-01")
		for _, b := range g.branches {
			p := entity.Payroll{BranchID: b.ID, Period: code, Status: entity.PayrollStatusDraft, CreatedAt: g.capNow(next.Add(-24 * time.Hour)), UpdatedAt: g.now}
			paid := period.Before(cur)
			if paid {
				lockedAt := g.at(addDays(next, 2), 10, 0)
				paidAt := g.at(addDays(next, 4), 15, 30)
				p.Status, p.LockedBy, p.LockedAt, p.PaidBy, p.PaidAt = entity.PayrollStatusPaid, &g.ownerID, &lockedAt, &g.ownerID, &paidAt
				p.CreatedAt, p.UpdatedAt = g.at(addDays(next, 1), 9, 0), paidAt
			}
			if err := g.tx.Create(&p).Error; err != nil {
				return err
			}
			var rows []struct {
				StaffID  uint
				Base     float64
				WorkDays float64
				LateDays int
				PTComm   float64
				Sales    float64
				Position string
				Rate     float64
			}
			// Gắn hoa hồng PT của kỳ vào bảng lương (như payrollService.Generate); kỳ đã chi → hoa hồng đã chi cùng lương.
			commUpd := map[string]any{"payroll_id": p.ID}
			if paid {
				commUpd["status"], commUpd["paid_at"], commUpd["paid_by"] = entity.CommissionPaid, *p.PaidAt, g.ownerID
			}
			if err := g.tx.Table("trainer_commissions").Where("branch_id = ? AND period = ? AND payroll_id IS NULL", b.ID, code).
				Updates(commUpd).Error; err != nil {
				return err
			}
			// Cùng nguồn số liệu với payrollRepository.CollectStaffPayrollSources: ngày công theo ngày (giờ VN),
			// hoa hồng PT đã gắn bảng lương, hoa hồng bán gói chỉ tính gói có hóa đơn PAID.
			if err := g.tx.Raw(`SELECT sp.user_id AS staff_id, sp.base_salary AS base, sp.position, sp.commission_rate AS rate,
					(SELECT COUNT(DISTINCT (sa.check_in_at AT TIME ZONE 'Asia/Ho_Chi_Minh')::date) FROM staff_attendances sa
						WHERE sa.staff_id = sp.user_id AND sa.branch_id = ? AND sa.check_in_at >= ? AND sa.check_in_at < ?) AS work_days,
					(SELECT COUNT(*) FROM staff_attendances sa WHERE sa.staff_id = sp.user_id AND sa.branch_id = ? AND sa.check_in_at >= ? AND sa.check_in_at < ? AND sa.late_minutes > 15) AS late_days,
					(SELECT COALESCE(SUM(amount), 0) FROM trainer_commissions tc WHERE tc.trainer_id = sp.user_id AND tc.payroll_id = ?) AS pt_comm,
					(SELECT COALESCE(SUM(s.price), 0) FROM subscriptions s JOIN invoices i ON i.id = s.invoice_id AND i.status = 'PAID'
						WHERE s.sold_by = sp.user_id AND s.branch_id = ? AND s.status <> 'CANCELLED' AND s.created_at >= ? AND s.created_at < ?) AS sales
				FROM staff_profiles sp
				JOIN user_roles ur ON ur.user_id = sp.user_id AND ur.branch_id = ?
				ORDER BY sp.user_id`,
				b.ID, period, next, b.ID, period, next, p.ID, b.ID, period, next, b.ID).Scan(&rows).Error; err != nil {
				return err
			}
			var items []entity.PayrollItem
			for _, row := range rows {
				rate := row.Rate
				if rate <= 0 || row.Position == "TRAINER" {
					rate = 2
				}
				salesComm := roundVND(row.Sales * rate / 100)
				it := entity.PayrollItem{PayrollID: p.ID, StaffID: row.StaffID, BaseSalary: row.Base, WorkDays: row.WorkDays,
					PTCommission: row.PTComm, SalesCommission: salesComm, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
				if paid {
					var notes []string
					switch {
					case row.Position == "MANAGER":
						it.Bonus = float64(g.r.between(3, 6)) * 500000
						notes = append(notes, "Thưởng hoàn thành chỉ tiêu chi nhánh")
					case row.PTComm >= 3000000:
						it.Bonus = 1000000
						notes = append(notes, "Thưởng HLV vượt KPI buổi dạy")
					case row.Sales >= 15000000:
						it.Bonus = 500000
						notes = append(notes, "Thưởng doanh số bán gói")
					}
					if row.LateDays > 0 {
						it.Penalty = float64(row.LateDays) * 50000
						notes = append(notes, fmt.Sprintf("Trừ đi muộn %d lần", row.LateDays))
					}
					it.Note = strings.Join(notes, "; ")
				}
				it.RecalcTotal()
				items = append(items, it)
			}
			if len(items) > 0 {
				if err := g.tx.Create(&items).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------
// Sprint 21 – Chi phí vận hành
// ---------------------------------------------------------------------

func (g *richGen) seedExpenses() error {
	var cats []entity.ExpenseCategory
	if err := g.tx.Order("id ASC").Find(&cats).Error; err != nil {
		return err
	}
	catID := map[string]uint{}
	for _, c := range cats {
		switch {
		case strings.Contains(c.Name, "thuê mặt bằng"):
			catID["rent"] = c.ID
		case strings.Contains(c.Name, "Điện"):
			catID["util"] = c.ID
		case strings.Contains(c.Name, "Bảo trì"):
			catID["maint"] = c.ID
		case strings.Contains(c.Name, "Marketing"):
			catID["mkt"] = c.ID
		case strings.Contains(c.Name, "Vật tư"):
			catID["supply"] = c.ID
		}
	}
	if len(catID) < 5 {
		return nil
	}
	rent := []float64{45000000, 38000000, 60000000}
	mkt := []string{
		"Chạy quảng cáo Facebook Ads chương trình hội viên mới",
		"In tờ rơi, standee ưu đãi tháng",
		"Quảng cáo Google Maps & TikTok",
		"Tài trợ giải chạy bộ cộng đồng quận",
	}
	maint := []string{
		"Thay dây cáp máy kéo xô",
		"Bảo dưỡng định kỳ 6 máy chạy bộ",
		"Thay đệm ghế tập ngực",
		"Sửa điều hòa phòng Yoga",
		"Thay bóng đèn LED khu tạ tự do",
	}
	supply := []string{
		"Khăn tập, nước rửa tay, giấy lau",
		"Hóa chất vệ sinh sàn & dụng cụ",
		"Nước uống bình 20L cho khu chờ",
	}
	cur := g.monthStart(g.today)
	var list []entity.Expense
	for mi := 5; mi >= 0; mi-- {
		month := cur.AddDate(0, -mi, 0)
		isCur := mi == 0
		rejectDay := g.r.between(14, 26)
		for bi, b := range g.branches {
			add := func(day int, key string, amount float64, desc string) {
				spent := time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, g.loc)
				if spent.After(g.today) {
					return
				}
				e := entity.Expense{BranchID: b.ID, CategoryID: catID[key], Amount: amount, SpentAt: pgDate(spent), Description: desc,
					CreatedBy: &b.manager.ID, Status: entity.ExpenseStatusApproved, ApprovedBy: &g.ownerID,
					CreatedAt: g.at(spent, 10, g.r.between(0, 59)), UpdatedAt: g.at(spent, 16, 0)}
				if isCur && key != "rent" && key != "util" && g.r.chance(0.6) {
					e.Status, e.ApprovedBy = entity.ExpenseStatusPending, nil
				}
				list = append(list, e)
			}
			label := month.Format("01/2006")
			add(5, "rent", rent[bi], "Tiền thuê mặt bằng tháng "+label)
			add(8, "util", float64(g.r.between(90, 150))*100000, "Tiền điện, nước & Internet tháng "+label)
			add(12, "supply", float64(g.r.between(15, 40))*100000, supply[g.r.Intn(len(supply))])
			add(25, "supply", float64(g.r.between(10, 30))*100000, supply[g.r.Intn(len(supply))])
			add(15, "mkt", float64(g.r.between(5, 20))*1000000, mkt[g.r.Intn(len(mkt))])
			add(18, "maint", float64(g.r.between(15, 80))*100000, maint[g.r.Intn(len(maint))])
			if mi == 1 {
				spent := time.Date(month.Year(), month.Month(), rejectDay, 0, 0, 0, 0, g.loc)
				list = append(list, entity.Expense{BranchID: b.ID, CategoryID: catID["maint"], Amount: 86000000, SpentAt: pgDate(spent),
					Description: "Đề xuất mua thêm 2 máy chạy bộ – hoãn sang quý sau", CreatedBy: &b.manager.ID, ApprovedBy: &g.ownerID,
					Status: entity.ExpenseStatusRejected, CreatedAt: g.at(spent, 9, 30), UpdatedAt: g.at(spent, 17, 0)})
			}
		}
	}
	return g.tx.CreateInBatches(&list, 200).Error
}

// ---------------------------------------------------------------------
// Sprint 21 – Ca thu ngân: tiền đầu ca + tiền mặt thực thu trong ngày
// ---------------------------------------------------------------------

func (g *richGen) seedCashSessions() error {
	from := g.day(-45)
	var rows []struct {
		BranchID uint
		D        string
		Amount   float64
	}
	if err := g.tx.Raw(`SELECT i.branch_id, to_char(p.paid_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM-DD') AS d, SUM(p.amount) AS amount
		FROM payments p JOIN invoices i ON i.id = p.invoice_id
		WHERE p.method = 'CASH' AND p.paid_at >= ?
		GROUP BY 1, 2`, from).Scan(&rows).Error; err != nil {
		return err
	}
	cash := map[string]float64{}
	for _, r := range rows {
		cash[fmt.Sprintf("%d|%s", r.BranchID, r.D)] = r.Amount
	}
	const opening = 2000000
	var list []entity.CashSession
	for _, b := range g.branches {
		openH, openM := hhmmToMin(b.OpenTime)/60, hhmmToMin(b.OpenTime)%60
		closeH, closeM := hhmmToMin(b.CloseTime)/60, hhmmToMin(b.CloseTime)%60
		for off := -45; off <= 0; off++ {
			d := g.day(off)
			staff := b.receps[(off+45)%len(b.receps)]
			expected := opening + cash[fmt.Sprintf("%d|%s", b.ID, ymd(d))]
			s := entity.CashSession{BranchID: b.ID, StaffID: staff.ID, OpenedAt: g.at(d, openH, openM), OpeningCash: opening,
				ExpectedCash: expected, Status: entity.CashSessionClosed, CreatedAt: g.at(d, openH, openM)}
			if off == 0 {
				s.Status = entity.CashSessionOpen
				s.Note = "Ca thu ngân đang mở tại quầy " + b.short
				s.UpdatedAt = g.now
				list = append(list, s)
				continue
			}
			closed := g.at(d, closeH, closeM)
			diff := 0.0
			if g.r.chance(0.15) {
				diff = float64(g.r.between(-10, 4)) * 5000
			}
			s.ClosedAt = &closed
			s.CountedCash = expected + diff
			s.Difference = diff
			s.UpdatedAt = closed
			switch {
			case diff < 0:
				s.Note = "Thiếu tiền lẻ trả lại khách, đã ghi biên bản"
			case diff > 0:
				s.Note = "Thừa tiền do khách không lấy tiền thừa"
			default:
				s.Note = "Khớp sổ"
			}
			list = append(list, s)
		}
	}
	return g.tx.CreateInBatches(&list, 200).Error
}

// ---------------------------------------------------------------------
// Sprint 26 – Chỉ tiêu KPI: đặt quanh số thực tế để báo cáo có tỷ lệ đạt/không đạt hợp lý
// ---------------------------------------------------------------------

func (g *richGen) seedSalesTargets() error {
	cur := g.monthStart(g.today)
	actual := func(b *rsBranch, metric string, from, to time.Time) (float64, error) {
		var v float64
		var q string
		switch metric {
		case entity.KPIMetricRevenue:
			q = `SELECT COALESCE(SUM(p.amount), 0) FROM payments p JOIN invoices i ON i.id = p.invoice_id WHERE i.branch_id = ? AND p.paid_at >= ? AND p.paid_at < ?`
		case entity.KPIMetricNewMembers:
			q = `SELECT COUNT(*) FROM members WHERE home_branch_id = ? AND created_at >= ? AND created_at < ?`
		case entity.KPIMetricRenewals:
			q = `SELECT COUNT(*) FROM subscription_events e JOIN subscriptions s ON s.id = e.subscription_id WHERE s.branch_id = ? AND e.event_type = 'RENEWED' AND e.created_at >= ? AND e.created_at < ?`
		case entity.KPIMetricPTSales:
			q = `SELECT COALESCE(SUM(price), 0) FROM pt_contracts WHERE branch_id = ? AND status <> 'CANCELLED' AND created_at >= ? AND created_at < ?`
		}
		err := g.tx.Raw(q, b.ID, from, to).Scan(&v).Error
		return v, err
	}
	round := func(metric string, v float64) float64 {
		if metric == entity.KPIMetricRevenue || metric == entity.KPIMetricPTSales {
			return math.Max(5000000, math.Round(v/5000000)*5000000)
		}
		return math.Max(3, math.Round(v))
	}
	metrics := []string{entity.KPIMetricRevenue, entity.KPIMetricNewMembers, entity.KPIMetricRenewals, entity.KPIMetricPTSales}
	var list []entity.SalesTarget
	for _, b := range g.branches {
		for mi := -4; mi <= 1; mi++ {
			month := cur.AddDate(0, mi, 0)
			for _, metric := range metrics {
				var base float64
				var err error
				if mi < 0 {
					base, err = actual(b, metric, month, month.AddDate(0, 1, 0))
					base *= float64(g.r.between(85, 118)) / 100
				} else {
					prev := cur.AddDate(0, -1, 0)
					base, err = actual(b, metric, prev, cur)
					base *= float64(g.r.between(105, 115)) / 100
				}
				if err != nil {
					return err
				}
				created := g.at(addDays(month, -3), 9, 0)
				if created.After(g.now) {
					created = g.now.Add(-2 * time.Hour)
				}
				list = append(list, entity.SalesTarget{BranchID: b.ID, Period: month.Format("2006-01"), Metric: metric,
					TargetValue: round(metric, base), CreatedAt: created, UpdatedAt: created})
			}
		}
		for _, rc := range b.receps {
			id := rc.ID
			list = append(list, entity.SalesTarget{BranchID: b.ID, StaffID: &id, Period: cur.Format("2006-01"), Metric: entity.KPIMetricNewMembers,
				TargetValue: float64(g.r.between(6, 12)), CreatedAt: g.at(addDays(cur, -3), 9, 0), UpdatedAt: g.at(addDays(cur, -3), 9, 0)})
		}
	}
	return g.tx.Create(&list).Error
}

// ---------------------------------------------------------------------
// Sprint 25 – Ngày lễ & giờ hoạt động đặc biệt (chỉ ngày tương lai hoặc đã qua, không trùng hôm nay)
// ---------------------------------------------------------------------

func (g *richGen) seedHolidays() error {
	str := func(s string) *string { return &s }
	y := g.today.Year()
	type hd struct {
		branch      *rsBranch
		date        time.Time
		name        string
		closed      bool
		open, close string
	}
	d := func(year int, m time.Month, day int) time.Time { return time.Date(year, m, day, 0, 0, 0, 0, g.loc) }
	defs := []hd{
		{nil, d(y, time.April, 30), "Ngày Giải phóng miền Nam 30/4", false, "07:00", "17:00"},
		{nil, d(y, time.May, 1), "Quốc tế Lao động 1/5", false, "07:00", "17:00"},
		{nil, d(y, time.September, 2), "Quốc khánh 2/9", false, "07:00", "17:00"},
		{nil, d(y+1, time.January, 1), "Tết Dương lịch", false, "07:00", "17:00"},
		{nil, d(y+1, time.February, 5), "29 Tết – mở cửa nửa ngày", false, "06:00", "12:00"},
		{nil, d(y+1, time.February, 6), "Mùng 1 Tết Nguyên đán", true, "", ""},
		{nil, d(y+1, time.February, 7), "Mùng 2 Tết Nguyên đán", true, "", ""},
		{nil, d(y+1, time.February, 8), "Mùng 3 Tết Nguyên đán", true, "", ""},
		{nil, d(y+1, time.February, 9), "Mùng 4 Tết – mở cửa rút ngắn", false, "08:00", "18:00"},
		{nil, d(y+1, time.April, 30), "Ngày Giải phóng miền Nam 30/4", false, "07:00", "17:00"},
	}
	if len(g.branches) >= 3 {
		defs = append(defs,
			hd{g.branches[0], d(y, time.December, 24), "Đêm Giáng sinh – đóng cửa sớm", false, "05:30", "20:00"},
			hd{g.branches[2], addDays(g.today, 24), "Bảo trì hệ thống điện toàn tòa nhà", true, "", ""},
			hd{g.branches[1], addDays(g.today, -33), "Phun khử khuẩn định kỳ – mở muộn", false, "09:00", "22:00"},
		)
	}
	var list []entity.BranchHoliday
	for _, h := range defs {
		if h.date.Equal(g.today) {
			continue
		}
		bh := entity.BranchHoliday{Date: pgDate(h.date), Name: h.name, Closed: h.closed,
			CreatedAt: g.capNow(h.date.AddDate(0, 0, -20)), UpdatedAt: g.capNow(h.date.AddDate(0, 0, -20))}
		if h.branch != nil {
			bh.BranchID = &h.branch.ID
		}
		if !h.closed {
			bh.OpenTime, bh.CloseTime = str(h.open), str(h.close)
		}
		list = append(list, bh)
	}
	return g.tx.Create(&list).Error
}

// ---------------------------------------------------------------------
// Sprint 25 – Lịch rảnh HLV (khớp ngày nghỉ cố định trong tuần của ca làm)
// ---------------------------------------------------------------------

func (g *richGen) seedAvailabilities() error {
	var list []entity.TrainerAvailability
	for _, b := range g.branches {
		for ti, t := range b.trainers {
			rest := (7 - ti%7) % 7 // isoWeekday nghỉ: (wd+ti)%7 == 0
			if rest == 0 {
				rest = 7
			}
			for iso := 1; iso <= 7; iso++ {
				if iso == rest {
					continue
				}
				dow := iso % 7 // 0 = Chủ nhật
				if iso == 7 {
					list = append(list, entity.TrainerAvailability{TrainerID: t.ID, BranchID: b.ID, DOW: dow, StartTime: "07:00", EndTime: "11:30"})
					continue
				}
				list = append(list,
					entity.TrainerAvailability{TrainerID: t.ID, BranchID: b.ID, DOW: dow, StartTime: "06:00", EndTime: "12:00"},
					entity.TrainerAvailability{TrainerID: t.ID, BranchID: b.ID, DOW: dow, StartTime: "16:00", EndTime: "21:00"},
				)
			}
		}
	}
	return g.tx.Create(&list).Error
}

// ---------------------------------------------------------------------
// Sprint 22 – Giới thiệu bạn bè, huy hiệu, thử thách, điểm thưởng
// ---------------------------------------------------------------------

func (g *richGen) seedReferrals() error {
	byBranch := map[uint][]*rsMember{}
	for _, m := range g.members {
		if m.ReferralCode != nil {
			byBranch[m.HomeBranchID] = append(byBranch[m.HomeBranchID], m)
		}
	}
	for _, m := range g.members {
		if m.Source != "REFERRAL" {
			continue
		}
		var cands []*rsMember
		for _, c := range byBranch[m.HomeBranchID] {
			if c.ID != m.ID && c.CreatedAt.Before(m.CreatedAt) {
				cands = append(cands, c)
			}
		}
		if len(cands) == 0 {
			continue
		}
		ref := cands[g.r.Intn(len(cands))]
		if g.showcase != nil && g.showcase.ReferralCode != nil && g.showcase.HomeBranchID == m.HomeBranchID &&
			g.showcase.ID != m.ID && g.showcase.CreatedAt.Before(m.CreatedAt) && g.r.chance(0.15) {
			ref = g.showcase
		}
		if err := g.tx.Exec(`UPDATE members SET referred_by = ? WHERE id = ?`, ref.ID, m.ID).Error; err != nil {
			return err
		}
		id := m.ID
		g.addLedger(ref.ID, 100, entity.LoyaltyReasonReferral, "MEMBER", &id, "Giới thiệu thành công hội viên "+m.FullName, nil, m.CreatedAt.Add(2*time.Hour))
	}
	return nil
}

func (g *richGen) seedBadges() error {
	var badges []entity.Badge
	if err := g.tx.Find(&badges).Error; err != nil {
		return err
	}
	badgeID := map[string]uint{}
	for _, b := range badges {
		badgeID[b.Code] = b.ID
	}
	var rows []struct {
		MemberID uint
		Rn       int
		At       time.Time
		Kind     string
	}
	if err := g.tx.Raw(`
		SELECT member_id, rn, at, 'C' AS kind FROM (
			SELECT member_id, checked_in_at AS at, ROW_NUMBER() OVER (PARTITION BY member_id ORDER BY checked_in_at) AS rn
			FROM checkins WHERE result = 'ALLOWED' AND member_id IS NOT NULL) t WHERE rn IN (7, 30, 100)
		UNION ALL
		SELECT member_id, rn, at, 'K' AS kind FROM (
			SELECT member_id, attended_at AS at, ROW_NUMBER() OVER (PARTITION BY member_id ORDER BY attended_at) AS rn
			FROM class_bookings WHERE status = 'ATTENDED' AND attended_at IS NOT NULL) t WHERE rn = 10`).Scan(&rows).Error; err != nil {
		return err
	}
	codeOf := map[string]string{"C7": "STREAK_7", "C30": "STREAK_30", "C100": "CENTURY_100", "K10": "CLASS_LOVER"}
	bulk := newBulk("member_badges", "member_id", "badge_id", "earned_at")
	for _, r := range rows {
		id := badgeID[codeOf[fmt.Sprintf("%s%d", r.Kind, r.Rn)]]
		if id == 0 {
			continue
		}
		bulk.add(r.MemberID, id, r.At)
	}
	return bulk.flush(g.ctx, g.tx)
}

func (g *richGen) seedChallenges() error {
	cur := g.monthStart(g.today)
	prev := cur.AddDate(0, -1, 0)
	summerStart := time.Date(g.today.Year(), time.July, 1, 0, 0, 0, 0, g.loc)
	type chDef struct {
		c      entity.Challenge
		branch *rsBranch
	}
	var cg *rsBranch
	if len(g.branches) > 0 {
		cg = g.branches[0]
	}
	defs := []chDef{
		{c: entity.Challenge{Name: "Thử thách mùa hè – 30 buổi tập", Description: "Hoàn thành 30 lượt check-in trong tháng 7 và tháng 8 để nhận 500 điểm thưởng.",
			Metric: entity.ChallengeMetricCheckins, Target: 30, StartDate: summerStart, EndDate: summerStart.AddDate(0, 2, -1), RewardPoints: 500, Status: "ENDED"}},
		{c: entity.Challenge{Name: "Chiến binh tháng " + prev.Format("01/2006"), Description: "Đạt 12 buổi tập trong tháng để nhận 300 điểm thưởng.",
			Metric: entity.ChallengeMetricCheckins, Target: 12, StartDate: prev, EndDate: cur.AddDate(0, 0, -1), RewardPoints: 300, Status: "ENDED"}},
		{c: entity.Challenge{Name: "16 buổi tập tháng " + cur.Format("01/2006"), Description: "Check-in đủ 16 buổi trong tháng này để nhận 300 điểm và huy hiệu tinh thần thép.",
			Metric: entity.ChallengeMetricCheckins, Target: 16, StartDate: cur, EndDate: cur.AddDate(0, 1, -1), RewardPoints: 300, Status: "ACTIVE"}},
		{c: entity.Challenge{Name: "Ngôi sao lớp nhóm Cầu Giấy", Description: "Tham gia 8 buổi lớp nhóm (Yoga, Zumba, Spinning…) trong tháng tại Cầu Giấy.",
			Metric: entity.ChallengeMetricClasses, Target: 8, StartDate: cur, EndDate: cur.AddDate(0, 1, -1), RewardPoints: 200, Status: "ACTIVE"}, branch: cg},
		{c: entity.Challenge{Name: "Kỷ luật cùng PT – 10 buổi", Description: "Hoàn thành 10 buổi tập cá nhân với HLV trong 2 tháng.",
			Metric: entity.ChallengeMetricPTSessions, Target: 10, StartDate: prev, EndDate: cur.AddDate(0, 1, -1), RewardPoints: 250, Status: "ACTIVE"}},
		{c: entity.Challenge{Name: "Về đích cuối năm – 20 buổi tháng 12", Description: "Khởi động từ 01/12: tập đủ 20 buổi trong tháng 12 để nhận 400 điểm.",
			Metric: entity.ChallengeMetricCheckins, Target: 20, StartDate: time.Date(g.today.Year(), time.December, 1, 0, 0, 0, 0, g.loc),
			EndDate: time.Date(g.today.Year(), time.December, 31, 0, 0, 0, 0, g.loc), RewardPoints: 400, Status: "ACTIVE"}},
	}
	events := map[string]string{
		entity.ChallengeMetricCheckins:   `SELECT member_id, checked_in_at AS at FROM checkins WHERE result = 'ALLOWED' AND member_id IS NOT NULL`,
		entity.ChallengeMetricClasses:    `SELECT member_id, attended_at AS at FROM class_bookings WHERE status = 'ATTENDED' AND attended_at IS NOT NULL`,
		entity.ChallengeMetricPTSessions: `SELECT member_id, completed_at AS at FROM pt_sessions WHERE status = 'COMPLETED' AND completed_at IS NOT NULL`,
	}
	bulk := newBulk("challenge_participants", "challenge_id", "member_id", "progress", "completed_at", "joined_at")
	for _, d := range defs {
		c := d.c
		if d.branch != nil {
			c.BranchID = &d.branch.ID
		}
		start, end := c.StartDate, c.EndDate
		c.StartDate, c.EndDate = pgDate(start), pgDate(end)
		c.CreatedAt = g.capNow(start.AddDate(0, 0, -5))
		c.UpdatedAt = c.CreatedAt
		if err := g.tx.Create(&c).Error; err != nil {
			return err
		}
		if start.After(g.now) {
			continue
		}
		var rows []struct {
			MemberID uint
			Cnt      int
			DoneAt   *time.Time
		}
		if err := g.tx.Raw(`SELECT member_id, COUNT(*) AS cnt, MAX(CASE WHEN rn = ? THEN at END) AS done_at FROM (
				SELECT member_id, at, ROW_NUMBER() OVER (PARTITION BY member_id ORDER BY at) AS rn
				FROM (`+events[c.Metric]+`) e WHERE at >= ? AND at < ?) x GROUP BY member_id`,
			c.Target, start, addDays(end, 1)).Scan(&rows).Error; err != nil {
			return err
		}
		chID := c.ID
		for _, r := range rows {
			m := g.memberByID[r.MemberID]
			if m == nil || (d.branch != nil && m.HomeBranchID != d.branch.ID) {
				continue
			}
			if m != g.showcase && !g.r.chance(0.55) {
				continue
			}
			joined := g.at(addDays(start, g.r.between(0, 3)), g.r.between(7, 21), g.r.between(0, 59))
			if r.DoneAt != nil && joined.After(*r.DoneAt) {
				joined = r.DoneAt.Add(-24 * time.Hour)
			}
			joined = g.capNow(joined)
			progress := r.Cnt
			if progress > c.Target {
				progress = c.Target
			}
			var done any
			if r.DoneAt != nil {
				done = *r.DoneAt
				g.addLedger(m.ID, c.RewardPoints, entity.LoyaltyReasonChallenge, "CHALLENGE", &chID, "Hoàn thành thử thách: "+c.Name, nil, *r.DoneAt)
			}
			bulk.add(c.ID, m.ID, progress, done, joined)
		}
	}
	return bulk.flush(g.ctx, g.tx)
}

// seedLoyalty: tích điểm từ hóa đơn đã thanh toán (1 điểm/10.000đ), check-in (5 điểm/ngày, gộp theo tháng),
// đánh giá (+20); sau đó đổi quà cho một phần hội viên và cập nhật số dư members.loyalty_points.
func (g *richGen) seedLoyalty() error {
	var invs []struct {
		ID        uint
		MemberID  uint
		InvoiceNo string
		Amt       float64
		IssuedAt  time.Time
	}
	if err := g.tx.Raw(`SELECT id, member_id, invoice_no, paid_amount - refunded_amount AS amt, issued_at FROM invoices
		WHERE member_id IS NOT NULL AND status = 'PAID' ORDER BY issued_at`).Scan(&invs).Error; err != nil {
		return err
	}
	for _, iv := range invs {
		pts := int(iv.Amt / 10000)
		if pts <= 0 {
			continue
		}
		id := iv.ID
		g.addLedger(iv.MemberID, pts, entity.LoyaltyReasonRenew, "INVOICE", &id, "Tích điểm thanh toán hóa đơn "+iv.InvoiceNo, nil, iv.IssuedAt.Add(time.Minute))
	}

	var cks []struct {
		MemberID uint
		M        string
		Days     int
	}
	if err := g.tx.Raw(`SELECT member_id, to_char(checked_in_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM') AS m,
			COUNT(DISTINCT (checked_in_at AT TIME ZONE 'Asia/Ho_Chi_Minh')::date) AS days
		FROM checkins WHERE result = 'ALLOWED' AND member_id IS NOT NULL GROUP BY 1, 2`).Scan(&cks).Error; err != nil {
		return err
	}
	for _, c := range cks {
		ms, err := time.ParseInLocation("2006-01", c.M, g.loc)
		if err != nil {
			continue
		}
		at := g.capNow(g.at(ms.AddDate(0, 1, -1), 23, 0))
		g.addLedger(c.MemberID, c.Days*5, entity.LoyaltyReasonCheckin, "", nil,
			fmt.Sprintf("Tích điểm check-in tháng %s (%d buổi × 5 điểm)", ms.Format("01/2006"), c.Days), nil, at)
	}

	var revs []struct {
		ID        uint
		MemberID  uint
		CreatedAt time.Time
	}
	if err := g.tx.Raw(`SELECT id, member_id, created_at FROM reviews`).Scan(&revs).Error; err != nil {
		return err
	}
	for _, rv := range revs {
		id := rv.ID
		g.addLedger(rv.MemberID, 20, entity.LoyaltyReasonReview, "REVIEW", &id, "Cảm ơn bạn đã đánh giá dịch vụ", nil, rv.CreatedAt.Add(time.Minute))
	}

	// Đổi quà: nước khoáng (sản phẩm) và voucher giảm giá (sinh mã khuyến mãi dùng 1 lần).
	var rewards []entity.LoyaltyReward
	if err := g.tx.Where("status = ?", consts.StatusActive).Order("cost_points ASC").Find(&rewards).Error; err != nil {
		return err
	}
	var water, voucher *entity.LoyaltyReward
	for i := range rewards {
		if rewards[i].RewardType == entity.RewardTypeProduct && water == nil {
			water = &rewards[i]
		}
		if rewards[i].RewardType == entity.RewardTypeVoucher && voucher == nil {
			voucher = &rewards[i]
		}
	}
	used := map[uint]int{}
	var promos []entity.Promotion
	ids := make([]uint, 0, len(g.points))
	for id := range g.points {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, mid := range ids {
		m := g.memberByID[mid]
		if m == nil || m.UserID == nil {
			continue
		}
		isShow := m == g.showcase
		if voucher != nil && g.points[mid] >= voucher.CostPoints+100 && (isShow || g.r.chance(0.25)) {
			at := g.now.Add(-time.Duration(g.r.between(2, 40*24)) * time.Hour)
			if isShow {
				at = g.now.Add(-50 * time.Hour)
			}
			code := fmt.Sprintf("RWD-%s-%04X", strings.ToUpper(m.MemberCode), g.r.Intn(0xFFFF))
			one := 1
			usedCount := 0
			if !isShow && g.r.chance(0.4) {
				usedCount = 1
			}
			disc := voucher.DiscountPct
			if disc <= 0 {
				disc = 10
			}
			promos = append(promos, entity.Promotion{
				BaseEntity: entity.BaseEntity{CreatedAt: at, UpdatedAt: at, CreatedBy: m.UserID, UpdatedBy: m.UserID},
				Code:       code, Name: fmt.Sprintf("Voucher đổi điểm (%s) - %s", m.MemberCode, voucher.Name),
				Type: entity.PromotionPercent, Value: disc, AppliesTo: entity.AppliesAll, StartAt: at, EndAt: at.AddDate(0, 0, 30),
				UsageLimit: &one, UsedCount: usedCount, Status: consts.StatusActive})
			rid := voucher.ID
			g.addLedger(mid, -voucher.CostPoints, entity.LoyaltyReasonRedeem, "REWARD", &rid,
				fmt.Sprintf("Đổi quà: %s (Mã: %s)", voucher.Name, code), m.UserID, at)
			used[voucher.ID]++
		}
		if water != nil && g.points[mid] >= water.CostPoints && g.r.chance(0.2) {
			at := g.now.Add(-time.Duration(g.r.between(2, 60*24)) * time.Hour)
			rid := water.ID
			g.addLedger(mid, -water.CostPoints, entity.LoyaltyReasonRedeem, "REWARD", &rid, "Đổi quà: "+water.Name, m.UserID, at)
			used[water.ID]++
		}
	}
	// Một vài điều chỉnh thủ công của quản lý.
	adj := g.shuffled(g.activeMembers(0, ""))
	for i := 0; i < 6 && i < len(adj); i++ {
		at := g.now.Add(-time.Duration(g.r.between(24, 30*24)) * time.Hour)
		g.addLedger(adj[i].ID, 50, entity.LoyaltyReasonAdjust, "", nil, "Bù điểm do lỗi đồng bộ check-in", &g.ownerID, at)
	}

	if len(promos) > 0 {
		if err := g.tx.Create(&promos).Error; err != nil {
			return err
		}
	}
	for rid, n := range used {
		if err := g.tx.Exec(`UPDATE loyalty_rewards SET stock = GREATEST(stock - ?, 0) WHERE id = ?`, n, rid).Error; err != nil {
			return err
		}
	}
	if err := g.ledger.flush(g.ctx, g.tx); err != nil {
		return err
	}
	return g.tx.Exec(`UPDATE members m SET loyalty_points = s.pts
		FROM (SELECT member_id, GREATEST(SUM(points), 0) AS pts FROM loyalty_ledgers GROUP BY member_id) s
		WHERE s.member_id = m.id`).Error
}

// ---------------------------------------------------------------------
// Sprint 23 – Thực đơn dinh dưỡng & nhật ký ăn uống
// ---------------------------------------------------------------------

func (g *richGen) seedMealPlans() error {
	var foods []entity.Food
	if err := g.tx.Order("id ASC").Find(&foods).Error; err != nil {
		return err
	}
	fid := map[string]uint{}
	for _, f := range foods {
		for _, key := range []string{"Ức gà", "Cơm trắng", "Khoai lang", "Trứng gà", "Thịt bò", "Cá hồi", "Phở bò", "Bún bò", "Chuối", "Sữa chua", "Whey", "Yến mạch", "Bông cải", "Đậu phụ"} {
			if strings.HasPrefix(f.Name, key) {
				fid[key] = f.ID
			}
		}
	}
	if len(fid) < 14 {
		return nil
	}
	type portion struct {
		key  string
		qty  float64
		carb bool
	}
	menus := map[string][][]portion{
		entity.MealBreakfast: {
			{{"Yến mạch", 1, true}, {"Chuối", 1, true}, {"Sữa chua", 1, false}},
			{{"Phở bò", 1, true}},
			{{"Trứng gà", 2, false}, {"Khoai lang", 1.5, true}},
			{{"Whey", 1, false}, {"Chuối", 1, true}, {"Yến mạch", 1, true}},
		},
		entity.MealLunch: {
			{{"Cơm trắng", 1, true}, {"Ức gà", 1.5, false}, {"Bông cải", 1, false}},
			{{"Cơm trắng", 1, true}, {"Thịt bò", 1, false}, {"Bông cải", 1, false}},
			{{"Cơm trắng", 1, true}, {"Cá hồi", 1.2, false}, {"Bông cải", 1, false}},
			{{"Bún bò", 1, true}},
		},
		entity.MealDinner: {
			{{"Khoai lang", 2, true}, {"Ức gà", 1.5, false}, {"Bông cải", 1, false}},
			{{"Đậu phụ", 2, false}, {"Cơm trắng", 0.7, true}, {"Bông cải", 1, false}},
			{{"Cá hồi", 1, false}, {"Khoai lang", 1.5, true}},
			{{"Trứng gà", 2, false}, {"Đậu phụ", 1.5, false}, {"Bông cải", 1.5, false}},
		},
		entity.MealSnack: {
			{{"Sữa chua", 1, false}}, {{"Whey", 1, false}}, {{"Chuối", 1, true}},
		},
	}
	goals := []struct {
		name, goal string
		kcal       int
		carbF      float64
	}{
		{"Thực đơn Giảm mỡ (Cutting %d kcal)", "Giảm 4–6kg mỡ trong 3 tháng, giữ khối cơ", 1700, 0.75},
		{"Thực đơn Tăng cơ (Clean Bulk %d kcal)", "Tăng cơ nạc, kiểm soát tỷ lệ mỡ dưới 15%", 2600, 1.35},
		{"Thực đơn Duy trì vóc dáng (%d kcal)", "Giữ cân nặng ổn định, đủ năng lượng tập luyện", 2100, 1},
	}
	var contracts []struct {
		ID        uint
		MemberID  uint
		TrainerID uint
		CreatedAt time.Time
	}
	if err := g.tx.Raw(`SELECT DISTINCT ON (member_id) id, member_id, trainer_id, created_at FROM pt_contracts
		WHERE status = 'ACTIVE' ORDER BY member_id, id DESC`).Scan(&contracts).Error; err != nil {
		return err
	}
	if g.showcase != nil {
		found := false
		for _, c := range contracts {
			if c.MemberID == g.showcase.ID {
				found = true
			}
		}
		if !found {
			b := g.branchByID[g.showcase.HomeBranchID]
			if b != nil {
				contracts = append([]struct {
					ID        uint
					MemberID  uint
					TrainerID uint
					CreatedAt time.Time
				}{{MemberID: g.showcase.ID, TrainerID: b.trainers[0].ID, CreatedAt: g.now.AddDate(0, 0, -20)}}, contracts...)
			}
		}
	}

	type planItems struct {
		plan  entity.MealPlan
		items []entity.MealPlanItem
		carbF float64
	}
	var plans []*planItems
	logMembers := map[uint]*planItems{}
	for i, c := range contracts {
		if i >= 45 {
			break
		}
		gl := goals[g.r.Intn(len(goals))]
		kcal := gl.kcal + g.r.between(-2, 2)*50
		start := dayOf(c.CreatedAt.In(g.loc), g.loc).AddDate(0, 0, g.r.between(1, 5))
		if start.After(g.today) {
			start = g.today
		}
		end := start.AddDate(0, 0, 56)
		trainer := c.TrainerID
		p := &planItems{carbF: gl.carbF, plan: entity.MealPlan{MemberID: c.MemberID, TrainerID: &trainer, Name: fmt.Sprintf(gl.name, kcal), Goal: gl.goal,
			TargetCalories: kcal, StartDate: ptrTime(pgDate(start)), EndDate: ptrTime(pgDate(end)), Status: "ACTIVE",
			CreatedAt: g.capNow(g.at(start, 9, 0)), UpdatedAt: g.capNow(g.at(start, 9, 0))}}
		for day := 1; day <= 7; day++ {
			for _, meal := range []string{entity.MealBreakfast, entity.MealLunch, entity.MealDinner, entity.MealSnack} {
				if meal == entity.MealSnack && (day+i)%2 == 1 {
					continue
				}
				opts := menus[meal]
				menu := opts[(day+i+len(meal))%len(opts)]
				for _, pt := range menu {
					q := pt.qty
					if pt.carb {
						q = math.Round(q*gl.carbF*10) / 10
					}
					note := ""
					if meal == entity.MealBreakfast && day == 1 {
						note = "Ăn trước buổi tập 60 phút"
					}
					p.items = append(p.items, entity.MealPlanItem{DayIndex: day, Meal: meal, FoodID: fid[pt.key], Qty: q, Note: note})
				}
			}
		}
		plans = append(plans, p)
		if (g.showcase != nil && c.MemberID == g.showcase.ID) || len(logMembers) < 20 {
			logMembers[c.MemberID] = p
		}
		// Thực đơn cũ đã lưu trữ cho một số hội viên.
		if i%4 == 0 {
			oldStart := start.AddDate(0, 0, -70)
			oldEnd := start.AddDate(0, 0, -14)
			plans = append(plans, &planItems{carbF: 1, plan: entity.MealPlan{MemberID: c.MemberID, TrainerID: &trainer, Name: "Thực đơn làm quen (2000 kcal)",
				Goal: "Xây dựng thói quen ăn uống điều độ", TargetCalories: 2000, StartDate: ptrTime(pgDate(oldStart)), EndDate: ptrTime(pgDate(oldEnd)),
				Status: "ARCHIVED", CreatedAt: g.at(oldStart, 9, 0), UpdatedAt: g.at(oldEnd, 9, 0)},
				items: []entity.MealPlanItem{
					{DayIndex: 1, Meal: entity.MealBreakfast, FoodID: fid["Phở bò"], Qty: 1},
					{DayIndex: 1, Meal: entity.MealLunch, FoodID: fid["Cơm trắng"], Qty: 1},
					{DayIndex: 1, Meal: entity.MealLunch, FoodID: fid["Ức gà"], Qty: 1},
					{DayIndex: 1, Meal: entity.MealDinner, FoodID: fid["Khoai lang"], Qty: 2},
				}})
		}
	}
	for _, p := range plans {
		if err := g.tx.Create(&p.plan).Error; err != nil {
			return err
		}
		for i := range p.items {
			p.items[i].PlanID = p.plan.ID
		}
		if err := g.tx.Create(&p.items).Error; err != nil {
			return err
		}
	}

	logs := newBulk("food_logs", "member_id", "food_id", "qty", "meal", "eaten_at", "created_at")
	for mid, p := range logMembers {
		days := 7
		if g.showcase != nil && mid == g.showcase.ID {
			days = 21
		}
		for off := -days; off <= 0; off++ {
			if off < 0 && !g.r.chance(0.8) {
				continue
			}
			d := g.day(off)
			idx := isoWeekday(d)
			for _, it := range p.items {
				if it.DayIndex != idx {
					continue
				}
				hour := map[string]int{entity.MealBreakfast: 7, entity.MealLunch: 12, entity.MealDinner: 19, entity.MealSnack: 16}[it.Meal]
				at := g.at(d, hour, g.r.between(0, 40))
				if at.After(g.now) {
					continue
				}
				qty := math.Round(it.Qty*float64(g.r.between(85, 115))) / 100
				logs.add(mid, it.FoodID, qty, it.Meal, pgDate(d), at)
			}
		}
	}
	return logs.flush(g.ctx, g.tx)
}

// ---------------------------------------------------------------------
// Sprint 23 – Bảng tin & khảo sát NPS
// ---------------------------------------------------------------------

func (g *richGen) seedAnnouncements() error {
	type an struct {
		bi        int
		title     string
		body      string
		daysAgo   int
		pinned    bool
		expireIn  int
		scheduled bool
	}
	defs := []an{
		{-1, "Lịch hoạt động dịp Tết Dương lịch", "Ngày 01/01 toàn hệ thống mở cửa từ 07:00 đến 17:00. Các lớp nhóm buổi tối tạm nghỉ, lớp PT vui lòng xác nhận lại với HLV.", 2, true, 75, false},
		{-1, "Ra mắt chương trình Tích điểm – Đổi quà", "Mỗi buổi check-in được cộng 5 điểm, mỗi 10.000đ thanh toán gói tập được cộng 1 điểm. Đổi voucher giảm giá và quà tặng tại mục Điểm thưởng trên Cổng hội viên.", 18, true, 0, false},
		{0, "Khai trương khu Functional & máy đo InBody mới", "Hội viên Cầu Giấy được đo InBody miễn phí 2 tuần/lần và trải nghiệm dàn tạ Rogue mới tại tầng 3.", 6, true, 0, false},
		{0, "Tạm ngưng phòng xông hơi để bảo trì", "Phòng xông hơi nam tạm ngưng từ 20/10 đến 24/10 để thay hệ thống đá nóng. Mong hội viên thông cảm.", 4, false, 3, false},
		{1, "Lớp Yoga bay (Aerial Yoga) khai giảng", "Lớp Aerial Yoga dành cho người mới bắt đầu khai giảng vào tối thứ 3–5 hằng tuần, giới hạn 10 học viên/lớp. Đăng ký tại quầy hoặc trên Cổng hội viên.", 9, false, 0, false},
		{1, "Ưu đãi giới thiệu bạn bè tháng này", "Giới thiệu 1 người bạn đăng ký gói từ 3 tháng, bạn nhận 100 điểm thưởng và người được giới thiệu giảm 10% học phí.", 12, false, 20, false},
		{2, "Thông báo bảo trì hệ thống điện", "Chi nhánh Quận 1 tạm đóng cửa 1 ngày để bảo trì điện toàn tòa nhà. Gói tập của hội viên sẽ được cộng bù 1 ngày.", 1, true, 25, false},
		{2, "Giải chạy cộng đồng GymMaster Run 5K", "Đăng ký miễn phí cho hội viên, nhận áo và huy chương hoàn thành. Hạn đăng ký: hết tháng này.", 15, false, 0, false},
		{-1, "Cập nhật nội quy phòng tập", "Vui lòng mang khăn cá nhân, trả tạ về giá sau khi tập và không giữ máy quá 20 phút vào giờ cao điểm 17:00–20:00.", 40, false, 0, false},
		{-1, "Workshop dinh dưỡng: Ăn sạch kiểu Việt", "Chuyên gia dinh dưỡng chia sẻ cách xây dựng thực đơn giàu đạm từ món Việt. Miễn phí cho hội viên, đăng ký tại quầy.", -5, false, 0, true},
	}
	var list []entity.Announcement
	for _, d := range defs {
		a := entity.Announcement{Title: d.title, Body: d.body, Pinned: d.pinned, CreatedBy: &g.ownerID}
		pub := g.now.Add(-time.Duration(d.daysAgo*24+g.r.between(1, 8)) * time.Hour)
		a.PublishAt = pub
		a.CreatedAt = g.capNow(pub.Add(-3 * time.Hour))
		a.UpdatedAt = a.CreatedAt
		if d.bi >= 0 && d.bi < len(g.branches) {
			b := g.branches[d.bi]
			a.BranchID = &b.ID
			a.CreatedBy = &b.manager.ID
		}
		if d.expireIn > 0 {
			exp := g.now.AddDate(0, 0, d.expireIn)
			a.ExpireAt = &exp
		}
		list = append(list, a)
	}
	return g.tx.Create(&list).Error
}

func (g *richGen) seedSurveys() error {
	y := g.today.Year()
	q3s := time.Date(y, time.July, 1, 8, 0, 0, 0, g.loc)
	q3e := time.Date(y, time.September, 30, 23, 0, 0, 0, g.loc)
	q4s := time.Date(y, time.October, 1, 8, 0, 0, 0, g.loc)
	if q4s.After(g.now) {
		q4s = g.now.AddDate(0, 0, -20)
	}
	fac := g.now.AddDate(0, 0, -12)
	surveys := []struct {
		s     entity.Survey
		count int
	}{
		{entity.Survey{Title: "Khảo sát mức độ hài lòng hội viên Quý 3 (NPS)", Description: "Bạn sẵn sàng giới thiệu GymMaster cho bạn bè ở mức nào (0–10)?",
			Type: entity.SurveyTypeNPS, StartAt: q3s, EndAt: &q3e, Status: "CLOSED"}, 140},
		{entity.Survey{Title: "Khảo sát mức độ hài lòng hội viên Quý 4 (NPS)", Description: "Bạn sẵn sàng giới thiệu GymMaster cho bạn bè/đồng nghiệp ở mức nào (0–10)?",
			Type: entity.SurveyTypeNPS, StartAt: q4s, Status: "ACTIVE"}, 75},
		{entity.Survey{Title: "Đánh giá cơ sở vật chất & vệ sinh", Description: "Chấm điểm 0–10 cho chất lượng máy tập, vệ sinh phòng thay đồ và không gian tập.",
			Type: entity.SurveyTypeCustom, StartAt: fac, Status: "ACTIVE"}, 45},
	}
	comments := map[int][]string{
		2: {"Phòng tập sạch sẽ, HLV nhiệt tình, sẽ giới thiệu cho đồng nghiệp!", "Rất hài lòng, đặt lớp trên web tiện lợi.", "Máy móc mới, check-in QR nhanh.", "HLV PT theo sát, sau 2 tháng giảm 5kg."},
		1: {"Ổn nhưng giờ cao điểm hơi đông.", "Muốn có thêm lớp Yoga buổi trưa.", "Phòng thay đồ đôi khi thiếu tủ.", "Gửi xe hơi chật vào buổi tối."},
		0: {"Điều hòa khu cardio không mát.", "Một số máy chạy bộ hay báo lỗi.", "Lễ tân xử lý gia hạn hơi chậm."},
	}
	for _, sv := range surveys {
		s := sv.s
		s.CreatedAt = s.StartAt.Add(-time.Hour)
		s.UpdatedAt = s.CreatedAt
		if err := g.tx.Create(&s).Error; err != nil {
			return err
		}
		end := g.now
		if s.EndAt != nil && s.EndAt.Before(end) {
			end = *s.EndAt
		}
		bulk := newBulk("survey_responses", "survey_id", "member_id", "score", "comment", "answers", "created_at")
		pool := g.shuffled(g.activeMembers(0, ""))
		for i := 0; i < sv.count && i < len(pool); i++ {
			m := pool[i]
			var score, cat int
			switch p := g.r.Float64(); {
			case p < 0.58:
				score, cat = g.r.between(9, 10), 2
			case p < 0.86:
				score, cat = g.r.between(7, 8), 1
			default:
				score, cat = g.r.between(2, 6), 0
			}
			comment := ""
			if g.r.chance(0.55) {
				cs := comments[cat]
				comment = cs[g.r.Intn(len(cs))]
			}
			span := end.Sub(s.StartAt)
			at := s.StartAt.Add(time.Duration(g.r.Int63n(int64(span))))
			ans := fmt.Sprintf(`{"cleanliness":%d,"equipment":%d,"trainers":%d,"staff":%d}`,
				clampInt(score/2+g.r.between(-1, 1), 1, 5), clampInt(score/2+g.r.between(-1, 1), 1, 5),
				clampInt(score/2+g.r.between(0, 1), 1, 5), clampInt(score/2+g.r.between(-1, 1), 1, 5))
			bulk.add(s.ID, m.ID, score, comment, datatypes.JSON(ans), at)
		}
		if err := bulk.flush(g.ctx, g.tx); err != nil {
			return err
		}
	}
	return nil
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ---------------------------------------------------------------------
// Sprint 24 – Vé khách mời (theo quota gói tập) + lượt check-in tương ứng
// ---------------------------------------------------------------------

func (g *richGen) seedGuestPasses() error {
	guests := []string{"Nguyễn Văn Khánh", "Trần Thị Mỹ Linh", "Lê Quang Vinh", "Phạm Thu Trang", "Hoàng Minh Đức", "Vũ Ngọc Ánh",
		"Đặng Hải Yến", "Bùi Tuấn Kiệt", "Đỗ Phương Thảo", "Ngô Gia Bảo", "Dương Thanh Tâm", "Lý Hoài Nam"}
	quota := map[string]int{}
	seq := 0
	var passes []entity.DayPass
	type ck struct {
		pass   int
		branch uint
		at     time.Time
	}
	var cks []ck
	for _, b := range g.branches {
		var sponsors []*rsMember
		for _, m := range g.activeMembers(b.ID, "") {
			if m.GuestPasses != nil && *m.GuestPasses > 0 {
				sponsors = append(sponsors, m)
			}
		}
		if len(sponsors) == 0 {
			continue
		}
		for off := -60; off <= 2; off++ {
			n := 0
			if g.r.chance(0.55) {
				n = g.r.between(1, 2)
			}
			for k := 0; k < n; k++ {
				sp := g.pickMember(sponsors)
				d := g.day(off)
				key := fmt.Sprintf("%d|%s", sp.ID, d.Format("2006-01"))
				if quota[key] >= *sp.GuestPasses {
					continue
				}
				quota[key]++
				seq++
				spID := sp.ID
				recep := g.r.pickStaff(b.receps)
				created := g.capNow(g.at(addDays(d, -g.r.between(0, 2)), g.r.between(8, 20), g.r.between(0, 59)))
				p := entity.DayPass{BranchID: b.ID, BuyerName: guests[g.r.Intn(len(guests))], Phone: fmt.Sprintf("09%08d", g.r.Intn(100000000)),
					SponsorMemberID: &spID, Type: entity.DayPassTypeGuest, ValidDate: pgDate(d),
					QRToken: fmt.Sprintf("DP-%s-%05d", d.Format("060102"), seq), CreatedBy: &recep.ID, CreatedAt: created}
				if off < 0 && g.r.chance(0.82) {
					used := g.at(d, g.r.between(7, 20), g.r.between(0, 59))
					p.UsedAt = &used
					cks = append(cks, ck{len(passes), b.ID, used})
				}
				passes = append(passes, p)
			}
		}
	}
	if len(passes) == 0 {
		return nil
	}
	if err := g.tx.CreateInBatches(&passes, 200).Error; err != nil {
		return err
	}
	bulk := newBulk("checkins", "branch_id", "day_pass_id", "checked_in_at", "checked_out_at", "method", "result", "note", "created_at")
	for _, c := range cks {
		out := c.at.Add(time.Duration(g.r.between(60, 120)) * time.Minute)
		bulk.add(c.branch, passes[c.pass].ID, c.at, out, "QR", "ALLOWED", "Khách mời của hội viên", c.at)
	}
	return bulk.flush(g.ctx, g.tx)
}

// ---------------------------------------------------------------------
// Sprint 24 – Hợp đồng điện tử
// ---------------------------------------------------------------------

func (g *richGen) seedContracts() error {
	var tpl entity.ContractTemplate
	if err := g.tx.Where("status = ?", "ACTIVE").Order("id ASC").First(&tpl).Error; err != nil {
		return nil //nolint:nilerr // chưa có mẫu hợp đồng thì bỏ qua
	}
	sigs := make([]string, 6)
	for i := range sigs {
		s, err := demoSignaturePNG(int64(i + 1))
		if err != nil {
			return err
		}
		sigs[i] = s
	}
	ips := []string{"113.190.232.%d", "14.232.166.%d", "27.72.98.%d", "171.244.10.%d"}
	uas := []string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Linux; Android 15; SM-S928B) AppleWebKit/537.36 Chrome/128.0 Mobile Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6) AppleWebKit/537.36 Chrome/128.0 Safari/537.36",
		"Kiosk GymMaster (iPad; iPadOS 18)",
	}
	since := g.now.AddDate(0, 0, -120)
	var list []entity.MemberContract
	for _, m := range g.members {
		if !m.active() || m.SubCreated == nil || (m.SubCreated.Before(since) && m != g.showcase) {
			continue
		}
		plan := ""
		if m.PlanName != nil {
			plan = *m.PlanName
		}
		replacer := strings.NewReplacer(
			"{{member_name}}", html.EscapeString(m.FullName),
			"{{member_code}}", html.EscapeString(m.MemberCode),
			"{{member_phone}}", html.EscapeString(m.Phone),
			"{{plan_name}}", html.EscapeString(plan),
		)
		created := m.SubCreated.Add(2 * time.Minute)
		c := entity.MemberContract{MemberID: m.ID, TemplateID: tpl.ID, TemplateVersion: tpl.Version, SubscriptionID: m.SubID,
			RenderedHTML: replacer.Replace(htmlsafe.Sanitize(tpl.BodyHTML)), Status: entity.ContractStatusPending,
			CreatedAt: created, UpdatedAt: created}
		if m == g.showcase || g.r.chance(0.88) {
			signed := g.capNow(created.Add(time.Duration(g.r.between(3, 40)) * time.Minute))
			c.Status, c.SignedAt, c.SignatureURL = entity.ContractStatusSigned, &signed, sigs[g.r.Intn(len(sigs))]
			c.IP = fmt.Sprintf(ips[g.r.Intn(len(ips))], g.r.between(2, 250))
			c.UserAgent = uas[g.r.Intn(len(uas))]
			c.UpdatedAt = signed
		}
		list = append(list, c)
	}
	if len(list) == 0 {
		return nil
	}
	return g.tx.CreateInBatches(&list, 100).Error
}

// demoSignaturePNG vẽ một nét chữ ký giả lập (PNG nền trong suốt) và trả về data URL.
func demoSignaturePNG(seed int64) (string, error) {
	r := rand.New(rand.NewSource(seed)) // #nosec G404
	const w, h = 360, 120
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	ink := color.NRGBA{R: 20, G: 40, B: 120, A: 255}
	a1, a2 := 18+r.Float64()*14, 6+r.Float64()*8
	f1, f2 := 1.5+r.Float64()*2, 5+r.Float64()*4
	ph := r.Float64() * math.Pi
	dot := func(x, y float64) {
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				px, py := int(x)+dx, int(y)+dy
				if px >= 0 && px < w && py >= 0 && py < h {
					img.SetNRGBA(px, py, ink)
				}
			}
		}
	}
	for i := 0; i <= 2400; i++ {
		t := float64(i) / 2400
		x := 25 + t*300 + 10*math.Sin(t*f2*math.Pi)
		y := 62 + a1*math.Sin(t*f1*2*math.Pi+ph)*(1-t*0.5) + a2*math.Sin(t*f2*2*math.Pi)
		dot(x, y)
	}
	for i := 0; i <= 400; i++ { // gạch chân
		t := float64(i) / 400
		dot(60+t*240, 100+3*math.Sin(t*math.Pi))
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// ---------------------------------------------------------------------
// Sprint 27 – Đồng ý xử lý dữ liệu cá nhân (Nghị định 13/2023)
// ---------------------------------------------------------------------

func (g *richGen) seedConsents() error {
	bulk := newBulk("consents", "member_id", "type", "granted", "granted_at", "ip")
	for _, m := range g.members {
		ip := ""
		if m.UserID != nil {
			ip = fmt.Sprintf("113.190.%d.%d", g.r.between(1, 254), g.r.between(1, 254))
		}
		bulk.add(m.ID, entity.ConsentTypeTerms, true, m.CreatedAt, ip)
		if m == g.showcase || g.r.chance(0.65) {
			bulk.add(m.ID, entity.ConsentTypeMarketing, true, m.CreatedAt, ip)
		}
		if m == g.showcase || g.r.chance(0.3) {
			bulk.add(m.ID, "PHOTO", true, m.CreatedAt.Add(24*time.Hour), ip)
		}
	}
	return bulk.flush(g.ctx, g.tx)
}

// ---------------------------------------------------------------------
// Sprint 29 – Hồ sơ HLV công khai, cảm nhận hội viên, blog, liên hệ, tin nhắn PT
// ---------------------------------------------------------------------

func (g *richGen) seedTrainerProfiles() error {
	var rows []struct {
		UserID          uint
		FullName        string
		Level           string
		Specialties     string
		Bio             string
		Certifications  string
		YearsExperience int
		BranchID        uint
	}
	if err := g.tx.Raw(`SELECT tp.user_id, u.full_name, tp.level, array_to_string(tp.specialties, ', ') AS specialties,
			tp.bio, tp.certifications, tp.years_experience, ur.branch_id
		FROM trainer_profiles tp
		JOIN users u ON u.id = tp.user_id AND u.deleted_at IS NULL
		JOIN user_roles ur ON ur.user_id = u.id AND ur.branch_id IS NOT NULL
		ORDER BY tp.user_id`).Scan(&rows).Error; err != nil {
		return err
	}
	headline := map[string]string{
		"Gym": "Chuyên gia Tăng cơ – Giảm mỡ & Sức mạnh", "Yoga": "Huấn luyện viên Yoga & Phục hồi cột sống",
		"Zumba": "HLV Zumba & Dance Fitness năng động", "Boxing": "HLV Boxing Fitness & HIIT đốt mỡ",
		"Spinning": "HLV Spinning & Cardio sức bền", "Pilates": "Chuyên gia Pilates & Core",
	}
	var list []entity.TrainerPublicProfile
	seen := map[string]bool{}
	for _, r := range rows {
		first := strings.TrimSpace(strings.Split(r.Specialties, ",")[0])
		hl := headline[first]
		if hl == "" {
			hl = "Huấn luyện viên cá nhân " + first
		}
		slug := "hlv-" + slugVN(r.FullName)
		if seen[slug] {
			slug = fmt.Sprintf("%s-%d", slug, r.UserID)
		}
		seen[slug] = true
		bio := r.Bio
		if b := g.branchByID[r.BranchID]; b != nil {
			bio += " Hiện phụ trách lớp nhóm và huấn luyện cá nhân 1-1 tại GymMaster " + b.short + "."
		}
		years := r.YearsExperience
		if years <= 0 {
			years = 1
		}
		list = append(list, entity.TrainerPublicProfile{UserID: r.UserID, Slug: slug, Headline: hl, Bio: bio,
			Specialties: r.Specialties, Certifications: r.Certifications, ExperienceYears: years,
			Featured: r.Level == "MASTER" || (r.Level == "SENIOR" && years >= 6), IsPublished: true,
			CreatedAt: g.now.AddDate(0, -2, 0), UpdatedAt: g.now.AddDate(0, 0, -g.r.between(1, 30))})
	}
	if len(list) == 0 {
		return nil
	}
	return g.tx.Create(&list).Error
}

func (g *richGen) seedTestimonials() error {
	contents := []struct {
		text, note string
		rating     int
	}{
		{"Sau 4 tháng tập cùng HLV, mình giảm 7kg mỡ, hết đau mỏi vai gáy và duy trì được 4 buổi/tuần.", "Giảm 7kg mỡ · Tăng 2,5kg cơ", 5},
		{"Đặt lớp, check-in QR qua Kiosk và theo dõi giáo án ngay trên web cực kỳ tiện, không phải chờ ở quầy.", "Hoàn thành thử thách 12 buổi/tháng", 5},
		{"Lớp Yoga buổi sáng rất chất lượng, cô giáo sửa tư thế kỹ nên lưng mình đỡ đau hẳn.", "Hết đau lưng sau 8 tuần", 5},
		{"Phòng tập sạch, máy mới, giờ cao điểm hơi đông nhưng lễ tân luôn hỗ trợ nhiệt tình.", "Hội viên trung thành", 4},
		{"HLV lên thực đơn món Việt dễ nấu, không phải ăn ức gà luộc mãi. Vòng eo giảm 6cm.", "Giảm 6cm vòng eo", 5},
		{"Zumba vui lắm, tập xong như đi quẩy, tuần nào cũng mong đến tối thứ 3.", "Tham gia 40+ buổi Zumba", 5},
		{"Mình là dân văn phòng, nhờ lịch PT linh hoạt nên tập đều được, sức bền cải thiện rõ.", "Chạy 5K dưới 30 phút", 5},
		{"Bảo lưu gói khi đi công tác rất nhanh gọn, về là tập tiếp không mất ngày nào.", "Bảo lưu 2 lần không mất phí", 4},
		{"Boxing giúp mình xả stress sau giờ làm, HLV vui tính và chuyên nghiệp.", "Giảm 4kg sau 2 tháng", 5},
		{"Nhận được email nhắc gia hạn và ưu đãi đúng lúc, tích điểm đổi được voucher 10%.", "Đổi 3 voucher giảm giá", 4},
		{"Khu tạ tự do rộng, đủ dụng cụ, nhân viên dọn dẹp thường xuyên.", "Deadlift 120kg", 5},
		{"Spinning cường độ cao nhưng HLV hướng dẫn kỹ, người mới như mình vẫn theo kịp.", "Tăng sức bền tim mạch", 4},
		{"Máy chạy bộ hay bị lỗi, mong phòng tập bảo trì sớm hơn.", "", 3},
		{"Mình muốn có thêm lớp Pilates buổi trưa cho dân văn phòng.", "", 4},
	}
	pool := g.shuffled(g.activeMembers(0, ""))
	var list []entity.Testimonial
	for i, c := range contents {
		if i >= len(pool) {
			break
		}
		m := pool[i]
		if i == 0 && g.showcase != nil {
			m = g.showcase
		}
		months := int(g.now.Sub(m.CreatedAt).Hours()/24/30) + 1
		b := g.branchByID[m.HomeBranchID]
		role := fmt.Sprintf("Hội viên %d tháng", months)
		if b != nil {
			role += " · " + b.short
		}
		created := g.now.Add(-time.Duration(g.r.between(2, 80)*24) * time.Hour)
		mid := m.ID
		t := entity.Testimonial{MemberID: &mid, AuthorName: m.FullName, AuthorRole: role, Rating: c.rating, Content: c.text,
			BeforeAfterNote: c.note, Status: entity.TestimonialApproved, ApprovedBy: &g.ownerID, CreatedAt: created, UpdatedAt: created}
		switch i {
		case 12:
			t.Status, t.ApprovedBy = entity.TestimonialRejected, &g.ownerID
		case 13, 11:
			t.Status, t.ApprovedBy = entity.TestimonialPending, nil
		}
		if t.Status == entity.TestimonialApproved {
			at := created.Add(5 * time.Hour)
			t.ApprovedAt = &at
		}
		list = append(list, t)
	}
	return g.tx.Create(&list).Error
}

func (g *richGen) seedPosts() error {
	type pd struct {
		slug, title, excerpt, cat, status string
		daysAgo                           int
		body                              []string
	}
	defs := []pd{
		{"lo-trinh-tang-co-giam-mo-cho-nguoi-moi", "Lộ trình Tăng cơ Giảm mỡ (Body Recomposition) chuẩn khoa học cho người mới",
			"Kết hợp tập kháng lực tăng tiến và tính TDEE/Macro để vừa giảm mỡ vừa giữ cơ.", "TAP_LUYEN", "PUBLISHED", 3,
			[]string{"<p>Body Recomposition là quá trình giảm tỷ lệ mỡ đồng thời tăng khối lượng cơ nạc. Người mới tập có lợi thế lớn vì cơ thể phản ứng rất nhanh với bài tập kháng lực.</p>",
				"<h2>3 nguyên tắc cốt lõi</h2><ul><li>Thâm hụt calo nhẹ 200–300 kcal/ngày so với TDEE.</li><li>Nạp 1,8–2,2g protein/kg cân nặng.</li><li>Tập kháng lực 3–4 buổi/tuần, tăng dần mức tạ (Progressive Overload).</li></ul>",
				"<p>Hãy đo InBody mỗi 2–4 tuần để theo dõi tiến độ thay vì chỉ nhìn cân nặng.</p>"}},
		{"thuc-don-eat-clean-viet-nam-2000-kcal", "Thực đơn Eat Clean món Việt 2000 kcal giàu protein, dễ chuẩn bị",
			"Không cần ức gà luộc nhàm chán: phở bò, cơm gạo lứt, cá hồi và khoai lang cho người tập gym.", "DINH_DUONG", "PUBLISHED", 10,
			[]string{"<p>Ẩm thực Việt có rất nhiều món giàu đạm và cân bằng dinh dưỡng nếu biết kiểm soát khẩu phần.</p>",
				"<h2>Gợi ý một ngày</h2><ul><li>Sáng: phở bò tái (450 kcal).</li><li>Trưa: cơm + ức gà áp chảo + bông cải (600 kcal).</li><li>Chiều: sữa chua Hy Lạp + chuối (205 kcal).</li><li>Tối: cá hồi + khoai lang (580 kcal).</li></ul>",
				"<p>Uống đủ 2–2,5 lít nước mỗi ngày và hạn chế nước ngọt có đường.</p>"}},
		{"5-sai-lam-khi-tap-squat", "5 sai lầm khi tập Squat khiến bạn đau gối",
			"Squat là “vua” của các bài tập chân – nhưng sai kỹ thuật có thể gây chấn thương.", "TAP_LUYEN", "PUBLISHED", 17,
			[]string{"<p>Squat giúp phát triển đùi, mông và core. Tuy nhiên, rất nhiều người tập sai mà không biết.</p>",
				"<h2>Những lỗi thường gặp</h2><ul><li>Gối đổ vào trong khi đứng lên.</li><li>Nhón gót, dồn trọng tâm lên mũi chân.</li><li>Võng lưng dưới ở điểm thấp nhất.</li><li>Xuống quá nhanh, không kiểm soát nhịp âm.</li><li>Tăng tạ quá vội khi kỹ thuật chưa vững.</li></ul>",
				"<p>Hãy nhờ HLV quay video và chỉnh tư thế ở những buổi đầu.</p>"}},
		{"bao-nhieu-protein-la-du", "Một ngày cần bao nhiêu protein để tăng cơ?",
			"Công thức đơn giản theo cân nặng và mục tiêu tập luyện.", "DINH_DUONG", "PUBLISHED", 24,
			[]string{"<p>Nhu cầu protein phụ thuộc cân nặng, mức độ vận động và mục tiêu.</p>",
				"<ul><li>Người ít vận động: 0,8g/kg.</li><li>Người tập gym duy trì: 1,2–1,6g/kg.</li><li>Tăng cơ hoặc giảm mỡ giữ cơ: 1,8–2,2g/kg.</li></ul>",
				"<p>Chia đều protein vào 3–5 bữa để cơ thể hấp thu tốt hơn.</p>"}},
		{"gymmaster-run-5k", "GymMaster Run 5K – Giải chạy cộng đồng cho hội viên",
			"Đăng ký miễn phí, nhận áo, huy chương và quà từ nhà tài trợ.", "SU_KIEN", "PUBLISHED", 6,
			[]string{"<p>GymMaster Run 5K là sự kiện thường niên dành cho hội viên và người thân.</p>",
				"<h2>Thông tin chính</h2><ul><li>Cự ly: 5km quanh hồ.</li><li>Xuất phát: 6:00 sáng Chủ nhật.</li><li>Đăng ký: tại quầy lễ tân hoặc Cổng hội viên.</li></ul>"}},
		{"yoga-cho-dan-van-phong", "6 động tác Yoga giảm đau lưng cho dân văn phòng",
			"Chỉ 15 phút mỗi tối giúp giải tỏa căng cơ vùng cổ, vai và lưng dưới.", "TAP_LUYEN", "PUBLISHED", 31,
			[]string{"<p>Ngồi máy tính 8 tiếng/ngày khiến cơ lưng và hông bị co cứng.</p>",
				"<ul><li>Tư thế em bé (Child's Pose).</li><li>Mèo – Bò (Cat-Cow).</li><li>Chó úp mặt (Downward Dog).</li><li>Vặn mình nằm ngửa.</li><li>Rắn hổ mang (Cobra).</li><li>Gập người về trước.</li></ul>",
				"<p>Giữ mỗi tư thế 30–60 giây và thở đều bằng mũi.</p>"}},
		{"hiit-dot-mo-20-phut", "HIIT 20 phút: đốt mỡ hiệu quả cho người bận rộn",
			"Bài tập cường độ cao ngắt quãng giúp tăng tiêu hao năng lượng sau tập (EPOC).", "TAP_LUYEN", "PUBLISHED", 45,
			[]string{"<p>HIIT xen kẽ những pha vận động tối đa với nghỉ ngắn, phù hợp người ít thời gian.</p>",
				"<p>Gợi ý: 40 giây tập – 20 giây nghỉ, 5 động tác × 4 vòng: burpee, jumping squat, mountain climber, kettlebell swing, plank jack.</p>"}},
		{"uu-dai-hoi-vien-cuoi-nam", "Ưu đãi cuối năm: tặng 1 tháng khi gia hạn gói 12 tháng",
			"Chương trình áp dụng cho hội viên gia hạn trong tháng 12.", "SU_KIEN", "DRAFT", 0,
			[]string{"<p>Nội dung đang được bộ phận marketing hoàn thiện.</p>"}},
		{"khai-truong-chi-nhanh-hai-chau", "Khai trương chi nhánh Hải Châu (Đà Nẵng)",
			"Thông báo cũ về kế hoạch mở rộng chi nhánh.", "SU_KIEN", "ARCHIVED", 120,
			[]string{"<p>Kế hoạch mở chi nhánh Hải Châu đã tạm hoãn.</p>"}},
	}
	var list []entity.Post
	for _, d := range defs {
		created := g.now.Add(-time.Duration(d.daysAgo*24+g.r.between(2, 10)) * time.Hour)
		p := entity.Post{Slug: d.slug, Title: d.title, Excerpt: d.excerpt, Category: d.cat, Content: strings.Join(d.body, "\n"),
			Status: d.status, AuthorID: &g.ownerID, CreatedAt: created, UpdatedAt: created}
		if d.status != entity.PostStatusDraft {
			pub := created.Add(time.Hour)
			p.PublishedAt = &pub
		}
		list = append(list, p)
	}
	return g.tx.Create(&list).Error
}

func (g *richGen) seedContactMessages() error {
	defs := []struct {
		name, phone, email, subject, msg, status string
	}{
		{"Nguyễn Thị Thu Hà", "0912345601", "thuha.nguyen@gmail.com", "Hỏi giá gói PT", "Cho mình hỏi gói PT 12 buổi giá bao nhiêu và có HLV nữ không ạ?", "NEW"},
		{"Trần Văn Hùng", "0987123402", "", "Tập thử", "Mình muốn đăng ký tập thử 1 buổi vào cuối tuần.", "NEW"},
		{"Lê Minh Châu", "0903456703", "chau.le@outlook.com", "Gói doanh nghiệp", "Công ty mình có 25 nhân viên muốn đăng ký gói tập doanh nghiệp, nhờ tư vấn giúp.", "CONTACTED"},
		{"Phạm Quốc Toàn", "0978456704", "", "Lớp Boxing", "Lớp Boxing buổi tối còn chỗ không ạ?", "CLOSED"},
		{"Hoàng Mai Phương", "0934567805", "maiphuong.h@gmail.com", "Bảo lưu gói", "Mình sắp đi công tác 1 tháng, muốn bảo lưu gói thì cần làm gì?", "CONTACTED"},
		{"Vũ Đức Anh", "0961234506", "", "Giờ mở cửa", "Chủ nhật phòng tập mở cửa đến mấy giờ?", "CLOSED"},
		{"Đinh Thị Lan", "0945678907", "lan.dinh@yahoo.com", "Lớp Yoga cho mẹ bầu", "Phòng tập có lớp Yoga cho mẹ bầu không ạ?", "NEW"},
		{"Bùi Thanh Sơn", "0919876508", "", "Gửi xe", "Phòng tập có chỗ gửi ô tô không?", "CLOSED"},
		{"Ngô Thảo Vy", "0988765409", "thaovy.ngo@gmail.com", "Chuyển chi nhánh", "Mình chuyển nhà sang Quận 1, có thể chuyển gói sang chi nhánh Quận 1 không?", "CONTACTED"},
		{"Đỗ Hải Long", "0977654310", "", "Hợp tác", "Bên mình cung cấp thực phẩm bổ sung, muốn hợp tác đặt quầy tại phòng tập.", "NEW"},
		{"Lý Kim Ngân", "0932109811", "kimngan.ly@gmail.com", "Hoàn tiền", "Mình thanh toán VNPay 2 lần cho cùng 1 gói, nhờ kiểm tra hoàn tiền giúp.", "CONTACTED"},
		{"Trương Gia Huy", "0901928312", "", "Tư vấn giảm cân", "Mình nặng 92kg muốn giảm cân an toàn, nên chọn gói nào?", "NEW"},
	}
	var list []entity.ContactMessage
	for i, d := range defs {
		b := g.branches[i%len(g.branches)]
		created := g.now.Add(-time.Duration(i*31+g.r.between(1, 20)) * time.Hour)
		c := entity.ContactMessage{BranchID: &b.ID, FullName: d.name, Phone: d.phone, Email: d.email, Subject: d.subject, Message: d.msg,
			Status: d.status, CreatedAt: created, UpdatedAt: created}
		if d.status != entity.ContactStatusNew {
			c.HandledBy = &b.receps[0].ID
			c.HandledNote = "Đã gọi điện tư vấn, khách hẹn qua phòng tập"
			if d.status == entity.ContactStatusClosed {
				c.HandledNote = "Đã giải đáp qua điện thoại"
			}
			c.UpdatedAt = created.Add(3 * time.Hour)
		}
		list = append(list, c)
	}
	return g.tx.Create(&list).Error
}

func (g *richGen) seedConversations() error {
	var rows []struct {
		ID           uint
		MemberID     uint
		TrainerID    uint
		CreatedAt    time.Time
		MemberUserID uint
		MemberName   string
		TrainerName  string
	}
	if err := g.tx.Raw(`SELECT c.id, c.member_id, c.trainer_id, c.created_at, m.user_id AS member_user_id,
			m.full_name AS member_name, u.full_name AS trainer_name
		FROM pt_contracts c
		JOIN members m ON m.id = c.member_id AND m.user_id IS NOT NULL
		JOIN users u ON u.id = c.trainer_id
		WHERE c.status = 'ACTIVE' ORDER BY c.id`).Scan(&rows).Error; err != nil {
		return err
	}
	scripts := [][][2]string{
		{{"M", "Chào anh/chị, em muốn dời buổi tập thứ 5 sang thứ 6 được không ạ?"}, {"T", "Được em nhé, thứ 6 lúc 18:00 anh/chị còn trống."}, {"M", "Dạ vậy em đặt 18:00 thứ 6 ạ."}, {"T", "Ok em, nhớ ăn nhẹ trước buổi tập 1 tiếng nhé."}},
		{{"T", "Buổi hôm qua em làm tốt lắm, hôm nay nhớ giãn cơ đùi sau 10 phút nhé."}, {"M", "Dạ, chân em hơi mỏi nhưng ổn ạ."}, {"T", "Bình thường thôi, uống nhiều nước và ngủ đủ giấc nhé."}},
		{{"M", "Cho em hỏi tuần này nên ăn bao nhiêu protein ạ?"}, {"T", "Em nặng 62kg thì khoảng 110–120g protein/ngày. Anh/chị đã cập nhật thực đơn trên app."}, {"M", "Em cảm ơn, em xem rồi ạ."}, {"T", "Cuối tuần chụp lại bữa ăn gửi anh/chị xem nhé."}},
		{{"T", "Tuần sau mình bắt đầu tăng mức tạ Squat lên 50kg nhé."}, {"M", "Dạ em hơi lo, nhưng sẽ cố gắng ạ!"}, {"T", "Yên tâm, anh/chị sẽ đứng spot cho em."}, {"M", "Hôm nay em đo InBody giảm được 1,2kg mỡ rồi ạ 🎉"}, {"T", "Tuyệt vời! Giữ phong độ nhé."}},
		{{"M", "Em bị cảm, xin nghỉ buổi tối nay ạ."}, {"T", "Em nghỉ ngơi cho khỏe, anh/chị đã hủy buổi trên hệ thống, không trừ buổi nhé."}, {"M", "Dạ em cảm ơn ạ."}},
	}
	for i, r := range rows {
		if i >= 60 {
			break
		}
		script := scripts[i%len(scripts)]
		start := r.CreatedAt.In(g.loc).Add(24 * time.Hour)
		latest := g.now.Add(-time.Duration(g.r.between(1, 6*24)) * time.Hour)
		if start.After(latest) {
			start = latest.Add(-time.Duration(len(script)) * time.Hour)
		}
		step := latest.Sub(start) / time.Duration(len(script))
		conv := entity.Conversation{PTContractID: r.ID, MemberID: r.MemberID, TrainerID: r.TrainerID, CreatedAt: start, UpdatedAt: latest}
		if err := g.tx.Create(&conv).Error; err != nil {
			return err
		}
		unread := g.r.chance(0.4)
		msgs := newBulk("messages", "conversation_id", "sender_user_id", "sender_role", "content", "read_at", "created_at")
		var lastAt time.Time
		var lastText string
		for k, line := range script {
			at := start.Add(time.Duration(k)*step + time.Duration(g.r.between(0, 30))*time.Minute)
			sender, role := r.TrainerID, "TRAINER"
			if line[0] == "M" {
				sender, role = r.MemberUserID, "MEMBER"
			}
			var read any = at.Add(time.Duration(g.r.between(5, 120)) * time.Minute)
			if k == len(script)-1 && unread {
				read = nil
			}
			msgs.add(conv.ID, sender, role, line[1], read, at)
			lastAt, lastText = at, line[1]
		}
		if err := msgs.flush(g.ctx, g.tx); err != nil {
			return err
		}
		if err := g.tx.Model(&entity.Conversation{}).Where("id = ?", conv.ID).Updates(map[string]any{
			"last_message_at": lastAt, "last_message_preview": truncRunes(lastText, 120), "updated_at": lastAt,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}
