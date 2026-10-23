package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

// Exercise là bài tập trong thư viện.
type Exercise struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	MuscleGroup string    `json:"muscle_group"`
	Equipment   string    `json:"equipment"`
	VideoURL    string    `json:"video_url"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ExerciseInput dùng chung cho tạo/cập nhật bài tập.
type ExerciseInput struct {
	Name        string `json:"name"         v:"required|max-length:150#Vui lòng nhập tên bài tập|Tên bài tập tối đa 150 ký tự"`
	MuscleGroup string `json:"muscle_group" v:"required|in:CHEST,BACK,SHOULDERS,LEGS,ARMS,CORE,CARDIO,FULL_BODY#Vui lòng chọn nhóm cơ|Nhóm cơ không hợp lệ"`
	Equipment   string `json:"equipment"    v:"max-length:100#Dụng cụ tối đa 100 ký tự"`
	VideoURL    string `json:"video_url"    v:"max-length:500#Đường dẫn video tối đa 500 ký tự"`
	Description string `json:"description"`
}

// WorkoutPlanItem là một bài tập trong ngày của giáo án.
type WorkoutPlanItem struct {
	ID           uint    `json:"id"`
	DayID        uint    `json:"day_id"`
	ExerciseID   uint    `json:"exercise_id"`
	ExerciseName string  `json:"exercise_name"`
	MuscleGroup  string  `json:"muscle_group"`
	Equipment    string  `json:"equipment"`
	VideoURL     string  `json:"video_url"`
	Sets         int     `json:"sets"`
	Reps         int     `json:"reps"`
	Weight       float64 `json:"weight"`
	RestSec      int     `json:"rest_sec"`
	Note         string  `json:"note"`
	Sort         int     `json:"sort"`
}

// WorkoutPlanDay là một buổi/ngày trong giáo án.
type WorkoutPlanDay struct {
	ID       uint               `json:"id"`
	PlanID   uint               `json:"plan_id"`
	DayIndex int                `json:"day_index"`
	Title    string             `json:"title"`
	Items    []*WorkoutPlanItem `json:"items"`
}

// WorkoutPlan là giáo án tập luyện.
type WorkoutPlan struct {
	ID           uint               `json:"id"`
	MemberID     *uint              `json:"member_id"`
	MemberCode   string             `json:"member_code,omitempty"`
	MemberName   string             `json:"member_name,omitempty"`
	TrainerID    uint               `json:"trainer_id"`
	TrainerName  string             `json:"trainer_name,omitempty"`
	PTContractID *uint              `json:"pt_contract_id"`
	Name         string             `json:"name"`
	Goal         string             `json:"goal"`
	StartDate    *string            `json:"start_date"`
	EndDate      *string            `json:"end_date"`
	Status       string             `json:"status"`
	Days         []*WorkoutPlanDay  `json:"days,omitempty"`
	Items        []*WorkoutPlanItem `json:"items,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// WorkoutItemInput là dữ liệu đầu vào của một bài tập trong buổi.
type WorkoutItemInput struct {
	ExerciseID uint    `json:"exercise_id" v:"required|min:1#Vui lòng chọn bài tập|Bài tập không hợp lệ"`
	Sets       int     `json:"sets"        d:"3"  v:"min:1#Số hiệp phải >= 1"`
	Reps       int     `json:"reps"        d:"10" v:"min:1#Số lần lặp phải >= 1"`
	Weight     float64 `json:"weight"      v:"min:0|max:500#Mức tạ không được âm|Mức tạ tối đa 500 kg"`
	RestSec    int     `json:"rest_sec"    d:"60" v:"min:0#Thời gian nghỉ không được âm"`
	Note       string  `json:"note"        v:"max-length:255#Ghi chú tối đa 255 ký tự"`
	Sort       int     `json:"sort"`
}

// WorkoutDayInput là dữ liệu đầu vào của một buổi trong giáo án.
type WorkoutDayInput struct {
	DayIndex int                `json:"day_index" v:"required|between:1,14#Vui lòng nhập thứ tự ngày|Thứ tự ngày phải từ 1 đến 14"`
	Title    string             `json:"title"     v:"required|max-length:150#Vui lòng nhập tiêu đề buổi tập|Tiêu đề tối đa 150 ký tự"`
	Items    []WorkoutItemInput `json:"items"`
}

// WorkoutPlanInput dùng chung cho tạo/cập nhật giáo án.
type WorkoutPlanInput struct {
	MemberID     *uint             `json:"member_id"`
	TrainerID    uint              `json:"trainer_id"`
	PTContractID *uint             `json:"pt_contract_id"`
	Name         string            `json:"name"       v:"required|max-length:150#Vui lòng nhập tên giáo án|Tên giáo án tối đa 150 ký tự"`
	Goal         string            `json:"goal"       v:"max-length:255#Mục tiêu tối đa 255 ký tự"`
	StartDate    string            `json:"start_date"`
	EndDate      string            `json:"end_date"`
	Status       string            `json:"status"     d:"ACTIVE" v:"in:DRAFT,ACTIVE,COMPLETED,ARCHIVED#Trạng thái giáo án không hợp lệ"`
	Days         []WorkoutDayInput `json:"days"`
}

// WorkoutLogItem là một bản ghi nhật ký tập luyện của hội viên.
type WorkoutLogItem struct {
	ID           uint      `json:"id"`
	MemberID     uint      `json:"member_id"`
	PlanItemID   *uint     `json:"plan_item_id"`
	ExerciseID   uint      `json:"exercise_id"`
	ExerciseName string    `json:"exercise_name"`
	MuscleGroup  string    `json:"muscle_group"`
	PerformedAt  time.Time `json:"performed_at"`
	Sets         int       `json:"sets"`
	Reps         int       `json:"reps"`
	Weight       float64   `json:"weight"`
	Volume       float64   `json:"volume"`
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"created_at"`
}

// WorkoutProgressPoint là điểm thống kê tiến bộ theo ngày.
type WorkoutProgressPoint struct {
	Date        string  `json:"date"`
	MaxWeight   float64 `json:"max_weight"`
	TotalVolume float64 `json:"total_volume"`
	TotalSets   int     `json:"total_sets"`
}

// ---- Admin / HLV (/v1/admin) ----------------------------------------------------

type ListExercisesReq struct {
	g.Meta `path:"/exercises" method:"get" tags:"Giáo án & Tập luyện" summary:"Danh sách bài tập" perm:"workout.view,workout.manage"`
	dto.PageReq
	MuscleGroup string `json:"muscle_group" in:"query" v:"in:CHEST,BACK,SHOULDERS,LEGS,ARMS,CORE,CARDIO,FULL_BODY#Nhóm cơ không hợp lệ"`
	Search      string `json:"search"       in:"query"`
}
type ListExercisesRes = dto.PageRes[*Exercise]

type CreateExerciseReq struct {
	g.Meta `path:"/exercises" method:"post" tags:"Giáo án & Tập luyện" summary:"Thêm bài tập mới" perm:"workout.manage"`
	ExerciseInput
}
type CreateExerciseRes = Exercise

type UpdateExerciseReq struct {
	g.Meta `path:"/exercises/{id}" method:"put" tags:"Giáo án & Tập luyện" summary:"Cập nhật bài tập" perm:"workout.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	ExerciseInput
}
type UpdateExerciseRes = Exercise

type DeleteExerciseReq struct {
	g.Meta `path:"/exercises/{id}" method:"delete" tags:"Giáo án & Tập luyện" summary:"Xóa bài tập" perm:"workout.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteRes struct{}

type ListWorkoutPlansReq struct {
	g.Meta `path:"/workout-plans" method:"get" tags:"Giáo án & Tập luyện" summary:"Danh sách giáo án tập luyện" perm:"workout.view,workout.manage"`
	dto.PageReq
	MemberID  uint   `json:"member_id"  in:"query"`
	TrainerID uint   `json:"trainer_id" in:"query"`
	Status    string `json:"status"     in:"query" v:"in:DRAFT,ACTIVE,COMPLETED,ARCHIVED#Trạng thái không hợp lệ"`
	Search    string `json:"search"     in:"query"`
}
type ListWorkoutPlansRes = dto.PageRes[*WorkoutPlan]

type GetWorkoutPlanReq struct {
	g.Meta `path:"/workout-plans/{id}" method:"get" tags:"Giáo án & Tập luyện" summary:"Chi tiết giáo án tập luyện" perm:"workout.view,workout.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetWorkoutPlanRes = WorkoutPlan

type CreateWorkoutPlanReq struct {
	g.Meta `path:"/workout-plans" method:"post" tags:"Giáo án & Tập luyện" summary:"Tạo giáo án tập luyện" perm:"workout.manage"`
	WorkoutPlanInput
}
type CreateWorkoutPlanRes = WorkoutPlan

type UpdateWorkoutPlanReq struct {
	g.Meta `path:"/workout-plans/{id}" method:"put" tags:"Giáo án & Tập luyện" summary:"Cập nhật giáo án tập luyện" perm:"workout.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	WorkoutPlanInput
}
type UpdateWorkoutPlanRes = WorkoutPlan

type DeleteWorkoutPlanReq struct {
	g.Meta `path:"/workout-plans/{id}" method:"delete" tags:"Giáo án & Tập luyện" summary:"Xóa giáo án tập luyện" perm:"workout.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteWorkoutPlanRes struct{}

type CloneWorkoutPlanReq struct {
	g.Meta    `path:"/workout-plans/{id}/clone" method:"post" tags:"Giáo án & Tập luyện" summary:"Nhân bản giáo án cho hội viên" perm:"workout.manage"`
	ID        uint   `json:"id"         in:"path" v:"required|min:1"`
	MemberID  uint   `json:"member_id"  v:"required|min:1#Vui lòng chọn hội viên"`
	Name      string `json:"name"       v:"max-length:150"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}
type CloneWorkoutPlanRes = WorkoutPlan

// ---- Member (/v1/member) --------------------------------------------------------

type MemberExercisesReq struct {
	g.Meta      `path:"/exercises" method:"get" tags:"Cổng hội viên" summary:"Thư viện bài tập" role:"MEMBER"`
	MuscleGroup string `json:"muscle_group" in:"query"`
	Search      string `json:"search"       in:"query"`
}
type MemberExercisesRes []*Exercise

type MemberWorkoutPlansReq struct {
	g.Meta `path:"/workout-plans" method:"get" tags:"Cổng hội viên" summary:"Giáo án tập luyện của tôi" role:"MEMBER"`
}
type MemberWorkoutPlansRes []*WorkoutPlan

type CreateWorkoutLogReq struct {
	g.Meta      `path:"/workout-logs" method:"post" tags:"Cổng hội viên" summary:"Ghi nhật ký buổi tập" role:"MEMBER"`
	PlanItemID  *uint   `json:"plan_item_id"`
	ExerciseID  uint    `json:"exercise_id"  v:"required|min:1#Vui lòng chọn bài tập"`
	Sets        int     `json:"sets"         v:"required|min:1#Số hiệp phải >= 1"`
	Reps        int     `json:"reps"         v:"required|min:1#Số lần lặp phải >= 1"`
	Weight      float64 `json:"weight"       v:"min:0|max:500#Mức tạ không được âm|Mức tạ tối đa 500 kg"`
	Note        string  `json:"note"         v:"max-length:255"`
	PerformedAt string  `json:"performed_at"`
}
type CreateWorkoutLogRes = WorkoutLogItem

type MemberWorkoutLogsReq struct {
	g.Meta     `path:"/workout-logs" method:"get" tags:"Cổng hội viên" summary:"Lịch sử và tiến độ tập luyện của tôi" role:"MEMBER"`
	ExerciseID uint `json:"exercise_id" in:"query"`
	Limit      int  `json:"limit"       in:"query" d:"50"`
}
type MemberWorkoutLogsRes struct {
	Logs     []*WorkoutLogItem       `json:"logs"`
	Progress []*WorkoutProgressPoint `json:"progress"`
}
