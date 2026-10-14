package workout

import (
	"context"

	v1 "gymmaster-api/api/workout/v1"
	"gymmaster-api/internal/service"
)

type AdminControllerV1 struct {
	svc service.WorkoutService
}

type MemberControllerV1 struct {
	svc service.WorkoutService
}

func NewAdminV1(svc service.WorkoutService) *AdminControllerV1 {
	return &AdminControllerV1{svc: svc}
}

func NewMemberV1(svc service.WorkoutService) *MemberControllerV1 {
	return &MemberControllerV1{svc: svc}
}

func (c *AdminControllerV1) ListExercises(ctx context.Context, req *v1.ListExercisesReq) (res *v1.ListExercisesRes, err error) {
	return c.svc.ListExercises(ctx, req)
}

func (c *AdminControllerV1) CreateExercise(ctx context.Context, req *v1.CreateExerciseReq) (res *v1.CreateExerciseRes, err error) {
	return c.svc.CreateExercise(ctx, req.ExerciseInput)
}

func (c *AdminControllerV1) UpdateExercise(ctx context.Context, req *v1.UpdateExerciseReq) (res *v1.UpdateExerciseRes, err error) {
	return c.svc.UpdateExercise(ctx, req.ID, req.ExerciseInput)
}

func (c *AdminControllerV1) DeleteExercise(ctx context.Context, req *v1.DeleteExerciseReq) (res *v1.DeleteRes, err error) {
	return &v1.DeleteRes{}, c.svc.DeleteExercise(ctx, req.ID)
}

func (c *AdminControllerV1) ListWorkoutPlans(ctx context.Context, req *v1.ListWorkoutPlansReq) (res *v1.ListWorkoutPlansRes, err error) {
	return c.svc.ListWorkoutPlans(ctx, req)
}

func (c *AdminControllerV1) GetWorkoutPlan(ctx context.Context, req *v1.GetWorkoutPlanReq) (res *v1.GetWorkoutPlanRes, err error) {
	return c.svc.GetWorkoutPlan(ctx, req.ID)
}

func (c *AdminControllerV1) CreateWorkoutPlan(ctx context.Context, req *v1.CreateWorkoutPlanReq) (res *v1.CreateWorkoutPlanRes, err error) {
	return c.svc.CreateWorkoutPlan(ctx, req.WorkoutPlanInput)
}

func (c *AdminControllerV1) UpdateWorkoutPlan(ctx context.Context, req *v1.UpdateWorkoutPlanReq) (res *v1.UpdateWorkoutPlanRes, err error) {
	return c.svc.UpdateWorkoutPlan(ctx, req.ID, req.WorkoutPlanInput)
}

func (c *AdminControllerV1) DeleteWorkoutPlan(ctx context.Context, req *v1.DeleteWorkoutPlanReq) (res *v1.DeleteWorkoutPlanRes, err error) {
	return &v1.DeleteWorkoutPlanRes{}, c.svc.DeleteWorkoutPlan(ctx, req.ID)
}

func (c *AdminControllerV1) CloneWorkoutPlan(ctx context.Context, req *v1.CloneWorkoutPlanReq) (res *v1.CloneWorkoutPlanRes, err error) {
	return c.svc.CloneWorkoutPlan(ctx, req)
}

func (c *MemberControllerV1) MemberExercises(ctx context.Context, req *v1.MemberExercisesReq) (res v1.MemberExercisesRes, err error) {
	return c.svc.MemberExercises(ctx, req)
}

func (c *MemberControllerV1) MemberWorkoutPlans(ctx context.Context, _ *v1.MemberWorkoutPlansReq) (res v1.MemberWorkoutPlansRes, err error) {
	return c.svc.MemberWorkoutPlans(ctx)
}

func (c *MemberControllerV1) CreateWorkoutLog(ctx context.Context, req *v1.CreateWorkoutLogReq) (res *v1.CreateWorkoutLogRes, err error) {
	return c.svc.CreateWorkoutLog(ctx, req)
}

func (c *MemberControllerV1) MemberWorkoutLogs(ctx context.Context, req *v1.MemberWorkoutLogsReq) (res *v1.MemberWorkoutLogsRes, err error) {
	return c.svc.MemberWorkoutLogs(ctx, req)
}
