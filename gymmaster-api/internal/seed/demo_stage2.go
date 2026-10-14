package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/gorm"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
)

// seedStage2Demo bổ sung dữ liệu mẫu Giai đoạn 2 (Sprint 12–17):
// giáo án & nhật ký tập, đánh giá HLV/lớp, thiết bị & bảo trì, khách tiềm năng CRM & vé tập thử,
// giao dịch VNPay trực tuyến và hóa đơn chờ thanh toán cho tài khoản hội viên mẫu.
func seedStage2Demo(ctx context.Context, tx *gorm.DB) error {
	var planCount int64
	if err := tx.WithContext(ctx).Model(&entity.WorkoutPlan{}).Count(&planCount).Error; err != nil {
		return err
	}
	if planCount > 0 {
		return nil
	}

	loc := demoLoc()
	now := time.Now().In(loc)
	today := dayOf(now, loc)

	var branches []entity.Branch
	if err := tx.WithContext(ctx).Order("id ASC").Find(&branches).Error; err != nil || len(branches) == 0 {
		return err
	}

	var showcase entity.Member
	_ = tx.WithContext(ctx).Where("lower(email) = ?", "hoivien@gymmaster.vn").First(&showcase).Error

	var trainers []entity.User
	_ = tx.WithContext(ctx).
		Joins("JOIN staff_profiles sp ON sp.user_id = users.id").
		Where("sp.position = ?", "TRAINER").
		Order("users.id ASC").
		Find(&trainers).Error

	var exercises []entity.Exercise
	_ = tx.WithContext(ctx).Order("id ASC").Find(&exercises).Error

	// 1. Giáo án tập luyện mẫu & giáo án cá nhân cho hội viên mẫu + nhật ký tập (Sprint 14)
	if len(trainers) > 0 && len(exercises) >= 8 {
		mainTrainer := trainers[0]
		exByGroup := map[string][]entity.Exercise{}
		for _, ex := range exercises {
			exByGroup[ex.MuscleGroup] = append(exByGroup[ex.MuscleGroup], ex)
		}
		pickEx := func(group string, idx int) entity.Exercise {
			list := exByGroup[group]
			if len(list) > 0 {
				return list[idx%len(list)]
			}
			return exercises[idx%len(exercises)]
		}

		// Giáo án mẫu 1: Push - Pull - Legs
		tpl1 := entity.WorkoutPlan{
			TrainerID: mainTrainer.ID,
			Name:      "Giáo án Tăng cơ Push – Pull – Legs (Mẫu 3 buổi/tuần)",
			Goal:      "Tăng cơ toàn thân, phát triển sức mạnh nền tảng cho hội viên trung cấp",
			Status:    entity.WorkoutPlanActive,
		}
		if err := tx.WithContext(ctx).Create(&tpl1).Error; err != nil {
			return err
		}
		tpl1Days := []struct {
			idx   int
			title string
			items []entity.WorkoutPlanItem
		}{
			{
				idx:   1,
				title: "Buổi 1 – Thân trên Đẩy (Ngực – Vai – Tay sau)",
				items: []entity.WorkoutPlanItem{
					{ExerciseID: pickEx(entity.MuscleChest, 0).ID, Sets: 4, Reps: 10, Weight: 50, RestSec: 90, Note: "Hạ thanh đòn chạm nhẹ ngực giữa", Sort: 1},
					{ExerciseID: pickEx(entity.MuscleChest, 1).ID, Sets: 3, Reps: 12, Weight: 18, RestSec: 75, Note: "Ghế dốc 30 độ, kiểm soát nhịp âm", Sort: 2},
					{ExerciseID: pickEx(entity.MuscleShoulders, 0).ID, Sets: 4, Reps: 10, Weight: 16, RestSec: 75, Note: "Giữ thẳng lưng, không võng lưng dưới", Sort: 3},
					{ExerciseID: pickEx(entity.MuscleArms, 1).ID, Sets: 3, Reps: 12, Weight: 20, RestSec: 60, Note: "Khép khuỷu tay cố định sát thân người", Sort: 4},
				},
			},
			{
				idx:   3,
				title: "Buổi 2 – Thân trên Kéo (Lưng – Xô – Tay trước)",
				items: []entity.WorkoutPlanItem{
					{ExerciseID: pickEx(entity.MuscleBack, 0).ID, Sets: 4, Reps: 10, Weight: 45, RestSec: 90, Note: "Kéo thanh đòn về phía ngực trên, ép bả vai", Sort: 1},
					{ExerciseID: pickEx(entity.MuscleBack, 1).ID, Sets: 4, Reps: 10, Weight: 40, RestSec: 90, Note: "Giữ lưng phẳng 45 độ", Sort: 2},
					{ExerciseID: pickEx(entity.MuscleArms, 0).ID, Sets: 3, Reps: 12, Weight: 12, RestSec: 60, Note: "Không đung đưa thân người", Sort: 3},
				},
			},
			{
				idx:   5,
				title: "Buổi 3 – Chân – Mông & Core",
				items: []entity.WorkoutPlanItem{
					{ExerciseID: pickEx(entity.MuscleLegs, 0).ID, Sets: 4, Reps: 8, Weight: 70, RestSec: 120, Note: "Đùi song song sàn, mở gối theo mũi chân", Sort: 1},
					{ExerciseID: pickEx(entity.MuscleLegs, 1).ID, Sets: 4, Reps: 10, Weight: 60, RestSec: 90, Note: "Cảm nhận căng cơ đùi sau", Sort: 2},
					{ExerciseID: pickEx(entity.MuscleCore, 0).ID, Sets: 3, Reps: 60, Weight: 0, RestSec: 45, Note: "Giữ mỗi hiệp 60 giây, siết chặt bụng", Sort: 3},
				},
			},
		}
		for _, d := range tpl1Days {
			day := entity.WorkoutPlanDay{PlanID: tpl1.ID, DayIndex: d.idx, Title: d.title}
			if err := tx.WithContext(ctx).Create(&day).Error; err != nil {
				return err
			}
			for i := range d.items {
				d.items[i].DayID = day.ID
				if err := tx.WithContext(ctx).Create(&d.items[i]).Error; err != nil {
					return err
				}
			}
		}

		// Giáo án mẫu 2: Giảm mỡ Full Body
		tpl2 := entity.WorkoutPlan{
			TrainerID: mainTrainer.ID,
			Name:      "Giáo án Giảm mỡ & Săn chắc Full Body (Mẫu)",
			Goal:      "Đốt mỡ thừa, tăng sức bền tim mạch và săn chắc cơ thể",
			Status:    entity.WorkoutPlanActive,
		}
		if err := tx.WithContext(ctx).Create(&tpl2).Error; err != nil {
			return err
		}
		dayFB := entity.WorkoutPlanDay{PlanID: tpl2.ID, DayIndex: 2, Title: "Tổ hợp Full Body & HIIT"}
		if err := tx.WithContext(ctx).Create(&dayFB).Error; err != nil {
			return err
		}
		fbItems := []entity.WorkoutPlanItem{
			{DayID: dayFB.ID, ExerciseID: pickEx(entity.MuscleFullBody, 0).ID, Sets: 4, Reps: 12, Weight: 16, RestSec: 60, Note: "Bật hông mạnh, giữ thẳng lưng", Sort: 1},
			{DayID: dayFB.ID, ExerciseID: pickEx(entity.MuscleCardio, 0).ID, Sets: 4, Reps: 15, Weight: 0, RestSec: 45, Note: "Duy trì nhịp tim vùng đốt mỡ", Sort: 2},
			{DayID: dayFB.ID, ExerciseID: pickEx(entity.MuscleCore, 1).ID, Sets: 3, Reps: 20, Weight: 0, RestSec: 45, Note: "Thở đều trong suốt hiệp tập", Sort: 3},
		}
		for i := range fbItems {
			if err := tx.WithContext(ctx).Create(&fbItems[i]).Error; err != nil {
				return err
			}
		}

		// Giáo án riêng cho hội viên mẫu hoivien@gymmaster.vn
		if showcase.ID > 0 {
			var ptContract entity.PTContract
			var contractID *uint
			trainerID := mainTrainer.ID
			if err := tx.WithContext(ctx).Where("member_id = ?", showcase.ID).Order("id DESC").First(&ptContract).Error; err == nil {
				contractID = ptrUint(ptContract.ID)
				trainerID = ptContract.TrainerID
			}
			stDate := addDays(today, -28)
			enDate := addDays(today, 28)
			memberPlan := entity.WorkoutPlan{
				MemberID:     ptrUint(showcase.ID),
				TrainerID:    trainerID,
				PTContractID: contractID,
				Name:         "Giáo án Tăng cơ – Giảm mỡ 8 tuần (Nguyễn Minh Quân)",
				Goal:         "Tăng 2kg cơ nạc, giảm tỷ lệ mỡ xuống dưới 16%, cải thiện sức mạnh Bench Press & Squat",
				StartDate:    &stDate,
				EndDate:      &enDate,
				Status:       entity.WorkoutPlanActive,
			}
			if err := tx.WithContext(ctx).Create(&memberPlan).Error; err != nil {
				return err
			}

			benchEx := pickEx(entity.MuscleChest, 0)
			squatEx := pickEx(entity.MuscleLegs, 0)
			latEx := pickEx(entity.MuscleBack, 0)
			shoulderEx := pickEx(entity.MuscleShoulders, 0)

			mDays := []struct {
				idx   int
				title string
				items []entity.WorkoutPlanItem
			}{
				{
					idx:   1,
					title: "Thứ 2 – Ngực & Vai (Upper Push)",
					items: []entity.WorkoutPlanItem{
						{ExerciseID: benchEx.ID, Sets: 4, Reps: 10, Weight: 60, RestSec: 90, Note: "Khởi động kỹ khớp vai trước khi vào hiệp chính", Sort: 1},
						{ExerciseID: shoulderEx.ID, Sets: 4, Reps: 10, Weight: 18, RestSec: 75, Note: "Đẩy tạ đơn có kiểm soát", Sort: 2},
					},
				},
				{
					idx:   3,
					title: "Thứ 4 – Lưng Xô & Tay trước (Upper Pull)",
					items: []entity.WorkoutPlanItem{
						{ExerciseID: latEx.ID, Sets: 4, Reps: 10, Weight: 50, RestSec: 90, Note: "Giữ nhịp kéo 2 giây – thả 3 giây", Sort: 1},
						{ExerciseID: pickEx(entity.MuscleArms, 0).ID, Sets: 3, Reps: 12, Weight: 14, RestSec: 60, Note: "Siết cơ nhị đầu ở đỉnh chuyển động", Sort: 2},
					},
				},
				{
					idx:   5,
					title: "Thứ 6 – Chân, Mông & Bụng (Lower Body & Core)",
					items: []entity.WorkoutPlanItem{
						{ExerciseID: squatEx.ID, Sets: 4, Reps: 8, Weight: 80, RestSec: 120, Note: "Gồng chặt core trước mỗi lần xuống tạ", Sort: 1},
						{ExerciseID: pickEx(entity.MuscleCore, 0).ID, Sets: 3, Reps: 60, Weight: 0, RestSec: 45, Note: "Plank 60s mỗi hiệp", Sort: 2},
					},
				},
			}

			var firstBenchItemID, firstSquatItemID, firstLatItemID *uint
			for _, md := range mDays {
				day := entity.WorkoutPlanDay{PlanID: memberPlan.ID, DayIndex: md.idx, Title: md.title}
				if err := tx.WithContext(ctx).Create(&day).Error; err != nil {
					return err
				}
				for i := range md.items {
					md.items[i].DayID = day.ID
					if err := tx.WithContext(ctx).Create(&md.items[i]).Error; err != nil {
						return err
					}
					if md.items[i].ExerciseID == benchEx.ID && firstBenchItemID == nil {
						firstBenchItemID = ptrUint(md.items[i].ID)
					}
					if md.items[i].ExerciseID == squatEx.ID && firstSquatItemID == nil {
						firstSquatItemID = ptrUint(md.items[i].ID)
					}
					if md.items[i].ExerciseID == latEx.ID && firstLatItemID == nil {
						firstLatItemID = ptrUint(md.items[i].ID)
					}
				}
			}

			// Tạo chuỗi nhật ký tập luyện tăng tiến mức tạ 4 tuần qua cho hội viên mẫu
			benchWeights := []float64{50, 52.5, 55, 57.5, 60}
			squatWeights := []float64{65, 70, 72.5, 77.5, 80}
			latWeights := []float64{40, 42.5, 45, 47.5, 50}
			for w := 0; w < 5; w++ {
				dayOffset := -28 + w*6
				tBench := atMin(addDays(today, dayOffset), 18*60+15)
				tLat := atMin(addDays(today, dayOffset+2), 18*60+20)
				tSquat := atMin(addDays(today, dayOffset+4), 18*60+10)
				logs := []entity.WorkoutLog{
					{
						MemberID:    showcase.ID,
						PlanItemID:  firstBenchItemID,
						ExerciseID:  benchEx.ID,
						Sets:        4,
						Reps:        10,
						Weight:      benchWeights[w],
						Note:        fmt.Sprintf("Tuần %d: Hoàn thành đủ 4 hiệp, cảm giác cơ ngực tốt", w+1),
						PerformedAt: tBench,
					},
					{
						MemberID:    showcase.ID,
						PlanItemID:  firstLatItemID,
						ExerciseID:  latEx.ID,
						Sets:        4,
						Reps:        10,
						Weight:      latWeights[w],
						Note:        fmt.Sprintf("Tuần %d: Kéo xô chuẩn form", w+1),
						PerformedAt: tLat,
					},
					{
						MemberID:    showcase.ID,
						PlanItemID:  firstSquatItemID,
						ExerciseID:  squatEx.ID,
						Sets:        4,
						Reps:        8,
						Weight:      squatWeights[w],
						Note:        fmt.Sprintf("Tuần %d: Squat sâu, giữ lưng vững", w+1),
						PerformedAt: tSquat,
					},
				}
				for i := range logs {
					if err := tx.WithContext(ctx).Create(&logs[i]).Error; err != nil {
						return err
					}
				}
			}
		}
	}

	// 2. Đánh giá HLV & Lớp học (Sprint 15)
	var completedPTSessions []entity.PTSession
	_ = tx.WithContext(ctx).
		Where("status = ?", entity.PTSessionCompleted).
		Order("end_at DESC").
		Limit(8).
		Find(&completedPTSessions).Error

	ptComments := []string{
		"HLV hướng dẫn rất nhiệt tình, chỉnh từng động tác Squat và Deadlift chuẩn kỹ thuật.",
		"Buổi tập chất lượng, bài tập vừa sức nhưng đốt rất nhiều năng lượng.",
		"HLV theo sát từng hiệp, nhắc nhở nhịp thở và tư thế lưng rất kỹ.",
		"Giáo án cá nhân hóa hợp lý, sau 1 tháng thấy cơ thể săn chắc rõ rệt.",
	}
	for i, sess := range completedPTSessions {
		// Giữ lại ít nhất 1 buổi PT hoàn thành gần nhất của hội viên mẫu chưa đánh giá để hội viên trải nghiệm nút Đánh giá trên /me/pt
		if showcase.ID > 0 && sess.MemberID == showcase.ID && i == 0 {
			continue
		}
		rating := 5
		if i%3 == 2 {
			rating = 4
		}
		rev := entity.Review{
			MemberID:   sess.MemberID,
			TargetType: entity.ReviewTargetPTSession,
			TargetID:   sess.ID,
			TrainerID:  sess.TrainerID,
			BranchID:   ptrUint(sess.BranchID),
			Rating:     rating,
			Comment:    ptComments[i%len(ptComments)],
			CreatedAt:  sess.EndAt.Add(2 * time.Hour),
		}
		_ = tx.WithContext(ctx).Create(&rev).Error
	}

	var attendedBookings []entity.ClassBooking
	_ = tx.WithContext(ctx).
		Where("status = ?", entity.BookingStatusAttended).
		Order("id DESC").
		Limit(6).
		Find(&attendedBookings).Error

	classComments := []string{
		"Lớp học sôi động, âm nhạc cuốn hút và HLV truyền cảm hứng rất tốt.",
		"Phòng tập sạch sẽ, thoáng mát, bài tập phù hợp cho cả người mới.",
		"HLV hướng dẫn dễ hiểu, sửa tư thế cho từng học viên trong lớp.",
	}
	for i, bk := range attendedBookings {
		if showcase.ID > 0 && bk.MemberID == showcase.ID && i == 0 {
			continue
		}
		var sch entity.ClassSchedule
		if err := tx.WithContext(ctx).First(&sch, bk.ScheduleID).Error; err != nil {
			continue
		}
		rev := entity.Review{
			MemberID:   bk.MemberID,
			TargetType: entity.ReviewTargetClassSchedule,
			TargetID:   sch.ID,
			TrainerID:  sch.TrainerID,
			BranchID:   ptrUint(sch.BranchID),
			Rating:     5 - (i % 2),
			Comment:    classComments[i%len(classComments)],
			CreatedAt:  sch.EndAt.Add(3 * time.Hour),
		}
		_ = tx.WithContext(ctx).Create(&rev).Error
	}

	// Cập nhật lại rating_avg và rating_count trên trainer_profiles
	_ = tx.WithContext(ctx).Exec(`
		UPDATE trainer_profiles tp
		SET rating_avg = sub.avg_rating,
		    rating_count = sub.cnt
		FROM (
			SELECT trainer_id, COALESCE(ROUND(AVG(rating)::numeric, 2), 0) AS avg_rating, COUNT(*) AS cnt
			FROM reviews
			GROUP BY trainer_id
		) sub
		WHERE tp.user_id = sub.trainer_id
	`).Error

	// 3. Thiết bị phòng tập & Nhật ký bảo trì (Sprint 15)
	equipDefs := []struct {
		codePrefix string
		name       string
		cat        string
		price      float64
		interval   int
		status     string
		nextOffset int
		note       string
	}{
		{"TB-RUN01", "Máy chạy bộ thương mại Technogym Excite Run", entity.EquipmentCategoryCardio, 95000000, 60, entity.EquipmentStatusActive, 12, "Khu Cardio tầng 1"},
		{"TB-RUN02", "Máy chạy bộ thương mại Matrix T7xe", entity.EquipmentCategoryCardio, 88000000, 60, entity.EquipmentStatusBroken, -2, "Đang chờ thay băng tải"},
		{"TB-BIKE01", "Xe đạp tập kháng lực từ Assault AirBike", entity.EquipmentCategoryCardio, 32000000, 90, entity.EquipmentStatusActive, 0, "Đến kỳ bảo dưỡng định kỳ hôm nay"},
		{"TB-RACK01", "Khung gánh đùi Power Rack Rogue RML-490", entity.EquipmentCategoryStrength, 48000000, 120, entity.EquipmentStatusActive, 45, "Khu Strength chính"},
		{"TB-CABLE01", "Giàn cáp đôi đa năng Dual Adjustable Pulley", entity.EquipmentCategoryStrength, 65000000, 90, entity.EquipmentStatusMaintenance, -1, "Đang bảo dưỡng dây cáp"},
		{"TB-DBSET01", "Bộ tạ đơn Urethane 2.5kg – 40kg kèm giá đỡ", entity.EquipmentCategoryFreeWeight, 54000000, 180, entity.EquipmentStatusActive, 80, "Khu Free Weight"},
	}

	for bIdx, b := range branches {
		var room entity.Room
		var roomID *uint
		if err := tx.WithContext(ctx).Where("branch_id = ?", b.ID).Order("id ASC").First(&room).Error; err == nil {
			roomID = ptrUint(room.ID)
		}
		for eIdx, ed := range equipDefs {
			// Chỉ chi nhánh đầu tiên có đủ cả BROKEN & MAINTENANCE, các chi nhánh khác chủ yếu ACTIVE
			st := ed.status
			if bIdx > 0 && st != entity.EquipmentStatusActive {
				st = entity.EquipmentStatusActive
			}
			pDate := addDays(today, -300+eIdx*15)
			wUntil := addDays(pDate, 730)
			nextMaint := addDays(today, ed.nextOffset+bIdx*5)
			eq := entity.Equipment{
				BranchID:                b.ID,
				RoomID:                  roomID,
				Code:                    fmt.Sprintf("%s-%s", b.Code, ed.codePrefix),
				Name:                    ed.name,
				Category:                ed.cat,
				PurchaseDate:            &pDate,
				Price:                   ed.price,
				WarrantyUntil:           &wUntil,
				Status:                  st,
				MaintenanceIntervalDays: ed.interval,
				NextMaintenanceDate:     &nextMaint,
				Note:                    ed.note,
			}
			if err := tx.WithContext(ctx).Create(&eq).Error; err != nil {
				return err
			}

			// Tạo nhật ký bảo trì hoàn tất trong quá khứ cho máy chạy bộ 01
			if eIdx == 0 {
				sAt := atMin(addDays(today, -45), 9*60)
				fAt := atMin(addDays(today, -45), 11*60+30)
				logDone := entity.MaintenanceLog{
					EquipmentID: eq.ID,
					Type:        entity.MaintenanceTypePeriodic,
					Description: "Bảo dưỡng định kỳ: tra dầu băng tải, vệ sinh cảm biến nhịp tim và cân chỉnh độ căng",
					Cost:        1200000,
					Status:      entity.MaintenanceStatusDone,
					StartedAt:   &sAt,
					FinishedAt:  &fAt,
					CreatedAt:   sAt,
				}
				_ = tx.WithContext(ctx).Create(&logDone).Error
			}
			if st == entity.EquipmentStatusBroken {
				sAt := atMin(addDays(today, -2), 15*60)
				logOpen := entity.MaintenanceLog{
					EquipmentID: eq.ID,
					Type:        entity.MaintenanceTypeRepair,
					Description: "Băng tải phát tiếng kêu lớn khi chạy tốc độ trên 14km/h, cần kiểm tra bạc đạn rulo",
					Cost:        2500000,
					Status:      entity.MaintenanceStatusOpen,
					StartedAt:   &sAt,
					CreatedAt:   sAt,
				}
				_ = tx.WithContext(ctx).Create(&logOpen).Error
			}
			if st == entity.EquipmentStatusMaintenance {
				sAt := atMin(addDays(today, -1), 10*60)
				logProg := entity.MaintenanceLog{
					EquipmentID: eq.ID,
					Type:        entity.MaintenanceTypePeriodic,
					Description: "Thay cáp chịu lực và bôi trơn hệ thống ròng rọc",
					Cost:        1800000,
					Status:      entity.MaintenanceStatusInProgress,
					StartedAt:   &sAt,
					CreatedAt:   sAt,
				}
				_ = tx.WithContext(ctx).Create(&logProg).Error
			}
		}
	}

	// 4. CRM Khách tiềm năng (Leads), Lịch sử chăm sóc & Vé tập thử QR (Sprint 16)
	var firstPlan entity.MembershipPlan
	var interestPlanID *uint
	if err := tx.WithContext(ctx).Where("status = ?", consts.StatusActive).Order("sort_order ASC, id ASC").First(&firstPlan).Error; err == nil {
		interestPlanID = ptrUint(firstPlan.ID)
	}

	leadDefs := []struct {
		branchIdx int
		name      string
		phone     string
		email     string
		source    string
		status    string
		note      string
		trialQR   string
		acts      []struct{ typ, content string }
	}{
		{
			branchIdx: 0,
			name:      "Trần Hoàng Nam",
			phone:     "0938112233",
			email:     "nam.tran.lead@gmail.com",
			source:    entity.LeadSourceWebsite,
			status:    entity.LeadStatusNew,
			note:      "Đăng ký tập thử từ trang chủ, quan tâm khung giờ tối 18h30 sau giờ làm",
		},
		{
			branchIdx: 0,
			name:      "Lê Ngọc Ánh",
			phone:     "0938223344",
			email:     "anh.le.lead@gmail.com",
			source:    entity.LeadSourceFacebook,
			status:    entity.LeadStatusContacted,
			note:      "Quan tâm gói 3 tháng kèm lớp Yoga & Zumba",
			acts: []struct{ typ, content string }{
				{entity.LeadActivityCall, "Đã gọi tư vấn chương trình ưu đãi tháng này, khách hẹn cuối tuần qua xem phòng tập."},
				{entity.LeadActivitySMS, "Đã gửi lịch lớp Yoga & Zumba qua Zalo cho khách tham khảo."},
			},
		},
		{
			branchIdx: 0,
			name:      "Phạm Quốc Khánh",
			phone:     "0938334455",
			email:     "khanh.pham.lead@gmail.com",
			source:    entity.LeadSourceWalkIn,
			status:    entity.LeadStatusTrial,
			note:      "Đã cấp vé tập thử 1 ngày tại cơ sở Cầu Giấy",
			trialQR:   "TP-DEMO0001",
			acts: []struct{ typ, content string }{
				{entity.LeadActivityVisit, "Khách ghé tham quan khu vực tạ rời và phòng tập PT."},
				{entity.LeadActivityNote, "Đã cấp mã QR tập thử TP-DEMO0001 có hiệu lực hôm nay."},
			},
		},
		{
			branchIdx: 1,
			name:      "Vũ Minh Châu",
			phone:     "0938445566",
			email:     "chau.vu.lead@gmail.com",
			source:    entity.LeadSourceReferral,
			status:    entity.LeadStatusTrial,
			note:      "Bạn của hội viên cũ giới thiệu, muốn trải nghiệm thử 1 buổi",
			trialQR:   "TP-DEMO0002",
			acts: []struct{ typ, content string }{
				{entity.LeadActivityCall, "Đã xác nhận lịch hẹn tập thử lúc 17h30 chiều nay."},
			},
		},
		{
			branchIdx: 1,
			name:      "Đặng Thu Trang",
			phone:     "0938556677",
			email:     "trang.dang.lead@gmail.com",
			source:    entity.LeadSourceWebsite,
			status:    entity.LeadStatusConverted,
			note:      "Đã tập thử hài lòng và đăng ký gói tập chính thức",
			acts: []struct{ typ, content string }{
				{entity.LeadActivityCall, "Tư vấn gói 6 tháng đa chi nhánh."},
				{entity.LeadActivityVisit, "Khách hoàn thành buổi tập thử và chốt đăng ký gói."},
			},
		},
		{
			branchIdx: 2,
			name:      "Ngô Gia Bảo",
			phone:     "0938667788",
			email:     "bao.ngo.lead@gmail.com",
			source:    entity.LeadSourceFacebook,
			status:    entity.LeadStatusLost,
			note:      "Khách chuyển công tác sang khu vực khác xa phòng tập",
			acts: []struct{ typ, content string }{
				{entity.LeadActivityCall, "Khách báo vừa chuyển văn phòng nên chưa sắp xếp đi tập được."},
			},
		},
	}

	for _, ld := range leadDefs {
		b := branches[ld.branchIdx%len(branches)]
		var trialDate *time.Time
		if ld.status == entity.LeadStatusTrial {
			trialDate = &today
		}
		var convMemberID *uint
		if ld.status == entity.LeadStatusConverted && showcase.ID > 0 {
			convMemberID = ptrUint(showcase.ID)
		}
		lead := entity.Lead{
			BranchID:       b.ID,
			FullName:       ld.name,
			Phone:          ld.phone,
			Email:          ld.email,
			Source:         ld.source,
			InterestPlanID: interestPlanID,
			Status:         ld.status,
			TrialDate:      trialDate,
			MemberID:       convMemberID,
			Note:           ld.note,
		}
		if err := tx.WithContext(ctx).Create(&lead).Error; err != nil {
			return err
		}
		for _, ac := range ld.acts {
			act := entity.LeadActivity{
				LeadID:  lead.ID,
				Type:    ac.typ,
				Content: ac.content,
			}
			if err := tx.WithContext(ctx).Create(&act).Error; err != nil {
				return err
			}
		}
		if ld.trialQR != "" {
			tp := entity.TrialPass{
				LeadID:    lead.ID,
				BranchID:  b.ID,
				ValidDate: today,
				QRToken:   ld.trialQR,
			}
			if err := tx.WithContext(ctx).Create(&tp).Error; err != nil {
				return err
			}
		}
	}

	// 5. Giao dịch VNPay trực tuyến & Hóa đơn UNPAID mẫu cho hội viên mẫu (Sprint 13)
	if showcase.ID > 0 {
		// Tạo 1 hóa đơn UNPAID nguồn PORTAL để hội viên mẫu có thể bấm "Thanh toán VNPay" ngay trên /me/invoices
		unpaidInv := entity.Invoice{
			InvoiceNo: fmt.Sprintf("HD%s99001", today.Format("060102")),
			MemberID:  ptrUint(showcase.ID),
			BranchID:  showcase.HomeBranchID,
			Subtotal:  800000,
			Discount:  0,
			Total:     800000,
			Status:    entity.InvoiceStatusUnpaid,
			Source:    entity.InvoiceSourcePortal,
			Note:      "Đăng ký Gói 1 tháng qua Cổng hội viên (chờ thanh toán VNPay)",
			IssuedAt:  now.Add(-2 * time.Hour),
		}
		if err := tx.WithContext(ctx).Create(&unpaidInv).Error; err == nil {
			_ = tx.WithContext(ctx).Create(&entity.InvoiceItem{
				InvoiceID:   unpaidInv.ID,
				ItemType:    entity.InvoiceItemMembership,
				Description: "Gói tập 1 tháng (Gia hạn trực tuyến)",
				Quantity:    1,
				UnitPrice:   800000,
				Amount:      800000,
			}).Error
		}

		// Ghi nhận 1 giao dịch VNPay thành công trong lịch sử
		var paidInv entity.Invoice
		if err := tx.WithContext(ctx).Where("member_id = ? AND status = ?", showcase.ID, entity.InvoiceStatusPaid).Order("id DESC").First(&paidInv).Error; err == nil {
			paidAt := paidInv.IssuedAt.Add(3 * time.Minute)
			txn := entity.OnlinePaymentTxn{
				InvoiceID:     paidInv.ID,
				MemberID:      showcase.ID,
				TxnRef:        fmt.Sprintf("VNP%s%04d", paidInv.IssuedAt.Format("060102"), paidInv.ID),
				Amount:        paidInv.Total,
				Status:        entity.OnlineTxnSuccess,
				ProviderTxnNo: "14589201",
				BankCode:      "NCB",
				RawIPN:        []byte(`{"vnp_ResponseCode":"00","vnp_TransactionStatus":"00","vnp_BankCode":"NCB"}`),
				PaidAt:        &paidAt,
				CreatedAt:     paidInv.IssuedAt,
			}
			_ = tx.WithContext(ctx).Create(&txn).Error
		}
	}

	if err := seedStage3And4Demo(ctx, tx); err != nil {
		return err
	}

	g.Log().Infof(ctx, "seed: đã khởi tạo dữ liệu mẫu Giai đoạn 2, 3 & 4")
	return nil
}

