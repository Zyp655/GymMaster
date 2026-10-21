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

func (f *fakeContractPassRepo) ListDayPasses(_ context.Context, _ dto.Query, _ []uint, _ string) ([]repository.DayPassRow, int64, error) {
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

func (f *fakeContractPassRepo) LockMember(context.Context, uint) error { return nil }

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

func (f *fakeContractPassRepo) ListActiveSubPlans(_ context.Context, memberID uint, _ time.Time) ([]repository.ActiveSubPlan, error) {
	if p, ok := f.plans[memberID]; ok {
		sub := entity.Subscription{MemberID: memberID, PlanID: p.ID, BranchID: 1, Status: entity.SubStatusActive}
		return []repository.ActiveSubPlan{{Sub: sub, Plan: *p}}, nil
	}
	return nil, nil
}

func (f *fakeContractPassRepo) FindBranch(_ context.Context, id uint) (*entity.Branch, error) {
	if id == 1 || id == 2 {
		b := &entity.Branch{Name: "CN", Status: consts.StatusActive}
		b.ID = id
		return b, nil
	}
	return nil, gorm.ErrRecordNotFound
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
		SignatureData: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUg==",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.ContractStatusSigned, signed.Status)
	assert.NotNil(t, signed.SignedAt)

	// Ký lại hợp đồng đã ký → Conflict, chữ ký cũ giữ nguyên.
	_, err = svc.SignMemberContract(memberCtx, &v1.SignMemberContractReq{ID: contracts[0].ID, SignatureData: "data:image/png;base64,KHAC"})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	assert.Equal(t, "data:image/png;base64,iVBORw0KGgoAAAANSUhEUg==", repo.contracts[contracts[0].ID].SignatureURL)

	// Chưa có gói tập hiệu lực → hạn mức 0, không tạo được vé mời.
	_, err = svc.CreateGuestPass(memberCtx, &v1.CreateGuestPassReq{BuyerName: "Bạn", Phone: "0988111222"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// Gói tập SINGLE_BRANCH (chi nhánh 1) cho 2 vé mời/tháng.
	plan := &entity.MembershipPlan{Name: "M1", GuestPassesPerMonth: 2, AccessScope: entity.AccessSingleBranch}
	plan.ID = 1
	repo.plans[10] = plan
	gpRes, err := svc.MemberGuestPasses(memberCtx)
	require.NoError(t, err)
	assert.Equal(t, 2, gpRes.Quota)

	// Ngày quá khứ / chi nhánh ngoài phạm vi gói / chi nhánh không tồn tại → 400.
	_, err = svc.CreateGuestPass(memberCtx, &v1.CreateGuestPassReq{BuyerName: "Bạn", Phone: "0988111222", ValidDate: "2000-01-01"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	_, err = svc.CreateGuestPass(memberCtx, &v1.CreateGuestPassReq{BuyerName: "Bạn", Phone: "0988111222", BranchID: 2})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	_, err = svc.CreateGuestPass(memberCtx, &v1.CreateGuestPassReq{BuyerName: "Bạn", Phone: "0988111222", BranchID: 999})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// SĐT khách mời không hợp lệ → 400; SĐT có khoảng trắng/+84 được chuẩn hóa.
	_, err = svc.CreateGuestPass(memberCtx, &v1.CreateGuestPassReq{BuyerName: "Bạn", Phone: "abc123"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// Tạo vé mời bạn bè (tối đa 2 vé/tháng theo gói)
	gp1, err := svc.CreateGuestPass(memberCtx, &v1.CreateGuestPassReq{
		BuyerName: "Bạn thân 1",
		Phone:     "0988111222",
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(gp1.QRToken, "DP-"))
	assert.Equal(t, entity.DayPassTypeGuest, gp1.Type)

	gp2, err := svc.CreateGuestPass(memberCtx, &v1.CreateGuestPassReq{
		BuyerName: "Bạn thân 2",
		Phone:     "+84 988 333 444",
	})
	require.NoError(t, err)
	assert.Equal(t, "0988333444", gp2.Phone)

	// Vé thứ 3 trong tháng -> vượt hạn mức (Conflict)
	_, err = svc.CreateGuestPass(memberCtx, &v1.CreateGuestPassReq{
		BuyerName: "Bạn thân 3",
		Phone:     "0988555666",
	})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
}

func TestContractPassService_DayPassScopeAndValidation(t *testing.T) {
	repo := newFakeContractPassRepo()
	svc := NewContractPassService(repo, nil, nil)
	staffCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 7, Roles: []string{consts.RoleReceptionist}, BranchIDs: []uint{1}})
	ownerCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})

	_, err := svc.CreateDayPass(staffCtx, &v1.CreateDayPassReq{BranchID: 2, BuyerName: "Khách", Phone: "0988111222", Price: 100000})
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err), "lễ tân không bán vé ở chi nhánh ngoài phạm vi")
	_, err = svc.CreateDayPass(ownerCtx, &v1.CreateDayPassReq{BranchID: 999, BuyerName: "Khách", Phone: "0988111222", Price: 100000})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "chi nhánh không tồn tại")
	_, err = svc.CreateDayPass(staffCtx, &v1.CreateDayPassReq{BranchID: 1, BuyerName: "Khách", Phone: "abcdefghij", Price: 100000})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "SĐT không hợp lệ")
	_, err = svc.CreateDayPass(staffCtx, &v1.CreateDayPassReq{BranchID: 1, BuyerName: "Khách", Phone: "0988111222", ValidDate: "2000-01-01"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "ngày quá khứ")
	dp, err := svc.CreateDayPass(staffCtx, &v1.CreateDayPassReq{BranchID: 1, BuyerName: "Khách", Phone: "0988 111 222", Price: 100000})
	require.NoError(t, err)
	assert.Equal(t, "0988111222", dp.Phone)
}

func TestContractPassService_XSSAndSignatureValidation(t *testing.T) {
	repo := newFakeContractPassRepo()
	svc := NewContractPassService(repo, nil, nil)
	uid := uint(202)
	m := &entity.Member{UserID: &uid, MemberCode: "GM000020", FullName: `X<img src=x onerror="window.__pwn=1">`, Phone: `09"><script>1</script>`, HomeBranchID: 1}
	m.ID = 20
	repo.members[20] = m
	adminCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 999, Roles: []string{consts.RoleOwner}})
	memberCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: uid, Roles: []string{consts.RoleMember}})

	// Mẫu chứa payload → lưu bản đã lọc.
	tpl, err := svc.CreateTemplate(adminCtx, v1.ContractTemplateInput{
		Code: "XSS", Name: "XSS", Status: consts.StatusActive,
		BodyHTML: `<p>{{member_name}} {{member_phone}}</p><script>alert(1)</script><svg onload=alert(1)></svg><a href="javascript:alert(1)">l</a><style>*{}</style><img src=x onerror=alert(1)>`,
	})
	require.NoError(t, err)
	for _, bad := range []string{"<script", "onload", "javascript:", "<style", "onerror", "<svg"} {
		assert.NotContains(t, strings.ToLower(tpl.BodyHTML), bad)
		assert.NotContains(t, strings.ToLower(repo.templates[tpl.ID].BodyHTML), bad)
	}
	// Mẫu chỉ toàn thẻ nguy hiểm → rỗng sau khi lọc → 400.
	_, err = svc.CreateTemplate(adminCtx, v1.ContractTemplateInput{Code: "XSS2", Name: "x", BodyHTML: "<script>alert(1)</script>"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// Giá trị thay vào được escape.
	contracts, err := svc.MemberContracts(memberCtx)
	require.NoError(t, err)
	require.Len(t, contracts, 1)
	html := contracts[0].RenderedHTML
	assert.NotContains(t, html, "<img src")
	assert.NotContains(t, html, "<script")
	assert.Contains(t, html, "X&lt;img src=x onerror=&#34;window.__pwn=1&#34;&gt;")

	// Dữ liệu cũ đã lưu chưa escape → lọc khi đọc.
	repo.contracts[contracts[0].ID].RenderedHTML = `<p>X<img src=x onerror="window.__pwn=1"></p>`
	again, err := svc.MemberContracts(memberCtx)
	require.NoError(t, err)
	assert.NotContains(t, again[0].RenderedHTML, "onerror")

	// Chữ ký: chỉ PNG data URL hợp lệ hoặc /uploads/.
	for _, bad := range []string{
		"javascript:alert(1)", "https://evil.example/x.png", "data:image/svg+xml;base64,PHN2Zz4=",
		"data:image/png;base64,!!!!", "data:image/png;base64,R0lGODlh", `/uploads/a.png" onerror="x`, "/uploads/../etc/passwd",
		"data:text/plain;base64,YWJj", "data:image/png;base64," + strings.Repeat("A", 500*1024),
	} {
		_, err = svc.SignMemberContract(memberCtx, &v1.SignMemberContractReq{ID: contracts[0].ID, SignatureData: bad})
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), bad)
	}
	assert.Equal(t, entity.ContractStatusPending, repo.contracts[contracts[0].ID].Status)
	signed, err := svc.SignMemberContract(memberCtx, &v1.SignMemberContractReq{ID: contracts[0].ID, SignatureData: "/uploads/2026/10/sig.png"})
	require.NoError(t, err)
	assert.Equal(t, "/uploads/2026/10/sig.png", signed.SignatureURL)
}
