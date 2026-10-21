package engagement

import (
	"context"

	v1 "gymmaster-api/api/engagement/v1"
	"gymmaster-api/internal/service"
)

type AdminControllerV1 struct {
	svc service.EngagementService
}

type MemberControllerV1 struct {
	svc service.EngagementService
}

func NewAdminV1(svc service.EngagementService) *AdminControllerV1 {
	return &AdminControllerV1{svc: svc}
}

func NewMemberV1(svc service.EngagementService) *MemberControllerV1 {
	return &MemberControllerV1{svc: svc}
}

// ---- Admin ----------------------------------------------------------------------

func (c *AdminControllerV1) ListFoods(ctx context.Context, req *v1.ListFoodsReq) (res *v1.ListFoodsRes, err error) {
	return c.svc.ListFoods(ctx, req)
}

func (c *AdminControllerV1) CreateFood(ctx context.Context, req *v1.CreateFoodReq) (res *v1.CreateFoodRes, err error) {
	return c.svc.CreateFood(ctx, req.FoodInput)
}

func (c *AdminControllerV1) UpdateFood(ctx context.Context, req *v1.UpdateFoodReq) (res *v1.UpdateFoodRes, err error) {
	return c.svc.UpdateFood(ctx, req.ID, req.FoodInput)
}

func (c *AdminControllerV1) DeleteFood(ctx context.Context, req *v1.DeleteFoodReq) (res *v1.DeleteEngagementRes, err error) {
	return &v1.DeleteEngagementRes{}, c.svc.DeleteFood(ctx, req.ID)
}

func (c *AdminControllerV1) ListMealPlans(ctx context.Context, req *v1.ListMealPlansReq) (res *v1.ListMealPlansRes, err error) {
	return c.svc.ListMealPlans(ctx, req)
}

func (c *AdminControllerV1) GetMealPlan(ctx context.Context, req *v1.GetMealPlanReq) (res *v1.GetMealPlanRes, err error) {
	return c.svc.GetMealPlan(ctx, req.ID)
}

func (c *AdminControllerV1) CreateMealPlan(ctx context.Context, req *v1.CreateMealPlanReq) (res *v1.CreateMealPlanRes, err error) {
	return c.svc.CreateMealPlan(ctx, req.MealPlanInput)
}

func (c *AdminControllerV1) UpdateMealPlan(ctx context.Context, req *v1.UpdateMealPlanReq) (res *v1.UpdateMealPlanRes, err error) {
	return c.svc.UpdateMealPlan(ctx, req.ID, req.MealPlanInput)
}

func (c *AdminControllerV1) DeleteMealPlan(ctx context.Context, req *v1.DeleteMealPlanReq) (res *v1.DeleteEngagementRes, err error) {
	return &v1.DeleteEngagementRes{}, c.svc.DeleteMealPlan(ctx, req.ID)
}

func (c *AdminControllerV1) ListAnnouncements(ctx context.Context, req *v1.ListAnnouncementsReq) (res *v1.ListAnnouncementsRes, err error) {
	return c.svc.ListAnnouncements(ctx, req)
}

func (c *AdminControllerV1) CreateAnnouncement(ctx context.Context, req *v1.CreateAnnouncementReq) (res *v1.CreateAnnouncementRes, err error) {
	return c.svc.CreateAnnouncement(ctx, req.AnnouncementInput)
}

func (c *AdminControllerV1) UpdateAnnouncement(ctx context.Context, req *v1.UpdateAnnouncementReq) (res *v1.UpdateAnnouncementRes, err error) {
	return c.svc.UpdateAnnouncement(ctx, req.ID, req.AnnouncementInput)
}

func (c *AdminControllerV1) DeleteAnnouncement(ctx context.Context, req *v1.DeleteAnnouncementReq) (res *v1.DeleteEngagementRes, err error) {
	return &v1.DeleteEngagementRes{}, c.svc.DeleteAnnouncement(ctx, req.ID)
}

func (c *AdminControllerV1) ListSurveys(ctx context.Context, req *v1.ListSurveysReq) (res *v1.ListSurveysRes, err error) {
	return c.svc.ListSurveys(ctx, req)
}

func (c *AdminControllerV1) CreateSurvey(ctx context.Context, req *v1.CreateSurveyReq) (res *v1.CreateSurveyRes, err error) {
	return c.svc.CreateSurvey(ctx, req.SurveyInput)
}

func (c *AdminControllerV1) UpdateSurvey(ctx context.Context, req *v1.UpdateSurveyReq) (res *v1.UpdateSurveyRes, err error) {
	return c.svc.UpdateSurvey(ctx, req.ID, req.SurveyInput)
}

func (c *AdminControllerV1) DeleteSurvey(ctx context.Context, req *v1.DeleteSurveyReq) (res *v1.DeleteEngagementRes, err error) {
	return &v1.DeleteEngagementRes{}, c.svc.DeleteSurvey(ctx, req.ID)
}

func (c *AdminControllerV1) NPSReport(ctx context.Context, req *v1.NPSReportReq) (res *v1.NPSReportRes, err error) {
	return c.svc.NPSReport(ctx, req)
}

// ---- Member ---------------------------------------------------------------------

func (c *MemberControllerV1) MemberNutritionOverview(ctx context.Context, _ *v1.MemberNutritionOverviewReq) (res *v1.MemberNutritionOverviewRes, err error) {
	return c.svc.MemberNutritionOverview(ctx)
}

func (c *MemberControllerV1) MemberFoodLogs(ctx context.Context, req *v1.MemberFoodLogsReq) (res *v1.MemberFoodLogsRes, err error) {
	return c.svc.MemberFoodLogs(ctx, req)
}

func (c *MemberControllerV1) CreateFoodLog(ctx context.Context, req *v1.CreateFoodLogReq) (res *v1.CreateFoodLogRes, err error) {
	return c.svc.CreateFoodLog(ctx, req)
}

func (c *MemberControllerV1) UpdateFoodLog(ctx context.Context, req *v1.UpdateFoodLogReq) (res *v1.UpdateFoodLogRes, err error) {
	return c.svc.UpdateFoodLog(ctx, req)
}

func (c *MemberControllerV1) DeleteFoodLog(ctx context.Context, req *v1.DeleteFoodLogReq) (res *v1.DeleteEngagementRes, err error) {
	return &v1.DeleteEngagementRes{}, c.svc.DeleteFoodLog(ctx, req.ID)
}

func (c *MemberControllerV1) MemberAnnouncements(ctx context.Context, _ *v1.MemberAnnouncementsReq) (res v1.MemberAnnouncementsRes, err error) {
	return c.svc.MemberAnnouncements(ctx)
}

func (c *MemberControllerV1) MemberSurveys(ctx context.Context, _ *v1.MemberSurveysReq) (res v1.MemberSurveysRes, err error) {
	return c.svc.MemberSurveys(ctx)
}

func (c *MemberControllerV1) SubmitSurveyResponse(ctx context.Context, req *v1.SubmitSurveyResponseReq) (res *v1.SubmitSurveyResponseRes, err error) {
	return c.svc.SubmitSurveyResponse(ctx, req)
}
