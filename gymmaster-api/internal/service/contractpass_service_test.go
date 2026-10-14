package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/contractpass/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

type fakeContractPassRepo struct {
	nextTplID      uint
	nextContractID uint
	nextPassID     uint

	templates map[uint]*entity.ContractTemplate
	contracts map[uint]*entity.MemberContract
	passes    map[uint]*entity.DayPass
	members   map[uint]*entity.Member
	plans     map[uint]*entity.MembershipPlan
}

func newFakeContractPassRepo() *fakeContractPassRepo {
	return &fakeContractPassRepo{
		templates: map[uint]*entity.ContractTemplate{},
		contracts: map[uint]*entity.MemberContract{},
		passes:    map[uint]*entity.DayPass{},
		members:   map[uint]*entity.Member{},
		plans:     map[uint]*entity.MembershipPlan{},
	}
}

func (f *fakeContractPassRepo) ListTemplates(_ context.Context, _ dto.Query, _ string) ([]entity.ContractTemplate, int64, error) {
	var out []entity.ContractTemplate
	for _, t := range f.templates {
		out = append(out, *t)
	}
	return out, int64(len(out)), nil
}

func (f *fakeContractPassRepo) FindTemplateByID(_ context.Context, id uint) (*entity.ContractTemplate, error) {
	if t, ok := f.templates[id]; ok {
		cp := *t
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeContractPassRepo) FindFirstActiveTemplate(_ context.Context) (*entity.ContractTemplate, error) {
	for _, t := range f.templates {
		if t.Status == consts.StatusActive {
			cp := *t
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeContractPassRepo) ExistsTemplateCode(_ context.Context, code string, excludeID uint) (bool, error) {
	for _, t := range f.templates {
		if strings.EqualFold(t.Code, strings.TrimSpace(code)) && t.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeContractPassRepo) CreateTemplate(_ context.Context, t *entity.ContractTemplate) error {
	f.nextTplID++
	t.ID = f.nextTplID
	cp := *t
	f.templates[t.ID] = &cp
	return nil
}

func (f *fakeContractPassRepo) UpdateTemplate(_ context.Context, t *entity.ContractTemplate) error {
	cp := *t
	f.templates[t.ID] = &cp
	return nil
}

func (f *fakeContractPassRepo) DeleteTemplate(_ context.Context, id uint) error {
	delete(f.templates, id)
	return nil
}

func (f *fakeContractPassRepo) ListMemberContracts(_ context.Context, _ dto.Query, memberID uint, _ string) ([]repository.MemberContractRow, int64, error) {
	rows, _ := f.ListContractsByMember(context.Background(), memberID)
	return rows, int64(len(rows)), nil
}

func (f *fakeContractPassRepo) ListContractsByMember(_ context.Context, memberID uint) ([]repository.MemberContractRow, error) {
	var out []repository.MemberContractRow
	for _, mc := range f.contracts {
		if memberID == 0 || mc.MemberID == memberID {
			out = append(out, repository.MemberContractRow{MemberContract: *mc})
		}
	}
	return out, nil
}

func (f *fakeContractPassRepo) FindMemberContractByID(_ context.Context, id uint) (*repository.MemberContractRow, error) {
	if mc, ok := f.contracts[id]; ok {
		return &repository.MemberContractRow{MemberContract: *mc}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeContractPassRepo) CreateMemberContract(_ context.Context, mc *entity.MemberContract) error {
	f.nextContractID++
	mc.ID = f.nextContractID
	cp := *mc
	f.contracts[mc.ID] = &cp
	return nil
}

func (f *fakeContractPassRepo) UpdateMemberContract(_ context.Context, mc *entity.MemberContract) error {
	cp := *mc
	f.contracts[mc.ID] = &cp
	return nil
}

func (f *fakeContractPassRepo) ListDayPasses(_ context.Context, _ dto.Query, _ uint, _ string) ([]repository.DayPassRow, int64, error) {
	var out []repository.DayPassRow
	for _, dp := range f.passes {
		out = append(out, repository.DayPassRow{DayPass: *dp})
	}
	return out, int64(len(out)), nil
}

func (f *fakeContractPassRepo) ListSponsorGuestPasses(_ context.Context, sponsorMemberID uint) ([]repository.DayPassRow, error) {
	var out []repository.DayPassRow
	for _, dp := range f.passes {
		if dp.SponsorMemberID != nil && *dp.SponsorMemberID == sponsorMemberID {
			out = append(out, repository.DayPassRow{DayPass: *dp})
		}
	}
	return out, nil
}

func (f *fakeContractPassRepo) CountSponsorGuestPassesInMonth(_ context.Context, sponsorMemberID uint, _, _ time.Time) (int, error) {
	cnt := 0
	for _, dp := range f.passes {
		if dp.SponsorMemberID != nil && *dp.SponsorMemberID == sponsorMemberID {
			cnt++
		}
	}
	return cnt, nil
}

func (f *fakeContractPassRepo) FindDayPassByID(_ context.Context, id uint) (*repository.DayPassRow, error) {
	if dp, ok := f.passes[id]; ok {
		return &repository.DayPassRow{DayPass: *dp}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeContractPassRepo) CreateDayPass(_ context.Context, dp *entity.DayPass) error {
	f.nextPassID++
	dp.ID = f.nextPassID
	dp.CreatedAt = time.Now()
	cp := *dp
	f.passes[dp.ID] = &cp
	return nil
}

func (f *fakeContractPassRepo) FindMemberByID(_ context.Context, id uint) (*entity.Member, error) {
	if m, ok := f.members[id]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeContractPassRepo) FindMemberByUserID(_ context.Context, userID uint) (*entity.Member, error) {
	for _, m := range f.members {
		if m.UserID != nil && *m.UserID == userID {
			cp := *m
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeContractPassRepo) FindActiveMemberPlan(_ context.Context, memberID uint) (*entity.MembershipPlan, *uint, error) {
	if p, ok := f.plans[memberID]; ok {
		subID := uint(1)
		return p, &subID, nil
	}
	return nil, nil, gorm.ErrRecordNotFound
}

func TestContractPassService_SignAndGuestPassQuota(t *testing.T) {
	repo := newFakeContractPassRepo()
	svc := NewContractPassService(repo, nil, nil)

	uid := uint(101)
	m := &entity.Member{
		UserID:       &uid,
		MemberCode:   "GM000010",
		FullName:     "Trần Thị B",
		Phone:        "0912345678",
		HomeBranchID: 1,
	}
	m.ID = 10
	repo.members[10] = m

	adminCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 999, Roles: []string{consts.RoleOwner}})
	memberCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: uid, Roles: []string{consts.RoleMember}})

	// Tạo mẫu hợp đồng
	_, err := svc.CreateTemplate(adminCtx, v1.ContractTemplateInput{
		Code:     "TERMS_V1",
		Name:     "Hợp đồng hội viên",
		BodyHTML: "<p>Xin chào {{member_name}} ({{member_code}}) - {{member_phone}}</p>",
		Status:   consts.StatusActive,
	})
	require.NoError(t, err)

	// Member gọi danh sách hợp đồng -> tự khởi tạo 1 hợp đồng PENDING đã render HTML
	contracts, err := svc.MemberContracts(memberCtx)
	require.NoError(t, err)
	require.Len(t, contracts, 1)
	assert.Equal(t, entity.ContractStatusPending, contracts[0].Status)
	assert.Contains(t, contracts[0].RenderedHTML, "Trần Thị B")
	assert.Contains(t, contracts[0].RenderedHTML, "GM000010")

	// Member ký hợp đồng
	signed, err := svc.SignMemberContract(memberCtx, &v1.SignMemberContractReq{
		ID:            contracts[0].ID,
		SignatureData: "data:image/png;base64,iVBORw0KGgoAAAANS...",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.ContractStatusSigned, signed.Status)
	assert.NotNil(t, signed.SignedAt)

	// Tạo vé mời bạn bè (mặc định tối đa 2 vé/tháng)
	gp1, err := svc.CreateGuestPass(memberCtx, &v1.CreateGuestPassReq{
		BuyerName: "Bạn thân 1",
		Phone:     "0988111222",
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(gp1.QRToken, "DP-"))
	assert.Equal(t, entity.DayPassTypeGuest, gp1.Type)

	_, err = svc.CreateGuestPass(memberCtx, &v1.CreateGuestPassReq{
		BuyerName: "Bạn thân 2",
		Phone:     "0988333444",
	})
	require.NoError(t, err)

	// Vé thứ 3 trong tháng -> vượt hạn mức (Conflict)
	_, err = svc.CreateGuestPass(memberCtx, &v1.CreateGuestPassReq{
		BuyerName: "Bạn thân 3",
		Phone:     "0988555666",
	})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
}
