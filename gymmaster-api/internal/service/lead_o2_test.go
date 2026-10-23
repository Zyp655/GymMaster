package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/lead/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
)

func (f *fakeLeadRepo) FindOpenLeadByPhone(_ context.Context, branchID uint, phone string) (*entity.Lead, error) {
	var found *entity.Lead
	for _, l := range f.leads {
		if l.BranchID != branchID || l.Phone != phone {
			continue
		}
		switch l.Status {
		case entity.LeadStatusNew, entity.LeadStatusContacted, entity.LeadStatusTrial:
			if found == nil || l.ID > found.ID {
				cp := l.Lead
				found = &cp
			}
		}
	}
	return found, nil
}

// LD-02: chi nhánh ngừng hoạt động → 400; LD-08: rate-limit → HTTP 429; FA-04: gộp lead trùng SĐT đang mở.
func TestLeadService_PublicCreate_BranchInactiveRateLimitAndDedup(t *testing.T) {
	repo := newFakeLeadRepo()
	repo.branches[2] = &entity.Branch{Code: "HN02", Name: "CN ngừng", Status: consts.StatusInactive}
	svc := NewLeadService(repo, nil, nil, nil, nil)
	svc.(*leadService).now = func() time.Time { return time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC) }
	ctx := ownerCtx()

	_, err := svc.PublicCreateLead(ctx, &v1.PublicCreateLeadReq{BranchID: 2, FullName: "Khách", Phone: "0988000111"}, "198.51.100.1")
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	assert.Empty(t, repo.leads)

	// Admin tạo lead / cấp vé tập thử ở chi nhánh ngừng hoạt động cũng bị chặn.
	_, err = svc.CreateLead(ctx, &v1.CreateLeadReq{LeadInput: v1.LeadInput{BranchID: 2, FullName: "Khách", Phone: "0988000111"}})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	first, err := svc.PublicCreateLead(ctx, &v1.PublicCreateLeadReq{BranchID: 1, FullName: "Khách A", Phone: "0988000222"}, "198.51.100.2")
	require.NoError(t, err)
	second, err := svc.PublicCreateLead(ctx, &v1.PublicCreateLeadReq{
		BranchID: 1, FullName: "Khách A", Phone: "+84988000222", Email: "a@example.com", Note: "Muốn tập buổi tối",
	}, "198.51.100.2")
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "không tạo lead mới khi đã có lead đang mở cùng SĐT")
	assert.Len(t, repo.leads, 1)
	require.Len(t, repo.activities[first.ID], 1)
	assert.Contains(t, repo.activities[first.ID][0].Content, "Muốn tập buổi tối")
	assert.Equal(t, "a@example.com", repo.leads[first.ID].Email)

	for i := 0; i < 3; i++ {
		_, _ = svc.PublicCreateLead(ctx, &v1.PublicCreateLeadReq{BranchID: 1, FullName: "Khách A", Phone: "0988000222"}, "198.51.100.2")
	}
	_, err = svc.PublicCreateLead(ctx, &v1.PublicCreateLeadReq{BranchID: 1, FullName: "Khách A", Phone: "0988000222"}, "198.51.100.2")
	require.Error(t, err)
	status, _ := rescode.HTTPStatus(codeOf(err))
	assert.Equal(t, http.StatusTooManyRequests, status)
}
