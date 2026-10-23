package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/locker/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

type fakeLockerRepo struct {
	nextLockerID uint
	nextRentalID uint
	lockers      map[uint]*entity.Locker
	rentals      map[uint]*entity.LockerRental
	members      map[uint]*entity.Member
}

func newFakeLockerRepo() *fakeLockerRepo {
	uid := uint(101)
	return &fakeLockerRepo{
		lockers: map[uint]*entity.Locker{},
		rentals: map[uint]*entity.LockerRental{},
		members: map[uint]*entity.Member{
			10: {BaseEntity: entity.BaseEntity{ID: 10}, UserID: &uid, MemberCode: "HV00010", FullName: "Trần Thị B", Status: entity.MemberStatusActive},
		},
	}
}

func (f *fakeLockerRepo) ListLockers(_ context.Context, filter repository.LockerFilter) ([]repository.LockerRow, error) {
	var out []repository.LockerRow
	for _, l := range f.lockers {
		if filter.Status != "" && l.Status != filter.Status {
			continue
		}
		row := repository.LockerRow{Locker: *l, BranchName: "CN Quận 1"}
		for _, r := range f.rentals {
			if r.LockerID == l.ID && r.Status == entity.LockerRentalActive {
				rid := r.ID
				mid := r.MemberID
				st := r.StartAt
				row.RentalID = &rid
				row.MemberID = &mid
				row.RentalType = r.Type
				row.StartAt = &st
				row.EndAt = r.EndAt
				if m, ok := f.members[r.MemberID]; ok {
					row.MemberName = m.FullName
					row.MemberCode = m.MemberCode
				}
				break
			}
		}
		out = append(out, row)
	}
	return out, nil
}

func (f *fakeLockerRepo) FindLockerByID(_ context.Context, id uint) (*repository.LockerRow, error) {
	if l, ok := f.lockers[id]; ok {
		return &repository.LockerRow{Locker: *l, BranchName: "CN Quận 1"}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeLockerRepo) ExistsLockerCode(_ context.Context, branchID uint, code string, excludeID uint) (bool, error) {
	for _, l := range f.lockers {
		if l.BranchID == branchID && strings.EqualFold(l.Code, strings.TrimSpace(code)) && l.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeLockerRepo) CreateLocker(_ context.Context, l *entity.Locker) error {
	f.nextLockerID++
	l.ID = f.nextLockerID
	cp := *l
	f.lockers[l.ID] = &cp
	return nil
}

func (f *fakeLockerRepo) UpdateLocker(_ context.Context, l *entity.Locker) error {
	cp := *l
	f.lockers[l.ID] = &cp
	return nil
}

func (f *fakeLockerRepo) DeleteLocker(_ context.Context, id uint) error {
	delete(f.lockers, id)
	return nil
}

func (f *fakeLockerRepo) FindMemberByID(_ context.Context, id uint) (*entity.Member, error) {
	if m, ok := f.members[id]; ok {
		return m, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeLockerRepo) FindMemberByUserID(_ context.Context, userID uint) (*entity.Member, error) {
	for _, m := range f.members {
		if m.UserID != nil && *m.UserID == userID {
			return m, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeLockerRepo) CreateRental(_ context.Context, r *entity.LockerRental) error {
	f.nextRentalID++
	r.ID = f.nextRentalID
	cp := *r
	f.rentals[r.ID] = &cp
	return nil
}

func (f *fakeLockerRepo) EndActiveRentalsByLocker(_ context.Context, lockerID uint, endedAt time.Time) error {
	for _, r := range f.rentals {
		if r.LockerID == lockerID && r.Status == entity.LockerRentalActive {
			r.Status = entity.LockerRentalEnded
			t := endedAt
			r.EndAt = &t
		}
	}
	return nil
}

func (f *fakeLockerRepo) ListRentals(_ context.Context, filter repository.LockerRentalFilter) ([]repository.LockerRentalRow, error) {
	var out []repository.LockerRentalRow
	for _, r := range f.rentals {
		if filter.MemberID > 0 && r.MemberID != filter.MemberID {
			continue
		}
		if filter.Status != "" && r.Status != filter.Status {
			continue
		}
		l := f.lockers[r.LockerID]
		row := repository.LockerRentalRow{LockerRental: *r}
		if l != nil {
			row.LockerCode = l.Code
			row.Zone = l.Zone
			row.Size = l.Size
			row.BranchID = l.BranchID
		}
		if m, ok := f.members[r.MemberID]; ok {
			row.MemberCode = m.MemberCode
			row.MemberName = m.FullName
		}
		out = append(out, row)
	}
	return out, nil
}

func (f *fakeLockerRepo) ListExpiredActiveRentals(_ context.Context, now time.Time) ([]entity.LockerRental, error) {
	var out []entity.LockerRental
	for _, r := range f.rentals {
		if r.Status == entity.LockerRentalActive && r.EndAt != nil && r.EndAt.Before(now) {
			out = append(out, *r)
		}
	}
	return out, nil
}

func (f *fakeLockerRepo) ExpireRental(_ context.Context, rentalID, lockerID uint, _ time.Time) error {
	if r, ok := f.rentals[rentalID]; ok {
		r.Status = entity.LockerRentalExpired
	}
	if l, ok := f.lockers[lockerID]; ok {
		l.Status = entity.LockerStatusAvailable
	}
	return nil
}

func TestLockerService_AssignReleaseBoardAndExpire(t *testing.T) {
	repo := newFakeLockerRepo()
	svc := NewLockerService(repo, nil, nil)
	fixedNow := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	svc.(*lockerService).now = func() time.Time { return fixedNow }

	adminCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	memberCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 101, Roles: []string{consts.RoleMember}})

	// Create 2 lockers
	l1, err := svc.CreateLocker(adminCtx, &v1.CreateLockerReq{
		BranchID:   1,
		Code:       "A-01",
		Zone:       "Khu Nam",
		Size:       "STANDARD",
		MonthlyFee: lockerFeePtr(150000),
	})
	require.NoError(t, err)

	l2, err := svc.CreateLocker(adminCtx, &v1.CreateLockerReq{
		BranchID:   1,
		Code:       "A-02",
		Zone:       "Khu Nam",
		Size:       "LARGE",
		MonthlyFee: lockerFeePtr(200000),
	})
	require.NoError(t, err)

	// Assign l1 as DAILY -> OCCUPIED
	r1, err := svc.AssignLocker(adminCtx, &v1.AssignLockerReq{
		ID:       l1.ID,
		MemberID: 10,
		Type:     entity.LockerRentalDaily,
	})
	require.NoError(t, err)
	assert.Equal(t, entity.LockerRentalDaily, r1.Type)
	assert.Equal(t, entity.LockerStatusOccupied, repo.lockers[l1.ID].Status)

	// Assigning an already OCCUPIED locker -> LockerUnavailable
	_, err = svc.AssignLocker(adminCtx, &v1.AssignLockerReq{
		ID:       l1.ID,
		MemberID: 10,
		Type:     entity.LockerRentalDaily,
	})
	assert.Equal(t, rescode.LockerUnavailable.Code(), codeOf(err))

	// Assign l2 as MONTHLY (2 months) -> RENTED, price = 400000, endAt = now + 60 days
	r2, err := svc.AssignLocker(adminCtx, &v1.AssignLockerReq{
		ID:       l2.ID,
		MemberID: 10,
		Type:     entity.LockerRentalMonthly,
		Months:   2,
	})
	require.NoError(t, err)
	assert.Equal(t, float64(400000), r2.Price)
	assert.Equal(t, entity.LockerStatusRented, repo.lockers[l2.ID].Status)
	require.NotNil(t, r2.EndAt)
	assert.Equal(t, fixedNow.Add(60*24*time.Hour), *r2.EndAt)

	// Board stats: 1 occupied, 1 rented
	board, err := svc.GetBoard(adminCtx, &v1.GetLockerBoardReq{BranchID: 1})
	require.NoError(t, err)
	assert.Equal(t, 0, board.Available)
	assert.Equal(t, 1, board.Occupied)
	assert.Equal(t, 1, board.Rented)

	// Member sees 2 active lockers
	myLockers, err := svc.MemberLockers(memberCtx)
	require.NoError(t, err)
	assert.Len(t, myLockers, 2)

	// Release l1 -> AVAILABLE
	rel, err := svc.ReleaseLocker(adminCtx, l1.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.LockerStatusAvailable, rel.Status)

	// Expire due rentals after 61 days -> l2 becomes AVAILABLE
	expiredCount, err := svc.ExpireDueRentals(adminCtx, fixedNow.Add(61*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, expiredCount)
	assert.Equal(t, entity.LockerStatusAvailable, repo.lockers[l2.ID].Status)
}

func TestCheckLockerStatusChange(t *testing.T) {
	rid := uint(1)
	rented := &repository.LockerRow{RentalID: &rid, RentalType: entity.LockerRentalMonthly}
	free := &repository.LockerRow{}

	assert.Error(t, checkLockerStatusChange(rented, entity.LockerStatusAvailable), "tủ đang thuê không được đặt AVAILABLE")
	assert.Error(t, checkLockerStatusChange(rented, entity.LockerStatusOccupied))
	assert.NoError(t, checkLockerStatusChange(rented, entity.LockerStatusBroken))
	assert.NoError(t, checkLockerStatusChange(rented, entity.LockerStatusRented))

	assert.NoError(t, checkLockerStatusChange(free, entity.LockerStatusAvailable))
	assert.NoError(t, checkLockerStatusChange(free, entity.LockerStatusBroken))
	assert.Error(t, checkLockerStatusChange(free, entity.LockerStatusRented), "RENTED chỉ có qua gán tủ")
}

func (f *fakeLockerRepo) EndRentalForVoidedInvoice(_ context.Context, rentalID uint, now time.Time) (bool, error) {
	r, ok := f.rentals[rentalID]
	if !ok || r.Status != entity.LockerRentalActive {
		return false, nil
	}
	r.Status, r.EndAt = entity.LockerRentalEnded, &now
	if l, ok := f.lockers[r.LockerID]; ok && l.Status != entity.LockerStatusBroken {
		l.Status = entity.LockerStatusAvailable
	}
	return true, nil
}

func TestLockerService_InvoiceVoidEndsRental(t *testing.T) {
	repo := newFakeLockerRepo()
	svc := NewLockerService(repo, nil, nil).(*lockerService)
	now := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	repo.lockers[1] = &entity.Locker{ID: 1, BranchID: 1, Code: "A-01", Status: entity.LockerStatusRented}
	repo.lockers[2] = &entity.Locker{ID: 2, BranchID: 1, Code: "A-02", Status: entity.LockerStatusBroken}
	repo.rentals[5] = &entity.LockerRental{ID: 5, LockerID: 1, MemberID: 10, Type: entity.LockerRentalMonthly, Status: entity.LockerRentalActive}
	repo.rentals[6] = &entity.LockerRental{ID: 6, LockerID: 2, MemberID: 10, Type: entity.LockerRentalMonthly, Status: entity.LockerRentalActive}
	inv := &entity.Invoice{}
	ref := func(id uint) *entity.InvoiceItem {
		return &entity.InvoiceItem{ItemType: entity.InvoiceItemLocker, RefID: &id}
	}

	require.NoError(t, svc.onInvoiceVoid(context.Background(), inv, ref(5)))
	assert.Equal(t, entity.LockerRentalEnded, repo.rentals[5].Status)
	assert.Equal(t, entity.LockerStatusAvailable, repo.lockers[1].Status)

	require.NoError(t, svc.onInvoiceVoid(context.Background(), inv, ref(6)))
	assert.Equal(t, entity.LockerRentalEnded, repo.rentals[6].Status)
	assert.Equal(t, entity.LockerStatusBroken, repo.lockers[2].Status, "tủ hỏng giữ BROKEN")

	// Idempotent + dòng không có ref_id.
	require.NoError(t, svc.onInvoiceVoid(context.Background(), inv, ref(5)))
	require.NoError(t, svc.onInvoiceVoid(context.Background(), inv, &entity.InvoiceItem{ItemType: entity.InvoiceItemLocker}))
}

func TestLockerService_CreateRespectsZeroFee(t *testing.T) {
	repo := newFakeLockerRepo()
	svc := NewLockerService(repo, nil, nil)
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	l, err := svc.CreateLocker(ctx, &v1.CreateLockerReq{BranchID: 1, Code: "free-01", MonthlyFee: new(float64)})
	require.NoError(t, err)
	assert.Equal(t, 0.0, l.MonthlyFee, "phí 0 phải được giữ nguyên")
	def, err := svc.CreateLocker(ctx, &v1.CreateLockerReq{BranchID: 1, Code: "def-01"})
	require.NoError(t, err)
	assert.Equal(t, 150000.0, def.MonthlyFee, "bỏ trống → 150.000đ")
	neg := -1.0
	_, err = svc.CreateLocker(ctx, &v1.CreateLockerReq{BranchID: 1, Code: "neg-01", MonthlyFee: &neg})
	require.Error(t, err)
}

func lockerFeePtr(v float64) *float64 { return &v }
