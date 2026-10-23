package seed

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"

	"gymmaster-api/internal/entity"
)

// Bổ sung dữ liệu cho các màn hình còn thưa sau seed lõi: khách tiềm năng CRM, đánh giá HLV/lớp,
// thiết bị & bảo trì, giáo án cá nhân + nhật ký tập theo buổi PT, thông báo và nhật ký thao tác.

// ---------------------------------------------------------------------
// Sprint 16 – Khách tiềm năng, lịch sử chăm sóc & vé tập thử
// ---------------------------------------------------------------------

func (g *richGen) seedLeads() error {
	var planIDs []uint
	if err := g.tx.Raw(`SELECT id FROM membership_plans WHERE status = 'ACTIVE' AND deleted_at IS NULL ORDER BY sort_order`).Scan(&planIDs).Error; err != nil {
		return err
	}
	names := &demoGen{r: g.r}
	sources := []string{entity.LeadSourceWebsite, entity.LeadSourceFacebook, entity.LeadSourceWalkIn, entity.LeadSourceReferral, entity.LeadSourceOther}
	sourceW := []float64{0.35, 0.3, 0.2, 0.12, 0.03}
	notes := map[string][]string{
		entity.LeadStatusNew:       {"Để lại số qua form website, chưa liên hệ", "Inbox fanpage hỏi giá gói 3 tháng", "Khách đi ngang qua lấy tờ rơi"},
		entity.LeadStatusContacted: {"Đã gọi tư vấn, khách đang cân nhắc giữa gói 3 và 6 tháng", "Khách hỏi lớp Yoga buổi tối", "Hẹn gọi lại sau khi nhận lương"},
		entity.LeadStatusTrial:     {"Đã đặt lịch tập thử, ưu tiên khung 18:00", "Khách muốn thử lớp Zumba trước khi đăng ký"},
		entity.LeadStatusConverted: {"Đã đăng ký gói sau buổi tập thử", "Chốt gói kèm 4 buổi PT"},
		entity.LeadStatusLost:      {"Khách chọn phòng tập gần nhà hơn", "Ngân sách chưa phù hợp", "Không liên lạc được sau 3 lần gọi"},
	}
	usedMember := map[uint]bool{}
	seq := 0
	for _, b := range g.branches {
		var branchMembers []*rsMember
		for _, m := range g.members {
			if m.HomeBranchID == b.ID {
				branchMembers = append(branchMembers, m)
			}
		}
		for i := 0; i < 42; i++ {
			gender := "FEMALE"
			if g.r.chance(0.45) {
				gender = "MALE"
			}
			full, _, _, given := names.personName(gender)
			created := g.now.Add(-time.Duration(g.r.between(2, 100*24)) * time.Hour)
			age := g.now.Sub(created).Hours() / 24
			p := g.r.Float64()
			var status string
			switch {
			case age < 7:
				status = map[bool]string{true: entity.LeadStatusNew, false: entity.LeadStatusContacted}[p < 0.6]
			case age < 21:
				status = []string{entity.LeadStatusContacted, entity.LeadStatusTrial, entity.LeadStatusLost}[g.r.weighted([]float64{0.45, 0.4, 0.15})]
			case age < 45:
				status = []string{entity.LeadStatusTrial, entity.LeadStatusConverted, entity.LeadStatusLost}[g.r.weighted([]float64{0.15, 0.6, 0.25})]
			default:
				status = map[bool]string{true: entity.LeadStatusConverted, false: entity.LeadStatusLost}[p < 0.68]
			}
			recep := g.r.pickStaff(b.receps)
			planID := planIDs[g.r.Intn(len(planIDs))]
			l := entity.Lead{BranchID: b.ID, FullName: full, Phone: fmt.Sprintf("08%08d", g.r.Intn(100000000)),
				Source: sources[g.r.weighted(sourceW)], InterestPlanID: &planID, Status: status, AssignedTo: &recep.ID}
			if g.r.chance(0.5) {
				l.Email = fmt.Sprintf("%s.%d@gmail.com", asciiFold(given), g.r.between(80, 9999))
			}
			if status == entity.LeadStatusConverted {
				var cands []*rsMember
				for _, m := range branchMembers {
					if !usedMember[m.ID] && m.CreatedAt.After(created) && m.CreatedAt.Before(created.AddDate(0, 0, 30)) {
						cands = append(cands, m)
					}
				}
				var cand *rsMember
				if len(cands) > 0 {
					cand = cands[g.r.Intn(len(cands))]
				}
				if cand == nil {
					status = entity.LeadStatusLost
					l.Status = status
				} else {
					usedMember[cand.ID] = true
					l.MemberID = &cand.ID
					l.FullName, l.Phone = cand.FullName, cand.Phone
				}
			}
			l.Note = notes[status][g.r.Intn(len(notes[status]))]
			var trial time.Time
			if status == entity.LeadStatusTrial || status == entity.LeadStatusConverted {
				trial = dayOf(created.AddDate(0, 0, g.r.between(1, 6)), g.loc)
				l.TrialDate = ptrTime(pgDate(trial))
			}
			l.CreatedAt, l.UpdatedAt, l.CreatedBy = created, g.capNow(created.Add(time.Duration(g.r.between(1, 72))*time.Hour)), &recep.ID
			if err := g.tx.Create(&l).Error; err != nil {
				return err
			}

			acts := newBulk("lead_activities", "lead_id", "type", "content", "created_by", "created_at")
			steps := map[string]int{entity.LeadStatusNew: 0, entity.LeadStatusContacted: 1, entity.LeadStatusTrial: 2, entity.LeadStatusConverted: 3, entity.LeadStatusLost: 2}[status]
			at := created
			contents := []struct{ typ, text string }{
				{entity.LeadActivityCall, "Gọi tư vấn bảng giá, giới thiệu ưu đãi tháng"},
				{entity.LeadActivitySMS, "Gửi SMS lịch tập thử và địa chỉ chi nhánh"},
				{entity.LeadActivityVisit, "Khách đến tập thử, đã hướng dẫn máy và đo InBody"},
				{entity.LeadActivityNote, "Khách đồng ý đăng ký gói tập"},
			}
			for s := 0; s < steps; s++ {
				at = g.capNow(at.Add(time.Duration(g.r.between(3, 60)) * time.Hour))
				c := contents[s]
				if status == entity.LeadStatusLost && s == steps-1 {
					c = struct{ typ, text string }{entity.LeadActivityNote, "Khách từ chối: " + strings.ToLower(l.Note)}
				}
				acts.add(l.ID, c.typ, c.text, recep.ID, at)
			}
			if err := acts.flush(g.ctx, g.tx); err != nil {
				return err
			}
			if !trial.IsZero() {
				seq++
				tp := entity.TrialPass{LeadID: l.ID, BranchID: b.ID, ValidDate: pgDate(trial), QRToken: fmt.Sprintf("TP-%s-%04d", trial.Format("060102"), seq),
					CreatedBy: &recep.ID, CreatedAt: g.capNow(created.Add(2 * time.Hour))}
				if trial.Before(g.today) && (status == entity.LeadStatusConverted || g.r.chance(0.6)) {
					used := g.at(trial, g.r.between(7, 20), g.r.between(0, 59))
					tp.UsedAt = &used
				}
				if err := g.tx.Create(&tp).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------
// Sprint 15 – Đánh giá buổi lớp & buổi PT (suy ra từ lượt tham gia thật)
// ---------------------------------------------------------------------

func (g *richGen) seedReviews() error {
	var rows []struct {
		MemberID   uint
		TargetType string
		TargetID   uint
		TrainerID  uint
		BranchID   uint
		At         time.Time
	}
	since := g.now.AddDate(0, 0, -90)
	if err := g.tx.Raw(`
		SELECT cb.member_id, 'CLASS_SCHEDULE' AS target_type, cs.id AS target_id, COALESCE(cs.substitute_trainer_id, cs.trainer_id) AS trainer_id,
			cs.branch_id, cb.attended_at AS at
		FROM class_bookings cb JOIN class_schedules cs ON cs.id = cb.schedule_id
		WHERE cb.status = 'ATTENDED' AND cb.attended_at >= ?
		UNION ALL
		SELECT ps.member_id, 'PT_SESSION', ps.id, ps.trainer_id, ps.branch_id, ps.completed_at
		FROM pt_sessions ps WHERE ps.status = 'COMPLETED' AND ps.completed_at >= ?`, since, since).Scan(&rows).Error; err != nil {
		return err
	}
	var existing []struct {
		MemberID   uint
		TargetType string
		TargetID   uint
	}
	if err := g.tx.Raw(`SELECT member_id, target_type, target_id FROM reviews`).Scan(&existing).Error; err != nil {
		return err
	}
	done := map[string]bool{}
	for _, e := range existing {
		done[fmt.Sprintf("%d|%s|%d", e.MemberID, e.TargetType, e.TargetID)] = true
	}
	comments := map[int][]string{
		5: {"HLV nhiệt tình, chỉnh tư thế rất kỹ!", "Buổi tập năng lượng, nhạc hay.", "Rất hài lòng, sẽ tiếp tục đăng ký.", "Giải thích dễ hiểu, theo sát từng động tác.", ""},
		4: {"Buổi tập tốt, phòng hơi đông.", "Ổn, mong có thêm bài giãn cơ cuối buổi.", "HLV vui vẻ, bài hơi nặng với người mới.", ""},
		3: {"Bình thường, điều hòa chưa đủ mát.", "Lớp đông quá, HLV khó bao quát."},
		2: {"Bắt đầu muộn 10 phút.", "Âm thanh quá to."},
		1: {"Không hài lòng, HLV đến muộn."},
	}
	bulk := newBulk("reviews", "member_id", "target_type", "target_id", "trainer_id", "branch_id", "rating", "comment", "created_at")
	for _, r := range rows {
		p := 0.1
		if r.TargetType == entity.ReviewTargetPTSession {
			p = 0.3
		}
		key := fmt.Sprintf("%d|%s|%d", r.MemberID, r.TargetType, r.TargetID)
		if done[key] || !g.r.chance(p) {
			continue
		}
		done[key] = true
		rating := []int{5, 4, 3, 2, 1}[g.r.weighted([]float64{0.56, 0.3, 0.1, 0.03, 0.01})]
		cs := comments[rating]
		at := g.capNow(r.At.Add(time.Duration(g.r.between(1, 20)) * time.Hour))
		bulk.add(r.MemberID, r.TargetType, r.TargetID, r.TrainerID, r.BranchID, rating, cs[g.r.Intn(len(cs))], at)
	}
	if err := bulk.flush(g.ctx, g.tx); err != nil {
		return err
	}
	return g.tx.Exec(`UPDATE trainer_profiles tp SET rating_avg = s.avg, rating_count = s.cnt
		FROM (SELECT trainer_id, ROUND(AVG(rating)::numeric, 2) AS avg, COUNT(*) AS cnt FROM reviews GROUP BY trainer_id) s
		WHERE s.trainer_id = tp.user_id`).Error
}

// ---------------------------------------------------------------------
// Sprint 15 – Thiết bị phòng tập & nhật ký bảo trì
// ---------------------------------------------------------------------

func (g *richGen) seedEquipment() error {
	var codes []string
	if err := g.tx.Raw(`SELECT code FROM equipments`).Scan(&codes).Error; err != nil {
		return err
	}
	exists := map[string]bool{}
	for _, c := range codes {
		exists[c] = true
	}
	catalog := []struct {
		key, name, cat string
		price          float64
		count          int
		interval       int
	}{
		{"RUN", "Máy chạy bộ thương mại Life Fitness Integrity", entity.EquipmentCategoryCardio, 145000000, 6, 60},
		{"ELP", "Máy tập Elliptical Precor EFX 835", entity.EquipmentCategoryCardio, 120000000, 3, 60},
		{"BIKE", "Xe đạp Spinning Keiser M3i", entity.EquipmentCategoryCardio, 42000000, 8, 90},
		{"ROW", "Máy chèo thuyền Concept2 RowErg", entity.EquipmentCategoryCardio, 32000000, 2, 90},
		{"LAT", "Máy kéo xô Hammer Strength Iso-Lateral", entity.EquipmentCategoryStrength, 98000000, 2, 120},
		{"PRESS", "Máy đạp đùi 45 độ Leg Press", entity.EquipmentCategoryStrength, 115000000, 2, 120},
		{"CABLE", "Khung cáp đa năng Cable Crossover", entity.EquipmentCategoryStrength, 135000000, 1, 90},
		{"SMITH", "Máy Smith Machine", entity.EquipmentCategoryStrength, 76000000, 1, 120},
		{"BENCH", "Ghế đẩy ngực điều chỉnh độ dốc", entity.EquipmentCategoryFreeWeight, 9500000, 6, 180},
		{"DB", "Bộ tạ đơn cao su 2,5–40kg kèm giá", entity.EquipmentCategoryFreeWeight, 68000000, 1, 180},
		{"KB", "Bộ tạ ấm Kettlebell 4–32kg", entity.EquipmentCategoryFunctional, 18000000, 1, 180},
		{"TRX", "Dây treo TRX Pro4", entity.EquipmentCategoryFunctional, 6500000, 4, 180},
		{"MAT", "Thảm Yoga TPU 8mm (bộ 20 tấm)", entity.EquipmentCategoryAccessory, 7000000, 1, 365},
	}
	repairs := []string{"Băng chạy bị lệch, phát tiếng kêu", "Màn hình không hiển thị nhịp tim", "Dây cáp sờn, cần thay mới", "Đệm ghế rách", "Bàn đạp lỏng", "Mất kết nối Bluetooth"}
	logs := newBulk("maintenance_logs", "equipment_id", "type", "reported_by", "description", "cost", "status", "started_at", "finished_at", "created_at", "updated_at")
	for _, b := range g.branches {
		recep := b.receps[0]
		var specials []string // một thiết bị đang bảo trì, một hỏng, một thanh lý mỗi chi nhánh
		for _, it := range catalog {
			for n := 1; n <= it.count; n++ {
				code := fmt.Sprintf("%s-TB-%s%02d", b.Code, it.key, n)
				if exists[code] {
					continue
				}
				purchase := dayOf(g.now.AddDate(0, -g.r.between(4, 30), -g.r.between(0, 27)), g.loc)
				warranty := purchase.AddDate(2, 0, 0)
				e := entity.Equipment{BranchID: b.ID, Code: code, Name: it.name, Category: it.cat, PurchaseDate: ptrTime(pgDate(purchase)),
					Price: it.price, WarrantyUntil: ptrTime(pgDate(warranty)), Status: entity.EquipmentStatusActive, MaintenanceIntervalDays: it.interval}
				switch {
				case len(specials) == 0 && it.cat == entity.EquipmentCategoryCardio && n == 2:
					e.Status, e.Note = entity.EquipmentStatusMaintenance, "Đang thay băng chạy"
					specials = append(specials, code)
				case len(specials) == 1 && it.cat == entity.EquipmentCategoryStrength:
					e.Status, e.Note = entity.EquipmentStatusBroken, "Đứt cáp, đã treo biển tạm ngưng"
					specials = append(specials, code)
				case len(specials) == 2 && it.key == "BENCH" && n == 6:
					e.Status, e.Note = entity.EquipmentStatusRetired, "Thanh lý do khung gỉ sét"
					specials = append(specials, code)
				}
				e.CreatedAt, e.UpdatedAt, e.CreatedBy = g.at(purchase, 10, 0), g.now.Add(-time.Duration(g.r.between(1, 200))*time.Hour), &b.manager.ID

				// Lịch sử bảo trì định kỳ + sửa chữa.
				last := purchase
				type lg struct {
					typ, desc, status string
					cost              float64
					start, finish     *time.Time
				}
				var hist []lg
				for d := addDays(purchase, it.interval); d.Before(g.today); d = addDays(d, it.interval) {
					st := g.at(d, 9, 0)
					fin := st.Add(time.Duration(g.r.between(1, 4)) * time.Hour)
					hist = append(hist, lg{entity.MaintenanceTypePeriodic, "Bảo dưỡng định kỳ: vệ sinh, tra dầu, kiểm tra an toàn", entity.MaintenanceStatusDone,
						float64(g.r.between(3, 15)) * 100000, &st, &fin})
					last = d
					if g.r.chance(0.15) {
						rs := g.at(addDays(d, g.r.between(5, 30)), 14, 0)
						if rs.Before(g.now) {
							rf := rs.Add(time.Duration(g.r.between(20, 72)) * time.Hour)
							if rf.After(g.now) {
								rf = g.now.Add(-time.Hour)
							}
							hist = append(hist, lg{entity.MaintenanceTypeRepair, repairs[g.r.Intn(len(repairs))], entity.MaintenanceStatusDone,
								float64(g.r.between(5, 60)) * 100000, &rs, &rf})
						}
					}
				}
				if len(hist) > 6 {
					hist = hist[len(hist)-6:]
				}
				switch e.Status {
				case entity.EquipmentStatusMaintenance:
					st := g.now.Add(-26 * time.Hour)
					hist = append(hist, lg{entity.MaintenanceTypeRepair, "Băng chạy mòn, đang chờ linh kiện thay thế", entity.MaintenanceStatusInProgress, 3500000, &st, nil})
				case entity.EquipmentStatusBroken:
					hist = append(hist, lg{entity.MaintenanceTypeRepair, "Đứt cáp kéo, cần thay toàn bộ dây cáp", entity.MaintenanceStatusOpen, 0, nil, nil})
				}
				next := addDays(last, it.interval)
				e.NextMaintenanceDate = ptrTime(pgDate(next))
				if err := g.tx.Create(&e).Error; err != nil {
					return err
				}
				for _, h := range hist {
					created := g.now.Add(-2 * time.Hour)
					if h.start != nil {
						created = h.start.Add(-time.Duration(g.r.between(1, 48)) * time.Hour)
					}
					updated := created
					if h.finish != nil {
						updated = *h.finish
					}
					logs.add(e.ID, h.typ, recep.ID, h.desc, h.cost, h.status, nTime(h.start), nTime(h.finish), created, updated)
				}
			}
		}
	}
	return logs.flush(g.ctx, g.tx)
}

// ---------------------------------------------------------------------
// Sprint 14 – Giáo án cá nhân theo hợp đồng PT + nhật ký tập theo buổi PT đã hoàn thành
// ---------------------------------------------------------------------

func (g *richGen) seedWorkoutPlans() error {
	var exercises []entity.Exercise
	if err := g.tx.Order("id ASC").Find(&exercises).Error; err != nil {
		return err
	}
	byGroup := map[string][]entity.Exercise{}
	for _, ex := range exercises {
		byGroup[ex.MuscleGroup] = append(byGroup[ex.MuscleGroup], ex)
	}
	pick := func(group string, i int) entity.Exercise {
		list := byGroup[group]
		if len(list) == 0 {
			return exercises[i%len(exercises)]
		}
		return list[i%len(list)]
	}
	if len(exercises) == 0 {
		return nil
	}
	type itemDef struct {
		group  string
		sets   int
		reps   int
		weight float64
		rest   int
	}
	days := []struct {
		idx   int
		title string
		items []itemDef
	}{
		{1, "Buổi A – Ngực, Vai, Tay sau", []itemDef{{entity.MuscleChest, 4, 10, 40, 90}, {entity.MuscleChest, 3, 12, 14, 75}, {entity.MuscleShoulders, 3, 12, 10, 60}, {entity.MuscleArms, 3, 12, 15, 60}}},
		{3, "Buổi B – Lưng, Xô, Tay trước", []itemDef{{entity.MuscleBack, 4, 10, 40, 90}, {entity.MuscleBack, 3, 12, 30, 75}, {entity.MuscleArms, 3, 12, 10, 60}, {entity.MuscleCore, 3, 15, 0, 45}}},
		{5, "Buổi C – Chân, Mông & Core", []itemDef{{entity.MuscleLegs, 4, 8, 50, 120}, {entity.MuscleLegs, 3, 10, 80, 90}, {entity.MuscleLegs, 3, 12, 20, 75}, {entity.MuscleCore, 3, 20, 0, 45}}},
	}
	var contracts []struct {
		ID        uint
		MemberID  uint
		TrainerID uint
		StartDate time.Time
		EndDate   time.Time
		FullName  string
		Gender    string
	}
	if err := g.tx.Raw(`SELECT c.id, c.member_id, c.trainer_id, c.start_date, c.end_date, m.full_name, m.gender
		FROM pt_contracts c JOIN members m ON m.id = c.member_id
		WHERE c.status IN ('ACTIVE', 'COMPLETED')
			AND NOT EXISTS (SELECT 1 FROM workout_plans wp WHERE wp.pt_contract_id = c.id OR (wp.member_id = c.member_id AND wp.status = 'ACTIVE'))
		ORDER BY c.id`).Scan(&contracts).Error; err != nil {
		return err
	}
	goals := []string{"Tăng cơ, tăng sức mạnh nền tảng", "Giảm mỡ, săn chắc toàn thân", "Cải thiện tư thế & sức bền"}
	logs := newBulk("workout_logs", "member_id", "plan_item_id", "exercise_id", "performed_at", "sets", "reps", "weight", "note", "created_at")
	for ci, c := range contracts {
		scale := 1.0
		if c.Gender == "FEMALE" {
			scale = 0.6
		}
		status := entity.WorkoutPlanActive
		if c.EndDate.Before(g.today) {
			status = "COMPLETED"
		}
		mid, cid := c.MemberID, c.ID
		start, end := c.StartDate, c.EndDate
		plan := entity.WorkoutPlan{MemberID: &mid, TrainerID: c.TrainerID, PTContractID: &cid, Name: "Giáo án cá nhân – " + c.FullName,
			Goal: goals[ci%len(goals)], StartDate: &start, EndDate: &end, Status: status}
		plan.CreatedAt = g.capNow(g.at(c.StartDate.In(g.loc), 9, 0))
		plan.UpdatedAt = plan.CreatedAt
		plan.CreatedBy = &c.TrainerID
		if err := g.tx.Create(&plan).Error; err != nil {
			return err
		}
		dayItems := make([][]entity.WorkoutPlanItem, len(days))
		for di, d := range days {
			day := entity.WorkoutPlanDay{PlanID: plan.ID, DayIndex: d.idx, Title: d.title}
			if err := g.tx.Create(&day).Error; err != nil {
				return err
			}
			for k, it := range d.items {
				ex := pick(it.group, k+ci)
				w := float64(int(it.weight*scale/2.5+0.5)) * 2.5
				dayItems[di] = append(dayItems[di], entity.WorkoutPlanItem{DayID: day.ID, ExerciseID: ex.ID, Sets: it.sets, Reps: it.reps,
					Weight: w, RestSec: it.rest, Sort: k + 1})
			}
			if err := g.tx.Create(&dayItems[di]).Error; err != nil {
				return err
			}
		}

		var sessions []time.Time
		if err := g.tx.Raw(`SELECT start_at FROM pt_sessions WHERE contract_id = ? AND status = 'COMPLETED' ORDER BY start_at`, c.ID).
			Scan(&sessions).Error; err != nil {
			return err
		}
		for si, at := range sessions {
			items := dayItems[si%len(days)]
			progress := 1 + 0.025*float64(si/len(days))
			for _, it := range items {
				w := it.Weight
				if w > 0 {
					w = float64(int(w*progress/2.5+0.5)) * 2.5
				}
				itemID := it.ID
				note := ""
				if si == len(sessions)-1 && w > 0 {
					note = "Kỷ lục cá nhân mới 💪"
				}
				perf := at.Add(time.Duration(10+it.Sort*12) * time.Minute)
				if perf.After(g.now) {
					continue
				}
				logs.add(c.MemberID, itemID, it.ExerciseID, perf, it.Sets, it.Reps, w, note, perf)
			}
		}
	}
	return logs.flush(g.ctx, g.tx)
}

// ---------------------------------------------------------------------
// Sprint 10/17 – Thông báo cho nhân viên & hội viên
// ---------------------------------------------------------------------

func (g *richGen) seedNotifications() error {
	type tpl struct{ typ, title, body, link string }
	staffTpl := map[string][]tpl{
		"MANAGER": {
			{"LEAD_NEW", "Có khách tiềm năng mới từ website", "Khách vừa để lại thông tin đăng ký tập thử, vui lòng phân công lễ tân liên hệ.", "/app/leads"},
			{"FREEZE_REQUEST", "Yêu cầu bảo lưu gói tập chờ duyệt", "Một hội viên vừa gửi yêu cầu bảo lưu gói tập.", "/app/subscription-freezes"},
			{"LOW_STOCK", "Sản phẩm sắp hết hàng", "Có mặt hàng dưới định mức tồn kho tại chi nhánh.", "/app/inventory"},
			{"MAINTENANCE_DUE", "Thiết bị đến hạn bảo trì", "Một số thiết bị đến lịch bảo dưỡng định kỳ trong tuần này.", "/app/equipments"},
			{"SYSTEM", "Báo cáo tổng hợp ngày đã sẵn sàng", "Hệ thống đã tổng hợp doanh thu và lượt check-in hôm qua.", "/app/reports"},
		},
		"RECEPTIONIST": {
			{"LEAD_NEW", "Bạn được giao một khách tiềm năng mới", "Vui lòng gọi tư vấn trong vòng 24 giờ.", "/app/leads"},
			{"SUBSCRIPTION_EXPIRING", "Hội viên sắp hết hạn gói tập", "Có hội viên hết hạn gói trong 7 ngày tới, hãy liên hệ gia hạn.", "/app/subscriptions"},
			{"BIRTHDAY", "Hôm nay có hội viên sinh nhật", "Gửi lời chúc và voucher sinh nhật cho hội viên khi check-in.", "/app/members"},
			{"FREEZE_REQUEST", "Yêu cầu bảo lưu mới", "Một hội viên vừa gửi yêu cầu bảo lưu.", "/app/subscription-freezes"},
		},
		"TRAINER": {
			{"PT_SCHEDULED", "Hội viên vừa đặt buổi PT", "Bạn có buổi PT mới trong lịch dạy tuần này.", "/app/pt"},
			{"PT_CANCELLED", "Hội viên hủy buổi PT", "Một buổi PT đã được hủy trước hạn, buổi tập không bị trừ.", "/app/pt"},
			{"CLASS_CANCELLED", "Lớp nhóm bị hủy do không đủ học viên", "Lớp đã được hủy, hội viên đã được thông báo.", "/app/classes"},
			{"COMMISSION_PAID", "Hoa hồng PT đã được chi trả", "Hoa hồng kỳ trước đã được cộng vào bảng lương.", "/app/payrolls"},
			{"SYSTEM", "Bạn nhận được đánh giá 5★", "Hội viên vừa đánh giá rất cao buổi tập của bạn.", "/app/reviews"},
		},
	}
	memberTpl := []tpl{
		{"SUBSCRIPTION_EXPIRING", "Gói tập sắp hết hạn", "Gói tập của bạn sẽ hết hạn trong 7 ngày. Gia hạn ngay để nhận ưu đãi.", "/me/subscriptions"},
		{"PAYMENT_SUCCESS", "Thanh toán thành công", "Cảm ơn bạn! Hóa đơn đã được thanh toán đầy đủ.", "/me/invoices"},
		{"BOOKING_PROMOTED", "Bạn đã có chỗ trong lớp", "Một học viên vừa hủy, bạn được chuyển từ danh sách chờ vào lớp.", "/me/bookings"},
		{"PT_SCHEDULED", "Lịch tập PT đã được xác nhận", "Buổi PT của bạn đã được HLV xác nhận.", "/me/pt"},
		{"REVIEW_PROMPT", "Đánh giá buổi tập hôm nay", "Hãy cho chúng tôi biết cảm nhận của bạn về buổi tập.", "/me/history"},
		{"CHECKIN", "Check-in thành công", "Chúc bạn có buổi tập hiệu quả!", "/me/history"},
		{"SYSTEM", "Ưu đãi tích điểm tháng này", "Đổi 200 điểm lấy voucher giảm 10% khi gia hạn gói tập.", "/me/rewards"},
		{"CLASS_CANCELLED", "Lớp nhóm bị hủy", "Lớp bạn đặt đã bị hủy do HLV có việc đột xuất, lượt đặt đã được hoàn.", "/me/bookings"},
	}
	bulk := newBulk("notifications", "user_id", "type", "title", "body", "link", "read_at", "created_at")
	emit := func(userID uint, list []tpl, n int) {
		for k := 0; k < n; k++ {
			t := list[g.r.Intn(len(list))]
			at := g.now.Add(-time.Duration(g.r.between(10, 21*24*60)) * time.Minute)
			var read any
			if k >= 3 || g.r.chance(0.3) {
				read = at.Add(time.Duration(g.r.between(5, 600)) * time.Minute)
				if read.(time.Time).After(g.now) {
					read = g.now
				}
			}
			bulk.add(userID, t.typ, t.title, t.body, t.link, read, at)
		}
	}
	emit(g.ownerID, append(staffTpl["MANAGER"], tpl{"DEVICE_OFFLINE", "Thiết bị mất kết nối", "Khóa từ cửa phòng Yoga Quận 1 mất kết nối hơn 24 giờ.", "/app/access-devices"}), 14)
	for _, s := range g.staff {
		if list := staffTpl[s.Position]; len(list) > 0 {
			emit(s.ID, list, g.r.between(8, 14))
		}
	}
	for _, m := range g.members {
		if m.UserID != nil {
			emit(*m.UserID, memberTpl, g.r.between(4, 10))
		}
	}
	return bulk.flush(g.ctx, g.tx)
}

// ---------------------------------------------------------------------
// Sprint 10 – Nhật ký thao tác suy ra từ nghiệp vụ đã seed (90 ngày gần nhất)
// ---------------------------------------------------------------------

func (g *richGen) seedAuditLogs() error {
	since := g.now.AddDate(0, 0, -90)
	bulk := newBulk("audit_logs", "actor_id", "action", "entity", "entity_id", "before", "after", "ip", "created_at")
	ipOf := func(branchID uint) string {
		return fmt.Sprintf("192.168.%d.%d", 10+branchID%200, g.r.between(20, 80))
	}
	js := func(format string, args ...any) datatypes.JSON { return datatypes.JSON(fmt.Sprintf(format, args...)) }

	var invs []struct {
		ID        uint
		InvoiceNo string
		Total     float64
		BranchID  uint
		IssuedBy  *uint
		IssuedAt  time.Time
	}
	if err := g.tx.Raw(`SELECT id, invoice_no, total, branch_id, issued_by, issued_at FROM invoices WHERE issued_at >= ?`, since).Scan(&invs).Error; err != nil {
		return err
	}
	for _, iv := range invs {
		bulk.add(nUint(iv.IssuedBy), "CREATE", "invoice", iv.ID, nil, js(`{"invoice_no":%q,"total":%.0f}`, iv.InvoiceNo, iv.Total), ipOf(iv.BranchID), iv.IssuedAt)
	}

	var pays []struct {
		InvoiceID  uint
		Amount     float64
		Method     string
		ReceivedBy *uint
		BranchID   uint
		PaidAt     time.Time
	}
	if err := g.tx.Raw(`SELECT p.invoice_id, p.amount, p.method, p.received_by, i.branch_id, p.paid_at
		FROM payments p JOIN invoices i ON i.id = p.invoice_id WHERE p.paid_at >= ?`, since).Scan(&pays).Error; err != nil {
		return err
	}
	for _, p := range pays {
		bulk.add(nUint(p.ReceivedBy), "PAYMENT", "invoice", p.InvoiceID, nil, js(`{"amount":%.0f,"method":%q}`, p.Amount, p.Method), ipOf(p.BranchID), p.PaidAt.Add(time.Second))
	}

	var evs []struct {
		SubscriptionID uint
		EventType      string
		ActorID        *uint
		BranchID       uint
		CreatedAt      time.Time
	}
	if err := g.tx.Raw(`SELECT e.subscription_id, e.event_type, e.actor_id, s.branch_id, e.created_at
		FROM subscription_events e JOIN subscriptions s ON s.id = e.subscription_id WHERE e.created_at >= ?`, since).Scan(&evs).Error; err != nil {
		return err
	}
	action := map[string]string{"CREATED": "SELL", "RENEWED": "RENEW", "UPGRADED": "UPGRADE", "TRANSFERRED": "TRANSFER", "CANCELLED": "CANCEL",
		"FROZEN": "FREEZE_APPROVE", "FREEZE_REJECTED": "FREEZE_REJECT", "FREEZE_REQUESTED": "FREEZE_REQUEST", "UNFROZEN": "FREEZE_END"}
	for _, e := range evs {
		if a := action[e.EventType]; a != "" && e.ActorID != nil {
			bulk.add(*e.ActorID, a, "subscription", e.SubscriptionID, nil, js(`{"event":%q}`, e.EventType), ipOf(e.BranchID), e.CreatedAt)
		}
	}

	var mems []struct {
		ID           uint
		MemberCode   string
		HomeBranchID uint
		CreatedBy    *uint
		CreatedAt    time.Time
	}
	if err := g.tx.Raw(`SELECT id, member_code, home_branch_id, created_by, created_at FROM members WHERE created_at >= ? AND created_by IS NOT NULL`, since).
		Scan(&mems).Error; err != nil {
		return err
	}
	for _, m := range mems {
		bulk.add(*m.CreatedBy, "CREATE", "member", m.ID, nil, js(`{"member_code":%q}`, m.MemberCode), ipOf(m.HomeBranchID), m.CreatedAt)
	}

	var exps []struct {
		ID         uint
		BranchID   uint
		Amount     float64
		Status     string
		CreatedBy  *uint
		ApprovedBy *uint
		CreatedAt  time.Time
		UpdatedAt  time.Time
	}
	if err := g.tx.Raw(`SELECT id, branch_id, amount, status, created_by, approved_by, created_at, updated_at FROM expenses WHERE created_at >= ?`, since).
		Scan(&exps).Error; err != nil {
		return err
	}
	for _, e := range exps {
		bulk.add(nUint(e.CreatedBy), "CREATE", "expense", e.ID, nil, js(`{"amount":%.0f}`, e.Amount), ipOf(e.BranchID), e.CreatedAt)
		switch e.Status {
		case entity.ExpenseStatusApproved:
			bulk.add(nUint(e.ApprovedBy), "APPROVE", "expense", e.ID, js(`{"status":"PENDING"}`), js(`{"status":"APPROVED"}`), "14.232.166.20", e.UpdatedAt)
		case entity.ExpenseStatusRejected:
			bulk.add(nUint(e.ApprovedBy), "REJECT", "expense", e.ID, js(`{"status":"PENDING"}`), js(`{"status":"REJECTED"}`), "14.232.166.20", e.UpdatedAt)
		}
	}

	var prs []struct {
		ID       uint
		BranchID uint
		Period   string
		PaidAt   *time.Time
	}
	if err := g.tx.Raw(`SELECT id, branch_id, period, paid_at FROM payrolls WHERE status = 'PAID'`).Scan(&prs).Error; err != nil {
		return err
	}
	for _, p := range prs {
		if p.PaidAt != nil {
			bulk.add(g.ownerID, "LOCK", "payroll", p.ID, nil, js(`{"period":%q}`, p.Period), "14.232.166.20", p.PaidAt.Add(-50*time.Hour))
			bulk.add(g.ownerID, "PAY", "payroll", p.ID, js(`{"status":"LOCKED"}`), js(`{"status":"PAID"}`), "14.232.166.20", *p.PaidAt)
		}
	}

	// Đăng nhập của nhân viên khi vào ca (30 ngày gần nhất).
	var logins []struct {
		StaffID   uint
		BranchID  uint
		CheckInAt time.Time
	}
	if err := g.tx.Raw(`SELECT staff_id, branch_id, check_in_at FROM staff_attendances WHERE check_in_at >= ?`, g.now.AddDate(0, 0, -30)).
		Scan(&logins).Error; err != nil {
		return err
	}
	for _, l := range logins {
		at := l.CheckInAt.Add(time.Duration(g.r.between(1, 6)) * time.Minute)
		if at.After(g.now) {
			continue
		}
		bulk.add(l.StaffID, "LOGIN", "auth", l.StaffID, nil, nil, ipOf(l.BranchID), at)
	}
	for d := -30; d <= 0; d++ {
		at := g.at(g.day(d), g.r.between(7, 10), g.r.between(0, 59))
		if at.Before(g.now) {
			bulk.add(g.ownerID, "LOGIN", "auth", g.ownerID, nil, nil, "14.232.166.20", at)
		}
	}
	return bulk.flush(g.ctx, g.tx)
}

// ---------------------------------------------------------------------
// Sprint 13 – Hội viên tự mua gói trực tuyến qua VNPay: chuyển một phần hóa đơn gói (thu một lần, hội viên có
// tài khoản cổng) thành giao dịch online thành công – chạy TRƯỚC ca thu ngân/bảng lương để số liệu khớp.
// ---------------------------------------------------------------------

func (g *richGen) seedOnlinePayments() error {
	// Giao dịch VNPay mẫu có sẵn từ seed giai đoạn 2: khoản thu tương ứng phải ghi phương thức VNPAY.
	if err := g.tx.Exec(`UPDATE payments p SET method = 'VNPAY', received_by = NULL, reference = t.provider_txn_no
		FROM online_payment_txns t
		WHERE t.invoice_id = p.invoice_id AND t.status = 'SUCCESS' AND p.method <> 'VNPAY' AND p.amount = t.amount`).Error; err != nil {
		return err
	}
	var rows []struct {
		PaymentID uint
		InvoiceID uint
		MemberID  uint
		UserID    uint
		Amount    float64
		PaidAt    time.Time
	}
	if err := g.tx.Raw(`SELECT p.id AS payment_id, i.id AS invoice_id, i.member_id, m.user_id, p.amount, p.paid_at
		FROM payments p
		JOIN invoices i ON i.id = p.invoice_id AND i.status = 'PAID' AND p.amount = i.total
		JOIN members m ON m.id = i.member_id AND m.user_id IS NOT NULL
		WHERE p.method <> 'VNPAY' AND p.paid_at >= ?
			AND EXISTS (SELECT 1 FROM invoice_items it WHERE it.invoice_id = i.id AND it.item_type = 'MEMBERSHIP')
			AND NOT EXISTS (SELECT 1 FROM online_payment_txns t WHERE t.invoice_id = i.id)
			AND (SELECT COUNT(*) FROM payments p2 WHERE p2.invoice_id = i.id) = 1
		ORDER BY p.id`, g.now.AddDate(0, 0, -150)).Scan(&rows).Error; err != nil {
		return err
	}
	banks := []string{"NCB", "VCB", "TCB", "MB", "ACB", "VIB"}
	for _, r := range rows {
		if !g.r.chance(0.45) {
			continue
		}
		created := r.PaidAt.Add(-time.Duration(g.r.between(1, 4)) * time.Minute)
		txnRef := fmt.Sprintf("VNP%s%06d", created.In(g.loc).Format("060102"), r.InvoiceID)
		providerNo := fmt.Sprintf("14%06d", g.r.Intn(1000000))
		bank := banks[g.r.Intn(len(banks))]
		amount := r.Amount
		paidAt := r.PaidAt
		txn := entity.OnlinePaymentTxn{InvoiceID: r.InvoiceID, MemberID: r.MemberID, TxnRef: txnRef, Amount: amount,
			Status: entity.OnlineTxnSuccess, ProviderTxnNo: providerNo, BankCode: bank, RecordedAmount: &amount,
			RawIPN:    datatypes.JSON(fmt.Sprintf(`{"vnp_ResponseCode":"00","vnp_TransactionStatus":"00","vnp_BankCode":%q,"vnp_TransactionNo":%q}`, bank, providerNo)),
			CreatedAt: created, PaidAt: &paidAt}
		if err := g.tx.Create(&txn).Error; err != nil {
			return err
		}
		if err := g.tx.Exec(`UPDATE payments SET method = 'VNPAY', received_by = NULL, reference = ?, note = 'Thanh toán trực tuyến qua VNPay' WHERE id = ?`,
			providerNo, r.PaymentID).Error; err != nil {
			return err
		}
		if err := g.tx.Exec(`UPDATE invoices SET source = 'PORTAL', issued_by = ? WHERE id = ?`, r.UserID, r.InvoiceID).Error; err != nil {
			return err
		}
		if err := g.tx.Exec(`UPDATE subscriptions SET sold_by = ? WHERE invoice_id = ?`, r.UserID, r.InvoiceID).Error; err != nil {
			return err
		}
	}
	return nil
}
