package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/review/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

type fakeReviewRepo struct {
	nextID    uint
	members   map[uint]*entity.Member
	ptSess    map[uint]*entity.PTSession
	schedules map[uint]*entity.ClassSchedule
	bookings  []entity.ClassBooking
	reviews   []entity.Review
	trainers  map[uint]*entity.TrainerProfile
}

func newFakeReviewRepo() *fakeReviewRepo {
	return &fakeReviewRepo{
		members:   map[uint]*entity.Member{},
		ptSess:    map[uint]*entity.PTSession{},
		schedules: map[uint]*entity.ClassSchedule{},
		trainers:  map[uint]*entity.TrainerProfile{},
	}
}

func (f *fakeReviewRepo) FindMemberByUserID(_ context.Context, userID uint) (*entity.Member, error) {
	for _, m := range f.members {
		if m.UserID != nil && *m.UserID == userID {
			cp := *m
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeReviewRepo) FindPTSession(_ context.Context, id uint) (*entity.PTSession, error) {
	if s, ok := f.ptSess[id]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeReviewRepo) FindClassSchedule(_ context.Context, id uint) (*entity.ClassSchedule, error) {
	if s, ok := f.schedules[id]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeReviewRepo) FindMemberClassBooking(_ context.Context, memberID, scheduleID uint) (*entity.ClassBooking, error) {
	for _, b := range f.bookings {
		if b.MemberID == memberID && b.ScheduleID == scheduleID && b.Status == entity.BookingStatusAttended {
			cp := b
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeReviewRepo) ExistsByMemberTarget(_ context.Context, memberID uint, targetType string, targetID uint) (bool, error) {
	for _, r := range f.reviews {
		if r.MemberID == memberID && r.TargetType == targetType && r.TargetID == targetID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeReviewRepo) Create(_ context.Context, rev *entity.Review) error {
	f.nextID++
	rev.ID = f.nextID
	rev.CreatedAt = time.Now()
	f.reviews = append(f.reviews, *rev)
	return nil
}

func (f *fakeReviewRepo) UpdateTrainerRating(_ context.Context, trainerID uint) error {
	var sum, count int
	for _, r := range f.reviews {
		if r.TrainerID == trainerID {
			sum += r.Rating
			count++
		}
	}
	tp, ok := f.trainers[trainerID]
	if !ok {
		tp = &entity.TrainerProfile{UserID: trainerID}
		f.trainers[trainerID] = tp
	}
	tp.RatingCount = count
	if count > 0 {
		tp.RatingAvg = math.Round((float64(sum)/float64(count))*100) / 100
	} else {
		tp.RatingAvg = 0
	}
	return nil
}

func (f *fakeReviewRepo) ListByMember(_ context.Context, memberID uint) ([]repository.ReviewRow, error) {
	var out []repository.ReviewRow
	for _, r := range f.reviews {
		if r.MemberID == memberID {
			out = append(out, repository.ReviewRow{Review: r})
		}
	}
	return out, nil
}

func (f *fakeReviewRepo) List(_ context.Context, _ dto.Query, filter repository.ReviewFilter) ([]repository.ReviewRow, int64, error) {
	var out []repository.ReviewRow
	for _, r := range f.reviews {
		if filter.TrainerID > 0 && r.TrainerID != filter.TrainerID {
			continue
		}
		if filter.TargetType != "" && r.TargetType != filter.TargetType {
			continue
		}
		if filter.Rating > 0 && r.Rating != filter.Rating {
			continue
		}
		out = append(out, repository.ReviewRow{Review: r})
	}
	return out, int64(len(out)), nil
}

func TestReviewService_CreateAndTrainerRating(t *testing.T) {
	repo := newFakeReviewRepo()
	svc := NewReviewService(repo, nil, nil)

	now := time.Now()
	u1, u2 := uint(101), uint(102)
	repo.members[1] = &entity.Member{BaseEntity: entity.BaseEntity{ID: 1}, FullName: "Member 1", UserID: &u1}
	repo.members[2] = &entity.Member{BaseEntity: entity.BaseEntity{ID: 2}, FullName: "Member 2", UserID: &u2}
	repo.trainers[50] = &entity.TrainerProfile{UserID: 50}

	twoDaysAgo := now.Add(-2 * 24 * time.Hour)
	eightDaysAgo := now.Add(-8 * 24 * time.Hour)

	repo.ptSess[10] = &entity.PTSession{
		ID:          10,
		MemberID:    1,
		TrainerID:   50,
		BranchID:    1,
		EndAt:       twoDaysAgo,
		CompletedAt: &twoDaysAgo,
		Status:      entity.PTSessionCompleted,
	}
	repo.ptSess[11] = &entity.PTSession{
		ID:        11,
		MemberID:  1,
		TrainerID: 50,
		BranchID:  1,
		EndAt:     twoDaysAgo,
		Status:    entity.PTSessionScheduled,
	}
	repo.ptSess[12] = &entity.PTSession{
		ID:          12,
		MemberID:    1,
		TrainerID:   50,
		BranchID:    1,
		EndAt:       eightDaysAgo,
		CompletedAt: &eightDaysAgo,
		Status:      entity.PTSessionCompleted,
	}

	repo.schedules[20] = &entity.ClassSchedule{
		ID:        20,
		TrainerID: 50,
		BranchID:  1,
		EndAt:     twoDaysAgo,
		Status:    entity.ClassScheduleCompleted,
	}
	repo.bookings = append(repo.bookings, entity.ClassBooking{
		ID:         1,
		ScheduleID: 20,
		MemberID:   2,
		Status:     entity.BookingStatusAttended,
		AttendedAt: &twoDaysAgo,
	})

	ctxM1 := reqctx.WithAuth(context.Background(), &token.Payload{UserID: u1, Roles: []string{consts.RoleMember}})
	ctxM2 := reqctx.WithAuth(context.Background(), &token.Payload{UserID: u2, Roles: []string{consts.RoleMember}})

	// 1. Valid PT session review by Member 1 (rating 5)
	rev1, err := svc.Create(ctxM1, &v1.CreateReviewReq{
		TargetType: entity.ReviewTargetPTSession,
		TargetID:   10,
		Rating:     5,
		Comment:    "HLV rất nhiệt tình!",
	})
	require.NoError(t, err)
	assert.Equal(t, 5, rev1.Rating)
	assert.Equal(t, 5.0, repo.trainers[50].RatingAvg)
	assert.Equal(t, 1, repo.trainers[50].RatingCount)

	// 2. Duplicate review -> Conflict
	_, err = svc.Create(ctxM1, &v1.CreateReviewReq{
		TargetType: entity.ReviewTargetPTSession,
		TargetID:   10,
		Rating:     4,
	})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	// 3. Uncompleted PT session -> BadRequest
	_, err = svc.Create(ctxM1, &v1.CreateReviewReq{
		TargetType: entity.ReviewTargetPTSession,
		TargetID:   11,
		Rating:     4,
	})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// 4. Expired > 7 days PT session -> BadRequest
	_, err = svc.Create(ctxM1, &v1.CreateReviewReq{
		TargetType: entity.ReviewTargetPTSession,
		TargetID:   12,
		Rating:     4,
	})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// 5. Member 1 tries to review ClassSchedule 20 (not attended) -> BadRequest
	_, err = svc.Create(ctxM1, &v1.CreateReviewReq{
		TargetType: entity.ReviewTargetClassSchedule,
		TargetID:   20,
		Rating:     4,
	})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// 6. Member 2 reviews ClassSchedule 20 (attended, rating 4) -> recalculates trainer avg to 4.5
	_, err = svc.Create(ctxM2, &v1.CreateReviewReq{
		TargetType: entity.ReviewTargetClassSchedule,
		TargetID:   20,
		Rating:     4,
		Comment:    "Lớp học vui",
	})
	require.NoError(t, err)
	assert.Equal(t, 4.5, repo.trainers[50].RatingAvg)
	assert.Equal(t, 2, repo.trainers[50].RatingCount)
}
