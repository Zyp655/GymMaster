package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/locker/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	JobExpireLockerRentals = "expire_locker_rentals"
	auditEntityLocker      = "locker"
)

var (
	validLockerSizes        = []string{entity.LockerSizeSmall, entity.LockerSizeStandard, entity.LockerSizeLarge}
	validLockerStatuses     = []string{entity.LockerStatusAvailable, entity.LockerStatusOccupied, entity.LockerStatusRented, entity.LockerStatusBroken}
	validLockerRentalTypes  = []string{entity.LockerRentalDaily, entity.LockerRentalMonthly}
	defaultLockerMonthlyFee = 150000.0
)

type (
	LockerService interface {
		ListLockers(ctx context.Context, req *v1.ListLockersReq) (v1.ListLockersRes, error)
		CreateLocker(ctx context.Context, req *v1.CreateLockerReq) (*v1.LockerItem, error)
		UpdateLocker(ctx context.Context, req *v1.UpdateLockerReq) (*v1.LockerItem, error)
		DeleteLocker(ctx context.Context, id uint) error
		GetBoard(ctx context.Context, req *v1.GetLockerBoardReq) (*v1.GetLockerBoardRes, error)
		AssignLocker(ctx context.Context, req *v1.AssignLockerReq) (*v1.LockerRentalItem, error)
		ReleaseLocker(ctx context.Context, id uint) (*v1.LockerItem, error)
		CreateRental(ctx context.Context, req *v1.CreateLockerRentalReq) (*v1.LockerRentalItem, error)
		ListRentals(ctx context.Context, req *v1.ListLockerRentalsReq) (v1.ListLockerRentalsRes, error)
		MemberLockers(ctx context.Context) (v1.MemberLockersRes, error)
		ExpireDueRentals(ctx context.Context, now time.Time) (int, error)
	}

	lockerService struct {
		repo  repository.LockerRepository
		tx    postgres.TxManager
		audit AuditService
		now   func() time.Time
	}
)

func NewLockerService(repo repository.LockerRepository, tx postgres.TxManager, audit AuditService) LockerService {
	return &lockerService{
		repo:  repo,
		tx:    tx,
		audit: audit,
		now:   time.Now,
	}
}

func (s *lockerService) ListLockers(ctx context.Context, req *v1.ListLockersReq) (v1.ListLockersRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem tủ đồ tại chi nhánh này")
	}
	rows, err := s.repo.ListLockers(ctx, repository.LockerFilter{
		BranchIDs: branchIDs,
		Zone:      strings.TrimSpace(req.Zone),
		Size:      strings.ToUpper(strings.TrimSpace(req.Size)),
		Status:    strings.ToUpper(strings.TrimSpace(req.Status)),
	})
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toLockerItem), nil
}

func (s *lockerService) CreateLocker(ctx context.Context, req *v1.CreateLockerReq) (*v1.LockerItem, error) {
	if req.BranchID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn chi nhánh")
	}
	if !reqctx.CanAccessBranch(ctx, req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được tạo tủ đồ tại chi nhánh này")
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập mã tủ")
	}
	zone := strings.TrimSpace(req.Zone)
	if zone == "" {
		zone = "Khu A"
	}
	size := strings.ToUpper(strings.TrimSpace(req.Size))
	if size == "" {
		size = entity.LockerSizeStandard
	}
	if !slices.Contains(validLockerSizes, size) {
		return nil, rescode.New(rescode.BadRequest, "Kích cỡ tủ không hợp lệ")
	}
	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if status == "" {
		status = entity.LockerStatusAvailable
	}
	if !slices.Contains(validLockerStatuses, status) {
		return nil, rescode.New(rescode.BadRequest, "Trạng thái tủ không hợp lệ")
	}
	fee := req.MonthlyFee
	if fee < 0 {
		return nil, rescode.New(rescode.BadRequest, "Phí thuê tháng không được âm")
	}
	if fee == 0 {
		fee = defaultLockerMonthlyFee
	}
	dup, err := s.repo.ExistsLockerCode(ctx, req.BranchID, code, 0)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, rescode.New(rescode.Conflict, "Mã tủ đồ đã tồn tại tại chi nhánh này")
	}

	now := s.now()
	l := &entity.Locker{
		BranchID:   req.BranchID,
		Code:       code,
		Zone:       zone,
		Size:       size,
		MonthlyFee: fee,
		Status:     status,
		Note:       strings.TrimSpace(req.Note),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.CreateLocker(ctx, l); err != nil {
		return nil, mapDuplicate(err, "Mã tủ đồ đã tồn tại")
	}
	res := toLockerItem(repository.LockerRow{Locker: *l})
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityLocker, l.ID, nil, res)
	}
	return res, nil
}

func (s *lockerService) UpdateLocker(ctx context.Context, req *v1.UpdateLockerReq) (*v1.LockerItem, error) {
	row, err := s.repo.FindLockerByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy tủ đồ")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được thao tác tủ đồ tại chi nhánh này")
	}
	l := row.Locker
	before := toLockerItem(*row)
	if code := strings.ToUpper(strings.TrimSpace(req.Code)); code != "" && code != l.Code {
		dup, err := s.repo.ExistsLockerCode(ctx, l.BranchID, code, l.ID)
		if err != nil {
			return nil, err
		}
		if dup {
			return nil, rescode.New(rescode.Conflict, "Mã tủ đồ đã tồn tại tại chi nhánh này")
		}
		l.Code = code
	}
	if zone := strings.TrimSpace(req.Zone); zone != "" {
		l.Zone = zone
	}
	if size := strings.ToUpper(strings.TrimSpace(req.Size)); size != "" {
		if !slices.Contains(validLockerSizes, size) {
			return nil, rescode.New(rescode.BadRequest, "Kích cỡ tủ không hợp lệ")
		}
		l.Size = size
	}
	if req.MonthlyFee != nil {
		if *req.MonthlyFee < 0 {
			return nil, rescode.New(rescode.BadRequest, "Phí thuê tháng không được âm")
		}
		l.MonthlyFee = *req.MonthlyFee
	}
	if status := strings.ToUpper(strings.TrimSpace(req.Status)); status != "" {
		if !slices.Contains(validLockerStatuses, status) {
			return nil, rescode.New(rescode.BadRequest, "Trạng thái tủ không hợp lệ")
		}
		l.Status = status
	}
	if req.Note != nil {
		l.Note = strings.TrimSpace(*req.Note)
	}
	l.UpdatedAt = s.now()
	if err := s.repo.UpdateLocker(ctx, &l); err != nil {
		return nil, err
	}
	updatedRow := *row
	updatedRow.Locker = l
	res := toLockerItem(updatedRow)
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityLocker, l.ID, before, res)
	}
	return res, nil
}

func (s *lockerService) DeleteLocker(ctx context.Context, id uint) error {
	row, err := s.repo.FindLockerByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy tủ đồ")
		}
		return err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return rescode.New(rescode.PermissionDenied, "Bạn không được xóa tủ đồ tại chi nhánh này")
	}
	if row.Status == entity.LockerStatusOccupied || row.Status == entity.LockerStatusRented {
		return rescode.New(rescode.LockerUnavailable, "Tủ đồ đang có người sử dụng, không thể xóa")
	}
	if err := s.repo.DeleteLocker(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityLocker, id, toLockerItem(*row), nil)
	}
	return nil
}

func (s *lockerService) GetBoard(ctx context.Context, req *v1.GetLockerBoardReq) (*v1.GetLockerBoardRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem sơ đồ tủ tại chi nhánh này")
	}
	rows, err := s.repo.ListLockers(ctx, repository.LockerFilter{BranchIDs: branchIDs})
	if err != nil {
		return nil, err
	}
	items := dto.Map(rows, toLockerItem)
	stats := &v1.LockerBoardStats{Total: len(items)}
	for _, it := range items {
		switch it.Status {
		case entity.LockerStatusAvailable:
			stats.Available++
		case entity.LockerStatusOccupied:
			stats.Occupied++
		case entity.LockerStatusRented:
			stats.Rented++
		case entity.LockerStatusBroken:
			stats.Broken++
		}
	}
	return &v1.GetLockerBoardRes{
		Available: stats.Available,
		Occupied:  stats.Occupied,
		Rented:    stats.Rented,
		Broken:    stats.Broken,
		Total:     stats.Total,
		Stats:     stats,
		Lockers:   items,
		Items:     items,
	}, nil
}

func (s *lockerService) AssignLocker(ctx context.Context, req *v1.AssignLockerReq) (*v1.LockerRentalItem, error) {
	row, err := s.repo.FindLockerByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy tủ đồ")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được thao tác tủ đồ tại chi nhánh này")
	}
	if row.Status != entity.LockerStatusAvailable {
		return nil, rescode.New(rescode.LockerUnavailable, "Tủ đồ hiện không khả dụng (đang được sử dụng hoặc bị hỏng)")
	}
	m, err := s.repo.FindMemberByID(ctx, req.MemberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy hội viên")
		}
		return nil, err
	}

	rentalType := strings.ToUpper(strings.TrimSpace(req.Type))
	if rentalType == "" {
		rentalType = entity.LockerRentalDaily
	}
	if !slices.Contains(validLockerRentalTypes, rentalType) {
		return nil, rescode.New(rescode.BadRequest, "Loại thuê tủ không hợp lệ (DAILY hoặc MONTHLY)")
	}

	now := s.now()
	var (
		newStatus string
		endAt     *time.Time
		price     float64
	)
	if rentalType == entity.LockerRentalMonthly {
		months := req.Months
		if months <= 0 {
			months = 1
		}
		e := now.Add(time.Duration(months) * 30 * 24 * time.Hour)
		endAt = &e
		price = row.MonthlyFee * float64(months)
		newStatus = entity.LockerStatusRented
	} else {
		e := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		endAt = &e
		price = 0
		newStatus = entity.LockerStatusOccupied
	}

	rental := &entity.LockerRental{
		LockerID:  row.ID,
		MemberID:  m.ID,
		Type:      rentalType,
		StartAt:   now,
		EndAt:     endAt,
		Price:     price,
		Status:    entity.LockerRentalActive,
		Note:      strings.TrimSpace(req.Note),
		CreatedBy: reqctx.UserID(ctx),
		CreatedAt: now,
		UpdatedAt: now,
	}
	l := row.Locker
	l.Status = newStatus
	l.UpdatedAt = now

	run := func(txCtx context.Context) error {
		if err := s.repo.CreateRental(txCtx, rental); err != nil {
			return err
		}
		return s.repo.UpdateLocker(txCtx, &l)
	}
	if s.tx != nil {
		err = s.tx.WithTx(ctx, run)
	} else {
		err = run(ctx)
	}
	if err != nil {
		return nil, err
	}

	return toLockerRentalItem(repository.LockerRentalRow{
		LockerRental: *rental,
		LockerCode:   row.Code,
		Zone:         row.Zone,
		Size:         row.Size,
		BranchID:     row.BranchID,
		BranchName:   row.BranchName,
		MemberCode:   m.MemberCode,
		MemberName:   m.FullName,
	}), nil
}

func (s *lockerService) ReleaseLocker(ctx context.Context, id uint) (*v1.LockerItem, error) {
	row, err := s.repo.FindLockerByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy tủ đồ")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được thao tác tủ đồ tại chi nhánh này")
	}
	now := s.now()
	l := row.Locker
	l.Status = entity.LockerStatusAvailable
	l.UpdatedAt = now

	run := func(txCtx context.Context) error {
		if err := s.repo.EndActiveRentalsByLocker(txCtx, id, now); err != nil {
			return err
		}
		return s.repo.UpdateLocker(txCtx, &l)
	}
	if s.tx != nil {
		err = s.tx.WithTx(ctx, run)
	} else {
		err = run(ctx)
	}
	if err != nil {
		return nil, err
	}
	return toLockerItem(repository.LockerRow{Locker: l, BranchName: row.BranchName}), nil
}

func (s *lockerService) CreateRental(ctx context.Context, req *v1.CreateLockerRentalReq) (*v1.LockerRentalItem, error) {
	return s.AssignLocker(ctx, &v1.AssignLockerReq{
		ID:       req.LockerID,
		MemberID: req.MemberID,
		Type:     req.Type,
		Months:   req.Months,
		Note:     req.Note,
	})
}

func (s *lockerService) ListRentals(ctx context.Context, req *v1.ListLockerRentalsReq) (v1.ListLockerRentalsRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem lượt thuê tủ tại chi nhánh này")
	}
	rows, err := s.repo.ListRentals(ctx, repository.LockerRentalFilter{
		BranchIDs: branchIDs,
		LockerID:  req.LockerID,
		MemberID:  req.MemberID,
		Type:      strings.ToUpper(strings.TrimSpace(req.Type)),
		Status:    strings.ToUpper(strings.TrimSpace(req.Status)),
	})
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toLockerRentalItem), nil
}

func (s *lockerService) MemberLockers(ctx context.Context) (v1.MemberLockersRes, error) {
	uid := reqctx.UserID(ctx)
	if uid == nil {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	m, err := s.repo.FindMemberByUserID(ctx, *uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Tài khoản chưa được liên kết với hội viên")
		}
		return nil, err
	}
	rows, err := s.repo.ListRentals(ctx, repository.LockerRentalFilter{
		MemberID: m.ID,
		Status:   entity.LockerRentalActive,
	})
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toLockerRentalItem), nil
}

func (s *lockerService) ExpireDueRentals(ctx context.Context, now time.Time) (int, error) {
	expired, err := s.repo.ListExpiredActiveRentals(ctx, now)
	if err != nil {
		return 0, err
	}
	if len(expired) == 0 {
		return 0, nil
	}
	var count int
	var firstErr error
	for _, r := range expired {
		if err := s.repo.ExpireRental(ctx, r.ID, r.LockerID, now); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		count++
	}
	return count, firstErr
}

func toLockerItem(r repository.LockerRow) *v1.LockerItem {
	return &v1.LockerItem{
		ID:         r.ID,
		BranchID:   r.BranchID,
		BranchName: r.BranchName,
		Code:       r.Code,
		Zone:       r.Zone,
		Size:       r.Size,
		MonthlyFee: r.MonthlyFee,
		Status:     r.Status,
		Note:       r.Note,
		RentalID:   r.RentalID,
		MemberID:   r.MemberID,
		MemberName: r.MemberName,
		MemberCode: r.MemberCode,
		RentalType: r.RentalType,
		StartAt:    r.StartAt,
		EndAt:      r.EndAt,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

func toLockerRentalItem(r repository.LockerRentalRow) *v1.LockerRentalItem {
	return &v1.LockerRentalItem{
		ID:         r.ID,
		LockerID:   r.LockerID,
		LockerCode: r.LockerCode,
		Zone:       r.Zone,
		Size:       r.Size,
		BranchID:   r.BranchID,
		BranchName: r.BranchName,
		MemberID:   r.MemberID,
		MemberCode: r.MemberCode,
		MemberName: r.MemberName,
		Type:       r.Type,
		StartAt:    r.StartAt,
		EndAt:      r.EndAt,
		InvoiceID:  r.InvoiceID,
		Price:      r.Price,
		Status:     r.Status,
		Note:       r.Note,
		CreatedBy:  r.CreatedBy,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}
