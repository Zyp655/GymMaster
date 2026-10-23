package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type ReviewItem struct {
	ID          uint      `json:"id"`
	MemberID    uint      `json:"member_id"`
	MemberCode  string    `json:"member_code,omitempty"`
	MemberName  string    `json:"member_name,omitempty"`
	TargetType  string    `json:"target_type"`
	TargetID    uint      `json:"target_id"`
	TrainerID   uint      `json:"trainer_id"`
	TrainerName string    `json:"trainer_name,omitempty"`
	BranchID    *uint     `json:"branch_id"`
	BranchName  string    `json:"branch_name,omitempty"`
	Rating      int       `json:"rating"`
	Comment     string    `json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
}

// ---- Member (/v1/member) --------------------------------------------------------

type CreateReviewReq struct {
	g.Meta     `path:"/reviews" method:"post" tags:"Cổng hội viên" summary:"Đánh giá buổi tập PT hoặc lớp học" role:"MEMBER"`
	TargetType string `json:"target_type" v:"required|in:PT_SESSION,CLASS_SCHEDULE#Vui lòng chọn loại buổi tập|Loại buổi tập không hợp lệ"`
	TargetID   uint   `json:"target_id"   v:"required|min:1#Vui lòng chọn buổi tập"`
	Rating     int    `json:"rating"      v:"required|between:1,5#Vui lòng chọn số sao|Số sao phải từ 1 đến 5"`
	Comment    string `json:"comment"     v:"max-length:1000#Nhận xét tối đa 1000 ký tự"`
}
type CreateReviewRes = ReviewItem

type MemberReviewsReq struct {
	g.Meta `path:"/reviews" method:"get" tags:"Cổng hội viên" summary:"Danh sách đánh giá của tôi" role:"MEMBER"`
}
type MemberReviewsRes []*ReviewItem

// ---- Admin (/v1/admin) ----------------------------------------------------------

type ListReviewsReq struct {
	g.Meta `path:"/reviews" method:"get" tags:"Đánh giá" summary:"Danh sách đánh giá HLV & lớp học" perm:"review.view"`
	dto.PageReq
	TrainerID  uint   `json:"trainer_id"  in:"query"`
	TargetType string `json:"target_type" in:"query" v:"in:PT_SESSION,CLASS_SCHEDULE#Loại đánh giá không hợp lệ"`
	Rating     int    `json:"rating"      in:"query" v:"between:0,5"`
	BranchID   uint   `json:"branch_id"   in:"query"`
}
type ListReviewsRes = dto.PageRes[*ReviewItem]
