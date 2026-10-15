package cms

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"

	v1 "gymmaster-api/api/cms/v1"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/service"
)

type Controller struct {
	svc *service.CMSService
}

func New(svc *service.CMSService) *Controller {
	return &Controller{svc: svc}
}

type PublicController struct {
	svc *service.CMSService
}

func NewPublic(svc *service.CMSService) *PublicController {
	return &PublicController{svc: svc}
}

type SharedPortalController struct {
	svc *service.CMSService
}

func NewSharedPortal(svc *service.CMSService) *SharedPortalController {
	return &SharedPortalController{svc: svc}
}

// Public endpoints

func (c *PublicController) PublicTrainers(ctx context.Context, _ *v1.PublicTrainersReq) (v1.PublicTrainersRes, error) {
	return c.svc.PublicTrainers(ctx)
}

func (c *PublicController) PublicTrainerDetail(ctx context.Context, req *v1.PublicTrainerDetailReq) (*v1.PublicTrainerDetailRes, error) {
	return c.svc.PublicTrainerDetail(ctx, req.Slug)
}

func (c *PublicController) PublicClassTypes(ctx context.Context, _ *v1.PublicClassTypesReq) (v1.PublicClassTypesRes, error) {
	return c.svc.PublicClassTypes(ctx)
}

func (c *PublicController) PublicClassTypeDetail(ctx context.Context, req *v1.PublicClassTypeDetailReq) (*v1.PublicClassTypeDetailRes, error) {
	return c.svc.PublicClassTypeDetail(ctx, req.ID)
}

func (c *PublicController) PublicTestimonials(ctx context.Context, _ *v1.PublicTestimonialsReq) (v1.PublicTestimonialsRes, error) {
	return c.svc.PublicTestimonials(ctx)
}

func (c *PublicController) SubmitTestimonial(ctx context.Context, req *v1.PublicSubmitTestimonialReq) (*v1.PublicSubmitTestimonialRes, error) {
	return c.svc.SubmitTestimonial(ctx, req)
}

func (c *PublicController) PublicPosts(ctx context.Context, req *v1.PublicPostsReq) (v1.PublicPostsRes, error) {
	return c.svc.PublicPosts(ctx, req.Category, req.Search)
}

func (c *PublicController) PublicPostDetail(ctx context.Context, req *v1.PublicPostDetailReq) (*v1.PublicPostDetailRes, error) {
	return c.svc.PublicPostDetail(ctx, req.Slug)
}

func (c *PublicController) SubmitContactMessage(ctx context.Context, req *v1.PublicContactMessageReq) (*v1.PublicContactMessageRes, error) {
	return c.svc.SubmitContactMessage(ctx, req)
}

func (c *PublicController) CalendarICS(r *ghttp.Request) {
	tokenStr := r.GetQuery("token").String()
	ics, err := c.svc.BuildCalendarICS(r.Context(), tokenStr)
	if err != nil {
		r.SetError(err)
		return
	}
	r.Response.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	r.Response.Header().Set("Content-Disposition", `attachment; filename="gymmaster-calendar.ics"`)
	r.Response.WriteExit(ics)
}

// Admin endpoints

func (c *Controller) AdminListTrainerProfiles(ctx context.Context, _ *v1.AdminListTrainerProfilesReq) (v1.AdminListTrainerProfilesRes, error) {
	return c.svc.AdminListTrainerProfiles(ctx)
}

func (c *Controller) AdminSaveTrainerProfile(ctx context.Context, req *v1.AdminSaveTrainerProfileReq) (*v1.AdminSaveTrainerProfileRes, error) {
	return c.svc.AdminSaveTrainerProfile(ctx, req)
}

func (c *Controller) AdminDeleteTrainerProfile(ctx context.Context, req *v1.AdminDeleteTrainerProfileReq) (*v1.AdminDeleteTrainerProfileRes, error) {
	return &v1.AdminDeleteTrainerProfileRes{}, c.svc.AdminDeleteTrainerProfile(ctx, req.ID)
}

func (c *Controller) AdminListTestimonials(ctx context.Context, req *v1.AdminListTestimonialsReq) (v1.AdminListTestimonialsRes, error) {
	return c.svc.AdminListTestimonials(ctx, req.Status)
}

func (c *Controller) AdminApproveTestimonial(ctx context.Context, req *v1.AdminApproveTestimonialReq) (*v1.AdminApproveTestimonialRes, error) {
	item, err := c.svc.AdminUpdateTestimonialStatus(ctx, req.ID, entity.TestimonialApproved)
	if err != nil {
		return nil, err
	}
	return &v1.AdminApproveTestimonialRes{TestimonialItem: item}, nil
}

func (c *Controller) AdminRejectTestimonial(ctx context.Context, req *v1.AdminRejectTestimonialReq) (*v1.AdminRejectTestimonialRes, error) {
	item, err := c.svc.AdminUpdateTestimonialStatus(ctx, req.ID, entity.TestimonialRejected)
	if err != nil {
		return nil, err
	}
	return &v1.AdminRejectTestimonialRes{TestimonialItem: item}, nil
}

func (c *Controller) AdminDeleteTestimonial(ctx context.Context, req *v1.AdminDeleteTestimonialReq) (*v1.AdminDeleteTestimonialRes, error) {
	return &v1.AdminDeleteTestimonialRes{}, c.svc.AdminDeleteTestimonial(ctx, req.ID)
}

func (c *Controller) AdminListPosts(ctx context.Context, req *v1.AdminListPostsReq) (v1.AdminListPostsRes, error) {
	return c.svc.AdminListPosts(ctx, req.Status, req.Category)
}

func (c *Controller) AdminCreatePost(ctx context.Context, req *v1.AdminCreatePostReq) (*v1.AdminCreatePostRes, error) {
	return c.svc.AdminCreatePost(ctx, req)
}

func (c *Controller) AdminUpdatePost(ctx context.Context, req *v1.AdminUpdatePostReq) (*v1.AdminUpdatePostRes, error) {
	return c.svc.AdminUpdatePost(ctx, req)
}

func (c *Controller) AdminDeletePost(ctx context.Context, req *v1.AdminDeletePostReq) (*v1.AdminDeletePostRes, error) {
	return &v1.AdminDeletePostRes{}, c.svc.AdminDeletePost(ctx, req.ID)
}

func (c *Controller) AdminListContactMessages(ctx context.Context, req *v1.AdminListContactMessagesReq) (v1.AdminListContactMessagesRes, error) {
	return c.svc.AdminListContactMessages(ctx, req.Status)
}

func (c *Controller) AdminUpdateContactMessage(ctx context.Context, req *v1.AdminUpdateContactMessageReq) (*v1.AdminUpdateContactMessageRes, error) {
	return c.svc.AdminUpdateContactMessage(ctx, req)
}

// Shared endpoints (Conversations, Notification Preferences, Calendar Token)

func (c *SharedPortalController) ListConversations(ctx context.Context, _ *v1.ListConversationsReq) (v1.ListConversationsRes, error) {
	return c.svc.ListConversations(ctx)
}

func (c *SharedPortalController) ListMessages(ctx context.Context, req *v1.ListMessagesReq) (v1.ListMessagesRes, error) {
	return c.svc.ListMessages(ctx, req.ID)
}

func (c *SharedPortalController) SendMessage(ctx context.Context, req *v1.SendMessageReq) (*v1.SendMessageRes, error) {
	return c.svc.SendMessage(ctx, req)
}

func (c *SharedPortalController) GetNotificationPreferences(ctx context.Context, _ *v1.GetNotificationPreferencesReq) (v1.GetNotificationPreferencesRes, error) {
	return c.svc.GetNotificationPreferences(ctx)
}

func (c *SharedPortalController) SaveNotificationPreferences(ctx context.Context, req *v1.SaveNotificationPreferencesReq) (v1.SaveNotificationPreferencesRes, error) {
	return c.svc.SaveNotificationPreferences(ctx, req)
}

func (c *SharedPortalController) GetCalendarToken(ctx context.Context, _ *v1.GetCalendarTokenReq) (*v1.GetCalendarTokenRes, error) {
	res, err := c.svc.GetOrRotateCalendarToken(ctx, false)
	if err != nil {
		return nil, err
	}
	return &v1.GetCalendarTokenRes{CalendarTokenRes: res}, nil
}

func (c *SharedPortalController) RotateCalendarToken(ctx context.Context, _ *v1.RotateCalendarTokenReq) (*v1.RotateCalendarTokenRes, error) {
	res, err := c.svc.GetOrRotateCalendarToken(ctx, true)
	if err != nil {
		return nil, err
	}
	return &v1.RotateCalendarTokenRes{CalendarTokenRes: res}, nil
}
