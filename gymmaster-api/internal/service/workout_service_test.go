package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/workout/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

type fakeWorkoutRepo struct {
	nextExID   uint
	nextPlanID uint
	nextDayID  uint
	nextItemID uint
	nextLogID  uint

	exercises   map[uint]*entity.Exercise
	plans       map[uint]*entity.WorkoutPlan
	days        map[uint][]entity.WorkoutPlanDay
	items       map[uint][]entity.WorkoutPlanItem
	members     map[uint]*entity.Member
	ptContracts map[[2]uint]bool // [memberID, trainerID]
	logs        []entity.WorkoutLog
}

func newFakeWorkoutRepo() *fakeWorkoutRepo {
	return &fakeWorkoutRepo{
		exercises:   map[uint]*entity.Exercise{},
		plans:       map[uint]*entity.WorkoutPlan{},
		days:        map[uint][]entity.WorkoutPlanDay{},
		items:       map[uint][]entity.WorkoutPlanItem{},
		members:     map[uint]*entity.Member{},
		ptContracts: map[[2]uint]bool{},
	}
}

func (f *fakeWorkoutRepo) ListExercises(_ context.Context, _ dto.Query, muscleGroup, search string) ([]entity.Exercise, int64, error) {
	var out []entity.Exercise
	for _, e := range f.exercises {
		if muscleGroup != "" && e.MuscleGroup != muscleGroup {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(e.Name), strings.ToLower(search)) {
			continue
		}
		out = append(out, *e)
	}
	return out, int64(len(out)), nil
}

func (f *fakeWorkoutRepo) FindExerciseByID(_ context.Context, id uint) (*entity.Exercise, error) {
	if e, ok := f.exercises[id]; ok {
		cp := *e
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeWorkoutRepo) ExistsExerciseName(_ context.Context, name string, excludeID uint) (bool, error) {
	for _, e := range f.exercises {
		if strings.EqualFold(e.Name, strings.TrimSpace(name)) && e.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeWorkoutRepo) CreateExercise(_ context.Context, e *entity.Exercise) error {
	f.nextExID++
	e.ID = f.nextExID
	cp := *e
	f.exercises[e.ID] = &cp
	return nil
}

func (f *fakeWorkoutRepo) UpdateExercise(_ context.Context, e *entity.Exercise) error {
	cp := *e
	f.exercises[e.ID] = &cp
	return nil
}

func (f *fakeWorkoutRepo) DeleteExercise(_ context.Context, id uint, _ *uint) error {
	delete(f.exercises, id)
	return nil
}

func (f *fakeWorkoutRepo) ListPlans(_ context.Context, _ dto.Query, filter repository.WorkoutPlanFilter) ([]repository.WorkoutPlanRow, int64, error) {
	var out []repository.WorkoutPlanRow
	for _, p := range f.plans {
		if filter.MemberID > 0 && (p.MemberID == nil || *p.MemberID != filter.MemberID) {
			continue
		}
		if filter.TrainerID > 0 && p.TrainerID != filter.TrainerID {
			continue
		}
		if filter.Status != "" && p.Status != filter.Status {
			continue
		}
		out = append(out, repository.WorkoutPlanRow{WorkoutPlan: *p})
	}
	return out, int64(len(out)), nil
}

func (f *fakeWorkoutRepo) FindPlanByID(_ context.Context, id uint) (*repository.WorkoutPlanRow, error) {
	if p, ok := f.plans[id]; ok {
		return &repository.WorkoutPlanRow{WorkoutPlan: *p}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeWorkoutRepo) ListMemberPlans(_ context.Context, memberID uint) ([]repository.WorkoutPlanRow, error) {
	var out []repository.WorkoutPlanRow
	for _, p := range f.plans {
		if p.MemberID != nil && *p.MemberID == memberID {
			out = append(out, repository.WorkoutPlanRow{WorkoutPlan: *p})
		}
	}
	return out, nil
}

func (f *fakeWorkoutRepo) CreatePlan(_ context.Context, p *entity.WorkoutPlan) error {
	f.nextPlanID++
	p.ID = f.nextPlanID
	cp := *p
	f.plans[p.ID] = &cp
	return nil
}

func (f *fakeWorkoutRepo) UpdatePlan(_ context.Context, p *entity.WorkoutPlan) error {
	cp := *p
	f.plans[p.ID] = &cp
	return nil
}

func (f *fakeWorkoutRepo) DeletePlan(_ context.Context, id uint, _ *uint) error {
	delete(f.plans, id)
	delete(f.days, id)
	return nil
}

func (f *fakeWorkoutRepo) ReplacePlanDays(_ context.Context, planID uint, days []repository.WorkoutDayWithItems) error {
	f.days[planID] = nil
	for _, d := range days {
		f.nextDayID++
		day := entity.WorkoutPlanDay{
			ID:       f.nextDayID,
			PlanID:   planID,
			DayIndex: d.Day.DayIndex,
			Title:    d.Day.Title,
		}
		f.days[planID] = append(f.days[planID], day)
		f.items[day.ID] = nil
		for i, it := range d.Items {
			f.nextItemID++
			row := it
			row.ID = f.nextItemID
			row.DayID = day.ID
			if row.Sort == 0 {
				row.Sort = i + 1
			}
			f.items[day.ID] = append(f.items[day.ID], row)
		}
	}
	return nil
}

func (f *fakeWorkoutRepo) LoadPlanDetails(_ context.Context, planIDs []uint) (map[uint][]entity.WorkoutPlanDay, map[uint][]repository.WorkoutPlanItemRow, error) {
	daysByPlan := map[uint][]entity.WorkoutPlanDay{}
	itemsByDay := map[uint][]repository.WorkoutPlanItemRow{}
	for _, pid := range planIDs {
		for _, d := range f.days[pid] {
			daysByPlan[pid] = append(daysByPlan[pid], d)
			for _, it := range f.items[d.ID] {
				exName, mg := "", ""
				if ex, ok := f.exercises[it.ExerciseID]; ok {
					exName = ex.Name
					mg = ex.MuscleGroup
				}
				itemsByDay[d.ID] = append(itemsByDay[d.ID], repository.WorkoutPlanItemRow{
					WorkoutPlanItem: it,
					ExerciseName:    exName,
					MuscleGroup:     mg,
				})
			}
		}
	}
	return daysByPlan, itemsByDay, nil
}

func (f *fakeWorkoutRepo) FindMemberByUserID(_ context.Context, userID uint) (*entity.Member, error) {
	for _, m := range f.members {
		if m.UserID != nil && *m.UserID == userID {
			cp := *m
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeWorkoutRepo) FindMemberByID(_ context.Context, id uint) (*entity.Member, error) {
	if m, ok := f.members[id]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeWorkoutRepo) HasPTContractWithTrainer(_ context.Context, memberID, trainerID uint) (bool, error) {
	return f.ptContracts[[2]uint{memberID, trainerID}], nil
}

func (f *fakeWorkoutRepo) CreateLog(_ context.Context, l *entity.WorkoutLog) error {
	f.nextLogID++
	l.ID = f.nextLogID
	f.logs = append(f.logs, *l)
	return nil
}

func (f *fakeWorkoutRepo) ListMemberLogs(_ context.Context, memberID, exerciseID uint, limit int) ([]repository.WorkoutLogRow, error) {
	var out []repository.WorkoutLogRow
	for _, l := range f.logs {
		if l.MemberID != memberID {
			continue
		}
		if exerciseID > 0 && l.ExerciseID != exerciseID {
			continue
		}
		exName, mg := "", ""
		if ex, ok := f.exercises[l.ExerciseID]; ok {
			exName = ex.Name
			mg = ex.MuscleGroup
		}
		out = append(out, repository.WorkoutLogRow{
			WorkoutLog:   l,
			ExerciseName: exName,
			MuscleGroup:  mg,
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func TestWorkoutService_ExerciseCRUD(t *testing.T) {
	repo := newFakeWorkoutRepo()
	svc := NewWorkoutService(repo, nil, nil)
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})

	ex, err := svc.CreateExercise(ctx, v1.ExerciseInput{
		Name:        "Bench Press",
		MuscleGroup: "CHEST",
		Equipment:   "Barbell",
	})
	require.NoError(t, err)
	assert.Equal(t, "Bench Press", ex.Name)

	// Duplicate name
	_, err = svc.CreateExercise(ctx, v1.ExerciseInput{
		Name:        "bench press",
		MuscleGroup: "CHEST",
	})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	// Update
	up, err := svc.UpdateExercise(ctx, ex.ID, v1.ExerciseInput{
		Name:        "Incline Bench Press",
		MuscleGroup: "CHEST",
	})
	require.NoError(t, err)
	assert.Equal(t, "Incline Bench Press", up.Name)

	// Delete
	require.NoError(t, svc.DeleteExercise(ctx, ex.ID))
}

func TestWorkoutService_TrainerPermissionsAndClone(t *testing.T) {
	repo := newFakeWorkoutRepo()
	svc := NewWorkoutService(repo, nil, nil)

	userID10 := uint(100)
	repo.members[10] = &entity.Member{BaseEntity: entity.BaseEntity{ID: 10}, FullName: "Member A", UserID: &userID10}
	repo.members[20] = &entity.Member{BaseEntity: entity.BaseEntity{ID: 20}, FullName: "Member B"}
	repo.ptContracts[[2]uint{10, 5}] = true // Member 10 has PT contract with Trainer 5

	ownerCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	ex, err := svc.CreateExercise(ownerCtx, v1.ExerciseInput{Name: "Squat", MuscleGroup: "LEGS"})
	require.NoError(t, err)

	trainer5Ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 5, Roles: []string{consts.RoleTrainer}})
	trainer6Ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 6, Roles: []string{consts.RoleTrainer}})

	// Trainer 5 creates a template plan (MemberID == nil) -> allowed, TrainerID defaults to 5
	tpl, err := svc.CreateWorkoutPlan(trainer5Ctx, v1.WorkoutPlanInput{
		Name: "Giáo án mẫu Tăng cơ",
		Goal: "Hypertrophy",
		Days: []v1.WorkoutDayInput{
			{
				DayIndex: 1,
				Title:    "Leg Day",
				Items: []v1.WorkoutItemInput{
					{ExerciseID: ex.ID, Sets: 4, Reps: 8, Weight: 60},
				},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, uint(5), tpl.TrainerID)
	require.Len(t, tpl.Days, 1)
	require.Len(t, tpl.Days[0].Items, 1)

	// Trainer 5 tries to create plan for Member 20 (no PT contract) -> PermissionDenied
	m20 := uint(20)
	_, err = svc.CreateWorkoutPlan(trainer5Ctx, v1.WorkoutPlanInput{
		MemberID: &m20,
		Name:     "Plan M20",
	})
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	// Trainer 5 clones template for Member 10 (has PT contract) -> allowed
	cloned, err := svc.CloneWorkoutPlan(trainer5Ctx, &v1.CloneWorkoutPlanReq{
		ID:        tpl.ID,
		MemberID:  10,
		Name:      "Giáo án riêng Member A",
		StartDate: "2026-10-01",
		EndDate:   "2026-10-31",
	})
	require.NoError(t, err)
	assert.Equal(t, uint(10), *cloned.MemberID)
	assert.Equal(t, uint(5), cloned.TrainerID)
	require.Len(t, cloned.Days, 1)

	// Trainer 6 cannot delete Trainer 5's plan -> PermissionDenied
	err = svc.DeleteWorkoutPlan(trainer6Ctx, cloned.ID)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
}

func TestWorkoutService_MemberLogsAndProgress(t *testing.T) {
	repo := newFakeWorkoutRepo()
	svc := NewWorkoutService(repo, nil, nil)

	memberUID := uint(100)
	repo.members[10] = &entity.Member{BaseEntity: entity.BaseEntity{ID: 10}, FullName: "Member A", UserID: &memberUID}

	ownerCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	ex, err := svc.CreateExercise(ownerCtx, v1.ExerciseInput{Name: "Deadlift", MuscleGroup: "BACK"})
	require.NoError(t, err)

	memberCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: memberUID, Roles: []string{consts.RoleMember}})

	_, err = svc.CreateWorkoutLog(memberCtx, &v1.CreateWorkoutLogReq{
		ExerciseID:  ex.ID,
		Sets:        3,
		Reps:        5,
		Weight:      100,
		PerformedAt: "2026-10-01",
	})
	require.NoError(t, err)

	_, err = svc.CreateWorkoutLog(memberCtx, &v1.CreateWorkoutLogReq{
		ExerciseID:  ex.ID,
		Sets:        4,
		Reps:        5,
		Weight:      110,
		PerformedAt: "2026-10-05",
	})
	require.NoError(t, err)

	res, err := svc.MemberWorkoutLogs(memberCtx, &v1.MemberWorkoutLogsReq{ExerciseID: ex.ID, Limit: 20})
	require.NoError(t, err)
	require.Len(t, res.Logs, 2)
	require.Len(t, res.Progress, 2)
	assert.Equal(t, "2026-10-01", res.Progress[0].Date)
	assert.Equal(t, 100.0, res.Progress[0].MaxWeight)
	assert.Equal(t, 1500.0, res.Progress[0].TotalVolume)
	assert.Equal(t, "2026-10-05", res.Progress[1].Date)
	assert.Equal(t, 110.0, res.Progress[1].MaxWeight)
	assert.Equal(t, 2200.0, res.Progress[1].TotalVolume)

	foreignPlanItem := uint(999)
	_, err = svc.CreateWorkoutLog(memberCtx, &v1.CreateWorkoutLogReq{
		PlanItemID: &foreignPlanItem,
		ExerciseID: ex.ID,
		Sets:       3,
		Reps:       8,
		Weight:     80,
	})
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}
