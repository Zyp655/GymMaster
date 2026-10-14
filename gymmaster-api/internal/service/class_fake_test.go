package service

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

// ---- store dùng chung cho fake repo lớp nhóm ------------------------------------

type classFakePT struct {
	trainerID  uint
	start, end time.Time
}

type classFakeStore struct {
	mu        sync.Mutex
	nextSch   uint
	nextBk    uint
	schedules map[uint]*entity.ClassSchedule
	bookings  map[uint]*entity.ClassBooking
	members   map[uint]*entity.Member
	subs      map[uint][]repository.ClassBookingSubscription
	types     map[uint]*entity.ClassType
	branches  map[uint]*entity.Branch
	rooms     map[uint]*entity.Room
	trainers  map[uint][]uint // trainerID -> branchIDs
	pts       []classFakePT
}

func newClassFakeStore() *classFakeStore {
	return &classFakeStore{
		schedules: map[uint]*entity.ClassSchedule{},
		bookings:  map[uint]*entity.ClassBooking{},
		members:   map[uint]*entity.Member{},
		subs:      map[uint][]repository.ClassBookingSubscription{},
		types: map[uint]*entity.ClassType{
			1: {BaseEntity: entity.BaseEntity{ID: 1}, Code: "YOGA", Name: "Yoga", Color: "#22C55E", DefaultDurationMin: 60, DefaultCapacity: 20, Level: "ALL", Status: "ACTIVE"},
			2: {BaseEntity: entity.BaseEntity{ID: 2}, Code: "OLD", Name: "Old", Color: "#000000", DefaultDurationMin: 45, DefaultCapacity: 10, Level: "ALL", Status: "INACTIVE"},
		},
		branches: map[uint]*entity.Branch{
			1: {BaseEntity: entity.BaseEntity{ID: 1}, Name: "CN1", OpenTime: "05:00", CloseTime: "22:00", Status: "ACTIVE"},
			2: {BaseEntity: entity.BaseEntity{ID: 2}, Name: "CN2", OpenTime: "06:00", CloseTime: "21:00", Status: "ACTIVE"},
		},
		rooms: map[uint]*entity.Room{
			1: {BaseEntity: entity.BaseEntity{ID: 1}, BranchID: 1, Name: "P1", Capacity: 15, Status: "ACTIVE"},
			2: {BaseEntity: entity.BaseEntity{ID: 2}, BranchID: 1, Name: "P2", Capacity: 30, Status: "ACTIVE"},
			3: {BaseEntity: entity.BaseEntity{ID: 3}, BranchID: 2, Name: "P3", Capacity: 20, Status: "ACTIVE"},
		},
		trainers: map[uint][]uint{10: {1}, 11: {1}, 12: {2}},
	}
}

func (s *classFakeStore) detail(sc *entity.ClassSchedule) *entity.ClassScheduleDetail {
	d := &entity.ClassScheduleDetail{ClassSchedule: *sc}
	if t := s.types[sc.ClassTypeID]; t != nil {
		d.ClassTypeName, d.ClassTypeCode, d.ClassTypeColor = t.Name, t.Code, t.Color
	}
	for _, b := range s.bookings {
		if b.ScheduleID == sc.ID && b.Status == entity.BookingStatusWaitlist {
			d.WaitlistCount++
		}
		if b.ScheduleID == sc.ID && b.Status == entity.BookingStatusAttended {
			d.AttendedCount++
		}
	}
	return d
}

func (s *classFakeStore) bookingDetail(b *entity.ClassBooking) entity.ClassBookingDetail {
	d := entity.ClassBookingDetail{ClassBooking: *b}
	if m := s.members[b.MemberID]; m != nil {
		d.MemberName, d.MemberUserID = m.FullName, m.UserID
	}
	if sc := s.schedules[b.ScheduleID]; sc != nil {
		d.StartAt, d.EndAt, d.BranchID, d.ScheduleStatus, d.ClassTypeID = sc.StartAt, sc.EndAt, sc.BranchID, sc.Status, sc.ClassTypeID
		if t := s.types[sc.ClassTypeID]; t != nil {
			d.ClassTypeName = t.Name
		}
	}
	return d
}

// ---- fake ClassScheduleRepository --------------------------------------------

type classFakeScheduleRepo struct{ *classFakeStore }

func (r classFakeScheduleRepo) List(_ context.Context, f repository.ClassScheduleFilter) ([]entity.ClassScheduleDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []entity.ClassScheduleDetail{}
	for _, sc := range r.schedules {
		if sc.StartAt.Before(f.From) || !sc.StartAt.Before(f.To) {
			continue
		}
		if f.BranchIDs != nil && !classContainsUint(f.BranchIDs, sc.BranchID) {
			continue
		}
		if f.TrainerID > 0 && sc.TrainerID != f.TrainerID {
			continue
		}
		if f.Status != "" && sc.Status != f.Status {
			continue
		}
		out = append(out, *r.detail(sc))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartAt.Before(out[j].StartAt) })
	return out, nil
}

func classContainsUint(xs []uint, v uint) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func (r classFakeScheduleRepo) FindByID(_ context.Context, id uint) (*entity.ClassScheduleDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sc, ok := r.schedules[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return r.detail(sc), nil
}

func (r classFakeScheduleRepo) FindForUpdate(_ context.Context, id uint) (*entity.ClassSchedule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sc, ok := r.schedules[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *sc
	return &cp, nil
}

func (r classFakeScheduleRepo) overlapLocked(sc *entity.ClassSchedule) bool {
	for _, x := range r.schedules {
		if x.ID == sc.ID || x.Status == entity.ClassScheduleCancelled {
			continue
		}
		if (x.RoomID == sc.RoomID || x.TrainerID == sc.TrainerID) && x.StartAt.Before(sc.EndAt) && sc.StartAt.Before(x.EndAt) {
			return true
		}
	}
	return false
}

func (r classFakeScheduleRepo) Create(_ context.Context, sc *entity.ClassSchedule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextSch++
	sc.ID = r.nextSch
	cp := *sc
	r.schedules[sc.ID] = &cp
	return nil
}

func (r classFakeScheduleRepo) CreateIfFree(ctx context.Context, sc *entity.ClassSchedule) (bool, error) {
	r.mu.Lock()
	busy := r.overlapLocked(sc)
	r.mu.Unlock()
	if busy {
		return false, nil
	}
	return true, r.Create(ctx, sc)
}

func (r classFakeScheduleRepo) Update(_ context.Context, sc *entity.ClassSchedule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur := r.schedules[sc.ID]
	cur.RoomID, cur.TrainerID, cur.StartAt, cur.EndAt, cur.Capacity, cur.Note = sc.RoomID, sc.TrainerID, sc.StartAt, sc.EndAt, sc.Capacity, sc.Note
	return nil
}

func (r classFakeScheduleRepo) AddBooked(_ context.Context, id uint, delta int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schedules[id].BookedCount += delta
	return nil
}

func (r classFakeScheduleRepo) MarkCancelled(_ context.Context, id uint, reason string, _ *uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	sc := r.schedules[id]
	sc.Status, sc.CancelReason, sc.BookedCount = entity.ClassScheduleCancelled, reason, 0
	return nil
}

func (r classFakeScheduleRepo) SeriesFrom(_ context.Context, seriesID string, from time.Time) ([]entity.ClassSchedule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []entity.ClassSchedule
	for _, sc := range r.schedules {
		if sc.SeriesID != nil && *sc.SeriesID == seriesID && !sc.StartAt.Before(from) && sc.Status == entity.ClassScheduleScheduled {
			out = append(out, *sc)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartAt.Before(out[j].StartAt) })
	return out, nil
}

func (r classFakeScheduleRepo) CompleteEnded(_ context.Context, endedBefore time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, sc := range r.schedules {
		if sc.Status == entity.ClassScheduleScheduled && sc.EndAt.Before(endedBefore) {
			sc.Status = entity.ClassScheduleCompleted
			n++
		}
	}
	return n, nil
}

func (r classFakeScheduleRepo) overlap(match func(*entity.ClassSchedule) bool, start, end time.Time, excludeID uint) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, x := range r.schedules {
		if x.ID != excludeID && x.Status != entity.ClassScheduleCancelled && match(x) && x.StartAt.Before(end) && start.Before(x.EndAt) {
			return true
		}
	}
	return false
}

func (r classFakeScheduleRepo) RoomOverlap(_ context.Context, roomID uint, start, end time.Time, excludeID uint) (bool, error) {
	return r.overlap(func(x *entity.ClassSchedule) bool { return x.RoomID == roomID }, start, end, excludeID), nil
}

func (r classFakeScheduleRepo) TrainerOverlap(_ context.Context, trainerID uint, start, end time.Time, excludeID uint) (bool, error) {
	return r.overlap(func(x *entity.ClassSchedule) bool { return x.TrainerID == trainerID }, start, end, excludeID), nil
}

func (r classFakeScheduleRepo) TrainerPTOverlap(_ context.Context, trainerID uint, start, end time.Time) (bool, error) {
	for _, p := range r.pts {
		if p.trainerID == trainerID && p.start.Before(end) && start.Before(p.end) {
			return true, nil
		}
	}
	return false, nil
}

func (r classFakeScheduleRepo) FindBranch(_ context.Context, id uint) (*entity.Branch, error) {
	if b, ok := r.branches[id]; ok {
		cp := *b
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r classFakeScheduleRepo) FindRoom(_ context.Context, id uint) (*entity.Room, error) {
	if x, ok := r.rooms[id]; ok {
		cp := *x
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r classFakeScheduleRepo) FindClassType(_ context.Context, id uint) (*entity.ClassType, error) {
	if x, ok := r.types[id]; ok {
		cp := *x
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r classFakeScheduleRepo) TrainerInBranch(_ context.Context, trainerID, branchID uint) (bool, error) {
	return classContainsUint(r.trainers[trainerID], branchID), nil
}

// ---- fake ClassBookingRepository ---------------------------------------------

type bookingFakeRepo struct{ *classFakeStore }

func (r bookingFakeRepo) Create(_ context.Context, b *entity.ClassBooking) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, x := range r.bookings {
		if x.ScheduleID == b.ScheduleID && x.MemberID == b.MemberID &&
			(x.Status == entity.BookingStatusBooked || x.Status == entity.BookingStatusWaitlist || x.Status == entity.BookingStatusAttended) {
			return gorm.ErrDuplicatedKey
		}
	}
	r.nextBk++
	b.ID = r.nextBk
	cp := *b
	r.bookings[b.ID] = &cp
	return nil
}

func (r bookingFakeRepo) FindByID(_ context.Context, id uint) (*entity.ClassBooking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.bookings[id]; ok {
		cp := *b
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r bookingFakeRepo) FindDetail(_ context.Context, id uint) (*entity.ClassBookingDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.bookings[id]; ok {
		d := r.bookingDetail(b)
		return &d, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r bookingFakeRepo) list(match func(*entity.ClassBooking) bool) []entity.ClassBookingDetail {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []entity.ClassBookingDetail{}
	for _, b := range r.bookings {
		if match(b) {
			out = append(out, r.bookingDetail(b))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (r bookingFakeRepo) ListBySchedule(_ context.Context, scheduleID uint) ([]entity.ClassBookingDetail, error) {
	return r.list(func(b *entity.ClassBooking) bool { return b.ScheduleID == scheduleID }), nil
}

func (r bookingFakeRepo) ListByMember(_ context.Context, memberID uint, from, to time.Time) ([]entity.ClassBookingDetail, error) {
	return r.list(func(b *entity.ClassBooking) bool {
		sc := r.schedules[b.ScheduleID]
		return b.MemberID == memberID && !sc.StartAt.Before(from) && sc.StartAt.Before(to)
	}), nil
}

func (r bookingFakeRepo) ListActiveBySchedule(_ context.Context, scheduleID uint) ([]entity.ClassBookingDetail, error) {
	return r.list(func(b *entity.ClassBooking) bool {
		return b.ScheduleID == scheduleID && (b.Status == entity.BookingStatusBooked || b.Status == entity.BookingStatusWaitlist)
	}), nil
}

func (r bookingFakeRepo) ExistsActive(_ context.Context, scheduleID, memberID uint) (bool, error) {
	return len(r.list(func(b *entity.ClassBooking) bool {
		return b.ScheduleID == scheduleID && b.MemberID == memberID &&
			(b.Status == entity.BookingStatusBooked || b.Status == entity.BookingStatusWaitlist || b.Status == entity.BookingStatusAttended)
	})) > 0, nil
}

func (r bookingFakeRepo) MemberOverlap(_ context.Context, memberID uint, start, end time.Time, excludeScheduleID uint) (bool, error) {
	return len(r.list(func(b *entity.ClassBooking) bool {
		sc := r.schedules[b.ScheduleID]
		return b.MemberID == memberID && (b.Status == entity.BookingStatusBooked || b.Status == entity.BookingStatusWaitlist) &&
			sc.Status != entity.ClassScheduleCancelled && sc.ID != excludeScheduleID && sc.StartAt.Before(end) && start.Before(sc.EndAt)
	})) > 0, nil
}

func (r bookingFakeRepo) MaxWaitlistPosition(_ context.Context, scheduleID uint) (int, error) {
	m := 0
	for _, d := range r.list(func(b *entity.ClassBooking) bool {
		return b.ScheduleID == scheduleID && b.Status == entity.BookingStatusWaitlist
	}) {
		if d.WaitlistPosition != nil && *d.WaitlistPosition > m {
			m = *d.WaitlistPosition
		}
	}
	return m, nil
}

func (r bookingFakeRepo) FirstWaitlist(_ context.Context, scheduleID uint) (*entity.ClassBooking, error) {
	items := r.list(func(b *entity.ClassBooking) bool {
		return b.ScheduleID == scheduleID && b.Status == entity.BookingStatusWaitlist
	})
	if len(items) == 0 {
		return nil, nil
	}
	sort.Slice(items, func(i, j int) bool { return *items[i].WaitlistPosition < *items[j].WaitlistPosition })
	b := items[0].ClassBooking
	return &b, nil
}

func (r bookingFakeRepo) UpdateStatus(_ context.Context, b *entity.ClassBooking) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur := r.bookings[b.ID]
	cur.Status, cur.WaitlistPosition, cur.CancelledAt, cur.AttendedAt = b.Status, b.WaitlistPosition, b.CancelledAt, b.AttendedAt
	return nil
}

func (r bookingFakeRepo) MarkNoShows(_ context.Context, endedBefore time.Time) ([]uint, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var ids []uint
	for _, b := range r.bookings {
		sc := r.schedules[b.ScheduleID]
		if b.Status == entity.BookingStatusBooked && sc.Status != entity.ClassScheduleCancelled && sc.EndAt.Before(endedBefore) {
			b.Status = entity.BookingStatusNoShow
			ids = append(ids, b.MemberID)
		}
	}
	return ids, nil
}

func (r bookingFakeRepo) CountNoShows(_ context.Context, memberID uint, since time.Time) (int64, error) {
	return int64(len(r.list(func(b *entity.ClassBooking) bool {
		return b.MemberID == memberID && b.Status == entity.BookingStatusNoShow && !r.schedules[b.ScheduleID].StartAt.Before(since)
	}))), nil
}

func (r bookingFakeRepo) FindMember(_ context.Context, id uint) (*entity.Member, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.members[id]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r bookingFakeRepo) SetMemberBlockedUntil(_ context.Context, memberID uint, until time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.members[memberID].BookingBlockedUntil = &until
	return nil
}

func (r bookingFakeRepo) ActiveSubscriptions(_ context.Context, memberID uint) ([]repository.ClassBookingSubscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]repository.ClassBookingSubscription(nil), r.subs[memberID]...), nil
}

// ---- fake ClassTypeRepository ------------------------------------------------

type classFakeTypeRepo struct {
	*classFakeStore
	future map[uint]int64
}

func (r classFakeTypeRepo) List(context.Context, dto.Query) ([]entity.ClassType, int64, error) {
	return nil, 0, nil
}
func (r classFakeTypeRepo) FindByID(_ context.Context, id uint) (*entity.ClassType, error) {
	if t, ok := r.types[id]; ok {
		cp := *t
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r classFakeTypeRepo) ExistsCode(_ context.Context, code string, excludeID uint) (bool, error) {
	for _, t := range r.types {
		if t.Code == code && t.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}
func (r classFakeTypeRepo) Create(_ context.Context, t *entity.ClassType) error {
	t.ID = uint(len(r.types) + 100)
	cp := *t
	r.types[t.ID] = &cp
	return nil
}
func (r classFakeTypeRepo) Update(_ context.Context, t *entity.ClassType) error {
	cp := *t
	r.types[t.ID] = &cp
	return nil
}
func (r classFakeTypeRepo) Delete(_ context.Context, id uint, _ *uint) error {
	delete(r.types, id)
	return nil
}
func (r classFakeTypeRepo) CountFutureSchedules(_ context.Context, id uint, _ time.Time) (int64, error) {
	return r.future[id], nil
}
func (r classFakeTypeRepo) ActiveOptions(context.Context) ([]entity.ClassType, error) {
	return nil, nil
}

// ---- fake service dùng chung ---------------------------------------------------

type bookingFakeSettings struct{ values map[string]int }

func (f bookingFakeSettings) Int(_ context.Context, key string, def int) int {
	if v, ok := f.values[key]; ok {
		return v
	}
	return def
}
func (f bookingFakeSettings) IntSlice(_ context.Context, _ string, def []int) []int     { return def }
func (f bookingFakeSettings) All(context.Context) ([]entity.Setting, error)             { return nil, nil }
func (f bookingFakeSettings) Set(context.Context, string, json.RawMessage, *uint) error { return nil }

type bookingFakeNotice struct {
	userIDs []uint
	typ     string
}

type bookingFakeNotifier struct {
	mu   sync.Mutex
	sent []bookingFakeNotice
}

func (n *bookingFakeNotifier) Notify(_ context.Context, userIDs []uint, typ, _, _, _ string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, bookingFakeNotice{userIDs: userIDs, typ: typ})
	return nil
}
func (n *bookingFakeNotifier) NotifyPermission(context.Context, string, *uint, string, string, string, string) error {
	return nil
}

type classFakeAudit struct{ actions []string }

func (a *classFakeAudit) Log(_ context.Context, action, entityName string, _ uint, _, _ any) {
	a.actions = append(a.actions, action+":"+entityName)
}

// ---- context -------------------------------------------------------------------

func classOwnerCtx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
}

// classStaffCtx: lễ tân chi nhánh 1.
func classStaffCtx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{
		UserID: 2, Roles: []string{consts.RoleReceptionist}, BranchIDs: []uint{1},
		Permissions: []string{consts.PermClassView, consts.PermBookingManage},
	})
}

// classTrainerCtx: HLV 10 chỉ có vai trò TRAINER tại chi nhánh 1.
func classTrainerCtx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{
		UserID: 10, Roles: []string{consts.RoleTrainer}, BranchIDs: []uint{1},
		Permissions: []string{consts.PermClassView, consts.PermBookingManage},
	})
}
