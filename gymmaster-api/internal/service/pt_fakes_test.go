package service

import (
	"context"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

// ---- Fake dùng cho test module PT (tiền tố pt) -----------------------------------------

type ptFakeTrainer struct {
	name     string
	branches []uint // nil = toàn hệ thống
	rate     float64
}

type ptFakeSlot struct {
	id        uint
	ownerID   uint // trainer_id (lớp) hoặc member_id (đặt lớp)
	start     time.Time
	end       time.Time
	cancelled bool
}

type ptFakeRepo struct {
	next        uint
	members     map[uint]*entity.Member
	packages    map[uint]*entity.PTPackage
	branches    map[uint]*entity.Branch
	trainers    map[uint]*ptFakeTrainer
	contracts   map[uint]*entity.PTContract
	sessions    map[uint]*entity.PTSession
	commissions []*entity.TrainerCommission
	classes     []ptFakeSlot // lịch dạy lớp của HLV
	bookings    []ptFakeSlot // lớp hội viên đã đặt
}

func newPTFakeRepo() *ptFakeRepo {
	uid := uint(101)
	return &ptFakeRepo{
		next: 1000,
		members: map[uint]*entity.Member{
			1: {BaseEntity: entity.BaseEntity{ID: 1}, UserID: &uid, MemberCode: "GM000001", FullName: "Nguyễn Văn An", Phone: "0900000001", Status: entity.MemberStatusActive},
			2: {BaseEntity: entity.BaseEntity{ID: 2}, MemberCode: "GM000002", FullName: "Trần Bình", Phone: "0900000002", Status: entity.MemberStatusInactive},
			3: {BaseEntity: entity.BaseEntity{ID: 3}, MemberCode: "GM000003", FullName: "Lê Cường", Phone: "0900000003", Status: entity.MemberStatusActive},
		},
		packages: map[uint]*entity.PTPackage{
			1: {BaseEntity: entity.BaseEntity{ID: 1}, Code: "PT10", Name: "PT 10 buổi", SessionCount: 10, SessionMinutes: 60, ValidityDays: 45, Price: 3500000, Status: consts.StatusActive},
			2: {BaseEntity: entity.BaseEntity{ID: 2}, Code: "PTOLD", Name: "PT cũ", SessionCount: 5, SessionMinutes: 60, ValidityDays: 30, Price: 1000000, Status: consts.StatusInactive},
			3: {BaseEntity: entity.BaseEntity{ID: 3}, Code: "PT2", Name: "PT 2 buổi", SessionCount: 2, SessionMinutes: 90, ValidityDays: 30, Price: 1000000, Status: consts.StatusActive},
		},
		branches: map[uint]*entity.Branch{
			1: {BaseEntity: entity.BaseEntity{ID: 1}, Name: "CN Cầu Giấy", OpenTime: "05:30", CloseTime: "22:00", Status: consts.StatusActive},
			2: {BaseEntity: entity.BaseEntity{ID: 2}, Name: "CN Hai Bà Trưng", OpenTime: "05:30", CloseTime: "22:00", Status: consts.StatusActive},
		},
		trainers: map[uint]*ptFakeTrainer{
			6:  {name: "HLV Sáu", branches: []uint{1}, rate: 10},
			7:  {name: "HLV Bảy", branches: []uint{1}, rate: 12.5},
			14: {name: "HLV Mười Bốn", branches: []uint{2}, rate: 20},
		},
		contracts: map[uint]*entity.PTContract{},
		sessions:  map[uint]*entity.PTSession{},
	}
}

func (f *ptFakeRepo) id() uint { f.next++; return f.next }

func (f *ptFakeRepo) FindMember(_ context.Context, id uint) (*entity.Member, error) {
	if m, ok := f.members[id]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *ptFakeRepo) FindPackage(_ context.Context, id uint) (*entity.PTPackage, error) {
	if p, ok := f.packages[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *ptFakeRepo) FindBranch(_ context.Context, id uint) (*entity.Branch, error) {
	if b, ok := f.branches[id]; ok {
		cp := *b
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *ptFakeRepo) TrainerInBranch(_ context.Context, trainerID, branchID uint) (bool, error) {
	t, ok := f.trainers[trainerID]
	if !ok {
		return false, nil
	}
	if t.branches == nil {
		return true, nil
	}
	for _, b := range t.branches {
		if b == branchID {
			return true, nil
		}
	}
	return false, nil
}
func (f *ptFakeRepo) CommissionRate(_ context.Context, trainerID uint) (float64, error) {
	if t, ok := f.trainers[trainerID]; ok {
		return t.rate, nil
	}
	return 0, nil
}
func (f *ptFakeRepo) UserName(_ context.Context, id uint) (string, error) {
	if t, ok := f.trainers[id]; ok {
		return t.name, nil
	}
	return "", nil
}

// ---- Hợp đồng ----

func (f *ptFakeRepo) CreateContract(_ context.Context, c *entity.PTContract) error {
	c.ID = f.id()
	c.CreatedAt, c.UpdatedAt = time.Now(), time.Now()
	cp := *c
	f.contracts[c.ID] = &cp
	return nil
}
func (f *ptFakeRepo) FindContract(_ context.Context, id uint, _ bool) (*entity.PTContract, error) {
	if c, ok := f.contracts[id]; ok {
		cp := *c
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *ptFakeRepo) UpdateContractFields(_ context.Context, id uint, m map[string]any) error {
	c := f.contracts[id]
	for k, v := range m {
		switch k {
		case "invoice_id":
			x := v.(uint)
			c.InvoiceID = &x
		case "price":
			c.Price = v.(float64)
		case "status":
			c.Status = v.(string)
		case "used_sessions":
			c.UsedSessions = v.(int)
		case "trainer_id":
			c.TrainerID = v.(uint)
		default:
			panic("ptFakeRepo.UpdateContractFields: " + k)
		}
	}
	return nil
}
func (f *ptFakeRepo) contractRow(c *entity.PTContract) repository.PTContractRow {
	r := repository.PTContractRow{PTContract: *c, BranchName: f.branches[c.BranchID].Name}
	if m, ok := f.members[c.MemberID]; ok {
		r.MemberCode, r.MemberName, r.MemberPhone = m.MemberCode, m.FullName, m.Phone
	}
	if t, ok := f.trainers[c.TrainerID]; ok {
		r.TrainerName = t.name
	}
	if p, ok := f.packages[c.PTPackageID]; ok {
		r.PackageCode, r.PackageName = p.Code, p.Name
	}
	r.ScheduledSessions, _ = f.CountScheduled(context.Background(), c.ID)
	return r
}
func (f *ptFakeRepo) FindContractRow(_ context.Context, id uint) (*repository.PTContractRow, error) {
	c, ok := f.contracts[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	r := f.contractRow(c)
	return &r, nil
}
func (f *ptFakeRepo) sortedContracts() []*entity.PTContract {
	out := make([]*entity.PTContract, 0, len(f.contracts))
	for _, c := range f.contracts {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}
func ptFakeIn(ids []uint, id uint) bool {
	if ids == nil {
		return true
	}
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}
func (f *ptFakeRepo) ListContracts(_ context.Context, _ dto.Query, flt repository.PTContractFilter) ([]repository.PTContractRow, int64, error) {
	var out []repository.PTContractRow
	for _, c := range f.sortedContracts() {
		if !ptFakeIn(flt.BranchIDs, c.BranchID) || (flt.TrainerID > 0 && c.TrainerID != flt.TrainerID) ||
			(flt.MemberID > 0 && c.MemberID != flt.MemberID) || (flt.Status != "" && c.Status != flt.Status) {
			continue
		}
		out = append(out, f.contractRow(c))
	}
	return out, int64(len(out)), nil
}
func (f *ptFakeRepo) ListContractsByMember(_ context.Context, memberID uint, branchIDs []uint) ([]repository.PTContractRow, error) {
	var out []repository.PTContractRow
	for _, c := range f.sortedContracts() {
		if c.MemberID == memberID && ptFakeIn(branchIDs, c.BranchID) {
			out = append(out, f.contractRow(c))
		}
	}
	return out, nil
}
func (f *ptFakeRepo) CountScheduled(_ context.Context, contractID uint) (int, error) {
	n := 0
	for _, s := range f.sessions {
		if s.ContractID == contractID && s.Status == entity.PTSessionScheduled {
			n++
		}
	}
	return n, nil
}
func (f *ptFakeRepo) ContractsToExpire(_ context.Context, today time.Time) ([]entity.PTContract, error) {
	var out []entity.PTContract
	for _, c := range f.sortedContracts() {
		if c.Status == entity.PTContractActive && c.EndDate.Before(today) {
			out = append(out, *c)
		}
	}
	return out, nil
}

// ---- Buổi tập ----

func ptFakeOverlap(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && aEnd.After(bStart)
}

func ptFakeBlocks(s *entity.PTSession) bool {
	return s.Status == entity.PTSessionScheduled || s.Status == entity.PTSessionCompleted
}

// exclusion mô phỏng EXCLUDE ex_pt_trainer_overlap.
func (f *ptFakeRepo) exclusion(s *entity.PTSession) error {
	if !ptFakeBlocks(s) {
		return nil
	}
	for _, o := range f.sessions {
		if o.ID != s.ID && o.TrainerID == s.TrainerID && ptFakeBlocks(o) && ptFakeOverlap(o.StartAt, o.EndAt, s.StartAt, s.EndAt) {
			return &pgconn.PgError{Code: "23P01", ConstraintName: "ex_pt_trainer_overlap"}
		}
	}
	return nil
}

func (f *ptFakeRepo) CreateSession(_ context.Context, s *entity.PTSession) error {
	if err := f.exclusion(s); err != nil {
		return err
	}
	s.ID = f.id()
	s.CreatedAt, s.UpdatedAt = time.Now(), time.Now()
	cp := *s
	f.sessions[s.ID] = &cp
	return nil
}
func (f *ptFakeRepo) FindSession(_ context.Context, id uint, _ bool) (*entity.PTSession, error) {
	if s, ok := f.sessions[id]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *ptFakeRepo) UpdateSessionFields(_ context.Context, id uint, m map[string]any) error {
	cp := *f.sessions[id]
	for k, v := range m {
		switch k {
		case "status":
			cp.Status = v.(string)
		case "cancelled_at":
			t := v.(time.Time)
			cp.CancelledAt = &t
		case "completed_at":
			t := v.(time.Time)
			cp.CompletedAt = &t
		case "cancel_reason":
			cp.CancelReason = v.(string)
		case "workout_note":
			cp.WorkoutNote = v.(string)
		case "trainer_note":
			cp.TrainerNote = v.(string)
		case "start_at":
			cp.StartAt = v.(time.Time)
		case "end_at":
			cp.EndAt = v.(time.Time)
		case "trainer_id":
			cp.TrainerID = v.(uint)
		default:
			panic("ptFakeRepo.UpdateSessionFields: " + k)
		}
	}
	if err := f.exclusion(&cp); err != nil {
		return err
	}
	f.sessions[id] = &cp
	return nil
}
func (f *ptFakeRepo) sessionRow(s *entity.PTSession) repository.PTSessionRow {
	r := repository.PTSessionRow{PTSession: *s, BranchName: f.branches[s.BranchID].Name}
	if m, ok := f.members[s.MemberID]; ok {
		r.MemberCode, r.MemberName, r.MemberPhone, r.MemberUserID = m.MemberCode, m.FullName, m.Phone, m.UserID
	}
	if t, ok := f.trainers[s.TrainerID]; ok {
		r.TrainerName = t.name
	}
	return r
}
func (f *ptFakeRepo) FindSessionRow(_ context.Context, id uint) (*repository.PTSessionRow, error) {
	s, ok := f.sessions[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	r := f.sessionRow(s)
	return &r, nil
}
func (f *ptFakeRepo) sortedSessions() []*entity.PTSession {
	out := make([]*entity.PTSession, 0, len(f.sessions))
	for _, s := range f.sessions {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].StartAt.Equal(out[j].StartAt) {
			return out[i].StartAt.Before(out[j].StartAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}
func (f *ptFakeRepo) ListSessions(_ context.Context, flt repository.PTSessionFilter) ([]repository.PTSessionRow, error) {
	var out []repository.PTSessionRow
	for _, s := range f.sortedSessions() {
		if s.StartAt.Before(flt.From) || !s.StartAt.Before(flt.To) || !ptFakeIn(flt.BranchIDs, s.BranchID) ||
			(flt.TrainerID > 0 && s.TrainerID != flt.TrainerID) || (flt.MemberID > 0 && s.MemberID != flt.MemberID) ||
			(flt.ContractID > 0 && s.ContractID != flt.ContractID) || (flt.Status != "" && s.Status != flt.Status) {
			continue
		}
		out = append(out, f.sessionRow(s))
	}
	return out, nil
}
func (f *ptFakeRepo) SessionsByContract(_ context.Context, contractID uint) ([]repository.PTSessionRow, error) {
	var out []repository.PTSessionRow
	for _, s := range f.sortedSessions() {
		if s.ContractID == contractID {
			out = append(out, f.sessionRow(s))
		}
	}
	return out, nil
}
func (f *ptFakeRepo) ScheduledSessions(_ context.Context, contractID uint, after *time.Time) ([]entity.PTSession, error) {
	var out []entity.PTSession
	for _, s := range f.sortedSessions() {
		if s.ContractID == contractID && s.Status == entity.PTSessionScheduled && (after == nil || s.StartAt.After(*after)) {
			out = append(out, *s)
		}
	}
	return out, nil
}
func (f *ptFakeRepo) OverdueSessions(_ context.Context, before time.Time) ([]repository.PTSessionRow, error) {
	var out []repository.PTSessionRow
	for _, s := range f.sortedSessions() {
		if s.Status == entity.PTSessionScheduled && s.EndAt.Before(before) {
			out = append(out, f.sessionRow(s))
		}
	}
	return out, nil
}
func (f *ptFakeRepo) TrainerPTOverlap(_ context.Context, trainerID uint, start, end time.Time, excludeID uint) (bool, error) {
	for _, s := range f.sessions {
		if s.ID != excludeID && s.TrainerID == trainerID && ptFakeBlocks(s) && ptFakeOverlap(s.StartAt, s.EndAt, start, end) {
			return true, nil
		}
	}
	return false, nil
}
func (f *ptFakeRepo) TrainerClassOverlap(_ context.Context, trainerID uint, start, end time.Time) (bool, error) {
	for _, c := range f.classes {
		if c.ownerID == trainerID && !c.cancelled && ptFakeOverlap(c.start, c.end, start, end) {
			return true, nil
		}
	}
	return false, nil
}
func (f *ptFakeRepo) MemberPTOverlap(_ context.Context, memberID uint, start, end time.Time, excludeID uint) (bool, error) {
	for _, s := range f.sessions {
		if s.ID != excludeID && s.MemberID == memberID && ptFakeBlocks(s) && ptFakeOverlap(s.StartAt, s.EndAt, start, end) {
			return true, nil
		}
	}
	return false, nil
}
func (f *ptFakeRepo) MemberClassOverlap(_ context.Context, memberID uint, start, end time.Time) (bool, error) {
	for _, b := range f.bookings {
		if b.ownerID == memberID && !b.cancelled && ptFakeOverlap(b.start, b.end, start, end) {
			return true, nil
		}
	}
	return false, nil
}
func (f *ptFakeRepo) TrainerClasses(_ context.Context, trainerID uint, from, to time.Time) ([]repository.PTClassSlot, error) {
	var out []repository.PTClassSlot
	for _, c := range f.classes {
		if c.ownerID == trainerID && !c.cancelled && !c.start.Before(from) && c.start.Before(to) {
			out = append(out, repository.PTClassSlot{ID: c.id, TrainerID: trainerID, BranchID: 1, ClassTypeName: "Yoga",
				StartAt: c.start, EndAt: c.end, Status: entity.ClassScheduleScheduled, Capacity: 20})
		}
	}
	return out, nil
}

// ---- Hoa hồng ----

func (f *ptFakeRepo) CreateCommission(_ context.Context, c *entity.TrainerCommission) error {
	for _, o := range f.commissions {
		if o.SourceType == c.SourceType && o.SourceID == c.SourceID {
			return gorm.ErrDuplicatedKey
		}
	}
	c.ID = f.id()
	cp := *c
	f.commissions = append(f.commissions, &cp)
	return nil
}
func (f *ptFakeRepo) matchCommission(c *entity.TrainerCommission, flt repository.PTCommissionFilter) bool {
	return ptFakeIn(flt.BranchIDs, c.BranchID) && (flt.TrainerID == 0 || c.TrainerID == flt.TrainerID) &&
		(flt.Period == "" || c.Period == flt.Period) && (flt.Status == "" || c.Status == flt.Status) &&
		(flt.SourceType == "" || c.SourceType == flt.SourceType)
}
func (f *ptFakeRepo) ListCommissions(_ context.Context, _ dto.Query, flt repository.PTCommissionFilter) ([]repository.PTCommissionRow, int64, error) {
	var out []repository.PTCommissionRow
	for _, c := range f.commissions {
		if f.matchCommission(c, flt) {
			out = append(out, repository.PTCommissionRow{TrainerCommission: *c, TrainerName: f.trainers[c.TrainerID].name})
		}
	}
	return out, int64(len(out)), nil
}
func (f *ptFakeRepo) CommissionSummary(_ context.Context, flt repository.PTCommissionFilter) ([]repository.PTCommissionSummaryRow, error) {
	by := map[uint]*repository.PTCommissionSummaryRow{}
	var order []uint
	for _, c := range f.commissions {
		if !f.matchCommission(c, flt) {
			continue
		}
		r, ok := by[c.TrainerID]
		if !ok {
			r = &repository.PTCommissionSummaryRow{TrainerID: c.TrainerID, TrainerName: f.trainers[c.TrainerID].name}
			by[c.TrainerID] = r
			order = append(order, c.TrainerID)
		}
		r.Sessions++
		r.BaseTotal += c.BaseAmount
		r.TotalAmount += c.Amount
		if c.Status == entity.CommissionPaid {
			r.PaidAmount += c.Amount
		} else {
			r.PendingAmount += c.Amount
		}
	}
	out := make([]repository.PTCommissionSummaryRow, 0, len(order))
	for _, id := range order {
		out = append(out, *by[id])
	}
	return out, nil
}
func (f *ptFakeRepo) PayCommissions(_ context.Context, trainerID uint, period string, branchIDs []uint, paidBy *uint, at time.Time) (int, float64, error) {
	n, sum := 0, 0.0
	for _, c := range f.commissions {
		if c.TrainerID == trainerID && c.Period == period && c.Status == entity.CommissionPending && ptFakeIn(branchIDs, c.BranchID) {
			c.Status, c.PaidAt, c.PaidBy = entity.CommissionPaid, &at, paidBy
			n++
			sum += c.Amount
		}
	}
	return n, sum, nil
}

// ---- Notifier ----

type ptFakeNotifier struct {
	sent []ptFakeNotice
}

type ptFakeNotice struct {
	userIDs []uint
	typ     string
}

func (n *ptFakeNotifier) Notify(_ context.Context, userIDs []uint, typ, _, _, _ string) error {
	if len(userIDs) > 0 {
		n.sent = append(n.sent, ptFakeNotice{userIDs: append([]uint(nil), userIDs...), typ: typ})
	}
	return nil
}
func (n *ptFakeNotifier) NotifyPermission(_ context.Context, perm string, _ *uint, typ, _, _, _ string) error {
	n.sent = append(n.sent, ptFakeNotice{typ: typ + ":" + perm})
	return nil
}

// notifiedUser: user đã nhận thông báo loại typ.
func (n *ptFakeNotifier) notifiedUser(userID uint, typ string) bool {
	for _, s := range n.sent {
		if s.typ != typ {
			continue
		}
		for _, id := range s.userIDs {
			if id == userID {
				return true
			}
		}
	}
	return false
}

// ---- Fixture ----

type ptFixture struct {
	repo     *ptFakeRepo
	invRep   *invoiceFakeRepo
	inv      InvoiceService
	notifier *ptFakeNotifier
	svc      *ptService
	clock    time.Time
}

// ptVN tạo thời điểm theo giờ Việt Nam.
func ptVN(month, day, hour, minute int) time.Time {
	return time.Date(2026, time.Month(month), day, hour, minute, 0, 0, subVNLoc)
}

func newPTFixture() *ptFixture {
	fx := &ptFixture{repo: newPTFakeRepo(), notifier: &ptFakeNotifier{}, clock: ptVN(10, 14, 9, 0)}
	promos := &invoiceFakePromoRepo{items: map[uint]*entity.Promotion{}}
	fx.invRep = newInvoiceFakeRepo(promos)
	fx.inv = NewInvoiceService(fx.invRep, NewPromotionService(promos, nil), subNoTx{}, nil)
	fx.svc = NewPTService(fx.repo, fx.inv, subNoTx{}, nil, nil, fx.notifier).(*ptService)
	fx.svc.now = func() time.Time { return fx.clock }
	return fx
}

func ptTrainerCtx(userID uint) context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{UserID: userID, Roles: []string{consts.RoleTrainer}, BranchIDs: []uint{1}})
}

// activeContract bán gói và thu đủ tiền → hợp đồng ACTIVE.
func (fx *ptFixture) activeContract(memberID, packageID, trainerID uint) uint {
	pkg := fx.repo.packages[packageID]
	res, err := fx.svc.Sell(subOwnerCtx(), ptSell(memberID, packageID, trainerID), &ptPay{Amount: pkg.Price, Method: "CASH"})
	if err != nil {
		panic(err)
	}
	return res.ID
}
