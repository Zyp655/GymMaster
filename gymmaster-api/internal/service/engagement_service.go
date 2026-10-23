package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/engagement/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	auditEntityFood         = "food"
	auditEntityMealPlan     = "meal_plan"
	auditEntityAnnouncement = "announcement"
	auditEntitySurvey       = "survey"
)

var (
	foodOrderable         = []string{"id", "name", "calories", "protein", "carbs", "fat", "created_at"}
	mealPlanOrderable     = []string{"id", "name", "target_calories", "status", "created_at"}
	announcementOrderable = []string{"id", "title", "publish_at", "pinned", "created_at"}
	surveyOrderable       = []string{"id", "title", "start_at", "status", "created_at"}
	validMeals            = []string{entity.MealBreakfast, entity.MealLunch, entity.MealDinner, entity.MealSnack}
)

type (
	EngagementService interface {
		ListFoods(ctx context.Context, req *v1.ListFoodsReq) (*v1.ListFoodsRes, error)
		CreateFood(ctx context.Context, in v1.FoodInput) (*v1.FoodItem, error)
		UpdateFood(ctx context.Context, id uint, in v1.FoodInput) (*v1.FoodItem, error)
		DeleteFood(ctx context.Context, id uint) error

		ListMealPlans(ctx context.Context, req *v1.ListMealPlansReq) (*v1.ListMealPlansRes, error)
		GetMealPlan(ctx context.Context, id uint) (*v1.MealPlanDetail, error)
		CreateMealPlan(ctx context.Context, in v1.MealPlanInput) (*v1.MealPlanDetail, error)
		UpdateMealPlan(ctx context.Context, id uint, in v1.MealPlanInput) (*v1.MealPlanDetail, error)
		DeleteMealPlan(ctx context.Context, id uint) error

		ListAnnouncements(ctx context.Context, req *v1.ListAnnouncementsReq) (*v1.ListAnnouncementsRes, error)
		CreateAnnouncement(ctx context.Context, in v1.AnnouncementInput) (*v1.AnnouncementItem, error)
		UpdateAnnouncement(ctx context.Context, id uint, in v1.AnnouncementInput) (*v1.AnnouncementItem, error)
		DeleteAnnouncement(ctx context.Context, id uint) error

		ListSurveys(ctx context.Context, req *v1.ListSurveysReq) (*v1.ListSurveysRes, error)
		CreateSurvey(ctx context.Context, in v1.SurveyInput) (*v1.SurveyItem, error)
		UpdateSurvey(ctx context.Context, id uint, in v1.SurveyInput) (*v1.SurveyItem, error)
		DeleteSurvey(ctx context.Context, id uint) error
		NPSReport(ctx context.Context, req *v1.NPSReportReq) (*v1.NPSReportRes, error)

		MemberNutritionOverview(ctx context.Context, req *v1.MemberNutritionOverviewReq) (*v1.MemberNutritionOverviewRes, error)
		MemberFoodLogs(ctx context.Context, req *v1.MemberFoodLogsReq) (*v1.MemberFoodLogsRes, error)
		CreateFoodLog(ctx context.Context, req *v1.CreateFoodLogReq) (*v1.FoodLogItem, error)
		DeleteFoodLog(ctx context.Context, id uint) error
		UpdateFoodLog(ctx context.Context, req *v1.UpdateFoodLogReq) (*v1.FoodLogItem, error)
		MemberAnnouncements(ctx context.Context) (v1.MemberAnnouncementsRes, error)
		MemberSurveys(ctx context.Context) (v1.MemberSurveysRes, error)
		SubmitSurveyResponse(ctx context.Context, req *v1.SubmitSurveyResponseReq) (*v1.SurveyResponseItem, error)
	}

	engagementService struct {
		repo  repository.EngagementRepository
		tx    postgres.TxManager
		audit AuditService
		now   func() time.Time
	}
)

func NewEngagementService(repo repository.EngagementRepository, tx postgres.TxManager, audit AuditService) EngagementService {
	return &engagementService{repo: repo, tx: tx, audit: audit, now: time.Now}
}

func (s *engagementService) runTx(ctx context.Context, fn func(context.Context) error) error {
	if s.tx != nil {
		return s.tx.WithTx(ctx, fn)
	}
	return fn(ctx)
}

func (s *engagementService) currentMember(ctx context.Context) (*entity.Member, error) {
	uid := reqctx.UserID(ctx)
	if uid == nil {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	m, err := s.repo.FindMemberByUserID(ctx, *uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Tài khoản chưa được liên kết với hội viên")
		}
		return nil, err
	}
	return m, nil
}

// ---- Admin: Foods ---------------------------------------------------------------

func (s *engagementService) ListFoods(ctx context.Context, req *v1.ListFoodsReq) (*v1.ListFoodsRes, error) {
	q := req.PageReq.ToQuery(foodOrderable, "id")
	items, total, err := s.repo.ListFoods(ctx, q, req.Search)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(items, toFoodItem), total, q), nil
}

func (s *engagementService) CreateFood(ctx context.Context, in v1.FoodInput) (*v1.FoodItem, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên thực phẩm")
	}
	dup, err := s.repo.ExistsFoodName(ctx, name, 0)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, rescode.New(rescode.Conflict, "Tên thực phẩm đã tồn tại")
	}
	unit := strings.TrimSpace(in.Unit)
	if unit == "" {
		unit = "100g"
	}
	f := &entity.Food{
		Name:     name,
		Unit:     unit,
		Calories: in.Calories,
		Protein:  in.Protein,
		Carbs:    in.Carbs,
		Fat:      in.Fat,
	}
	if err := s.repo.CreateFood(ctx, f); err != nil {
		return nil, mapDuplicate(err, "Tên thực phẩm đã tồn tại")
	}
	res := toFoodItem(*f)
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityFood, f.ID, nil, res)
	}
	return res, nil
}

func (s *engagementService) UpdateFood(ctx context.Context, id uint, in v1.FoodInput) (*v1.FoodItem, error) {
	f, err := s.repo.FindFoodByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thực phẩm")
		}
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên thực phẩm")
	}
	dup, err := s.repo.ExistsFoodName(ctx, name, id)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, rescode.New(rescode.Conflict, "Tên thực phẩm đã tồn tại")
	}
	before := toFoodItem(*f)
	f.Name = name
	if strings.TrimSpace(in.Unit) != "" {
		f.Unit = strings.TrimSpace(in.Unit)
	}
	f.Calories = in.Calories
	f.Protein = in.Protein
	f.Carbs = in.Carbs
	f.Fat = in.Fat
	if err := s.repo.UpdateFood(ctx, f); err != nil {
		return nil, mapDuplicate(err, "Tên thực phẩm đã tồn tại")
	}
	res := toFoodItem(*f)
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityFood, f.ID, before, res)
	}
	return res, nil
}

func (s *engagementService) DeleteFood(ctx context.Context, id uint) error {
	f, err := s.repo.FindFoodByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy thực phẩm")
		}
		return err
	}
	if err := s.repo.DeleteFood(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityFood, id, toFoodItem(*f), nil)
	}
	return nil
}

// ---- Admin: Meal Plans ----------------------------------------------------------

func (s *engagementService) ListMealPlans(ctx context.Context, req *v1.ListMealPlansReq) (*v1.ListMealPlansRes, error) {
	q := req.PageReq.ToQuery(mealPlanOrderable, "id")
	q.OrderBy = "mp." + q.OrderBy
	rows, total, err := s.repo.ListMealPlans(ctx, q, req.MemberID, req.TrainerID, strings.TrimSpace(req.Status))
	if err != nil {
		return nil, err
	}
	planIDs := make([]uint, 0, len(rows))
	for _, r := range rows {
		planIDs = append(planIDs, r.ID)
	}
	itemsByPlan, err := s.repo.LoadMealPlanItems(ctx, planIDs)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, func(r repository.MealPlanRow) *v1.MealPlanDetail {
		return toMealPlanDetail(r, itemsByPlan[r.ID])
	}), total, q), nil
}

func (s *engagementService) GetMealPlan(ctx context.Context, id uint) (*v1.MealPlanDetail, error) {
	row, err := s.repo.FindMealPlanByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thực đơn dinh dưỡng")
		}
		return nil, err
	}
	itemsByPlan, err := s.repo.LoadMealPlanItems(ctx, []uint{id})
	if err != nil {
		return nil, err
	}
	return toMealPlanDetail(*row, itemsByPlan[id]), nil
}

func (s *engagementService) prepareMealPlanItems(ctx context.Context, inItems []v1.MealPlanItemInput) ([]entity.MealPlanItem, error) {
	items := make([]entity.MealPlanItem, 0, len(inItems))
	for _, it := range inItems {
		dayIdx := it.DayIndex
		if dayIdx <= 0 {
			dayIdx = 1
		}
		if dayIdx > 7 {
			return nil, rescode.New(rescode.BadRequest, "Thứ tự ngày trong thực đơn phải từ 1 đến 7")
		}
		meal := strings.ToUpper(strings.TrimSpace(it.Meal))
		if meal == "" {
			meal = entity.MealBreakfast
		}
		if !slices.Contains(validMeals, meal) {
			return nil, rescode.New(rescode.BadRequest, "Bữa ăn không hợp lệ")
		}
		if _, err := s.repo.FindFoodByID(ctx, it.FoodID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, rescode.New(rescode.BadRequest, "Thực phẩm trong thực đơn không tồn tại")
			}
			return nil, err
		}
		qty := it.Qty
		if qty <= 0 {
			qty = 1
		}
		items = append(items, entity.MealPlanItem{
			DayIndex: dayIdx,
			Meal:     meal,
			FoodID:   it.FoodID,
			Qty:      qty,
			Note:     strings.TrimSpace(it.Note),
		})
	}
	return items, nil
}

func (s *engagementService) CreateMealPlan(ctx context.Context, in v1.MealPlanInput) (*v1.MealPlanDetail, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên thực đơn")
	}
	if _, err := s.repo.FindMemberByID(ctx, in.MemberID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.BadRequest, "Hội viên không tồn tại")
		}
		return nil, err
	}
	startDate, err := workoutParseDateOpt(in.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := workoutParseDateOpt(in.EndDate)
	if err != nil {
		return nil, err
	}
	if startDate != nil && endDate != nil && endDate.Before(*startDate) {
		return nil, rescode.New(rescode.BadRequest, "Ngày kết thúc phải sau hoặc bằng ngày bắt đầu")
	}
	items, err := s.prepareMealPlanItems(ctx, in.Items)
	if err != nil {
		return nil, err
	}
	targetCal := in.TargetCalories
	if targetCal <= 0 {
		targetCal = 2000
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = consts.StatusActive
	}
	trainerID := in.TrainerID
	if trainerID == nil {
		trainerID = reqctx.UserID(ctx)
	}
	plan := &entity.MealPlan{
		MemberID:       in.MemberID,
		TrainerID:      trainerID,
		Name:           name,
		Goal:           strings.TrimSpace(in.Goal),
		TargetCalories: targetCal,
		StartDate:      startDate,
		EndDate:        endDate,
		Status:         status,
	}
	err = s.runTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.CreateMealPlan(txCtx, plan); err != nil {
			return err
		}
		return s.repo.ReplaceMealPlanItems(txCtx, plan.ID, items)
	})
	if err != nil {
		return nil, err
	}
	res, err := s.GetMealPlan(ctx, plan.ID)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityMealPlan, plan.ID, nil, res)
	}
	return res, nil
}

func (s *engagementService) UpdateMealPlan(ctx context.Context, id uint, in v1.MealPlanInput) (*v1.MealPlanDetail, error) {
	existing, err := s.repo.FindMealPlanByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thực đơn dinh dưỡng")
		}
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên thực đơn")
	}
	startDate, err := workoutParseDateOpt(in.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := workoutParseDateOpt(in.EndDate)
	if err != nil {
		return nil, err
	}
	items, err := s.prepareMealPlanItems(ctx, in.Items)
	if err != nil {
		return nil, err
	}
	plan := existing.MealPlan
	if in.MemberID > 0 {
		plan.MemberID = in.MemberID
	}
	if in.TrainerID != nil {
		plan.TrainerID = in.TrainerID
	}
	plan.Name = name
	plan.Goal = strings.TrimSpace(in.Goal)
	if in.TargetCalories > 0 {
		plan.TargetCalories = in.TargetCalories
	}
	plan.StartDate = startDate
	plan.EndDate = endDate
	if strings.TrimSpace(in.Status) != "" {
		plan.Status = strings.ToUpper(strings.TrimSpace(in.Status))
	}
	err = s.runTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.UpdateMealPlan(txCtx, &plan); err != nil {
			return err
		}
		return s.repo.ReplaceMealPlanItems(txCtx, plan.ID, items)
	})
	if err != nil {
		return nil, err
	}
	return s.GetMealPlan(ctx, id)
}

func (s *engagementService) DeleteMealPlan(ctx context.Context, id uint) error {
	existing, err := s.repo.FindMealPlanByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy thực đơn dinh dưỡng")
		}
		return err
	}
	if err := s.repo.DeleteMealPlan(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityMealPlan, id, existing, nil)
	}
	return nil
}

// ---- Admin: Announcements -------------------------------------------------------

func (s *engagementService) ListAnnouncements(ctx context.Context, req *v1.ListAnnouncementsReq) (*v1.ListAnnouncementsRes, error) {
	q := req.PageReq.ToQuery(announcementOrderable, "id")
	q.OrderBy = "a." + q.OrderBy
	rows, total, err := s.repo.ListAnnouncements(ctx, q, req.BranchID)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, toAnnouncementItem), total, q), nil
}

func (s *engagementService) CreateAnnouncement(ctx context.Context, in v1.AnnouncementInput) (*v1.AnnouncementItem, error) {
	title := strings.TrimSpace(in.Title)
	body := strings.TrimSpace(in.Body)
	if title == "" || body == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tiêu đề và nội dung thông báo")
	}
	publishAt := s.now()
	if strings.TrimSpace(in.PublishAt) != "" {
		if t, err := parseFlexibleTime(in.PublishAt, publishAt); err == nil {
			publishAt = t
		}
	}
	var expireAt *time.Time
	if strings.TrimSpace(in.ExpireAt) != "" {
		if t, err := parseFlexibleTime(in.ExpireAt, publishAt); err == nil {
			expireAt = &t
		}
	}
	a := &entity.Announcement{
		BranchID:  in.BranchID,
		Title:     title,
		Body:      body,
		ImageURL:  strings.TrimSpace(in.ImageURL),
		PublishAt: publishAt,
		ExpireAt:  expireAt,
		Pinned:    in.Pinned,
		CreatedBy: reqctx.UserID(ctx),
	}
	if err := s.repo.CreateAnnouncement(ctx, a); err != nil {
		return nil, err
	}
	row, err := s.repo.FindAnnouncementByID(ctx, a.ID)
	if err != nil {
		row = &repository.AnnouncementRow{Announcement: *a}
	}
	res := toAnnouncementItem(*row)
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityAnnouncement, a.ID, nil, res)
	}
	return res, nil
}

func (s *engagementService) UpdateAnnouncement(ctx context.Context, id uint, in v1.AnnouncementInput) (*v1.AnnouncementItem, error) {
	existing, err := s.repo.FindAnnouncementByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thông báo")
		}
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	body := strings.TrimSpace(in.Body)
	if title == "" || body == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tiêu đề và nội dung thông báo")
	}
	a := existing.Announcement
	a.BranchID = in.BranchID
	a.Title = title
	a.Body = body
	a.ImageURL = strings.TrimSpace(in.ImageURL)
	a.Pinned = in.Pinned
	if strings.TrimSpace(in.PublishAt) != "" {
		if t, err := parseFlexibleTime(in.PublishAt, a.PublishAt); err == nil {
			a.PublishAt = t
		}
	}
	if strings.TrimSpace(in.ExpireAt) != "" {
		if t, err := parseFlexibleTime(in.ExpireAt, a.PublishAt); err == nil {
			a.ExpireAt = &t
		}
	} else {
		a.ExpireAt = nil
	}
	if err := s.repo.UpdateAnnouncement(ctx, &a); err != nil {
		return nil, err
	}
	row, err := s.repo.FindAnnouncementByID(ctx, id)
	if err != nil {
		row = &repository.AnnouncementRow{Announcement: a}
	}
	return toAnnouncementItem(*row), nil
}

func (s *engagementService) DeleteAnnouncement(ctx context.Context, id uint) error {
	existing, err := s.repo.FindAnnouncementByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy thông báo")
		}
		return err
	}
	if err := s.repo.DeleteAnnouncement(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityAnnouncement, id, toAnnouncementItem(*existing), nil)
	}
	return nil
}

// ---- Admin: Surveys & NPS Report ------------------------------------------------

func (s *engagementService) ListSurveys(ctx context.Context, req *v1.ListSurveysReq) (*v1.ListSurveysRes, error) {
	q := req.PageReq.ToQuery(surveyOrderable, "id")
	q.OrderBy = "s." + q.OrderBy
	rows, total, err := s.repo.ListSurveys(ctx, q, strings.TrimSpace(req.Status))
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, func(r repository.SurveyRow) *v1.SurveyItem {
		return toSurveyItem(r, nil)
	}), total, q), nil
}

func (s *engagementService) CreateSurvey(ctx context.Context, in v1.SurveyInput) (*v1.SurveyItem, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tiêu đề khảo sát")
	}
	sType := strings.ToUpper(strings.TrimSpace(in.Type))
	if sType == "" {
		sType = entity.SurveyTypeNPS
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = consts.StatusActive
	}
	startAt := s.now()
	if strings.TrimSpace(in.StartAt) != "" {
		if t, err := parseFlexibleTime(in.StartAt, startAt); err == nil {
			startAt = t
		}
	}
	var endAt *time.Time
	if strings.TrimSpace(in.EndAt) != "" {
		if t, err := parseFlexibleTime(in.EndAt, startAt); err == nil {
			endAt = &t
		}
	}
	sv := &entity.Survey{
		Title:       title,
		Description: strings.TrimSpace(in.Description),
		Type:        sType,
		StartAt:     startAt,
		EndAt:       endAt,
		Status:      status,
	}
	if err := s.repo.CreateSurvey(ctx, sv); err != nil {
		return nil, err
	}
	row, err := s.repo.FindSurveyByID(ctx, sv.ID)
	if err != nil {
		row = &repository.SurveyRow{Survey: *sv}
	}
	res := toSurveyItem(*row, nil)
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntitySurvey, sv.ID, nil, res)
	}
	return res, nil
}

func (s *engagementService) UpdateSurvey(ctx context.Context, id uint, in v1.SurveyInput) (*v1.SurveyItem, error) {
	existing, err := s.repo.FindSurveyByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy khảo sát")
		}
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tiêu đề khảo sát")
	}
	sv := existing.Survey
	sv.Title = title
	sv.Description = strings.TrimSpace(in.Description)
	if strings.TrimSpace(in.Type) != "" {
		sv.Type = strings.ToUpper(strings.TrimSpace(in.Type))
	}
	if strings.TrimSpace(in.Status) != "" {
		sv.Status = strings.ToUpper(strings.TrimSpace(in.Status))
	}
	if strings.TrimSpace(in.StartAt) != "" {
		if t, err := parseFlexibleTime(in.StartAt, sv.StartAt); err == nil {
			sv.StartAt = t
		}
	}
	if strings.TrimSpace(in.EndAt) != "" {
		if t, err := parseFlexibleTime(in.EndAt, sv.StartAt); err == nil {
			sv.EndAt = &t
		}
	} else {
		sv.EndAt = nil
	}
	if err := s.repo.UpdateSurvey(ctx, &sv); err != nil {
		return nil, err
	}
	row, err := s.repo.FindSurveyByID(ctx, id)
	if err != nil {
		row = &repository.SurveyRow{Survey: sv}
	}
	return toSurveyItem(*row, nil), nil
}

func (s *engagementService) DeleteSurvey(ctx context.Context, id uint) error {
	existing, err := s.repo.FindSurveyByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy khảo sát")
		}
		return err
	}
	if err := s.repo.DeleteSurvey(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntitySurvey, id, toSurveyItem(*existing, nil), nil)
	}
	return nil
}

func (s *engagementService) NPSReport(ctx context.Context, req *v1.NPSReportReq) (*v1.NPSReportRes, error) {
	rows, err := s.repo.ListSurveyResponses(ctx, req.SurveyID)
	if err != nil {
		return nil, err
	}
	var (
		promoters  int64
		passives   int64
		detractors int64
		sumScore   int64
	)
	items := make([]*v1.SurveyResponseItem, 0, len(rows))
	for _, r := range rows {
		sumScore += int64(r.Score)
		switch {
		case r.Score >= 9:
			promoters++
		case r.Score >= 7:
			passives++
		default:
			detractors++
		}
		items = append(items, toSurveyResponseItem(r))
	}
	total := int64(len(rows))
	var nps, avg, promPct, passPct, detPct float64
	if total > 0 {
		promPct = math.Round((float64(promoters)/float64(total))*1000) / 10
		passPct = math.Round((float64(passives)/float64(total))*1000) / 10
		detPct = math.Round((float64(detractors)/float64(total))*1000) / 10
		nps = math.Round(((float64(promoters)-float64(detractors))/float64(total))*1000) / 10
		avg = math.Round((float64(sumScore)/float64(total))*100) / 100
	}
	recent := items
	if len(recent) > 50 {
		recent = recent[:50]
	}
	return &v1.NPSReportRes{
		NPSScore:        nps,
		AverageScore:    avg,
		TotalResponses:  total,
		Promoters:       promoters,
		Passives:        passives,
		Detractors:      detractors,
		PromoterPct:     promPct,
		PassivePct:      passPct,
		DetractorPct:    detPct,
		RecentResponses: recent,
		Responses:       recent,
	}, nil
}

// ---- Member Endpoints -----------------------------------------------------------

// nutritionActivityFactor: hệ số vận động mặc định (vận động vừa, 3–5 buổi/tuần).
const nutritionActivityFactor = 1.55

// nutritionDefaultTDEE: calo duy trì mặc định khi chưa có chỉ số cơ thể / kết quả bất thường.
const nutritionDefaultTDEE = 2000

// calculateMifflinBMR: BMR Mifflin-St Jeor (kcal/ngày); ok=false khi chưa có cân nặng/chiều cao.
// Nam: 10*weight + 6.25*height - 5*age + 5; Nữ: 10*weight + 6.25*height - 5*age - 161.
func calculateMifflinBMR(m *entity.Member, bm *entity.BodyMetric, now time.Time) (float64, bool) {
	if bm == nil || bm.WeightKg <= 0 || bm.HeightCm <= 0 {
		return 0, false
	}
	age := 25
	if m != nil && m.Dob != nil {
		if years := ageOnDate(*m.Dob, now); years >= 10 && years <= 100 {
			age = years
		}
	}
	bmr := 10*bm.WeightKg + 6.25*bm.HeightCm - 5*float64(age)
	if m != nil && strings.EqualFold(m.Gender, entity.MemberGenderFemale) {
		bmr -= 161
	} else {
		bmr += 5
	}
	return bmr, true
}

// calculateMifflinCalories tính calo duy trì (TDEE) = BMR Mifflin-St Jeor × 1.55 (vận động vừa);
// chưa có chỉ số hoặc kết quả < 1000 kcal → 2000.
func calculateMifflinCalories(m *entity.Member, bm *entity.BodyMetric, now time.Time) int {
	bmr, ok := calculateMifflinBMR(m, bm, now)
	if !ok {
		return nutritionDefaultTDEE
	}
	cal := int(math.Round(bmr * nutritionActivityFactor))
	if cal < 1000 {
		return nutritionDefaultTDEE
	}
	return cal
}

// Mục tiêu dinh dưỡng (NU-01 – PLAN.md Sprint 23: điều chỉnh calo theo mục tiêu giảm/tăng cân).
const (
	NutritionGoalCut      = "CUT"
	NutritionGoalMaintain = "MAINTAIN"
	NutritionGoalBulk     = "BULK"
)

// nutritionGoalRule: điều chỉnh calo so với TDEE + cách chia macro của từng mục tiêu.
// Giảm mỡ: -20% (thâm hụt vừa phải), đạm 2.0 g/kg giữ cơ; tăng cơ: +10% (dư nhẹ hạn chế tích mỡ), đạm 1.8 g/kg;
// duy trì: đạm 1.6 g/kg. Béo theo % năng lượng, carb là phần còn lại. Chưa có cân nặng → đạm theo % năng lượng.
type nutritionGoalRule struct {
	goal          string
	label         string
	adjustmentPct int
	proteinPerKg  float64
	proteinPct    float64 // dùng khi chưa có cân nặng
	fatPct        float64
}

var nutritionGoalRules = []nutritionGoalRule{
	{NutritionGoalCut, "Giảm mỡ", -20, 2.0, 0.30, 0.25},
	{NutritionGoalMaintain, "Duy trì", 0, 1.6, 0.25, 0.30},
	{NutritionGoalBulk, "Tăng cơ", 10, 1.8, 0.25, 0.25},
}

// nutritionMinCalories: sàn calo gợi ý (không khuyến nghị ăn dưới mức này khi không có chuyên gia theo dõi).
const nutritionMinCalories = 1200

// nutritionGoalTarget tính calo & macro (g/ngày) cho một mục tiêu từ TDEE và cân nặng (0 = chưa có).
func nutritionGoalTarget(r nutritionGoalRule, tdee int, weightKg float64) *v1.NutritionGoalTarget {
	cal := int(math.Round(float64(tdee) * float64(100+r.adjustmentPct) / 100))
	if cal < nutritionMinCalories {
		cal = nutritionMinCalories
	}
	protein := float64(cal) * r.proteinPct / 4
	if weightKg > 0 {
		protein = weightKg * r.proteinPerKg
	}
	fat := float64(cal) * r.fatPct / 9
	carbs := math.Max(0, (float64(cal)-protein*4-fat*9)/4)
	return &v1.NutritionGoalTarget{
		Goal: r.goal, Label: r.label, AdjustmentPct: r.adjustmentPct, Calories: cal,
		ProteinG: int(math.Round(protein)), CarbsG: int(math.Round(carbs)), FatG: int(math.Round(fat)),
	}
}

// parseNutritionGoal: CUT | MAINTAIN | BULK (không phân biệt hoa thường); rỗng → "".
func parseNutritionGoal(raw string) (string, error) {
	g := strings.ToUpper(strings.TrimSpace(raw))
	switch g {
	case "", NutritionGoalCut, NutritionGoalMaintain, NutritionGoalBulk:
		return g, nil
	}
	return "", rescode.New(rescode.BadRequest, "Mục tiêu dinh dưỡng phải là CUT (giảm mỡ), MAINTAIN (duy trì) hoặc BULK (tăng cơ)")
}

// inferNutritionGoal suy mục tiêu từ ô "Mục tiêu" (chữ tự do) của thực đơn HLV giao; không rõ → "".
// Vừa giảm vừa tăng (giảm mỡ tăng cơ – recomposition) → duy trì calo.
func inferNutritionGoal(text string) string {
	t := strings.ToLower(text)
	cut := strings.Contains(t, "giảm") || strings.Contains(t, "siết") || strings.Contains(t, "cut") || strings.Contains(t, "lose")
	bulk := strings.Contains(t, "tăng") || strings.Contains(t, "bulk") || strings.Contains(t, "gain")
	switch {
	case cut && bulk:
		return NutritionGoalMaintain
	case cut:
		return NutritionGoalCut
	case bulk:
		return NutritionGoalBulk
	case strings.Contains(t, "duy trì") || strings.Contains(t, "maintain"):
		return NutritionGoalMaintain
	}
	return ""
}

// buildNutritionTargets điền BMR/TDEE và gợi ý theo mục tiêu vào res (goal đã chuẩn hóa, rỗng = MAINTAIN).
func buildNutritionTargets(res *v1.MemberNutritionOverviewRes, m *entity.Member, bm *entity.BodyMetric, now time.Time, goal, source string) {
	tdee := calculateMifflinCalories(m, bm, now)
	bmr, ok := calculateMifflinBMR(m, bm, now)
	weight := 0.0
	if ok && bm != nil {
		weight = bm.WeightKg
	}
	if goal == "" {
		goal, source = NutritionGoalMaintain, "DEFAULT"
	}
	res.RecommendedCalories = tdee
	res.TDEE = tdee
	res.BMR = int(math.Round(math.Max(bmr, 0)))
	res.ActivityFactor = nutritionActivityFactor
	res.HasBodyMetric = ok
	res.Goal, res.GoalSource = goal, source
	res.GoalTargets = make([]*v1.NutritionGoalTarget, 0, len(nutritionGoalRules))
	for _, r := range nutritionGoalRules {
		t := nutritionGoalTarget(r, tdee, weight)
		res.GoalTargets = append(res.GoalTargets, t)
		if r.goal == goal {
			res.GoalTarget = t
		}
	}
}

// ageOnDate: số tuổi tròn tại ngày now (giờ Việt Nam) – chưa tới sinh nhật trong năm thì chưa cộng tuổi (NU-01).
// dob là cột DATE nên chỉ lấy năm/tháng/ngày.
func ageOnDate(dob, now time.Time) int {
	n := now.In(memberVNLoc)
	years := n.Year() - dob.Year()
	if n.Month() < dob.Month() || (n.Month() == dob.Month() && n.Day() < dob.Day()) {
		years--
	}
	return years
}

func (s *engagementService) MemberNutritionOverview(ctx context.Context, req *v1.MemberNutritionOverviewReq) (*v1.MemberNutritionOverviewRes, error) {
	goal, err := parseNutritionGoal(func() string {
		if req == nil {
			return ""
		}
		return req.Goal
	}())
	if err != nil {
		return nil, err
	}
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	bm, _ := s.repo.FindLatestBodyMetric(ctx, m.ID)

	planRows, err := s.repo.ListMemberMealPlans(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	planIDs := make([]uint, 0, len(planRows))
	for _, r := range planRows {
		planIDs = append(planIDs, r.ID)
	}
	itemsByPlan, err := s.repo.LoadMealPlanItems(ctx, planIDs)
	if err != nil {
		return nil, err
	}
	plans := make([]*v1.MealPlanDetail, 0, len(planRows))
	for _, r := range planRows {
		plans = append(plans, toMealPlanDetail(r, itemsByPlan[r.ID]))
	}
	foods, err := s.repo.ListAllFoods(ctx)
	if err != nil {
		return nil, err
	}
	source := "QUERY"
	if goal == "" {
		// Suy từ thực đơn đang áp dụng (ACTIVE) đầu tiên có ghi mục tiêu.
		for _, r := range planRows {
			if r.Status == consts.StatusActive {
				if g := inferNutritionGoal(r.Goal); g != "" {
					goal, source = g, "MEAL_PLAN"
					break
				}
			}
		}
	}
	res := &v1.MemberNutritionOverviewRes{
		Plans: plans,
		Foods: dto.Map(foods, toFoodItem),
	}
	buildNutritionTargets(res, m, bm, s.now(), goal, source)
	return res, nil
}

func (s *engagementService) MemberFoodLogs(ctx context.Context, req *v1.MemberFoodLogsReq) (*v1.MemberFoodLogsRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	date := s.now().In(memberVNLoc)
	if strings.TrimSpace(req.Date) != "" {
		parsed, err := time.ParseInLocation(consts.DateLayout, strings.TrimSpace(req.Date), memberVNLoc)
		if err != nil {
			return nil, rescode.New(rescode.BadRequest, "Ngày phải có định dạng YYYY-MM-DD")
		}
		date = parsed
	}
	rows, err := s.repo.ListFoodLogsByDate(ctx, m.ID, date)
	if err != nil {
		return nil, err
	}
	var totalCal, totalPro, totalCarb, totalFat float64
	logs := make([]*v1.FoodLogItem, 0, len(rows))
	for _, r := range rows {
		it := toFoodLogItem(r)
		totalCal += it.Calories
		totalPro += it.Protein
		totalCarb += it.Carbs
		totalFat += it.Fat
		logs = append(logs, it)
	}
	return &v1.MemberFoodLogsRes{
		Date:          date.Format(consts.DateLayout),
		TotalCalories: math.Round(totalCal*10) / 10,
		TotalProtein:  math.Round(totalPro*10) / 10,
		TotalCarbs:    math.Round(totalCarb*10) / 10,
		TotalFat:      math.Round(totalFat*10) / 10,
		Logs:          logs,
		Items:         logs,
	}, nil
}

func (s *engagementService) CreateFoodLog(ctx context.Context, req *v1.CreateFoodLogReq) (*v1.FoodLogItem, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	food, err := s.repo.FindFoodByID(ctx, req.FoodID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thực phẩm")
		}
		return nil, err
	}
	qty, err := normalizeFoodLogQty(req.Qty)
	if err != nil {
		return nil, err
	}
	meal := strings.ToUpper(strings.TrimSpace(req.Meal))
	if meal == "" {
		meal = entity.MealBreakfast
	}
	if !slices.Contains(validMeals, meal) {
		return nil, rescode.New(rescode.BadRequest, "Bữa ăn không hợp lệ")
	}
	eatenAt := s.now().In(memberVNLoc)
	if strings.TrimSpace(req.EatenAt) != "" {
		parsed, err := time.ParseInLocation(consts.DateLayout, strings.TrimSpace(req.EatenAt), memberVNLoc)
		if err != nil {
			return nil, rescode.New(rescode.BadRequest, "Ngày ăn phải có định dạng YYYY-MM-DD")
		}
		eatenAt = parsed
	}
	log := &entity.FoodLog{
		MemberID: m.ID,
		FoodID:   food.ID,
		Qty:      qty,
		Meal:     meal,
		EatenAt:  eatenAt,
	}
	if err := s.repo.CreateFoodLog(ctx, log); err != nil {
		return nil, err
	}
	return toFoodLogItem(repository.FoodLogRow{
		FoodLog:  *log,
		FoodName: food.Name,
		Unit:     food.Unit,
		Calories: food.Calories,
		Protein:  food.Protein,
		Carbs:    food.Carbs,
		Fat:      food.Fat,
	}), nil
}

// foodLogMaxQty: tối đa 100 khẩu phần mỗi bản ghi (món "100g" → 10 kg) – chặn số vô lý/tràn cột.
const foodLogMaxQty = 100

// normalizeFoodLogQty kiểm tra 0 < qty ≤ 100 và làm tròn 4 chữ số (cột numeric(10,4)).
func normalizeFoodLogQty(qty float64) (float64, error) {
	if math.IsNaN(qty) || math.IsInf(qty, 0) || qty <= 0 {
		return 0, rescode.New(rescode.BadRequest, "Khẩu phần phải lớn hơn 0")
	}
	if qty > foodLogMaxQty {
		return 0, rescode.New(rescode.BadRequest, "Khẩu phần tối đa 100 (VD: món 100g tối đa 10.000g)")
	}
	rounded := math.Round(qty*10000) / 10000
	if rounded <= 0 {
		return 0, rescode.New(rescode.BadRequest, "Khẩu phần phải lớn hơn 0")
	}
	return rounded, nil
}

// UpdateFoodLog sửa món/khẩu phần/bữa của bản ghi ăn uống do chính hội viên tạo.
func (s *engagementService) UpdateFoodLog(ctx context.Context, req *v1.UpdateFoodLogReq) (*v1.FoodLogItem, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	log, err := s.repo.FindFoodLogByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy bản ghi bữa ăn")
		}
		return nil, err
	}
	if log.MemberID != m.ID {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền sửa bản ghi này")
	}
	food, err := s.repo.FindFoodByID(ctx, req.FoodID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thực phẩm")
		}
		return nil, err
	}
	qty, err := normalizeFoodLogQty(req.Qty)
	if err != nil {
		return nil, err
	}
	meal := strings.ToUpper(strings.TrimSpace(req.Meal))
	if !slices.Contains(validMeals, meal) {
		return nil, rescode.New(rescode.BadRequest, "Bữa ăn không hợp lệ")
	}
	log.FoodID = food.ID
	log.Qty = qty
	log.Meal = meal
	if err := s.repo.UpdateFoodLog(ctx, log); err != nil {
		return nil, err
	}
	return toFoodLogItem(repository.FoodLogRow{
		FoodLog:  *log,
		FoodName: food.Name,
		Unit:     food.Unit,
		Calories: food.Calories,
		Protein:  food.Protein,
		Carbs:    food.Carbs,
		Fat:      food.Fat,
	}), nil
}

func (s *engagementService) DeleteFoodLog(ctx context.Context, id uint) error {
	m, err := s.currentMember(ctx)
	if err != nil {
		return err
	}
	log, err := s.repo.FindFoodLogByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy bản ghi bữa ăn")
		}
		return err
	}
	if log.MemberID != m.ID {
		return rescode.New(rescode.PermissionDenied, "Bạn không có quyền xóa bản ghi này")
	}
	return s.repo.DeleteFoodLog(ctx, id)
}

func (s *engagementService) MemberAnnouncements(ctx context.Context) (v1.MemberAnnouncementsRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListActiveAnnouncements(ctx, m.HomeBranchID, s.now())
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toAnnouncementItem), nil
}

func (s *engagementService) MemberSurveys(ctx context.Context) (v1.MemberSurveysRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListActiveSurveys(ctx, s.now())
	if err != nil {
		return nil, err
	}
	myResps, err := s.repo.ListMemberSurveyResponses(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	respBySurvey := make(map[uint]*entity.SurveyResponse, len(myResps))
	for i := range myResps {
		r := myResps[i]
		respBySurvey[r.SurveyID] = &r
	}
	out := make([]*v1.SurveyItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, toSurveyItem(r, respBySurvey[r.ID]))
	}
	return out, nil
}

func (s *engagementService) SubmitSurveyResponse(ctx context.Context, req *v1.SubmitSurveyResponseReq) (*v1.SurveyResponseItem, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	if req.Score < 0 || req.Score > 10 {
		return nil, rescode.New(rescode.BadRequest, "Điểm đánh giá NPS phải từ 0 đến 10")
	}
	sv, err := s.repo.FindSurveyByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy khảo sát")
		}
		return nil, err
	}
	if sv.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Khảo sát đã đóng")
	}
	ansBytes := []byte("{}")
	if req.Answers != nil {
		if b, err := json.Marshal(req.Answers); err == nil {
			ansBytes = b
		}
	}
	sr := &entity.SurveyResponse{
		SurveyID: sv.ID,
		MemberID: m.ID,
		Score:    req.Score,
		Comment:  strings.TrimSpace(req.Comment),
		Answers:  datatypes.JSON(ansBytes),
	}
	if err := s.repo.UpsertSurveyResponse(ctx, sr); err != nil {
		return nil, err
	}
	return toSurveyResponseItem(repository.SurveyResponseRow{
		SurveyResponse: *sr,
		SurveyTitle:    sv.Title,
		MemberCode:     m.MemberCode,
		MemberName:     m.FullName,
	}), nil
}

// ---- Mappers --------------------------------------------------------------------

func toFoodItem(f entity.Food) *v1.FoodItem {
	return &v1.FoodItem{
		ID:        f.ID,
		Name:      f.Name,
		Unit:      f.Unit,
		Calories:  f.Calories,
		Protein:   f.Protein,
		Carbs:     f.Carbs,
		Fat:       f.Fat,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}

func toMealPlanDetail(r repository.MealPlanRow, items []repository.MealPlanItemRow) *v1.MealPlanDetail {
	mapped := make([]*v1.MealPlanItemDetail, 0, len(items))
	for _, it := range items {
		mapped = append(mapped, &v1.MealPlanItemDetail{
			ID:       it.ID,
			PlanID:   it.PlanID,
			DayIndex: it.DayIndex,
			Meal:     it.Meal,
			FoodID:   it.FoodID,
			FoodName: it.FoodName,
			Unit:     it.Unit,
			Qty:      it.Qty,
			Calories: math.Round(it.Calories*it.Qty*10) / 10,
			Protein:  math.Round(it.Protein*it.Qty*10) / 10,
			Carbs:    math.Round(it.Carbs*it.Qty*10) / 10,
			Fat:      math.Round(it.Fat*it.Qty*10) / 10,
			Note:     it.Note,
		})
	}
	return &v1.MealPlanDetail{
		ID:             r.ID,
		MemberID:       r.MemberID,
		MemberCode:     r.MemberCode,
		MemberName:     r.MemberName,
		TrainerID:      r.TrainerID,
		TrainerName:    r.TrainerName,
		Name:           r.Name,
		Goal:           r.Goal,
		TargetCalories: r.TargetCalories,
		StartDate:      workoutFmtDateOpt(r.StartDate),
		EndDate:        workoutFmtDateOpt(r.EndDate),
		Status:         r.Status,
		Items:          mapped,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func toFoodLogItem(r repository.FoodLogRow) *v1.FoodLogItem {
	return &v1.FoodLogItem{
		ID:        r.ID,
		MemberID:  r.MemberID,
		FoodID:    r.FoodID,
		FoodName:  r.FoodName,
		Unit:      r.Unit,
		Qty:       r.Qty,
		Meal:      r.Meal,
		Calories:  math.Round(r.Calories*r.Qty*10) / 10,
		Protein:   math.Round(r.Protein*r.Qty*10) / 10,
		Carbs:     math.Round(r.Carbs*r.Qty*10) / 10,
		Fat:       math.Round(r.Fat*r.Qty*10) / 10,
		EatenAt:   r.EatenAt.Format(consts.DateLayout),
		CreatedAt: r.CreatedAt,
	}
}

func toAnnouncementItem(r repository.AnnouncementRow) *v1.AnnouncementItem {
	return &v1.AnnouncementItem{
		ID:          r.ID,
		BranchID:    r.BranchID,
		BranchName:  r.BranchName,
		Title:       r.Title,
		Body:        r.Body,
		ImageURL:    r.ImageURL,
		PublishAt:   r.PublishAt,
		ExpireAt:    r.ExpireAt,
		Pinned:      r.Pinned,
		CreatedBy:   r.CreatedBy,
		CreatorName: r.CreatorName,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func toSurveyItem(r repository.SurveyRow, myResp *entity.SurveyResponse) *v1.SurveyItem {
	item := &v1.SurveyItem{
		ID:            r.ID,
		Title:         r.Title,
		Description:   r.Description,
		Type:          r.Type,
		StartAt:       r.StartAt,
		EndAt:         r.EndAt,
		Status:        r.Status,
		ResponseCount: r.ResponseCount,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
	if myResp != nil {
		item.Responded = true
		sc := myResp.Score
		item.MyScore = &sc
		item.MyComment = myResp.Comment
	}
	return item
}

func toSurveyResponseItem(r repository.SurveyResponseRow) *v1.SurveyResponseItem {
	cat := "DETRACTOR"
	if r.Score >= 9 {
		cat = "PROMOTER"
	} else if r.Score >= 7 {
		cat = "PASSIVE"
	}
	return &v1.SurveyResponseItem{
		ID:          r.ID,
		SurveyID:    r.SurveyID,
		SurveyTitle: r.SurveyTitle,
		MemberID:    r.MemberID,
		MemberCode:  r.MemberCode,
		MemberName:  r.MemberName,
		Score:       r.Score,
		Category:    cat,
		Comment:     r.Comment,
		CreatedAt:   r.CreatedAt,
	}
}
