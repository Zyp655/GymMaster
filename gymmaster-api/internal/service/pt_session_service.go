package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/pt/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

const (
	ptTrainerPTOverlapMsg    = "HLV đã có buổi PT khác trùng giờ"
	ptTrainerClassOverlapMsg = "HLV có lịch dạy lớp nhóm trùng giờ"
	ptMemberPTOverlapMsg     = "Hội viên đã có buổi PT khác trùng giờ"
	ptMemberClassOverlapMsg  = "Hội viên đã đặt lớp nhóm trùng giờ"
	ptMemberLink             = "/member/pt-sessions"
)

// notifyParticipants: thông báo cho nhân viên (link trang admin) và hội viên có tài khoản cổng (link cổng hội viên).
func (s *ptService) notifyParticipants(ctx context.Context, staffIDs []uint, memberUserID *uint, typ, title, body, adminLink string) {
	self := ptSelf(ctx)
	staff := make([]uint, 0, len(staffIDs))
	for _, id := range staffIDs {
		if id != self {
			staff = append(staff, id)
		}
	}
	s.notify(ctx, staff, typ, title, body, adminLink)
	if memberUserID != nil && *memberUserID != self {
		s.notify(ctx, []uint{*memberUserID}, typ, title, body, ptMemberLink)
	}
}

// ---- Kiểm tra lịch --------------------------------------------------------------

// ptValidateSlot: hợp đồng ACTIVE, giờ bắt đầu ở tương lai và nằm trong thời hạn hợp đồng. Trả giờ kết thúc.
func ptValidateSlot(c *entity.PTContract, start, now time.Time) (time.Time, error) {
	switch c.Status {
	case entity.PTContractActive:
	case entity.PTContractPending:
		return time.Time{}, rescode.New(rescode.BadRequest, "Hợp đồng PT chưa được kích hoạt (chưa thanh toán đủ)")
	default:
		return time.Time{}, rescode.New(rescode.BadRequest, "Hợp đồng PT đã kết thúc, không thể đặt buổi tập")
	}
	if !start.After(now) {
		return time.Time{}, rescode.New(rescode.BadRequest, "Giờ bắt đầu buổi tập phải ở tương lai")
	}
	end := start.Add(time.Duration(max(1, c.SessionMinutes)) * time.Minute)
	d := subDateOf(start)
	if d.Before(subNorm(c.StartDate)) || d.After(subNorm(c.EndDate)) || subDateOf(end).After(subNorm(c.EndDate)) {
		return time.Time{}, rescode.New(rescode.BadRequest, fmt.Sprintf("Buổi tập phải nằm trong thời hạn hợp đồng (%s - %s)",
			invoiceDescDate(c.StartDate), invoiceDescDate(c.EndDate)))
	}
	return end, nil
}

// ptCheckOpeningHours: buổi PT nằm trọn trong giờ mở cửa của chi nhánh (trong cùng một ngày, giờ VN).
func ptCheckOpeningHours(b *entity.Branch, start, end time.Time) error {
	ls, le := start.In(subVNLoc), end.In(subVNLoc)
	if ls.Format("2006-01-02") != le.Format("2006-01-02") || ls.Format("15:04") < b.OpenTime || le.Format("15:04") > b.CloseTime {
		return rescode.New(rescode.BadRequest,
			fmt.Sprintf("Buổi PT phải nằm trong giờ mở cửa của chi nhánh (%s - %s)", b.OpenTime, b.CloseTime))
	}
	return nil
}

// trainerConflict trả lý do HLV bận ("" nếu trống): buổi PT khác (trừ excludeID) hoặc lớp nhóm chưa hủy.
func (s *ptService) trainerConflict(ctx context.Context, trainerID uint, start, end time.Time, excludeID uint) (string, error) {
	busy, err := s.repo.TrainerPTOverlap(ctx, trainerID, start, end, excludeID)
	if err != nil || busy {
		return ptTrainerPTOverlapMsg, err
	}
	busy, err = s.repo.TrainerClassOverlap(ctx, trainerID, start, end)
	if err != nil || busy {
		return ptTrainerClassOverlapMsg, err
	}
	return "", nil
}

// slotConflict: HLV và hội viên đều phải trống trong [start, end).
func (s *ptService) slotConflict(ctx context.Context, trainerID, memberID uint, start, end time.Time, excludeID uint) error {
	reason, err := s.trainerConflict(ctx, trainerID, start, end, excludeID)
	if err != nil {
		return err
	}
	if reason == "" {
		busy, err := s.repo.MemberPTOverlap(ctx, memberID, start, end, excludeID)
		if err != nil {
			return err
		}
		if busy {
			reason = ptMemberPTOverlapMsg
		} else if busy, err = s.repo.MemberClassOverlap(ctx, memberID, start, end); err != nil {
			return err
		} else if busy {
			reason = ptMemberClassOverlapMsg
		}
	}
	if reason != "" {
		return rescode.New(rescode.Conflict, reason)
	}
	return nil
}

func ptMapOverlap(err error) error {
	if isExclusionViolation(err) {
		return rescode.New(rescode.Conflict, ptTrainerPTOverlapMsg)
	}
	return err
}

// ---- Truy cập buổi tập ----------------------------------------------------------

// ptActor: ai đang thao tác. memberID > 0 = hội viên qua cổng (không kiểm tra phạm vi chi nhánh).
type ptActor struct {
	memberID uint
}

func (a ptActor) portal() bool { return a.memberID > 0 }

func (s *ptService) checkSessionAccess(ctx context.Context, ss *entity.PTSession, a ptActor) error {
	if a.portal() {
		if ss.MemberID != a.memberID {
			return errPTSessionNotFound
		}
		return nil
	}
	if !reqctx.CanAccessBranch(ctx, ss.BranchID) {
		return errPTSessionScope
	}
	if ptTrainerOnly(ctx) && ss.TrainerID != ptSelf(ctx) {
		return errPTNotOwnSession
	}
	return nil
}

// lockSession khóa hợp đồng rồi buổi tập (thứ tự khóa cố định: hợp đồng → buổi, tránh deadlock).
func (s *ptService) lockSession(ctx context.Context, id uint, a ptActor) (*entity.PTContract, *entity.PTSession, error) {
	ss, err := s.repo.FindSession(ctx, id, false)
	if err != nil {
		return nil, nil, ptNotFound(err, errPTSessionNotFound)
	}
	if err := s.checkSessionAccess(ctx, ss, a); err != nil {
		return nil, nil, err
	}
	c, err := s.repo.FindContract(ctx, ss.ContractID, true)
	if err != nil {
		return nil, nil, ptNotFound(err, errPTContractNotFound)
	}
	if ss, err = s.repo.FindSession(ctx, id, true); err != nil {
		return nil, nil, ptNotFound(err, errPTSessionNotFound)
	}
	return c, ss, nil
}

func (s *ptService) sessionRes(ctx context.Context, id uint) (*v1.Session, *repository.PTSessionRow, error) {
	row, err := s.repo.FindSessionRow(ctx, id)
	if err != nil {
		return nil, nil, ptNotFound(err, errPTSessionNotFound)
	}
	return ptSessionRes(*row), row, nil
}

// ---- Danh sách / chi tiết -------------------------------------------------------

func (s *ptService) ListSessions(ctx context.Context, req *v1.SessionListReq) ([]*v1.Session, error) {
	from, to, err := ptRange(req.From, req.To)
	if err != nil {
		return nil, err
	}
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu chi nhánh này")
	}
	trainerID := req.TrainerID
	if ptTrainerOnly(ctx) {
		if trainerID != 0 && trainerID != ptSelf(ctx) {
			return nil, errPTOnlyOwnData
		}
		trainerID = ptSelf(ctx)
	}
	rows, err := s.repo.ListSessions(ctx, repository.PTSessionFilter{
		BranchIDs: branchIDs, TrainerID: trainerID, MemberID: req.MemberID, ContractID: req.ContractID, Status: req.Status, From: from, To: to,
	})
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, ptSessionRes), nil
}

func (s *ptService) GetSession(ctx context.Context, id uint) (*v1.Session, error) {
	res, row, err := s.sessionRes(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.checkSessionAccess(ctx, &row.PTSession, ptActor{}); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *ptService) ListMemberSessions(ctx context.Context, memberID uint, from, to time.Time) ([]*v1.Session, error) {
	if err := ptCheckRange(from, to); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListSessions(ctx, repository.PTSessionFilter{MemberID: memberID, From: from, To: to})
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, ptSessionRes), nil
}

// ---- Đặt buổi -------------------------------------------------------------------

func (s *ptService) CreateSession(ctx context.Context, contractID uint, startAt string) (*v1.Session, error) {
	start, err := classParseTime(startAt, subVNLoc)
	if err != nil {
		return nil, err
	}
	return s.book(ctx, contractID, start, ptActor{})
}

func (s *ptService) BookSessionByMember(ctx context.Context, memberID, contractID uint, startAt time.Time) (*v1.Session, error) {
	if memberID == 0 {
		return nil, errPTContractNotFound
	}
	return s.book(ctx, contractID, startAt, ptActor{memberID: memberID})
}

func (s *ptService) book(ctx context.Context, contractID uint, start time.Time, a ptActor) (*v1.Session, error) {
	now := s.now()
	var created *entity.PTSession
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		c, err := s.repo.FindContract(ctx, contractID, true)
		if err != nil {
			return ptNotFound(err, errPTContractNotFound)
		}
		source := entity.PTSourceCounter
		if a.portal() {
			if c.MemberID != a.memberID {
				return errPTContractNotFound
			}
			source = entity.PTSourcePortal
		} else {
			if err := s.checkContractAccess(ctx, c); err != nil {
				return err
			}
			if c.TrainerID == ptSelf(ctx) {
				source = entity.PTSourceTrainer
			}
		}
		end, err := ptValidateSlot(c, start, now)
		if err != nil {
			return err
		}
		member, err := s.repo.FindMember(ctx, c.MemberID)
		if err != nil {
			return ptNotFound(err, rescode.New(rescode.NotFound, "Không tìm thấy hội viên"))
		}
		if member.Status != entity.MemberStatusActive {
			return rescode.New(rescode.BadRequest, "Hội viên đang không hoạt động")
		}
		branch, err := s.repo.FindBranch(ctx, c.BranchID)
		if err != nil {
			return ptNotFound(err, rescode.New(rescode.BadRequest, "Chi nhánh không tồn tại"))
		}
		if err := ptCheckOpeningHours(branch, start, end); err != nil {
			return err
		}
		scheduled, err := s.repo.CountScheduled(ctx, c.ID)
		if err != nil {
			return err
		}
		if c.UsedSessions+scheduled >= c.TotalSessions {
			return rescode.New(rescode.BadRequest, fmt.Sprintf(
				"Hợp đồng đã đặt/dùng hết số buổi (đã dùng %d, đang đặt %d / tổng %d buổi)", c.UsedSessions, scheduled, c.TotalSessions))
		}
		if err := s.slotConflict(ctx, c.TrainerID, c.MemberID, start, end, 0); err != nil {
			return err
		}
		ss := &entity.PTSession{
			ContractID: c.ID, TrainerID: c.TrainerID, MemberID: c.MemberID, BranchID: c.BranchID,
			StartAt: start, EndAt: end, Status: entity.PTSessionScheduled, Source: source, CreatedBy: reqctx.UserID(ctx),
		}
		if err := s.repo.CreateSession(ctx, ss); err != nil {
			return ptMapOverlap(err)
		}
		s.logAudit(ctx, ptActSchedule, ptAuditSession, ss.ID, nil, map[string]any{
			"contract_id": c.ID, "trainer_id": ss.TrainerID, "member_id": ss.MemberID, "start_at": ss.StartAt, "end_at": ss.EndAt, "source": source,
		})
		created = ss
		return nil
	})
	if err != nil {
		return nil, err
	}
	res, row, err := s.sessionRes(ctx, created.ID)
	if err != nil {
		return nil, err
	}
	s.notifyParticipants(ctx, []uint{row.TrainerID}, row.MemberUserID, NotifyPTScheduled, "Lịch tập PT mới",
		fmt.Sprintf("Buổi PT của %s với HLV %s lúc %s (%d phút) tại %s.", row.MemberName, row.TrainerName, ptFmtTime(row.StartAt),
			int(row.EndAt.Sub(row.StartAt).Minutes()), row.BranchName),
		fmt.Sprintf("/admin/pt-sessions/%d", row.ID))
	return res, nil
}

// ---- Đổi giờ --------------------------------------------------------------------

func (s *ptService) RescheduleSession(ctx context.Context, id uint, startAt string) (*v1.Session, error) {
	start, err := classParseTime(startAt, subVNLoc)
	if err != nil {
		return nil, err
	}
	now := s.now()
	var oldStart time.Time
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		c, ss, err := s.lockSession(ctx, id, ptActor{})
		if err != nil {
			return err
		}
		if ss.Status != entity.PTSessionScheduled {
			return rescode.New(rescode.BadRequest, "Chỉ đổi giờ được buổi đang chờ tập")
		}
		if !ss.StartAt.After(now) {
			return rescode.New(rescode.BadRequest, "Buổi tập đã bắt đầu, không thể đổi giờ")
		}
		end, err := ptValidateSlot(c, start, now)
		if err != nil {
			return err
		}
		branch, err := s.repo.FindBranch(ctx, ss.BranchID)
		if err != nil {
			return ptNotFound(err, rescode.New(rescode.BadRequest, "Chi nhánh không tồn tại"))
		}
		if err := ptCheckOpeningHours(branch, start, end); err != nil {
			return err
		}
		if err := s.slotConflict(ctx, ss.TrainerID, ss.MemberID, start, end, ss.ID); err != nil {
			return err
		}
		if err := s.repo.UpdateSessionFields(ctx, ss.ID, map[string]any{"start_at": start, "end_at": end}); err != nil {
			return ptMapOverlap(err)
		}
		oldStart = ss.StartAt
		s.logAudit(ctx, ptActReschedule, ptAuditSession, ss.ID, map[string]any{"start_at": ss.StartAt, "end_at": ss.EndAt},
			map[string]any{"start_at": start, "end_at": end})
		return nil
	})
	if err != nil {
		return nil, err
	}
	res, row, err := s.sessionRes(ctx, id)
	if err != nil {
		return nil, err
	}
	s.notifyParticipants(ctx, []uint{row.TrainerID}, row.MemberUserID, NotifyPTScheduled, "Đổi giờ buổi tập PT",
		fmt.Sprintf("Buổi PT của %s với HLV %s đổi từ %s sang %s.", row.MemberName, row.TrainerName, ptFmtTime(oldStart), ptFmtTime(row.StartAt)),
		fmt.Sprintf("/admin/pt-sessions/%d", row.ID))
	return res, nil
}

// ---- Hủy buổi -------------------------------------------------------------------

func (s *ptService) CancelSession(ctx context.Context, id uint, reason string, countAsUsed bool) (*v1.Session, error) {
	return s.cancelSession(ctx, id, reason, countAsUsed, ptActor{})
}

func (s *ptService) CancelSessionByMember(ctx context.Context, memberID, sessionID uint, reason string) (*v1.Session, error) {
	if memberID == 0 {
		return nil, errPTSessionNotFound
	}
	return s.cancelSession(ctx, sessionID, reason, false, ptActor{memberID: memberID})
}

// ptCancelOutcome quyết định kết quả hủy buổi:
//   - hủy đúng hạn (now ≤ start − cancelBefore): CANCELLED, không tính buổi;
//   - hủy muộn: hội viên tự hủy → bị từ chối; nhân viên → CANCELLED, hoặc NO_SHOW (tính buổi) nếu countAsUsed.
func ptCancelOutcome(start, now time.Time, cancelBefore int, countAsUsed, portal bool) (status string, late bool, err error) {
	late = now.After(start.Add(-time.Duration(cancelBefore) * time.Minute))
	switch {
	case !late:
		return entity.PTSessionCancelled, false, nil
	case portal:
		return "", true, rescode.New(rescode.BadRequest, fmt.Sprintf(
			"Chỉ được hủy buổi PT trước giờ tập ít nhất %s, vui lòng liên hệ lễ tân", ptFmtMinutes(cancelBefore)))
	case countAsUsed:
		return entity.PTSessionNoShow, true, nil
	default:
		return entity.PTSessionCancelled, true, nil
	}
}

func (s *ptService) cancelSession(ctx context.Context, id uint, reason string, countAsUsed bool, a ptActor) (*v1.Session, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập lý do hủy")
	}
	cancelBefore := s.cancelBeforeMinutes(ctx)
	now := s.now()
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		c, ss, err := s.lockSession(ctx, id, a)
		if err != nil {
			return err
		}
		if ss.Status != entity.PTSessionScheduled {
			return rescode.New(rescode.BadRequest, "Chỉ hủy được buổi đang chờ tập")
		}
		status, late, err := ptCancelOutcome(ss.StartAt, now, cancelBefore, countAsUsed, a.portal())
		if err != nil {
			return err
		}
		fields := map[string]any{"status": status, "cancelled_at": now, "cancel_reason": invoiceTruncate(reason, 255)}
		counted := status == entity.PTSessionNoShow
		if counted {
			if err := s.useSession(ctx, c); err != nil {
				return err
			}
			fields["completed_at"] = now
		}
		if err := s.repo.UpdateSessionFields(ctx, ss.ID, fields); err != nil {
			return err
		}
		s.logAudit(ctx, ptActCancel, ptAuditSession, ss.ID, map[string]any{"status": ss.Status},
			map[string]any{"status": status, "reason": reason, "late": late, "counted": counted, "by_member": a.portal()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	res, row, err := s.sessionRes(ctx, id)
	if err != nil {
		return nil, err
	}
	s.notifyParticipants(ctx, []uint{row.TrainerID}, row.MemberUserID, NotifyPTCancelled, "Hủy buổi tập PT",
		fmt.Sprintf("Buổi PT của %s với HLV %s lúc %s đã bị hủy. Lý do: %s", row.MemberName, row.TrainerName, ptFmtTime(row.StartAt), reason),
		fmt.Sprintf("/admin/pt-sessions/%d", row.ID))
	return res, nil
}

// useSession trừ 1 buổi vào hợp đồng (đã khóa FOR UPDATE); dùng hết → COMPLETED.
func (s *ptService) useSession(ctx context.Context, c *entity.PTContract) error {
	if c.Status != entity.PTContractActive && c.Status != entity.PTContractExpired {
		return rescode.New(rescode.BadRequest, "Hợp đồng PT không còn hiệu lực")
	}
	if c.UsedSessions >= c.TotalSessions {
		return rescode.New(rescode.BadRequest, "Hợp đồng PT đã dùng hết số buổi")
	}
	c.UsedSessions++
	fields := map[string]any{"used_sessions": c.UsedSessions}
	if c.UsedSessions >= c.TotalSessions {
		c.Status = entity.PTContractCompleted
		fields["status"] = c.Status
	}
	return s.repo.UpdateContractFields(ctx, c.ID, fields)
}

// ---- Hoàn thành / vắng mặt ------------------------------------------------------

// ptSettleable: buổi có thể chốt (hoàn thành / vắng mặt): SCHEDULED, hoặc NO_SHOW do cron đánh dấu (chưa tính buổi).
func ptSettleable(ss *entity.PTSession, now time.Time) error {
	switch {
	case ss.Status == entity.PTSessionCompleted:
		return rescode.New(rescode.Conflict, "Buổi tập đã được xác nhận hoàn thành")
	case ss.Status == entity.PTSessionScheduled, ss.Status == entity.PTSessionNoShow && !ss.Counted():
	case ss.Status == entity.PTSessionNoShow:
		return rescode.New(rescode.Conflict, "Buổi tập đã được chốt vắng mặt")
	default:
		return rescode.New(rescode.BadRequest, "Buổi tập đã bị hủy")
	}
	if now.Before(ss.StartAt) {
		return rescode.New(rescode.BadRequest, "Chỉ xác nhận được sau giờ bắt đầu buổi tập")
	}
	return nil
}

func (s *ptService) CompleteSession(ctx context.Context, id uint, workoutNote, trainerNote string) (*v1.Session, error) {
	now := s.now()
	var commission *entity.TrainerCommission
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		c, ss, err := s.lockSession(ctx, id, ptActor{})
		if err != nil {
			return err
		}
		if err := ptSettleable(ss, now); err != nil {
			return err
		}
		if err := s.useSession(ctx, c); err != nil {
			return err
		}
		if err := s.repo.UpdateSessionFields(ctx, ss.ID, map[string]any{
			"status": entity.PTSessionCompleted, "completed_at": now, "cancel_reason": "",
			"workout_note": strings.TrimSpace(workoutNote), "trainer_note": strings.TrimSpace(trainerNote),
		}); err != nil {
			return ptMapOverlap(err)
		}
		rate, err := s.repo.CommissionRate(ctx, ss.TrainerID)
		if err != nil {
			return err
		}
		base, amount := ptCommission(c.Price, c.TotalSessions, rate)
		commission = &entity.TrainerCommission{
			TrainerID: ss.TrainerID, BranchID: ss.BranchID, SourceType: entity.CommissionSourcePTSession, SourceID: ss.ID,
			BaseAmount: base, Rate: rate, Amount: amount, Period: ptPeriod(ss.StartAt), Status: entity.CommissionPending,
		}
		if err := s.repo.CreateCommission(ctx, commission); err != nil {
			return mapDuplicate(err, "Buổi tập đã được tính hoa hồng")
		}
		s.logAudit(ctx, ptActComplete, ptAuditSession, ss.ID, map[string]any{"status": ss.Status}, map[string]any{
			"status": entity.PTSessionCompleted, "contract_id": c.ID, "used_sessions": c.UsedSessions, "contract_status": c.Status,
			"commission_id": commission.ID, "commission_amount": amount, "rate": rate, "period": commission.Period,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	res, _, err := s.sessionRes(ctx, id)
	return res, err
}

func (s *ptService) MarkNoShow(ctx context.Context, id uint, note string) (*v1.Session, error) {
	now := s.now()
	note = strings.TrimSpace(note)
	if note == "" {
		note = "Hội viên vắng mặt"
	}
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		c, ss, err := s.lockSession(ctx, id, ptActor{})
		if err != nil {
			return err
		}
		if err := ptSettleable(ss, now); err != nil {
			return err
		}
		if err := s.useSession(ctx, c); err != nil {
			return err
		}
		if err := s.repo.UpdateSessionFields(ctx, ss.ID, map[string]any{
			"status": entity.PTSessionNoShow, "completed_at": now, "cancel_reason": invoiceTruncate(note, 255),
		}); err != nil {
			return err
		}
		s.logAudit(ctx, ptActNoShow, ptAuditSession, ss.ID, map[string]any{"status": ss.Status}, map[string]any{
			"status": entity.PTSessionNoShow, "counted": true, "contract_id": c.ID, "used_sessions": c.UsedSessions, "note": note,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	res, row, err := s.sessionRes(ctx, id)
	if err != nil {
		return nil, err
	}
	s.notifyParticipants(ctx, nil, row.MemberUserID, NotifyPTCancelled, "Vắng mặt buổi tập PT",
		fmt.Sprintf("Bạn vắng mặt buổi PT lúc %s, buổi tập được tính là đã sử dụng.", ptFmtTime(row.StartAt)), "")
	return res, nil
}

// ---- Lịch dạy HLV ---------------------------------------------------------------

func (s *ptService) TrainerScheduleAdmin(ctx context.Context, req *v1.TrainerScheduleReq) ([]*v1.ScheduleItem, error) {
	from, to, err := ptRange(req.From, req.To)
	if err != nil {
		return nil, err
	}
	trainerID := req.TrainerID
	if trainerID == 0 {
		trainerID = ptSelf(ctx)
	}
	if ptTrainerOnly(ctx) && trainerID != ptSelf(ctx) {
		return nil, errPTOnlyOwnData
	}
	items, err := s.TrainerSchedule(ctx, trainerID, from, to)
	if err != nil {
		return nil, err
	}
	allowed := reqctx.AllowedBranches(ctx)
	if allowed == nil {
		return items, nil
	}
	out := items[:0]
	for _, it := range items {
		if reqctx.CanAccessBranch(ctx, it.BranchID) {
			out = append(out, it)
		}
	}
	return out, nil
}

func (s *ptService) TrainerSchedule(ctx context.Context, trainerID uint, from, to time.Time) ([]*v1.ScheduleItem, error) {
	if err := ptCheckRange(from, to); err != nil {
		return nil, err
	}
	sessions, err := s.repo.ListSessions(ctx, repository.PTSessionFilter{TrainerID: trainerID, From: from, To: to})
	if err != nil {
		return nil, err
	}
	classes, err := s.repo.TrainerClasses(ctx, trainerID, from, to)
	if err != nil {
		return nil, err
	}
	items := make([]*v1.ScheduleItem, 0, len(sessions)+len(classes))
	for _, ss := range sessions {
		if ss.Status == entity.PTSessionCancelled {
			continue
		}
		contractID, memberID := ss.ContractID, ss.MemberID
		items = append(items, &v1.ScheduleItem{
			Type: "PT", ID: ss.ID, Title: "PT – " + ss.MemberName, StartAt: ss.StartAt, EndAt: ss.EndAt, Status: ss.Status,
			BranchID: ss.BranchID, BranchName: ss.BranchName, TrainerID: ss.TrainerID, TrainerName: ss.TrainerName,
			ContractID: &contractID, MemberID: &memberID,
		})
	}
	for _, cl := range classes {
		booked, capacity := cl.BookedCount, cl.Capacity
		items = append(items, &v1.ScheduleItem{
			Type: "CLASS", ID: cl.ID, Title: cl.ClassTypeName, StartAt: cl.StartAt, EndAt: cl.EndAt, Status: cl.Status,
			BranchID: cl.BranchID, BranchName: cl.BranchName, TrainerID: cl.TrainerID, TrainerName: cl.TrainerName,
			RoomName: cl.RoomName, Color: cl.Color, BookedCount: &booked, Capacity: &capacity,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].StartAt.Equal(items[j].StartAt) {
			return items[i].StartAt.Before(items[j].StartAt)
		}
		return items[i].Type > items[j].Type // PT trước CLASS khi cùng giờ
	})
	return items, nil
}

// ---- Cron -----------------------------------------------------------------------

func (s *ptService) ExpireContracts(ctx context.Context, today time.Time) (int, error) {
	day := subDateOf(today)
	contracts, err := s.repo.ContractsToExpire(ctx, day)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, c := range contracts {
		err := s.tx.WithTx(ctx, func(ctx context.Context) error {
			cur, err := s.repo.FindContract(ctx, c.ID, true)
			if err != nil {
				return err
			}
			if cur.Status != entity.PTContractActive || !subNorm(cur.EndDate).Before(day) {
				return nil
			}
			if err := s.repo.UpdateContractFields(ctx, cur.ID, map[string]any{"status": entity.PTContractExpired}); err != nil {
				return err
			}
			// Buổi SCHEDULED bắt đầu sau ngày hết hạn (00:00 ngày end_date + 1, giờ VN) bị hủy.
			e := subNorm(cur.EndDate)
			after := time.Date(e.Year(), e.Month(), e.Day()+1, 0, 0, 0, 0, subVNLoc).Add(-time.Nanosecond)
			cancelled, err := s.cancelScheduled(ctx, cur.ID, &after, "Hợp đồng PT hết hạn")
			if err != nil {
				return err
			}
			s.logAudit(ctx, ptActExpire, ptAuditContract, cur.ID, map[string]any{"status": cur.Status},
				map[string]any{"status": entity.PTContractExpired, "end_date": subFmt(cur.EndDate), "cancelled_sessions": len(cancelled)})
			n++
			return nil
		})
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (s *ptService) MarkPTNoShows(ctx context.Context, now time.Time) (int, error) {
	rows, err := s.repo.OverdueSessions(ctx, now.Add(-ptNoShowGrace))
	if err != nil {
		return 0, err
	}
	const reason = "Tự động: quá 2 giờ sau giờ kết thúc chưa xác nhận – chờ quản lý xử lý"
	n := 0
	perBranch := map[uint]int{}
	for _, row := range rows {
		marked := false
		err := s.tx.WithTx(ctx, func(ctx context.Context) error {
			if _, err := s.repo.FindContract(ctx, row.ContractID, true); err != nil {
				return err
			}
			ss, err := s.repo.FindSession(ctx, row.ID, true)
			if err != nil {
				return err
			}
			if ss.Status != entity.PTSessionScheduled {
				return nil
			}
			if err := s.repo.UpdateSessionFields(ctx, ss.ID, map[string]any{"status": entity.PTSessionNoShow, "cancel_reason": reason}); err != nil {
				return err
			}
			s.logAudit(ctx, ptActAutoNoShow, ptAuditSession, ss.ID, map[string]any{"status": ss.Status},
				map[string]any{"status": entity.PTSessionNoShow, "counted": false})
			marked = true
			return nil
		})
		if err != nil {
			return n, err
		}
		if marked {
			n++
			perBranch[row.BranchID]++
			s.notify(ctx, []uint{row.TrainerID}, NotifyPTNoShow, "Buổi PT chưa được xác nhận",
				fmt.Sprintf("Buổi PT với %s lúc %s chưa được xác nhận hoàn thành, hệ thống tạm đánh dấu vắng mặt (chưa trừ buổi).",
					row.MemberName, ptFmtTime(row.StartAt)),
				fmt.Sprintf("/admin/pt-sessions/%d", row.ID))
		}
	}
	if s.notifier != nil {
		for branchID, cnt := range perBranch {
			b := branchID
			if err := s.notifier.NotifyPermission(ctx, "pt.cancel_contract", &b, NotifyPTNoShow, "Buổi PT chờ xử lý vắng mặt",
				fmt.Sprintf("%d buổi PT quá giờ chưa được xác nhận đã được đánh dấu vắng mặt (chưa trừ buổi). Vui lòng xác nhận hoàn thành hoặc tính buổi vắng mặt.", cnt),
				"/admin/pt-sessions?status=NO_SHOW"); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return n, err
			}
		}
	}
	return n, nil
}
