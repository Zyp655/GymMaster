package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/review/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	reviewWindow      = 7 * 24 * time.Hour
	auditEntityReview = "review"
)

var reviewOrderable = []string{"id", "rating", "created_at"}

type (
	ReviewService interface {
		Create(ctx context.Context, req *v1.CreateReviewReq) (*v1.ReviewItem, error)
		ListMyReviews(ctx context.Context) (v1.MemberReviewsRes, error)
		List(ctx context.Context, req *v1.ListReviewsReq) (*v1.ListReviewsRes, error)
	}

	reviewService struct {
		repo  repository.ReviewRepository
		tx    postgres.TxManager
		audit AuditService
		now   func() time.Time
	}
)

func NewReviewService(repo repository.ReviewRepository, tx postgres.TxManager, audit AuditService) ReviewService {
	return &reviewService{repo: repo, tx: tx, audit: audit, now: time.Now}
}

func (s *reviewService) currentMember(ctx context.Context) (*entity.Member, error) {
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
	return m, nil
}

func isExpiredForReview(now time.Time, times ...*time.Time) bool {
	for _, t := range times {
		if t != nil && !t.IsZero() && now.Sub(*t) > reviewWindow {
			return true
		}
	}
	return false
}

func (s *reviewService) Create(ctx context.Context, req *v1.CreateReviewReq) (*v1.ReviewItem, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	if req.Rating < 1 || req.Rating > 5 {
		return nil, rescode.New(rescode.BadRequest, "Điểm đánh giá phải từ 1 đến 5 sao")
	}
	targetType := strings.ToUpper(strings.TrimSpace(req.TargetType))
	if targetType != entity.ReviewTargetPTSession && targetType != entity.ReviewTargetClassSchedule {
		return nil, rescode.New(rescode.BadRequest, "Loại buổi tập đánh giá không hợp lệ")
	}

	dup, err := s.repo.ExistsByMemberTarget(ctx, m.ID, targetType, req.TargetID)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, rescode.New(rescode.Conflict, "Bạn đã đánh giá buổi tập này rồi")
	}

	now := s.now()
	var (
		trainerID uint
		branchID  *uint
	)

	switch targetType {
	case entity.ReviewTargetPTSession:
		sess, err := s.repo.FindPTSession(ctx, req.TargetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, rescode.New(rescode.NotFound, "Không tìm thấy buổi tập PT")
			}
			return nil, err
		}
		if sess.MemberID != m.ID {
			return nil, rescode.New(rescode.PermissionDenied, "Bạn chỉ được đánh giá buổi tập PT của chính mình")
		}
		if sess.Status != entity.PTSessionCompleted {
			return nil, rescode.New(rescode.BadRequest, "Chỉ được đánh giá buổi tập PT đã hoàn thành")
		}
		endAt := sess.EndAt
		if isExpiredForReview(now, &endAt, sess.CompletedAt) {
			return nil, rescode.New(rescode.BadRequest, "Chỉ được đánh giá trong vòng 7 ngày kể từ khi buổi tập kết thúc")
		}
		trainerID = sess.TrainerID
		bid := sess.BranchID
		branchID = &bid

	case entity.ReviewTargetClassSchedule:
		sched, err := s.repo.FindClassSchedule(ctx, req.TargetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, rescode.New(rescode.NotFound, "Không tìm thấy buổi lớp học")
			}
			return nil, err
		}
		booking, err := s.repo.FindMemberClassBooking(ctx, m.ID, req.TargetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, rescode.New(rescode.BadRequest, "Chỉ được đánh giá lớp học mà bạn đã tham gia")
			}
			return nil, err
		}
		if booking.Status != entity.BookingStatusAttended {
			return nil, rescode.New(rescode.BadRequest, "Chỉ được đánh giá lớp học mà bạn đã tham gia")
		}
		endAt := sched.EndAt
		if isExpiredForReview(now, &endAt, booking.AttendedAt) {
			return nil, rescode.New(rescode.BadRequest, "Chỉ được đánh giá trong vòng 7 ngày kể từ khi buổi học kết thúc")
		}
		trainerID = sched.TrainerID
		bid := sched.BranchID
		branchID = &bid
	}

	rev := &entity.Review{
		MemberID:   m.ID,
		TargetType: targetType,
		TargetID:   req.TargetID,
		TrainerID:  trainerID,
		BranchID:   branchID,
		Rating:     req.Rating,
		Comment:    strings.TrimSpace(req.Comment),
	}

	run := func(txCtx context.Context) error {
		if err := s.repo.Create(txCtx, rev); err != nil {
			return mapDuplicate(err, "Bạn đã đánh giá buổi tập này rồi")
		}
		return s.repo.UpdateTrainerRating(txCtx, trainerID)
	}
	if s.tx != nil {
		err = s.tx.WithTx(ctx, run)
	} else {
		err = run(ctx)
	}
	if err != nil {
		return nil, err
	}

	res := toReviewItem(repository.ReviewRow{
		Review:     *rev,
		MemberCode: m.MemberCode,
		MemberName: m.FullName,
	})
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityReview, rev.ID, nil, res)
	}
	awardReviewPoints(ctx, m.ID, rev.ID)
	return res, nil
}

func (s *reviewService) ListMyReviews(ctx context.Context) (v1.MemberReviewsRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListByMember(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toReviewItem), nil
}

func (s *reviewService) List(ctx context.Context, req *v1.ListReviewsReq) (*v1.ListReviewsRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem đánh giá của chi nhánh này")
	}
	q := req.PageReq.ToQuery(reviewOrderable, "created_at")
	q.OrderBy = "r." + q.OrderBy
	rows, total, err := s.repo.List(ctx, q, repository.ReviewFilter{
		BranchIDs:  branchIDs,
		TrainerID:  req.TrainerID,
		TargetType: strings.TrimSpace(req.TargetType),
		Rating:     req.Rating,
	})
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, toReviewItem), total, q), nil
}

func toReviewItem(r repository.ReviewRow) *v1.ReviewItem {
	return &v1.ReviewItem{
		ID:          r.ID,
		MemberID:    r.MemberID,
		MemberCode:  r.MemberCode,
		MemberName:  r.MemberName,
		TargetType:  r.TargetType,
		TargetID:    r.TargetID,
		TrainerID:   r.TrainerID,
		TrainerName: r.TrainerName,
		BranchID:    r.BranchID,
		BranchName:  r.BranchName,
		Rating:      r.Rating,
		Comment:     r.Comment,
		CreatedAt:   r.CreatedAt,
	}
}
