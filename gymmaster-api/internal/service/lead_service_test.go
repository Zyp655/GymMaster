package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/lead/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

type fakeLeadRepo struct {
	branches   map[uint]*entity.Branch
	leads      map[uint]*repository.LeadRow
	activities map[uint][]repository.LeadActivityRow
	passes     map[uint][]repository.TrialPassRow
	members    map[string]*entity.Member
	promos     map[string]*entity.Promotion
	seq        uint
}

func newFakeLeadRepo() *fakeLeadRepo {
	return &fakeLeadRepo{
		branches: map[uint]*entity.Branch{
			1: {Code: "HN01", Name: "GymMaster Cầu Giấy"},
		},
		leads:      make(map[uint]*repository.LeadRow),
		activities: make(map[uint][]repository.LeadActivityRow),
		passes:     make(map[uint][]repository.TrialPassRow),
		members:    make(map[string]*entity.Member),
		promos:     make(map[string]*entity.Promotion),
	}
}

func (f *fakeLeadRepo) FindBranch(_ context.Context, id uint) (*entity.Branch, error) {
	b, ok := f.branches[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return b, nil
}

func (f *fakeLeadRepo) ListLeads(_ context.Context, _ dto.Query, _ repository.LeadFilter) ([]repository.LeadRow, int64, error) {
	out := make([]repository.LeadRow, 0, len(f.leads))
	for _, l := range f.leads {
		out = append(out, *l)
	}
	return out, int64(len(out)), nil
}

func (f *fakeLeadRepo) FindLeadByID(_ context.Context, id uint) (*repository.LeadRow, error) {
	l, ok := f.leads[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *l
	return &cp, nil
}

func (f *fakeLeadRepo) CreateLead(_ context.Context, l *entity.Lead) error {
	f.seq++
	l.ID = f.seq
	f.leads[l.ID] = &repository.LeadRow{
		Lead:       *l,
		BranchName: "GymMaster Cầu Giấy",
	}
	return nil
}

func (f *fakeLeadRepo) UpdateLead(_ context.Context, l *entity.Lead) error {
	row, ok := f.leads[l.ID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	row.Lead = *l
	return nil
}

func (f *fakeLeadRepo) DeleteLead(_ context.Context, id uint, _ *uint) error {
	delete(f.leads, id)
	return nil
}

func (f *fakeLeadRepo) ListLeadActivities(_ context.Context, leadID uint) ([]repository.LeadActivityRow, error) {
	return f.activities[leadID], nil
}

func (f *fakeLeadRepo) CreateLeadActivity(_ context.Context, a *entity.LeadActivity) error {
	f.seq++
	a.ID = f.seq
	f.activities[a.LeadID] = append(f.activities[a.LeadID], repository.LeadActivityRow{LeadActivity: *a})
	return nil
}

func (f *fakeLeadRepo) ListTrialPasses(_ context.Context, leadID uint) ([]repository.TrialPassRow, error) {
	return f.passes[leadID], nil
}

func (f *fakeLeadRepo) CreateTrialPass(_ context.Context, tp *entity.TrialPass) error {
	f.seq++
	tp.ID = f.seq
	f.passes[tp.LeadID] = append([]repository.TrialPassRow{{TrialPass: *tp, BranchName: "GymMaster Cầu Giấy"}}, f.passes[tp.LeadID]...)
	return nil
}

func (f *fakeLeadRepo) FindMemberByPhone(_ context.Context, phone string) (*entity.Member, error) {
	m, ok := f.members[phone]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return m, nil
}

func (f *fakeLeadRepo) CreateMember(_ context.Context, m *entity.Member) error {
	f.seq++
	m.ID = f.seq
	m.MemberCode = "GM000999"
	f.members[m.Phone] = m
	return nil
}

func (f *fakeLeadRepo) ListPublicSchedules(_ context.Context, _ uint, _, _ time.Time) ([]repository.PublicScheduleRow, error) {
	return nil, nil
}

func (f *fakeLeadRepo) ListAtRiskMembers(_ context.Context, _ []uint, _ time.Time, _ int) ([]repository.AtRiskMemberRow, error) {
	return []repository.AtRiskMemberRow{
		{
			MemberID:     10,
			MemberCode:   "GM000010",
			FullName:     "Hội viên Vắng Mặt",
			Phone:        "0911223344",
			BranchID:     1,
			BranchName:   "GymMaster Cầu Giấy",
			PlanName:     "Gói 3 tháng",
			EndDate:      time.Now().AddDate(0, 1, 0),
			DaysInactive: 18,
		},
	}, nil
}

func (f *fakeLeadRepo) LeadsReport(_ context.Context, _ []uint, _, _ *time.Time) (*repository.LeadsReportData, error) {
	return &repository.LeadsReportData{
		Total:     4,
		Converted: 1,
		ByStatus:  []repository.LeadCountRow{{Key: "NEW", Count: 3}, {Key: "CONVERTED", Count: 1}},
		BySource:  []repository.LeadCountRow{{Key: "WEBSITE", Count: 4}},
	}, nil
}

func (f *fakeLeadRepo) ListBirthdayMembers(_ context.Context, _ time.Time) ([]entity.Member, error) {
	uid := uint(88)
	return []entity.Member{
		{
			BaseEntity: entity.BaseEntity{ID: 5},
			UserID:     &uid,
			MemberCode: "GM000005",
			FullName:   "Nguyễn Sinh Nhật",
			Email:      "sinhnhat@example.com",
		},
	}, nil
}

func (f *fakeLeadRepo) EnsureBirthdayPromotion(_ context.Context, p *entity.Promotion) error {
	f.promos[p.Code] = p
	return nil
}

func TestLeadService_PublicCreateRateLimitAndFlow(t *testing.T) {
	repo := newFakeLeadRepo()
	svc := NewLeadService(repo, nil, nil, nil, nil)
	svc.(*leadService).now = func() time.Time { return time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC) }
	ctx := ownerCtx()

	// SĐT không hợp lệ bị từ chối ngay khi tạo lead.
	_, err0 := svc.PublicCreateLead(ctx, &v1.PublicCreateLeadReq{BranchID: 1, FullName: "Sai SĐT", Phone: "abcdefghij"}, "203.0.113.99")
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err0))

	// 1. Tạo lead từ trang chủ (tối đa 5 lần / 15 phút trên cùng IP)
	var firstLead *v1.Lead
	for i := 0; i < 5; i++ {
		res, err := svc.PublicCreateLead(ctx, &v1.PublicCreateLeadReq{
			BranchID: 1,
			FullName: "Khách Tập Thử",
			Phone:    "0988776655",
			Email:    "trial@example.com",
		}, "203.0.113.10")
		require.NoError(t, err)
		if i == 0 {
			firstLead = res
		}
	}
	require.NotNil(t, firstLead)
	assert.Equal(t, entity.LeadStatusNew, firstLead.Status)
	assert.Equal(t, entity.LeadSourceWebsite, firstLead.Source)

	// Lần thứ 6 cùng IP bị chặn rate-limit
	_, err := svc.PublicCreateLead(ctx, &v1.PublicCreateLeadReq{
		BranchID: 1,
		FullName: "Khách Tập Thử 6",
		Phone:    "0988776655",
	}, "203.0.113.10")
	require.Error(t, err)
	assert.Equal(t, rescode.TooManyAttempts.Code(), codeOf(err))

	// 2. Ghi nhận hoạt động chăm sóc tự chuyển trạng thái NEW -> CONTACTED
	act, err := svc.AddActivity(ctx, &v1.AddLeadActivityReq{
		ID:      firstLead.ID,
		Type:    entity.LeadActivityCall,
		Content: "Đã gọi tư vấn lịch tập thử",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.LeadActivityCall, act.Type)

	updated, err := svc.GetLead(ctx, firstLead.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.LeadStatusContacted, updated.Status)

	// 3. Cấp vé tập thử QR tự chuyển sang TRIAL và sinh mã TP-...
	tp, err := svc.IssueTrialPass(ctx, &v1.IssueTrialPassReq{
		ID:        firstLead.ID,
		ValidDate: "2026-10-15",
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(tp.QRToken, "TP-"))

	// Mỗi lead chỉ 1 vé tập thử còn hiệu lực; ngày quá khứ bị từ chối.
	_, err = svc.IssueTrialPass(ctx, &v1.IssueTrialPassReq{ID: firstLead.ID, ValidDate: "2026-10-16"})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	_, err = svc.IssueTrialPass(ctx, &v1.IssueTrialPassReq{ID: firstLead.ID, ValidDate: "2026-10-01"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// Không chuyển CONVERTED bằng đổi trạng thái thường.
	_, err = svc.UpdateLeadStatus(ctx, firstLead.ID, entity.LeadStatusConverted)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	updated, err = svc.GetLead(ctx, firstLead.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.LeadStatusTrial, updated.Status)
	require.NotNil(t, updated.TrialPass)

	// 4. Chuyển đổi thành hội viên (CONVERTED)
	converted, err := svc.ConvertLead(ctx, &v1.ConvertLeadReq{
		ID:     firstLead.ID,
		Gender: "MALE",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.LeadStatusConverted, converted.Status)
	require.NotNil(t, converted.MemberID)

	// CONVERTED là trạng thái cuối.
	_, err = svc.UpdateLeadStatus(ctx, firstLead.ID, entity.LeadStatusLost)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// 5. Báo cáo phễu chuyển đổi & Cron sinh nhật / churn risk
	rep, err := svc.LeadsReport(ctx, &v1.LeadsReportReq{})
	require.NoError(t, err)
	assert.Equal(t, int64(4), rep.Total)
	assert.Equal(t, 25.0, rep.ConversionRate)

	bdayCount, err := svc.SendBirthdayGreetings(ctx, time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 1, bdayCount)
	assert.Contains(t, repo.promos, "BDAY-GM000005-2026")

	churnCount, err := svc.ScanChurnRisk(ctx, time.Now())
	require.NoError(t, err)
	assert.Equal(t, 1, churnCount)
}
