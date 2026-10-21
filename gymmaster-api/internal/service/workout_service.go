package service

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/workout/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	auditEntityExercise    = "exercise"
	auditEntityWorkoutPlan = "workout_plan"
)

var (
	validMuscleGroups = []string{
		entity.MuscleChest, entity.MuscleBack, entity.MuscleShoulders, entity.MuscleLegs,
		entity.MuscleArms, entity.MuscleCore, entity.MuscleCardio, entity.MuscleFullBody,
	}
	validWorkoutStatuses = []string{
		entity.WorkoutPlanDraft, entity.WorkoutPlanActive, entity.WorkoutPlanCompleted, entity.WorkoutPlanArchived,
	}
	exerciseOrderable    = []string{"id", "name", "muscle_group", "created_at"}
	workoutPlanOrderable = []string{"id", "name", "status", "start_date", "end_date", "created_at"}
)

type (
	WorkoutService interface {
		ListExercises(ctx context.Context, req *v1.ListExercisesReq) (*v1.ListExercisesRes, error)
		CreateExercise(ctx context.Context, in v1.ExerciseInput) (*v1.Exercise, error)
		UpdateExercise(ctx context.Context, id uint, in v1.ExerciseInput) (*v1.Exercise, error)
		DeleteExercise(ctx context.Context, id uint) error

		ListWorkoutPlans(ctx context.Context, req *v1.ListWorkoutPlansReq) (*v1.ListWorkoutPlansRes, error)
		GetWorkoutPlan(ctx context.Context, id uint) (*v1.WorkoutPlan, error)
		CreateWorkoutPlan(ctx context.Context, in v1.WorkoutPlanInput) (*v1.WorkoutPlan, error)
		UpdateWorkoutPlan(ctx context.Context, id uint, in v1.WorkoutPlanInput) (*v1.WorkoutPlan, error)
		DeleteWorkoutPlan(ctx context.Context, id uint) error
		CloneWorkoutPlan(ctx context.Context, req *v1.CloneWorkoutPlanReq) (*v1.WorkoutPlan, error)

		MemberExercises(ctx context.Context, req *v1.MemberExercisesReq) (v1.MemberExercisesRes, error)
		MemberWorkoutPlans(ctx context.Context) (v1.MemberWorkoutPlansRes, error)
		CreateWorkoutLog(ctx context.Context, req *v1.CreateWorkoutLogReq) (*v1.WorkoutLogItem, error)
		MemberWorkoutLogs(ctx context.Context, req *v1.MemberWorkoutLogsReq) (*v1.MemberWorkoutLogsRes, error)
	}

	workoutService struct {
		repo  repository.WorkoutRepository
		tx    postgres.TxManager
		audit AuditService
		now   func() time.Time
	}
)

func NewWorkoutService(repo repository.WorkoutRepository, tx postgres.TxManager, audit AuditService) WorkoutService {
	return &workoutService{repo: repo, tx: tx, audit: audit, now: time.Now}
}

func workoutTrainerOnly(ctx context.Context) (uint, bool) {
	auth := reqctx.Auth(ctx)
	if auth != nil && auth.HasRole(consts.RoleTrainer) && !auth.HasRole(consts.RoleOwner) && !auth.HasRole(consts.RoleManager) {
		return auth.UserID, true
	}
	return 0, false
}

func workoutParseDateOpt(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(consts.DateLayout, s)
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Ngày phải có định dạng YYYY-MM-DD")
	}
	return &t, nil
}

func workoutFmtDateOpt(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(consts.DateLayout)
	return &s
}

// ---- Bài tập (Exercise) ---------------------------------------------------------

func (s *workoutService) ListExercises(ctx context.Context, req *v1.ListExercisesReq) (*v1.ListExercisesRes, error) {
	q := req.PageReq.ToQuery(exerciseOrderable, "id")
	items, total, err := s.repo.ListExercises(ctx, q, strings.TrimSpace(req.MuscleGroup), req.Search)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(items, toExerciseRes), total, q), nil
}

func (s *workoutService) CreateExercise(ctx context.Context, in v1.ExerciseInput) (*v1.Exercise, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên bài tập")
	}
	mg := strings.ToUpper(strings.TrimSpace(in.MuscleGroup))
	if !slices.Contains(validMuscleGroups, mg) {
		return nil, rescode.New(rescode.BadRequest, "Nhóm cơ không hợp lệ")
	}
	dup, err := s.repo.ExistsExerciseName(ctx, name, 0)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, rescode.New(rescode.Conflict, "Tên bài tập đã tồn tại")
	}
	uid := reqctx.UserID(ctx)
	e := &entity.Exercise{
		BaseEntity:  entity.BaseEntity{CreatedBy: uid, UpdatedBy: uid},
		Name:        name,
		MuscleGroup: mg,
		Equipment:   strings.TrimSpace(in.Equipment),
		VideoURL:    strings.TrimSpace(in.VideoURL),
		Description: strings.TrimSpace(in.Description),
	}
	if err := s.repo.CreateExercise(ctx, e); err != nil {
		return nil, mapDuplicate(err, "Tên bài tập đã tồn tại")
	}
	res := toExerciseRes(*e)
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityExercise, e.ID, nil, res)
	}
	return res, nil
}

func (s *workoutService) UpdateExercise(ctx context.Context, id uint, in v1.ExerciseInput) (*v1.Exercise, error) {
	e, err := s.repo.FindExerciseByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy bài tập")
		}
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên bài tập")
	}
	mg := strings.ToUpper(strings.TrimSpace(in.MuscleGroup))
	if !slices.Contains(validMuscleGroups, mg) {
		return nil, rescode.New(rescode.BadRequest, "Nhóm cơ không hợp lệ")
	}
	dup, err := s.repo.ExistsExerciseName(ctx, name, id)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, rescode.New(rescode.Conflict, "Tên bài tập đã tồn tại")
	}
	before := toExerciseRes(*e)
	e.Name = name
	e.MuscleGroup = mg
	e.Equipment = strings.TrimSpace(in.Equipment)
	e.VideoURL = strings.TrimSpace(in.VideoURL)
	e.Description = strings.TrimSpace(in.Description)
	e.UpdatedBy = reqctx.UserID(ctx)
	if err := s.repo.UpdateExercise(ctx, e); err != nil {
		return nil, mapDuplicate(err, "Tên bài tập đã tồn tại")
	}
	res := toExerciseRes(*e)
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityExercise, e.ID, before, res)
	}
	return res, nil
}

func (s *workoutService) DeleteExercise(ctx context.Context, id uint) error {
	e, err := s.repo.FindExerciseByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy bài tập")
		}
		return err
	}
	if err := s.repo.DeleteExercise(ctx, id, reqctx.UserID(ctx)); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityExercise, id, toExerciseRes(*e), nil)
	}
	return nil
}

func toExerciseRes(e entity.Exercise) *v1.Exercise {
	return &v1.Exercise{
		ID:          e.ID,
		Name:        e.Name,
		MuscleGroup: e.MuscleGroup,
		Equipment:   e.Equipment,
		VideoURL:    e.VideoURL,
		Description: e.Description,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// ---- Giáo án tập luyện (WorkoutPlan) --------------------------------------------

func (s *workoutService) ListWorkoutPlans(ctx context.Context, req *v1.ListWorkoutPlansReq) (*v1.ListWorkoutPlansRes, error) {
	q := req.PageReq.ToQuery(workoutPlanOrderable, "id")
	q.OrderBy = "wp." + q.OrderBy
	rows, total, err := s.repo.ListPlans(ctx, q, repository.WorkoutPlanFilter{
		MemberID:  req.MemberID,
		TrainerID: req.TrainerID,
		Status:    strings.TrimSpace(req.Status),
		Search:    req.Search,
	})
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, func(r repository.WorkoutPlanRow) *v1.WorkoutPlan {
		return toWorkoutPlanRes(r, nil, nil)
	}), total, q), nil
}

func (s *workoutService) GetWorkoutPlan(ctx context.Context, id uint) (*v1.WorkoutPlan, error) {
	row, err := s.repo.FindPlanByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy giáo án tập luyện")
		}
		return nil, err
	}
	daysByPlan, itemsByDay, err := s.repo.LoadPlanDetails(ctx, []uint{id})
	if err != nil {
		return nil, err
	}
	return toWorkoutPlanRes(*row, daysByPlan[id], itemsByDay), nil
}

func (s *workoutService) preparePlanDays(ctx context.Context, inDays []v1.WorkoutDayInput) ([]repository.WorkoutDayWithItems, error) {
	days := make([]repository.WorkoutDayWithItems, 0, len(inDays))
	for _, d := range inDays {
		if d.DayIndex < 1 || d.DayIndex > 14 {
			return nil, rescode.New(rescode.BadRequest, "Thứ tự ngày trong giáo án phải từ 1 đến 14")
		}
		title := strings.TrimSpace(d.Title)
		if title == "" {
			return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tiêu đề buổi tập")
		}
		items := make([]entity.WorkoutPlanItem, 0, len(d.Items))
		for i, it := range d.Items {
			if it.ExerciseID == 0 {
				return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn bài tập")
			}
			if _, err := s.repo.FindExerciseByID(ctx, it.ExerciseID); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, rescode.New(rescode.BadRequest, "Bài tập trong giáo án không tồn tại")
				}
				return nil, err
			}
			sets := it.Sets
			if sets <= 0 {
				sets = 3
			}
			reps := it.Reps
			if reps <= 0 {
				reps = 10
			}
			restSec := it.RestSec
			if restSec < 0 {
				restSec = 0
			}
			if it.Weight < 0 {
				return nil, rescode.New(rescode.BadRequest, "Mức tạ không được âm")
			}
			sortIdx := it.Sort
			if sortIdx == 0 {
				sortIdx = i + 1
			}
			items = append(items, entity.WorkoutPlanItem{
				ExerciseID: it.ExerciseID,
				Sets:       sets,
				Reps:       reps,
				Weight:     it.Weight,
				RestSec:    restSec,
				Note:       strings.TrimSpace(it.Note),
				Sort:       sortIdx,
			})
		}
		days = append(days, repository.WorkoutDayWithItems{
			Day:   entity.WorkoutPlanDay{DayIndex: d.DayIndex, Title: title},
			Items: items,
		})
	}
	return days, nil
}

func (s *workoutService) CreateWorkoutPlan(ctx context.Context, in v1.WorkoutPlanInput) (*v1.WorkoutPlan, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên giáo án")
	}
	trainerID := in.TrainerID
	if tUID, isTrainer := workoutTrainerOnly(ctx); isTrainer {
		if trainerID != 0 && trainerID != tUID {
			return nil, rescode.New(rescode.PermissionDenied, "HLV chỉ được tạo giáo án cho chính mình")
		}
		trainerID = tUID
		if in.MemberID != nil && *in.MemberID > 0 {
			ok, err := s.repo.HasPTContractWithTrainer(ctx, *in.MemberID, tUID)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, rescode.New(rescode.PermissionDenied, "HLV chỉ được tạo giáo án cho học viên có hợp đồng PT với mình")
			}
		}
	} else if trainerID == 0 {
		if uid := reqctx.UserID(ctx); uid != nil {
			trainerID = *uid
		}
	}
	if trainerID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn huấn luyện viên")
	}
	if in.MemberID != nil && *in.MemberID > 0 {
		if _, err := s.repo.FindMemberByID(ctx, *in.MemberID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, rescode.New(rescode.BadRequest, "Hội viên không tồn tại")
			}
			return nil, err
		}
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
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = entity.WorkoutPlanActive
	}
	if !slices.Contains(validWorkoutStatuses, status) {
		return nil, rescode.New(rescode.BadRequest, "Trạng thái giáo án không hợp lệ")
	}
	days, err := s.preparePlanDays(ctx, in.Days)
	if err != nil {
		return nil, err
	}

	uid := reqctx.UserID(ctx)
	plan := &entity.WorkoutPlan{
		BaseEntity:   entity.BaseEntity{CreatedBy: uid, UpdatedBy: uid},
		MemberID:     in.MemberID,
		TrainerID:    trainerID,
		PTContractID: in.PTContractID,
		Name:         name,
		Goal:         strings.TrimSpace(in.Goal),
		StartDate:    startDate,
		EndDate:      endDate,
		Status:       status,
	}

	run := func(txCtx context.Context) error {
		if err := s.repo.CreatePlan(txCtx, plan); err != nil {
			return err
		}
		return s.repo.ReplacePlanDays(txCtx, plan.ID, days)
	}
	if s.tx != nil {
		err = s.tx.WithTx(ctx, run)
	} else {
		err = run(ctx)
	}
	if err != nil {
		return nil, err
	}
	res, err := s.GetWorkoutPlan(ctx, plan.ID)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityWorkoutPlan, plan.ID, nil, res)
	}
	return res, nil
}

func (s *workoutService) UpdateWorkoutPlan(ctx context.Context, id uint, in v1.WorkoutPlanInput) (*v1.WorkoutPlan, error) {
	existing, err := s.repo.FindPlanByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy giáo án tập luyện")
		}
		return nil, err
	}
	trainerID := in.TrainerID
	if tUID, isTrainer := workoutTrainerOnly(ctx); isTrainer {
		if existing.TrainerID != tUID {
			return nil, rescode.New(rescode.PermissionDenied, "Bạn chỉ được chỉnh sửa giáo án của mình")
		}
		if trainerID != 0 && trainerID != tUID {
			return nil, rescode.New(rescode.PermissionDenied, "HLV không được chuyển giáo án sang HLV khác")
		}
		trainerID = tUID
		if in.MemberID != nil && *in.MemberID > 0 {
			ok, err := s.repo.HasPTContractWithTrainer(ctx, *in.MemberID, tUID)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, rescode.New(rescode.PermissionDenied, "HLV chỉ được gán giáo án cho học viên có hợp đồng PT với mình")
			}
		}
	} else if trainerID == 0 {
		trainerID = existing.TrainerID
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên giáo án")
	}
	if in.MemberID != nil && *in.MemberID > 0 {
		if _, err := s.repo.FindMemberByID(ctx, *in.MemberID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, rescode.New(rescode.BadRequest, "Hội viên không tồn tại")
			}
			return nil, err
		}
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
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = existing.Status
	}
	if !slices.Contains(validWorkoutStatuses, status) {
		return nil, rescode.New(rescode.BadRequest, "Trạng thái giáo án không hợp lệ")
	}
	days, err := s.preparePlanDays(ctx, in.Days)
	if err != nil {
		return nil, err
	}

	plan := existing.WorkoutPlan
	plan.MemberID = in.MemberID
	plan.TrainerID = trainerID
	plan.PTContractID = in.PTContractID
	plan.Name = name
	plan.Goal = strings.TrimSpace(in.Goal)
	plan.StartDate = startDate
	plan.EndDate = endDate
	plan.Status = status
	plan.UpdatedBy = reqctx.UserID(ctx)

	run := func(txCtx context.Context) error {
		if err := s.repo.UpdatePlan(txCtx, &plan); err != nil {
			return err
		}
		return s.repo.ReplacePlanDays(txCtx, plan.ID, days)
	}
	if s.tx != nil {
		err = s.tx.WithTx(ctx, run)
	} else {
		err = run(ctx)
	}
	if err != nil {
		return nil, err
	}
	return s.GetWorkoutPlan(ctx, id)
}

func (s *workoutService) DeleteWorkoutPlan(ctx context.Context, id uint) error {
	existing, err := s.repo.FindPlanByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy giáo án tập luyện")
		}
		return err
	}
	if tUID, isTrainer := workoutTrainerOnly(ctx); isTrainer {
		if existing.TrainerID != tUID {
			return rescode.New(rescode.PermissionDenied, "Bạn chỉ được xóa giáo án của mình")
		}
	}
	if err := s.repo.DeletePlan(ctx, id, reqctx.UserID(ctx)); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityWorkoutPlan, id, existing, nil)
	}
	return nil
}

func (s *workoutService) CloneWorkoutPlan(ctx context.Context, req *v1.CloneWorkoutPlanReq) (*v1.WorkoutPlan, error) {
	src, err := s.repo.FindPlanByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy giáo án gốc")
		}
		return nil, err
	}
	trainerID := src.TrainerID
	if tUID, isTrainer := workoutTrainerOnly(ctx); isTrainer {
		if src.MemberID != nil && src.TrainerID != tUID {
			return nil, rescode.New(rescode.PermissionDenied, "Bạn chỉ được nhân bản giáo án mẫu hoặc giáo án của mình")
		}
		ok, err := s.repo.HasPTContractWithTrainer(ctx, req.MemberID, tUID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, rescode.New(rescode.PermissionDenied, "HLV chỉ được nhân bản giáo án cho học viên có hợp đồng PT với mình")
		}
		trainerID = tUID
	}
	if _, err := s.repo.FindMemberByID(ctx, req.MemberID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.BadRequest, "Hội viên không tồn tại")
		}
		return nil, err
	}
	startDate, err := workoutParseDateOpt(req.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := workoutParseDateOpt(req.EndDate)
	if err != nil {
		return nil, err
	}
	if startDate != nil && endDate != nil && endDate.Before(*startDate) {
		return nil, rescode.New(rescode.BadRequest, "Ngày kết thúc phải sau hoặc bằng ngày bắt đầu")
	}

	daysByPlan, itemsByDay, err := s.repo.LoadPlanDetails(ctx, []uint{src.ID})
	if err != nil {
		return nil, err
	}
	srcDays := daysByPlan[src.ID]
	clonedDays := make([]repository.WorkoutDayWithItems, 0, len(srcDays))
	for _, d := range srcDays {
		srcItems := itemsByDay[d.ID]
		items := make([]entity.WorkoutPlanItem, 0, len(srcItems))
		for _, it := range srcItems {
			items = append(items, entity.WorkoutPlanItem{
				ExerciseID: it.ExerciseID,
				Sets:       it.Sets,
				Reps:       it.Reps,
				Weight:     it.Weight,
				RestSec:    it.RestSec,
				Note:       it.Note,
				Sort:       it.Sort,
			})
		}
		clonedDays = append(clonedDays, repository.WorkoutDayWithItems{
			Day:   entity.WorkoutPlanDay{DayIndex: d.DayIndex, Title: d.Title},
			Items: items,
		})
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = src.Name
	}
	uid := reqctx.UserID(ctx)
	memberID := req.MemberID
	cloned := &entity.WorkoutPlan{
		BaseEntity: entity.BaseEntity{CreatedBy: uid, UpdatedBy: uid},
		MemberID:   &memberID,
		TrainerID:  trainerID,
		Name:       name,
		Goal:       src.Goal,
		StartDate:  startDate,
		EndDate:    endDate,
		Status:     entity.WorkoutPlanActive,
	}

	run := func(txCtx context.Context) error {
		if err := s.repo.CreatePlan(txCtx, cloned); err != nil {
			return err
		}
		return s.repo.ReplacePlanDays(txCtx, cloned.ID, clonedDays)
	}
	if s.tx != nil {
		err = s.tx.WithTx(ctx, run)
	} else {
		err = run(ctx)
	}
	if err != nil {
		return nil, err
	}
	return s.GetWorkoutPlan(ctx, cloned.ID)
}

func toWorkoutPlanRes(r repository.WorkoutPlanRow, days []entity.WorkoutPlanDay, itemsByDay map[uint][]repository.WorkoutPlanItemRow) *v1.WorkoutPlan {
	res := &v1.WorkoutPlan{
		ID:           r.ID,
		MemberID:     r.MemberID,
		MemberCode:   r.MemberCode,
		MemberName:   r.MemberName,
		TrainerID:    r.TrainerID,
		TrainerName:  r.TrainerName,
		PTContractID: r.PTContractID,
		Name:         r.Name,
		Goal:         r.Goal,
		StartDate:    workoutFmtDateOpt(r.StartDate),
		EndDate:      workoutFmtDateOpt(r.EndDate),
		Status:       r.Status,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
	if days != nil {
		res.Days = make([]*v1.WorkoutPlanDay, 0, len(days))
		res.Items = []*v1.WorkoutPlanItem{}
		for _, d := range days {
			dayItems := itemsByDay[d.ID]
			mappedItems := make([]*v1.WorkoutPlanItem, 0, len(dayItems))
			for _, it := range dayItems {
				m := &v1.WorkoutPlanItem{
					ID:           it.ID,
					DayID:        it.DayID,
					ExerciseID:   it.ExerciseID,
					ExerciseName: it.ExerciseName,
					MuscleGroup:  it.MuscleGroup,
					Equipment:    it.Equipment,
					VideoURL:     it.VideoURL,
					Sets:         it.Sets,
					Reps:         it.Reps,
					Weight:       it.Weight,
					RestSec:      it.RestSec,
					Note:         it.Note,
					Sort:         it.Sort,
				}
				mappedItems = append(mappedItems, m)
				res.Items = append(res.Items, m)
			}
			res.Days = append(res.Days, &v1.WorkoutPlanDay{
				ID:       d.ID,
				PlanID:   d.PlanID,
				DayIndex: d.DayIndex,
				Title:    d.Title,
				Items:    mappedItems,
			})
		}
	}
	return res
}

// ---- Cổng hội viên (Member) -----------------------------------------------------

func (s *workoutService) currentMember(ctx context.Context) (*entity.Member, error) {
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

func (s *workoutService) MemberExercises(ctx context.Context, req *v1.MemberExercisesReq) (v1.MemberExercisesRes, error) {
	q := dto.Query{Page: 1, Limit: dto.MaxLimit, OrderBy: "name", OrderDir: "ASC"}
	items, _, err := s.repo.ListExercises(ctx, q, strings.TrimSpace(req.MuscleGroup), req.Search)
	if err != nil {
		return nil, err
	}
	return dto.Map(items, toExerciseRes), nil
}

func (s *workoutService) MemberWorkoutPlans(ctx context.Context) (v1.MemberWorkoutPlansRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListMemberPlans(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	planIDs := make([]uint, 0, len(rows))
	for _, r := range rows {
		planIDs = append(planIDs, r.ID)
	}
	daysByPlan, itemsByDay, err := s.repo.LoadPlanDetails(ctx, planIDs)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.WorkoutPlan, 0, len(rows))
	for _, r := range rows {
		days := daysByPlan[r.ID]
		if days == nil {
			days = []entity.WorkoutPlanDay{}
		}
		out = append(out, toWorkoutPlanRes(r, days, itemsByDay))
	}
	return out, nil
}

func workoutParsePerformedAt(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return now, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04", s, memberVNLoc); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation(consts.DateLayout, s, memberVNLoc); err == nil {
		return t, nil
	}
	return time.Time{}, rescode.New(rescode.BadRequest, "Thời gian tập luyện không hợp lệ")
}

func (s *workoutService) memberOwnsPlanItem(ctx context.Context, memberID, planItemID uint) (bool, error) {
	plans, err := s.repo.ListMemberPlans(ctx, memberID)
	if err != nil || len(plans) == 0 {
		return false, err
	}
	planIDs := make([]uint, 0, len(plans))
	for _, p := range plans {
		planIDs = append(planIDs, p.ID)
	}
	_, itemsByDay, err := s.repo.LoadPlanDetails(ctx, planIDs)
	if err != nil {
		return false, err
	}
	for _, items := range itemsByDay {
		for _, it := range items {
			if it.ID == planItemID {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *workoutService) CreateWorkoutLog(ctx context.Context, req *v1.CreateWorkoutLogReq) (*v1.WorkoutLogItem, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	if req.Sets <= 0 || req.Reps <= 0 || req.Weight < 0 {
		return nil, rescode.New(rescode.BadRequest, "Thông số hiệp, lần lặp và mức tạ không hợp lệ")
	}
	if req.PlanItemID != nil && *req.PlanItemID > 0 {
		owned, err := s.memberOwnsPlanItem(ctx, m.ID, *req.PlanItemID)
		if err != nil {
			return nil, err
		}
		if !owned {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy mục bài tập trong giáo án của bạn")
		}
	}
	ex, err := s.repo.FindExerciseByID(ctx, req.ExerciseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy bài tập")
		}
		return nil, err
	}
	performedAt, err := workoutParsePerformedAt(req.PerformedAt, s.now())
	if err != nil {
		return nil, err
	}
	log := &entity.WorkoutLog{
		MemberID:    m.ID,
		PlanItemID:  req.PlanItemID,
		ExerciseID:  req.ExerciseID,
		PerformedAt: performedAt,
		Sets:        req.Sets,
		Reps:        req.Reps,
		Weight:      req.Weight,
		Note:        strings.TrimSpace(req.Note),
	}
	if err := s.repo.CreateLog(ctx, log); err != nil {
		return nil, err
	}
	return toWorkoutLogItem(repository.WorkoutLogRow{
		WorkoutLog:   *log,
		ExerciseName: ex.Name,
		MuscleGroup:  ex.MuscleGroup,
	}), nil
}

func (s *workoutService) MemberWorkoutLogs(ctx context.Context, req *v1.MemberWorkoutLogsReq) (*v1.MemberWorkoutLogsRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}
	rows, err := s.repo.ListMemberLogs(ctx, m.ID, req.ExerciseID, limit)
	if err != nil {
		return nil, err
	}
	logs := dto.Map(rows, toWorkoutLogItem)
	byDate := map[string]*v1.WorkoutProgressPoint{}
	for _, l := range rows {
		d := l.PerformedAt.In(memberVNLoc).Format(consts.DateLayout)
		pt, ok := byDate[d]
		if !ok {
			pt = &v1.WorkoutProgressPoint{Date: d}
			byDate[d] = pt
		}
		if l.Weight > pt.MaxWeight {
			pt.MaxWeight = l.Weight
		}
		pt.TotalVolume += float64(l.Sets*l.Reps) * l.Weight
		pt.TotalSets += l.Sets
	}
	progress := make([]*v1.WorkoutProgressPoint, 0, len(byDate))
	for _, pt := range byDate {
		progress = append(progress, pt)
	}
	sort.Slice(progress, func(i, j int) bool { return progress[i].Date < progress[j].Date })
	return &v1.MemberWorkoutLogsRes{
		Logs:     logs,
		Progress: progress,
	}, nil
}

func toWorkoutLogItem(r repository.WorkoutLogRow) *v1.WorkoutLogItem {
	return &v1.WorkoutLogItem{
		ID:           r.ID,
		MemberID:     r.MemberID,
		PlanItemID:   r.PlanItemID,
		ExerciseID:   r.ExerciseID,
		ExerciseName: r.ExerciseName,
		MuscleGroup:  r.MuscleGroup,
		PerformedAt:  r.PerformedAt,
		Sets:         r.Sets,
		Reps:         r.Reps,
		Weight:       r.Weight,
		Volume:       float64(r.Sets*r.Reps) * r.Weight,
		Note:         r.Note,
		CreatedAt:    r.CreatedAt,
	}
}
