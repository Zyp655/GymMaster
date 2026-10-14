package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/room/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

type (
	RoomService interface {
		List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error)
		Get(ctx context.Context, id uint) (*v1.Room, error)
		Create(ctx context.Context, in v1.RoomInput) (*v1.Room, error)
		Update(ctx context.Context, id uint, in v1.RoomInput) (*v1.Room, error)
		Delete(ctx context.Context, id uint) error
		Options(ctx context.Context, branchID uint) ([]*v1.RoomOption, error)
	}

	roomService struct {
		roomRepo repository.RoomRepository
		now      func() time.Time
	}
)

const roomNameConflictMsg = "Tên phòng đã tồn tại trong chi nhánh"

func NewRoomService(roomRepo repository.RoomRepository) RoomService {
	return &roomService{roomRepo: roomRepo, now: time.Now}
}

var roomOrderable = []string{"id", "name", "capacity", "status", "branch_id", "created_at"}

func (s *roomService) List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem phòng tập của chi nhánh này")
	}
	q := req.PageReq.ToQuery(roomOrderable, "id")
	if req.Status != "" {
		q.Filters["status"] = req.Status
	}
	items, total, err := s.roomRepo.List(ctx, q, branchIDs)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(items, toRoomRes), total, q), nil
}

func (s *roomService) Get(ctx context.Context, id uint) (*v1.Room, error) {
	r, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	return toRoomRes(*r), nil
}

func (s *roomService) Create(ctx context.Context, in v1.RoomInput) (*v1.Room, error) {
	in.Name = strings.TrimSpace(in.Name)
	if err := s.validate(ctx, in, 0); err != nil {
		return nil, err
	}
	r := &entity.Room{}
	applyRoomInput(r, in)
	r.CreatedBy = reqctx.UserID(ctx)
	r.UpdatedBy = r.CreatedBy
	if err := s.roomRepo.Create(ctx, r); err != nil {
		return nil, mapDuplicate(err, roomNameConflictMsg)
	}
	return s.Get(ctx, r.ID)
}

func (s *roomService) Update(ctx context.Context, id uint, in v1.RoomInput) (*v1.Room, error) {
	r, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	in.Name = strings.TrimSpace(in.Name)
	if err := s.validate(ctx, in, id); err != nil {
		return nil, err
	}
	applyRoomInput(r, in)
	r.UpdatedBy = reqctx.UserID(ctx)
	if err := s.roomRepo.Update(ctx, r); err != nil {
		return nil, mapDuplicate(err, roomNameConflictMsg)
	}
	return s.Get(ctx, id)
}

func (s *roomService) Delete(ctx context.Context, id uint) error {
	if _, err := s.find(ctx, id); err != nil {
		return err
	}
	n, err := s.roomRepo.CountFutureSchedules(ctx, id, s.now())
	if err != nil {
		return err
	}
	if n > 0 {
		return rescode.New(rescode.Conflict, "Không thể xóa: phòng còn lịch lớp sắp diễn ra")
	}
	return s.roomRepo.Delete(ctx, id, reqctx.UserID(ctx))
}

func (s *roomService) Options(ctx context.Context, branchID uint) ([]*v1.RoomOption, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, branchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem phòng tập của chi nhánh này")
	}
	items, err := s.roomRepo.ActiveOptions(ctx, branchIDs)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.RoomOption, 0, len(items))
	for _, r := range items {
		out = append(out, &v1.RoomOption{ID: r.ID, BranchID: r.BranchID, Name: r.Name, Capacity: r.Capacity})
	}
	return out, nil
}

// find trả phòng nếu người dùng được thao tác trên chi nhánh của phòng (ngoài phạm vi → 404).
func (s *roomService) find(ctx context.Context, id uint) (*entity.Room, error) {
	r, err := s.roomRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy phòng tập")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, r.BranchID) {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy phòng tập")
	}
	return r, nil
}

func (s *roomService) validate(ctx context.Context, in v1.RoomInput, excludeID uint) error {
	if in.Name == "" {
		return rescode.New(rescode.BadRequest, "Vui lòng nhập tên phòng")
	}
	if in.Capacity < 1 {
		return rescode.New(rescode.BadRequest, "Sức chứa phải lớn hơn 0")
	}
	if !reqctx.CanAccessBranch(ctx, in.BranchID) {
		return rescode.New(rescode.PermissionDenied, "Bạn không có quyền quản lý phòng tập của chi nhánh này")
	}
	ok, err := s.roomRepo.BranchExists(ctx, in.BranchID)
	if err != nil {
		return err
	}
	if !ok {
		return rescode.New(rescode.BadRequest, "Chi nhánh không tồn tại")
	}
	exists, err := s.roomRepo.ExistsName(ctx, in.BranchID, in.Name, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return rescode.New(rescode.Conflict, roomNameConflictMsg)
	}
	return nil
}

func applyRoomInput(r *entity.Room, in v1.RoomInput) {
	r.BranchID = in.BranchID
	r.Name = in.Name
	r.Capacity = in.Capacity
	r.Description = strings.TrimSpace(in.Description)
	r.Status = in.Status
	if r.Status == "" {
		r.Status = "ACTIVE"
	}
}

func toRoomRes(r entity.Room) *v1.Room {
	res := &v1.Room{
		ID:          r.ID,
		BranchID:    r.BranchID,
		Name:        r.Name,
		Capacity:    r.Capacity,
		Description: r.Description,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	if r.Branch != nil {
		res.BranchName = r.Branch.Name
	}
	return res
}
