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
	WorkoutPlanRow struct {
		entity.WorkoutPlan
		MemberCode  string
		MemberName  string
		TrainerName string
	}

	WorkoutPlanItemRow struct {
		entity.WorkoutPlanItem
		ExerciseName string
		MuscleGroup  string
		Equipment    string
		VideoURL     string
	}

	WorkoutDayWithItems struct {
		Day   entity.WorkoutPlanDay
		Items []entity.WorkoutPlanItem
	}

	WorkoutPlanFilter struct {
		MemberID  uint
		TrainerID uint
		Status    string
		Search    string
	}

	WorkoutLogRow struct {
		entity.WorkoutLog
		ExerciseName string
		MuscleGroup  string
	}

	WorkoutRepository interface {
		ListExercises(ctx context.Context, q dto.Query, muscleGroup, search string) ([]entity.Exercise, int64, error)
		FindExerciseByID(ctx context.Context, id uint) (*entity.Exercise, error)
		ExistsExerciseName(ctx context.Context, name string, excludeID uint) (bool, error)
		CreateExercise(ctx context.Context, e *entity.Exercise) error
		UpdateExercise(ctx context.Context, e *entity.Exercise) error
		DeleteExercise(ctx context.Context, id uint, deletedBy *uint) error

		ListPlans(ctx context.Context, q dto.Query, f WorkoutPlanFilter) ([]WorkoutPlanRow, int64, error)
		FindPlanByID(ctx context.Context, id uint) (*WorkoutPlanRow, error)
		ListMemberPlans(ctx context.Context, memberID uint) ([]WorkoutPlanRow, error)
		CreatePlan(ctx context.Context, p *entity.WorkoutPlan) error
		UpdatePlan(ctx context.Context, p *entity.WorkoutPlan) error
		DeletePlan(ctx context.Context, id uint, deletedBy *uint) error
		ReplacePlanDays(ctx context.Context, planID uint, days []WorkoutDayWithItems) error
		LoadPlanDetails(ctx context.Context, planIDs []uint) (map[uint][]entity.WorkoutPlanDay, map[uint][]WorkoutPlanItemRow, error)

		FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error)
		FindMemberByID(ctx context.Context, id uint) (*entity.Member, error)
		HasPTContractWithTrainer(ctx context.Context, memberID, trainerID uint) (bool, error)

		CreateLog(ctx context.Context, l *entity.WorkoutLog) error
		ListMemberLogs(ctx context.Context, memberID, exerciseID uint, limit int) ([]WorkoutLogRow, error)
	}

	workoutRepository struct {
		db *gorm.DB
	}
)

func NewWorkoutRepository(db *gorm.DB) WorkoutRepository {
	return &workoutRepository{db: db}
}

func (r *workoutRepository) ListExercises(ctx context.Context, q dto.Query, muscleGroup, search string) ([]entity.Exercise, int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.Exercise{})
	if muscleGroup != "" {
		tx = tx.Where("muscle_group = ?", muscleGroup)
	}
	kw := strings.TrimSpace(search)
	if kw == "" {
		kw = q.Keyword
	}
	if kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("name ILIKE ? OR equipment ILIKE ? OR description ILIKE ?", like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entity.Exercise
	if err := q.Paginate(tx).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *workoutRepository) FindExerciseByID(ctx context.Context, id uint) (*entity.Exercise, error) {
	var e entity.Exercise
	if err := postgres.Conn(ctx, r.db).First(&e, id).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *workoutRepository) ExistsExerciseName(ctx context.Context, name string, excludeID uint) (bool, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.Exercise{}).Where("LOWER(name) = LOWER(?)", strings.TrimSpace(name))
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *workoutRepository) CreateExercise(ctx context.Context, e *entity.Exercise) error {
	return postgres.Conn(ctx, r.db).Create(e).Error
}

func (r *workoutRepository) UpdateExercise(ctx context.Context, e *entity.Exercise) error {
	return postgres.Conn(ctx, r.db).Save(e).Error
}

func (r *workoutRepository) DeleteExercise(ctx context.Context, id uint, deletedBy *uint) error {
	now := time.Now()
	return postgres.Conn(ctx, r.db).Model(&entity.Exercise{}).Where("id = ?", id).
		Updates(map[string]any{"deleted_at": now, "updated_by": deletedBy, "updated_at": now}).Error
}

const workoutPlanSelect = `wp.*, COALESCE(m.member_code, '') AS member_code,
	COALESCE(m.full_name, '') AS member_name, COALESCE(u.full_name, '') AS trainer_name`

const workoutPlanJoins = `LEFT JOIN members m ON m.id = wp.member_id
	LEFT JOIN users u ON u.id = wp.trainer_id`

func (r *workoutRepository) basePlan(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("workout_plans wp").
		Select(workoutPlanSelect).
		Joins(workoutPlanJoins).
		Where("wp.deleted_at IS NULL")
}

func (r *workoutRepository) ListPlans(ctx context.Context, q dto.Query, f WorkoutPlanFilter) ([]WorkoutPlanRow, int64, error) {
	tx := r.basePlan(ctx)
	if f.MemberID > 0 {
		tx = tx.Where("wp.member_id = ?", f.MemberID)
	}
	if f.TrainerID > 0 {
		tx = tx.Where("wp.trainer_id = ?", f.TrainerID)
	}
	if f.Status != "" {
		tx = tx.Where("wp.status = ?", f.Status)
	}
	kw := strings.TrimSpace(f.Search)
	if kw == "" {
		kw = q.Keyword
	}
	if kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("wp.name ILIKE ? OR wp.goal ILIKE ? OR m.full_name ILIKE ?", like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []WorkoutPlanRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *workoutRepository) FindPlanByID(ctx context.Context, id uint) (*WorkoutPlanRow, error) {
	var row WorkoutPlanRow
	res := r.basePlan(ctx).Where("wp.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *workoutRepository) ListMemberPlans(ctx context.Context, memberID uint) ([]WorkoutPlanRow, error) {
	var rows []WorkoutPlanRow
	err := r.basePlan(ctx).Where("wp.member_id = ?", memberID).Order("wp.id DESC").Scan(&rows).Error
	return rows, err
}

func (r *workoutRepository) CreatePlan(ctx context.Context, p *entity.WorkoutPlan) error {
	return postgres.Conn(ctx, r.db).Create(p).Error
}

func (r *workoutRepository) UpdatePlan(ctx context.Context, p *entity.WorkoutPlan) error {
	return postgres.Conn(ctx, r.db).Save(p).Error
}

func (r *workoutRepository) DeletePlan(ctx context.Context, id uint, deletedBy *uint) error {
	now := time.Now()
	return postgres.Conn(ctx, r.db).Model(&entity.WorkoutPlan{}).Where("id = ?", id).
		Updates(map[string]any{"deleted_at": now, "updated_by": deletedBy, "updated_at": now}).Error
}

func (r *workoutRepository) ReplacePlanDays(ctx context.Context, planID uint, days []WorkoutDayWithItems) error {
	conn := postgres.Conn(ctx, r.db)
	if err := conn.Where("plan_id = ?", planID).Delete(&entity.WorkoutPlanDay{}).Error; err != nil {
		return err
	}
	for _, d := range days {
		day := entity.WorkoutPlanDay{
			PlanID:   planID,
			DayIndex: d.Day.DayIndex,
			Title:    d.Day.Title,
		}
		if err := conn.Create(&day).Error; err != nil {
			return err
		}
		for i, item := range d.Items {
			row := item
			row.ID = 0
			row.DayID = day.ID
			if row.Sort == 0 {
				row.Sort = i + 1
			}
			if err := conn.Create(&row).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *workoutRepository) LoadPlanDetails(ctx context.Context, planIDs []uint) (map[uint][]entity.WorkoutPlanDay, map[uint][]WorkoutPlanItemRow, error) {
	daysByPlan := map[uint][]entity.WorkoutPlanDay{}
	itemsByDay := map[uint][]WorkoutPlanItemRow{}
	if len(planIDs) == 0 {
		return daysByPlan, itemsByDay, nil
	}
	conn := postgres.Conn(ctx, r.db)
	var days []entity.WorkoutPlanDay
	if err := conn.Where("plan_id IN ?", planIDs).Order("day_index ASC, id ASC").Find(&days).Error; err != nil {
		return nil, nil, err
	}
	dayIDs := make([]uint, 0, len(days))
	for _, d := range days {
		daysByPlan[d.PlanID] = append(daysByPlan[d.PlanID], d)
		dayIDs = append(dayIDs, d.ID)
	}
	if len(dayIDs) == 0 {
		return daysByPlan, itemsByDay, nil
	}
	var items []WorkoutPlanItemRow
	err := conn.Table("workout_plan_items wpi").
		Select("wpi.*, COALESCE(e.name, '') AS exercise_name, COALESCE(e.muscle_group, '') AS muscle_group, COALESCE(e.equipment, '') AS equipment, COALESCE(e.video_url, '') AS video_url").
		Joins("LEFT JOIN exercises e ON e.id = wpi.exercise_id").
		Where("wpi.day_id IN ?", dayIDs).
		Order("wpi.sort ASC, wpi.id ASC").
		Scan(&items).Error
	if err != nil {
		return nil, nil, err
	}
	for _, it := range items {
		itemsByDay[it.DayID] = append(itemsByDay[it.DayID], it)
	}
	return daysByPlan, itemsByDay, nil
}

func (r *workoutRepository) FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).Where("user_id = ?", userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *workoutRepository) FindMemberByID(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *workoutRepository) HasPTContractWithTrainer(ctx context.Context, memberID, trainerID uint) (bool, error) {
	var count int64
	err := postgres.Conn(ctx, r.db).Table("pt_contracts").
		Where("member_id = ? AND trainer_id = ? AND status IN ('ACTIVE', 'PENDING', 'COMPLETED')", memberID, trainerID).
		Count(&count).Error
	return count > 0, err
}

func (r *workoutRepository) CreateLog(ctx context.Context, l *entity.WorkoutLog) error {
	return postgres.Conn(ctx, r.db).Create(l).Error
}

func (r *workoutRepository) ListMemberLogs(ctx context.Context, memberID, exerciseID uint, limit int) ([]WorkoutLogRow, error) {
	tx := postgres.Conn(ctx, r.db).Table("workout_logs wl").
		Select("wl.*, COALESCE(e.name, '') AS exercise_name, COALESCE(e.muscle_group, '') AS muscle_group").
		Joins("LEFT JOIN exercises e ON e.id = wl.exercise_id").
		Where("wl.member_id = ?", memberID)
	if exerciseID > 0 {
		tx = tx.Where("wl.exercise_id = ?", exerciseID)
	}
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	var rows []WorkoutLogRow
	err := tx.Order("wl.performed_at DESC, wl.id DESC").Scan(&rows).Error
	return rows, err
}
