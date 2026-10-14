package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/engagement/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

type fakeEngagementRepo struct {
	nextFoodID   uint
	nextPlanID   uint
	nextLogID    uint
	nextSurveyID uint
	nextRespID   uint

	foods       map[uint]*entity.Food
	plans       map[uint]*entity.MealPlan
	planItems   map[uint][]entity.MealPlanItem
	foodLogs    map[uint]*entity.FoodLog
	surveys     map[uint]*entity.Survey
	responses   []entity.SurveyResponse
	members     map[uint]*entity.Member
	bodyMetrics map[uint]*entity.BodyMetric
}

func newFakeEngagementRepo() *fakeEngagementRepo {
	return &fakeEngagementRepo{
		foods:       map[uint]*entity.Food{},
		plans:       map[uint]*entity.MealPlan{},
		planItems:   map[uint][]entity.MealPlanItem{},
		foodLogs:    map[uint]*entity.FoodLog{},
		surveys:     map[uint]*entity.Survey{},
		members:     map[uint]*entity.Member{},
		bodyMetrics: map[uint]*entity.BodyMetric{},
	}
}

func (f *fakeEngagementRepo) ListFoods(_ context.Context, _ dto.Query, _ string) ([]entity.Food, int64, error) {
	items, _ := f.ListAllFoods(context.Background())
	return items, int64(len(items)), nil
}

func (f *fakeEngagementRepo) ListAllFoods(_ context.Context) ([]entity.Food, error) {
	var out []entity.Food
	for _, fd := range f.foods {
		out = append(out, *fd)
	}
	return out, nil
}

func (f *fakeEngagementRepo) FindFoodByID(_ context.Context, id uint) (*entity.Food, error) {
	if fd, ok := f.foods[id]; ok {
		cp := *fd
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeEngagementRepo) ExistsFoodName(_ context.Context, name string, excludeID uint) (bool, error) {
	for _, fd := range f.foods {
		if strings.EqualFold(fd.Name, strings.TrimSpace(name)) && fd.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeEngagementRepo) CreateFood(_ context.Context, fd *entity.Food) error {
	f.nextFoodID++
	fd.ID = f.nextFoodID
	cp := *fd
	f.foods[fd.ID] = &cp
	return nil
}

func (f *fakeEngagementRepo) UpdateFood(_ context.Context, fd *entity.Food) error {
	cp := *fd
	f.foods[fd.ID] = &cp
	return nil
}

func (f *fakeEngagementRepo) DeleteFood(_ context.Context, id uint) error {
	delete(f.foods, id)
	return nil
}

func (f *fakeEngagementRepo) ListMealPlans(_ context.Context, _ dto.Query, memberID, _ uint, _ string) ([]repository.MealPlanRow, int64, error) {
	rows, _ := f.ListMemberMealPlans(context.Background(), memberID)
	return rows, int64(len(rows)), nil
}

func (f *fakeEngagementRepo) ListMemberMealPlans(_ context.Context, memberID uint) ([]repository.MealPlanRow, error) {
	var out []repository.MealPlanRow
	for _, p := range f.plans {
		if memberID == 0 || p.MemberID == memberID {
			out = append(out, repository.MealPlanRow{MealPlan: *p})
		}
	}
	return out, nil
}

func (f *fakeEngagementRepo) FindMealPlanByID(_ context.Context, id uint) (*repository.MealPlanRow, error) {
	if p, ok := f.plans[id]; ok {
		return &repository.MealPlanRow{MealPlan: *p}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeEngagementRepo) CreateMealPlan(_ context.Context, p *entity.MealPlan) error {
	f.nextPlanID++
	p.ID = f.nextPlanID
	cp := *p
	f.plans[p.ID] = &cp
	return nil
}

func (f *fakeEngagementRepo) UpdateMealPlan(_ context.Context, p *entity.MealPlan) error {
	cp := *p
	f.plans[p.ID] = &cp
	return nil
}

func (f *fakeEngagementRepo) DeleteMealPlan(_ context.Context, id uint) error {
	delete(f.plans, id)
	return nil
}

func (f *fakeEngagementRepo) ReplaceMealPlanItems(_ context.Context, planID uint, items []entity.MealPlanItem) error {
	f.planItems[planID] = items
	return nil
}

func (f *fakeEngagementRepo) LoadMealPlanItems(_ context.Context, planIDs []uint) (map[uint][]repository.MealPlanItemRow, error) {
	out := map[uint][]repository.MealPlanItemRow{}
	for _, pid := range planIDs {
		for _, it := range f.planItems[pid] {
			row := repository.MealPlanItemRow{MealPlanItem: it}
			if fd, ok := f.foods[it.FoodID]; ok {
				row.FoodName = fd.Name
				row.Unit = fd.Unit
				row.Calories = fd.Calories
				row.Protein = fd.Protein
				row.Carbs = fd.Carbs
				row.Fat = fd.Fat
			}
			out[pid] = append(out[pid], row)
		}
	}
	return out, nil
}

func (f *fakeEngagementRepo) ListFoodLogsByDate(_ context.Context, memberID uint, date time.Time) ([]repository.FoodLogRow, error) {
	ds := date.Format(consts.DateLayout)
	var out []repository.FoodLogRow
	for _, l := range f.foodLogs {
		if l.MemberID == memberID && l.EatenAt.Format(consts.DateLayout) == ds {
			row := repository.FoodLogRow{FoodLog: *l}
			if fd, ok := f.foods[l.FoodID]; ok {
				row.FoodName = fd.Name
				row.Unit = fd.Unit
				row.Calories = fd.Calories
				row.Protein = fd.Protein
				row.Carbs = fd.Carbs
				row.Fat = fd.Fat
			}
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeEngagementRepo) FindFoodLogByID(_ context.Context, id uint) (*entity.FoodLog, error) {
	if l, ok := f.foodLogs[id]; ok {
		cp := *l
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeEngagementRepo) CreateFoodLog(_ context.Context, l *entity.FoodLog) error {
	f.nextLogID++
	l.ID = f.nextLogID
	cp := *l
	f.foodLogs[l.ID] = &cp
	return nil
}

func (f *fakeEngagementRepo) DeleteFoodLog(_ context.Context, id uint) error {
	delete(f.foodLogs, id)
	return nil
}

func (f *fakeEngagementRepo) ListAnnouncements(_ context.Context, _ dto.Query, _ uint) ([]repository.AnnouncementRow, int64, error) {
	return nil, 0, nil
}
func (f *fakeEngagementRepo) ListActiveAnnouncements(_ context.Context, _ uint, _ time.Time) ([]repository.AnnouncementRow, error) {
	return nil, nil
}
func (f *fakeEngagementRepo) FindAnnouncementByID(_ context.Context, _ uint) (*repository.AnnouncementRow, error) {
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeEngagementRepo) CreateAnnouncement(_ context.Context, _ *entity.Announcement) error {
	return nil
}
func (f *fakeEngagementRepo) UpdateAnnouncement(_ context.Context, _ *entity.Announcement) error {
	return nil
}
func (f *fakeEngagementRepo) DeleteAnnouncement(_ context.Context, _ uint) error { return nil }

func (f *fakeEngagementRepo) ListSurveys(_ context.Context, _ dto.Query, _ string) ([]repository.SurveyRow, int64, error) {
	return nil, 0, nil
}
func (f *fakeEngagementRepo) ListActiveSurveys(_ context.Context, _ time.Time) ([]repository.SurveyRow, error) {
	var out []repository.SurveyRow
	for _, sv := range f.surveys {
		out = append(out, repository.SurveyRow{Survey: *sv})
	}
	return out, nil
}
func (f *fakeEngagementRepo) FindSurveyByID(_ context.Context, id uint) (*repository.SurveyRow, error) {
	if sv, ok := f.surveys[id]; ok {
		return &repository.SurveyRow{Survey: *sv}, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeEngagementRepo) CreateSurvey(_ context.Context, sv *entity.Survey) error {
	f.nextSurveyID++
	sv.ID = f.nextSurveyID
	cp := *sv
	f.surveys[sv.ID] = &cp
	return nil
}
func (f *fakeEngagementRepo) UpdateSurvey(_ context.Context, sv *entity.Survey) error {
	cp := *sv
	f.surveys[sv.ID] = &cp
	return nil
}
func (f *fakeEngagementRepo) DeleteSurvey(_ context.Context, id uint) error {
	delete(f.surveys, id)
	return nil
}
func (f *fakeEngagementRepo) ListMemberSurveyResponses(_ context.Context, memberID uint) ([]entity.SurveyResponse, error) {
	var out []entity.SurveyResponse
	for _, r := range f.responses {
		if r.MemberID == memberID {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeEngagementRepo) UpsertSurveyResponse(_ context.Context, sr *entity.SurveyResponse) error {
	f.nextRespID++
	sr.ID = f.nextRespID
	f.responses = append(f.responses, *sr)
	return nil
}
func (f *fakeEngagementRepo) ListSurveyResponses(_ context.Context, surveyID uint) ([]repository.SurveyResponseRow, error) {
	var out []repository.SurveyResponseRow
	for _, r := range f.responses {
		if surveyID == 0 || r.SurveyID == surveyID {
			out = append(out, repository.SurveyResponseRow{SurveyResponse: r})
		}
	}
	return out, nil
}

func (f *fakeEngagementRepo) FindMemberByID(_ context.Context, id uint) (*entity.Member, error) {
	if m, ok := f.members[id]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeEngagementRepo) FindMemberByUserID(_ context.Context, userID uint) (*entity.Member, error) {
	for _, m := range f.members {
		if m.UserID != nil && *m.UserID == userID {
			cp := *m
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeEngagementRepo) FindLatestBodyMetric(_ context.Context, memberID uint) (*entity.BodyMetric, error) {
	if bm, ok := f.bodyMetrics[memberID]; ok {
		cp := *bm
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func TestEngagementService_NutritionBMRAndNPS(t *testing.T) {
	repo := newFakeEngagementRepo()
	svc := NewEngagementService(repo, nil, nil)

	dob := time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC)
	uid1 := uint(101)
	uid2 := uint(102)
	uid3 := uint(103)
	repo.members[1] = &entity.Member{
		BaseEntity: entity.BaseEntity{ID: 1},
		UserID:     &uid1,
		FullName:   "Nam Hội Viên",
		Gender:     entity.MemberGenderMale,
		Dob:        &dob,
	}
	repo.members[2] = &entity.Member{BaseEntity: entity.BaseEntity{ID: 2}, UserID: &uid2, FullName: "HV 2"}
	repo.members[3] = &entity.Member{BaseEntity: entity.BaseEntity{ID: 3}, UserID: &uid3, FullName: "HV 3"}

	// BMR Mifflin-St Jeor cho nam 30 tuổi (2026-1996=30), 70kg, 175cm:
	// BMR = 10*70 + 6.25*175 - 5*30 + 5 = 700 + 1093.75 - 150 + 5 = 1648.75
	// TDEE = 1648.75 * 1.55 = 2555.56 -> 2556 kcal
	repo.bodyMetrics[1] = &entity.BodyMetric{
		MemberID: 1,
		WeightKg: 70,
		HeightCm: 175,
	}

	adminCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 999, Roles: []string{consts.RoleOwner}})
	m1Ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: uid1, Roles: []string{consts.RoleMember}})
	m2Ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: uid2, Roles: []string{consts.RoleMember}})
	m3Ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: uid3, Roles: []string{consts.RoleMember}})

	chicken, err := svc.CreateFood(adminCtx, v1.FoodInput{
		Name:     "Ức gà 100g",
		Unit:     "100g",
		Calories: 165,
		Protein:  31,
		Carbs:    0,
		Fat:      3.6,
	})
	require.NoError(t, err)

	overview, err := svc.MemberNutritionOverview(m1Ctx)
	require.NoError(t, err)
	assert.Equal(t, 2556, overview.RecommendedCalories)

	// Ghi nhật ký ăn uống 2 khẩu phần ức gà
	_, err = svc.CreateFoodLog(m1Ctx, &v1.CreateFoodLogReq{
		FoodID:  chicken.ID,
		Qty:     2,
		Meal:    entity.MealLunch,
		EatenAt: "2026-10-14",
	})
	require.NoError(t, err)

	logsRes, err := svc.MemberFoodLogs(m1Ctx, &v1.MemberFoodLogsReq{Date: "2026-10-14"})
	require.NoError(t, err)
	assert.Equal(t, 330.0, logsRes.TotalCalories)
	assert.Equal(t, 62.0, logsRes.TotalProtein)

	// Tạo khảo sát NPS & 3 hội viên trả lời: 1 Promoter (10), 1 Passive (8), 1 Detractor (5)
	// NPS = (1 - 1) / 3 * 100 = 0; nếu thêm 1 Promoter nữa -> (2 - 1)/4 = +25
	sv, err := svc.CreateSurvey(adminCtx, v1.SurveyInput{
		Title:  "Khảo sát hài lòng tháng 10",
		Type:   entity.SurveyTypeNPS,
		Status: consts.StatusActive,
	})
	require.NoError(t, err)

	_, err = svc.SubmitSurveyResponse(m1Ctx, &v1.SubmitSurveyResponseReq{ID: sv.ID, Score: 10, Comment: "Tuyệt vời"})
	require.NoError(t, err)
	_, err = svc.SubmitSurveyResponse(m2Ctx, &v1.SubmitSurveyResponseReq{ID: sv.ID, Score: 9, Comment: "Tốt"})
	require.NoError(t, err)
	_, err = svc.SubmitSurveyResponse(m3Ctx, &v1.SubmitSurveyResponseReq{ID: sv.ID, Score: 5, Comment: "Cần thêm tạ"})
	require.NoError(t, err)

	nps, err := svc.NPSReport(adminCtx, &v1.NPSReportReq{SurveyID: sv.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(3), nps.TotalResponses)
	assert.Equal(t, int64(2), nps.Promoters)
	assert.Equal(t, int64(0), nps.Passives)
	assert.Equal(t, int64(1), nps.Detractors)
	assert.Equal(t, 33.3, nps.NPSScore) // (2-1)/3 * 100 = 33.3%
}
