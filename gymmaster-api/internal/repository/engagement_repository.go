package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	MealPlanRow struct {
		entity.MealPlan
		MemberCode  string
		MemberName  string
		TrainerName string
	}

	MealPlanItemRow struct {
		entity.MealPlanItem
		FoodName string
		Unit     string
		Calories float64
		Protein  float64
		Carbs    float64
		Fat      float64
	}

	FoodLogRow struct {
		entity.FoodLog
		FoodName string
		Unit     string
		Calories float64
		Protein  float64
		Carbs    float64
		Fat      float64
	}

	AnnouncementRow struct {
		entity.Announcement
		BranchName  string
		CreatorName string
	}

	SurveyRow struct {
		entity.Survey
		ResponseCount int64
	}

	SurveyResponseRow struct {
		entity.SurveyResponse
		SurveyTitle string
		MemberCode  string
		MemberName  string
	}

	EngagementRepository interface {
		ListFoods(ctx context.Context, q dto.Query, search string) ([]entity.Food, int64, error)
		ListAllFoods(ctx context.Context) ([]entity.Food, error)
		FindFoodByID(ctx context.Context, id uint) (*entity.Food, error)
		ExistsFoodName(ctx context.Context, name string, excludeID uint) (bool, error)
		CreateFood(ctx context.Context, f *entity.Food) error
		UpdateFood(ctx context.Context, f *entity.Food) error
		DeleteFood(ctx context.Context, id uint) error

		ListMealPlans(ctx context.Context, q dto.Query, memberID, trainerID uint, status string) ([]MealPlanRow, int64, error)
		ListMemberMealPlans(ctx context.Context, memberID uint) ([]MealPlanRow, error)
		FindMealPlanByID(ctx context.Context, id uint) (*MealPlanRow, error)
		CreateMealPlan(ctx context.Context, p *entity.MealPlan) error
		UpdateMealPlan(ctx context.Context, p *entity.MealPlan) error
		DeleteMealPlan(ctx context.Context, id uint) error
		ReplaceMealPlanItems(ctx context.Context, planID uint, items []entity.MealPlanItem) error
		LoadMealPlanItems(ctx context.Context, planIDs []uint) (map[uint][]MealPlanItemRow, error)

		ListFoodLogsByDate(ctx context.Context, memberID uint, date time.Time) ([]FoodLogRow, error)
		FindFoodLogByID(ctx context.Context, id uint) (*entity.FoodLog, error)
		CreateFoodLog(ctx context.Context, l *entity.FoodLog) error
		DeleteFoodLog(ctx context.Context, id uint) error
		UpdateFoodLog(ctx context.Context, l *entity.FoodLog) error

		ListAnnouncements(ctx context.Context, q dto.Query, branchID uint) ([]AnnouncementRow, int64, error)
		ListActiveAnnouncements(ctx context.Context, branchID uint, now time.Time) ([]AnnouncementRow, error)
		FindAnnouncementByID(ctx context.Context, id uint) (*AnnouncementRow, error)
		CreateAnnouncement(ctx context.Context, a *entity.Announcement) error
		UpdateAnnouncement(ctx context.Context, a *entity.Announcement) error
		DeleteAnnouncement(ctx context.Context, id uint) error

		ListSurveys(ctx context.Context, q dto.Query, status string) ([]SurveyRow, int64, error)
		ListActiveSurveys(ctx context.Context, now time.Time) ([]SurveyRow, error)
		FindSurveyByID(ctx context.Context, id uint) (*SurveyRow, error)
		CreateSurvey(ctx context.Context, sv *entity.Survey) error
		UpdateSurvey(ctx context.Context, sv *entity.Survey) error
		DeleteSurvey(ctx context.Context, id uint) error

		ListMemberSurveyResponses(ctx context.Context, memberID uint) ([]entity.SurveyResponse, error)
		UpsertSurveyResponse(ctx context.Context, sr *entity.SurveyResponse) error
		ListSurveyResponses(ctx context.Context, surveyID uint) ([]SurveyResponseRow, error)

		FindMemberByID(ctx context.Context, id uint) (*entity.Member, error)
		FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error)
		FindLatestBodyMetric(ctx context.Context, memberID uint) (*entity.BodyMetric, error)
	}

	engagementRepository struct {
		db *gorm.DB
	}
)

func NewEngagementRepository(db *gorm.DB) EngagementRepository {
	return &engagementRepository{db: db}
}

// ---- Foods ----------------------------------------------------------------------

func (r *engagementRepository) ListFoods(ctx context.Context, q dto.Query, search string) ([]entity.Food, int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.Food{})
	kw := strings.TrimSpace(search)
	if kw == "" {
		kw = q.Keyword
	}
	if kw != "" {
		tx = tx.Where("name ILIKE ?", "%"+kw+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entity.Food
	if err := q.Paginate(tx).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *engagementRepository) ListAllFoods(ctx context.Context) ([]entity.Food, error) {
	var items []entity.Food
	err := postgres.Conn(ctx, r.db).Order("name ASC, id ASC").Find(&items).Error
	return items, err
}

func (r *engagementRepository) FindFoodByID(ctx context.Context, id uint) (*entity.Food, error) {
	var f entity.Food
	if err := postgres.Conn(ctx, r.db).First(&f, id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *engagementRepository) ExistsFoodName(ctx context.Context, name string, excludeID uint) (bool, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.Food{}).Where("LOWER(name) = LOWER(?)", strings.TrimSpace(name))
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	var count int64
	err := tx.Count(&count).Error
	return count > 0, err
}

func (r *engagementRepository) CreateFood(ctx context.Context, f *entity.Food) error {
	return postgres.Conn(ctx, r.db).Create(f).Error
}

func (r *engagementRepository) UpdateFood(ctx context.Context, f *entity.Food) error {
	return postgres.Conn(ctx, r.db).Save(f).Error
}

func (r *engagementRepository) DeleteFood(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.Food{}, id).Error
}

// ---- Meal Plans -----------------------------------------------------------------

const mealPlanSelect = `mp.*, COALESCE(m.member_code, '') AS member_code,
	COALESCE(m.full_name, '') AS member_name, COALESCE(u.full_name, '') AS trainer_name`

func (r *engagementRepository) baseMealPlan(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("meal_plans mp").
		Select(mealPlanSelect).
		Joins("LEFT JOIN members m ON m.id = mp.member_id").
		Joins("LEFT JOIN users u ON u.id = mp.trainer_id")
}

func (r *engagementRepository) ListMealPlans(ctx context.Context, q dto.Query, memberID, trainerID uint, status string) ([]MealPlanRow, int64, error) {
	tx := r.baseMealPlan(ctx)
	if memberID > 0 {
		tx = tx.Where("mp.member_id = ?", memberID)
	}
	if trainerID > 0 {
		tx = tx.Where("mp.trainer_id = ?", trainerID)
	}
	if status != "" {
		tx = tx.Where("mp.status = ?", status)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("mp.name ILIKE ? OR mp.goal ILIKE ? OR m.full_name ILIKE ?", like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []MealPlanRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *engagementRepository) ListMemberMealPlans(ctx context.Context, memberID uint) ([]MealPlanRow, error) {
	var rows []MealPlanRow
	err := r.baseMealPlan(ctx).Where("mp.member_id = ? AND mp.status = 'ACTIVE'", memberID).
		Order("mp.id DESC").Scan(&rows).Error
	return rows, err
}

func (r *engagementRepository) FindMealPlanByID(ctx context.Context, id uint) (*MealPlanRow, error) {
	var row MealPlanRow
	res := r.baseMealPlan(ctx).Where("mp.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *engagementRepository) CreateMealPlan(ctx context.Context, p *entity.MealPlan) error {
	return postgres.Conn(ctx, r.db).Create(p).Error
}

func (r *engagementRepository) UpdateMealPlan(ctx context.Context, p *entity.MealPlan) error {
	return postgres.Conn(ctx, r.db).Save(p).Error
}

func (r *engagementRepository) DeleteMealPlan(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.MealPlan{}, id).Error
}

func (r *engagementRepository) ReplaceMealPlanItems(ctx context.Context, planID uint, items []entity.MealPlanItem) error {
	conn := postgres.Conn(ctx, r.db)
	if err := conn.Where("plan_id = ?", planID).Delete(&entity.MealPlanItem{}).Error; err != nil {
		return err
	}
	for _, it := range items {
		row := it
		row.ID = 0
		row.PlanID = planID
		if err := conn.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *engagementRepository) LoadMealPlanItems(ctx context.Context, planIDs []uint) (map[uint][]MealPlanItemRow, error) {
	itemsByPlan := map[uint][]MealPlanItemRow{}
	if len(planIDs) == 0 {
		return itemsByPlan, nil
	}
	var rows []MealPlanItemRow
	err := postgres.Conn(ctx, r.db).Table("meal_plan_items mpi").
		Select("mpi.*, COALESCE(f.name, '') AS food_name, COALESCE(f.unit, '100g') AS unit, COALESCE(f.calories, 0) AS calories, COALESCE(f.protein, 0) AS protein, COALESCE(f.carbs, 0) AS carbs, COALESCE(f.fat, 0) AS fat").
		Joins("LEFT JOIN foods f ON f.id = mpi.food_id").
		Where("mpi.plan_id IN ?", planIDs).
		Order("mpi.day_index ASC, mpi.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		itemsByPlan[row.PlanID] = append(itemsByPlan[row.PlanID], row)
	}
	return itemsByPlan, nil
}

// ---- Food Logs ------------------------------------------------------------------

func (r *engagementRepository) ListFoodLogsByDate(ctx context.Context, memberID uint, date time.Time) ([]FoodLogRow, error) {
	var rows []FoodLogRow
	err := postgres.Conn(ctx, r.db).Table("food_logs fl").
		Select("fl.*, COALESCE(f.name, '') AS food_name, COALESCE(f.unit, '100g') AS unit, COALESCE(f.calories, 0) AS calories, COALESCE(f.protein, 0) AS protein, COALESCE(f.carbs, 0) AS carbs, COALESCE(f.fat, 0) AS fat").
		Joins("LEFT JOIN foods f ON f.id = fl.food_id").
		Where("fl.member_id = ? AND fl.eaten_at = ?", memberID, date.Format("2006-01-02")).
		Order("fl.id ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *engagementRepository) FindFoodLogByID(ctx context.Context, id uint) (*entity.FoodLog, error) {
	var l entity.FoodLog
	if err := postgres.Conn(ctx, r.db).First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *engagementRepository) CreateFoodLog(ctx context.Context, l *entity.FoodLog) error {
	return postgres.Conn(ctx, r.db).Create(l).Error
}

func (r *engagementRepository) UpdateFoodLog(ctx context.Context, l *entity.FoodLog) error {
	return postgres.Conn(ctx, r.db).Model(&entity.FoodLog{}).Where("id = ?", l.ID).
		Updates(map[string]any{"food_id": l.FoodID, "qty": l.Qty, "meal": l.Meal}).Error
}

func (r *engagementRepository) DeleteFoodLog(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.FoodLog{}, id).Error
}

// ---- Announcements --------------------------------------------------------------

const announcementSelect = `a.*, COALESCE(b.name, '') AS branch_name, COALESCE(u.full_name, '') AS creator_name`

func (r *engagementRepository) baseAnnouncement(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("announcements a").
		Select(announcementSelect).
		Joins("LEFT JOIN branches b ON b.id = a.branch_id").
		Joins("LEFT JOIN users u ON u.id = a.created_by")
}

func (r *engagementRepository) ListAnnouncements(ctx context.Context, q dto.Query, branchID uint) ([]AnnouncementRow, int64, error) {
	tx := r.baseAnnouncement(ctx)
	if branchID > 0 {
		tx = tx.Where("(a.branch_id IS NULL OR a.branch_id = ?)", branchID)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("a.title ILIKE ? OR a.body ILIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []AnnouncementRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *engagementRepository) ListActiveAnnouncements(ctx context.Context, branchID uint, now time.Time) ([]AnnouncementRow, error) {
	tx := r.baseAnnouncement(ctx).
		Where("a.publish_at <= ? AND (a.expire_at IS NULL OR a.expire_at >= ?)", now, now)
	if branchID > 0 {
		tx = tx.Where("(a.branch_id IS NULL OR a.branch_id = ?)", branchID)
	}
	var rows []AnnouncementRow
	err := tx.Order("a.pinned DESC, a.publish_at DESC, a.id DESC").Scan(&rows).Error
	return rows, err
}

func (r *engagementRepository) FindAnnouncementByID(ctx context.Context, id uint) (*AnnouncementRow, error) {
	var row AnnouncementRow
	res := r.baseAnnouncement(ctx).Where("a.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *engagementRepository) CreateAnnouncement(ctx context.Context, a *entity.Announcement) error {
	return postgres.Conn(ctx, r.db).Create(a).Error
}

func (r *engagementRepository) UpdateAnnouncement(ctx context.Context, a *entity.Announcement) error {
	return postgres.Conn(ctx, r.db).Save(a).Error
}

func (r *engagementRepository) DeleteAnnouncement(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.Announcement{}, id).Error
}

// ---- Surveys --------------------------------------------------------------------

const surveySelect = `s.*, (SELECT COUNT(*) FROM survey_responses sr WHERE sr.survey_id = s.id) AS response_count`

func (r *engagementRepository) baseSurvey(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("surveys s").Select(surveySelect)
}

func (r *engagementRepository) ListSurveys(ctx context.Context, q dto.Query, status string) ([]SurveyRow, int64, error) {
	tx := r.baseSurvey(ctx)
	if status != "" {
		tx = tx.Where("s.status = ?", status)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("s.title ILIKE ? OR s.description ILIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []SurveyRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *engagementRepository) ListActiveSurveys(ctx context.Context, now time.Time) ([]SurveyRow, error) {
	var rows []SurveyRow
	err := r.baseSurvey(ctx).
		Where("s.status = 'ACTIVE' AND s.start_at <= ? AND (s.end_at IS NULL OR s.end_at >= ?)", now, now).
		Order("s.id DESC").
		Scan(&rows).Error
	return rows, err
}

func (r *engagementRepository) FindSurveyByID(ctx context.Context, id uint) (*SurveyRow, error) {
	var row SurveyRow
	res := r.baseSurvey(ctx).Where("s.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *engagementRepository) CreateSurvey(ctx context.Context, sv *entity.Survey) error {
	return postgres.Conn(ctx, r.db).Create(sv).Error
}

func (r *engagementRepository) UpdateSurvey(ctx context.Context, sv *entity.Survey) error {
	return postgres.Conn(ctx, r.db).Save(sv).Error
}

func (r *engagementRepository) DeleteSurvey(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.Survey{}, id).Error
}

func (r *engagementRepository) ListMemberSurveyResponses(ctx context.Context, memberID uint) ([]entity.SurveyResponse, error) {
	var items []entity.SurveyResponse
	err := postgres.Conn(ctx, r.db).Where("member_id = ?", memberID).Find(&items).Error
	return items, err
}

func (r *engagementRepository) UpsertSurveyResponse(ctx context.Context, sr *entity.SurveyResponse) error {
	conn := postgres.Conn(ctx, r.db)
	var existing entity.SurveyResponse
	err := conn.Where("survey_id = ? AND member_id = ?", sr.SurveyID, sr.MemberID).First(&existing).Error
	if err == nil {
		existing.Score = sr.Score
		existing.Comment = sr.Comment
		existing.Answers = sr.Answers
		if err := conn.Save(&existing).Error; err != nil {
			return err
		}
		*sr = existing
		return nil
	}
	return conn.Create(sr).Error
}

func (r *engagementRepository) ListSurveyResponses(ctx context.Context, surveyID uint) ([]SurveyResponseRow, error) {
	tx := postgres.Conn(ctx, r.db).Table("survey_responses sr").
		Select("sr.*, COALESCE(s.title, '') AS survey_title, COALESCE(m.member_code, '') AS member_code, COALESCE(m.full_name, '') AS member_name").
		Joins("LEFT JOIN surveys s ON s.id = sr.survey_id").
		Joins("LEFT JOIN members m ON m.id = sr.member_id")
	if surveyID > 0 {
		tx = tx.Where("sr.survey_id = ?", surveyID)
	}
	var rows []SurveyResponseRow
	err := tx.Order("sr.created_at DESC, sr.id DESC").Scan(&rows).Error
	return rows, err
}

// ---- Members & Body Metrics -----------------------------------------------------

func (r *engagementRepository) FindMemberByID(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *engagementRepository) FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).Where("user_id = ?", userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *engagementRepository) FindLatestBodyMetric(ctx context.Context, memberID uint) (*entity.BodyMetric, error) {
	var bm entity.BodyMetric
	if err := postgres.Conn(ctx, r.db).
		Where("member_id = ?", memberID).
		Order("measured_at DESC, id DESC").
		First(&bm).Error; err != nil {
		return nil, err
	}
	return &bm, nil
}
