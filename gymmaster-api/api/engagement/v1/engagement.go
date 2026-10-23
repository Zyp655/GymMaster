package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type FoodItem struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Unit      string    `json:"unit"`
	Calories  float64   `json:"calories"`
	Protein   float64   `json:"protein"`
	Carbs     float64   `json:"carbs"`
	Fat       float64   `json:"fat"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FoodInput struct {
	Name     string  `json:"name"     v:"required|max-length:150#Vui lòng nhập tên thực phẩm"`
	Unit     string  `json:"unit"     d:"100g" v:"max-length:50"`
	Calories float64 `json:"calories" v:"min:0#Calo không được âm"`
	Protein  float64 `json:"protein"  v:"min:0#Protein không được âm"`
	Carbs    float64 `json:"carbs"    v:"min:0#Carbs không được âm"`
	Fat      float64 `json:"fat"      v:"min:0#Fat không được âm"`
}

type MealPlanItemDetail struct {
	ID       uint    `json:"id"`
	PlanID   uint    `json:"plan_id"`
	DayIndex int     `json:"day_index"`
	Meal     string  `json:"meal"`
	FoodID   uint    `json:"food_id"`
	FoodName string  `json:"food_name"`
	Unit     string  `json:"unit"`
	Qty      float64 `json:"qty"`
	Calories float64 `json:"calories"`
	Protein  float64 `json:"protein"`
	Carbs    float64 `json:"carbs"`
	Fat      float64 `json:"fat"`
	Note     string  `json:"note"`
}

type MealPlanItemInput struct {
	DayIndex int     `json:"day_index" d:"1" v:"between:1,7#Thứ tự ngày phải từ 1 đến 7"`
	Meal     string  `json:"meal"      d:"BREAKFAST" v:"required|in:BREAKFAST,LUNCH,DINNER,SNACK#Bữa ăn không hợp lệ"`
	FoodID   uint    `json:"food_id"   v:"required|min:1#Vui lòng chọn thực phẩm"`
	Qty      float64 `json:"qty"       d:"1" v:"min:0.1#Khẩu phần phải lớn hơn 0"`
	Note     string  `json:"note"      v:"max-length:255"`
}

type MealPlanDetail struct {
	ID             uint                  `json:"id"`
	MemberID       uint                  `json:"member_id"`
	MemberCode     string                `json:"member_code,omitempty"`
	MemberName     string                `json:"member_name,omitempty"`
	TrainerID      *uint                 `json:"trainer_id"`
	TrainerName    string                `json:"trainer_name,omitempty"`
	Name           string                `json:"name"`
	Goal           string                `json:"goal"`
	TargetCalories int                   `json:"target_calories"`
	StartDate      *string               `json:"start_date"`
	EndDate        *string               `json:"end_date"`
	Status         string                `json:"status"`
	Items          []*MealPlanItemDetail `json:"items"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type MealPlanInput struct {
	MemberID       uint                `json:"member_id"       v:"required|min:1#Vui lòng chọn hội viên"`
	TrainerID      *uint               `json:"trainer_id"`
	Name           string              `json:"name"            v:"required|max-length:150#Vui lòng nhập tên thực đơn"`
	Goal           string              `json:"goal"            v:"max-length:255"`
	TargetCalories int                 `json:"target_calories" d:"2000" v:"min:500#Mục tiêu calo phải >= 500"`
	StartDate      string              `json:"start_date"`
	EndDate        string              `json:"end_date"`
	Status         string              `json:"status"          d:"ACTIVE" v:"in:ACTIVE,ARCHIVED"`
	Items          []MealPlanItemInput `json:"items"`
}

type FoodLogItem struct {
	ID        uint      `json:"id"`
	MemberID  uint      `json:"member_id"`
	FoodID    uint      `json:"food_id"`
	FoodName  string    `json:"food_name"`
	Unit      string    `json:"unit"`
	Qty       float64   `json:"qty"`
	Meal      string    `json:"meal"`
	Calories  float64   `json:"calories"`
	Protein   float64   `json:"protein"`
	Carbs     float64   `json:"carbs"`
	Fat       float64   `json:"fat"`
	EatenAt   string    `json:"eaten_at"`
	CreatedAt time.Time `json:"created_at"`
}

type AnnouncementItem struct {
	ID          uint       `json:"id"`
	BranchID    *uint      `json:"branch_id"`
	BranchName  string     `json:"branch_name,omitempty"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	ImageURL    string     `json:"image_url"`
	PublishAt   time.Time  `json:"publish_at"`
	ExpireAt    *time.Time `json:"expire_at"`
	Pinned      bool       `json:"pinned"`
	CreatedBy   *uint      `json:"created_by"`
	CreatorName string     `json:"creator_name,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type AnnouncementInput struct {
	BranchID  *uint  `json:"branch_id"`
	Title     string `json:"title"      v:"required|max-length:200#Vui lòng nhập tiêu đề thông báo"`
	Body      string `json:"body"       v:"required#Vui lòng nhập nội dung thông báo"`
	ImageURL  string `json:"image_url"  v:"max-length:500"`
	PublishAt string `json:"publish_at"`
	ExpireAt  string `json:"expire_at"`
	Pinned    bool   `json:"pinned"`
}

type SurveyItem struct {
	ID            uint       `json:"id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Type          string     `json:"type"`
	StartAt       time.Time  `json:"start_at"`
	EndAt         *time.Time `json:"end_at"`
	Status        string     `json:"status"`
	ResponseCount int64      `json:"response_count"`
	Responded     bool       `json:"responded"`
	MyScore       *int       `json:"my_score,omitempty"`
	MyComment     string     `json:"my_comment,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type SurveyInput struct {
	Title       string `json:"title"       v:"required|max-length:200#Vui lòng nhập tiêu đề khảo sát"`
	Description string `json:"description" v:"max-length:500"`
	Type        string `json:"type"        d:"NPS"    v:"in:NPS,CUSTOM"`
	StartAt     string `json:"start_at"`
	EndAt       string `json:"end_at"`
	Status      string `json:"status"      d:"ACTIVE" v:"in:ACTIVE,CLOSED"`
}

type SurveyResponseItem struct {
	ID          uint      `json:"id"`
	SurveyID    uint      `json:"survey_id"`
	SurveyTitle string    `json:"survey_title,omitempty"`
	MemberID    uint      `json:"member_id"`
	MemberCode  string    `json:"member_code,omitempty"`
	MemberName  string    `json:"member_name,omitempty"`
	Score       int       `json:"score"`
	Category    string    `json:"category"` // PROMOTER | PASSIVE | DETRACTOR
	Comment     string    `json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
}

// ---- Admin (/v1/admin) ----------------------------------------------------------

type ListFoodsReq struct {
	g.Meta `path:"/foods" method:"get" tags:"Dinh dưỡng & Tương tác" summary:"Danh mục thực phẩm dinh dưỡng" perm:"engagement.view,engagement.manage"`
	dto.PageReq
	Search string `json:"search" in:"query"`
}
type ListFoodsRes = dto.PageRes[*FoodItem]

type CreateFoodReq struct {
	g.Meta `path:"/foods" method:"post" tags:"Dinh dưỡng & Tương tác" summary:"Thêm thực phẩm mới" perm:"engagement.manage"`
	FoodInput
}
type CreateFoodRes = FoodItem

type UpdateFoodReq struct {
	g.Meta `path:"/foods/{id}" method:"put" tags:"Dinh dưỡng & Tương tác" summary:"Cập nhật thực phẩm" perm:"engagement.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	FoodInput
}
type UpdateFoodRes = FoodItem

type DeleteFoodReq struct {
	g.Meta `path:"/foods/{id}" method:"delete" tags:"Dinh dưỡng & Tương tác" summary:"Xóa thực phẩm" perm:"engagement.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteEngagementRes struct{}

type ListMealPlansReq struct {
	g.Meta `path:"/meal-plans" method:"get" tags:"Dinh dưỡng & Tương tác" summary:"Danh sách thực đơn dinh dưỡng" perm:"engagement.view,engagement.manage"`
	dto.PageReq
	MemberID  uint   `json:"member_id"  in:"query"`
	TrainerID uint   `json:"trainer_id" in:"query"`
	Status    string `json:"status"     in:"query"`
}
type ListMealPlansRes = dto.PageRes[*MealPlanDetail]

type GetMealPlanReq struct {
	g.Meta `path:"/meal-plans/{id}" method:"get" tags:"Dinh dưỡng & Tương tác" summary:"Chi tiết thực đơn dinh dưỡng" perm:"engagement.view,engagement.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetMealPlanRes = MealPlanDetail

type CreateMealPlanReq struct {
	g.Meta `path:"/meal-plans" method:"post" tags:"Dinh dưỡng & Tương tác" summary:"Tạo thực đơn dinh dưỡng" perm:"engagement.manage"`
	MealPlanInput
}
type CreateMealPlanRes = MealPlanDetail

type UpdateMealPlanReq struct {
	g.Meta `path:"/meal-plans/{id}" method:"put" tags:"Dinh dưỡng & Tương tác" summary:"Cập nhật thực đơn dinh dưỡng" perm:"engagement.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	MealPlanInput
}
type UpdateMealPlanRes = MealPlanDetail

type DeleteMealPlanReq struct {
	g.Meta `path:"/meal-plans/{id}" method:"delete" tags:"Dinh dưỡng & Tương tác" summary:"Xóa thực đơn dinh dưỡng" perm:"engagement.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type ListAnnouncementsReq struct {
	g.Meta `path:"/announcements" method:"get" tags:"Dinh dưỡng & Tương tác" summary:"Danh sách thông báo bảng tin" perm:"engagement.view,engagement.manage"`
	dto.PageReq
	BranchID uint `json:"branch_id" in:"query"`
}
type ListAnnouncementsRes = dto.PageRes[*AnnouncementItem]

type CreateAnnouncementReq struct {
	g.Meta `path:"/announcements" method:"post" tags:"Dinh dưỡng & Tương tác" summary:"Đăng thông báo mới" perm:"engagement.manage"`
	AnnouncementInput
}
type CreateAnnouncementRes = AnnouncementItem

type UpdateAnnouncementReq struct {
	g.Meta `path:"/announcements/{id}" method:"put" tags:"Dinh dưỡng & Tương tác" summary:"Cập nhật thông báo" perm:"engagement.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	AnnouncementInput
}
type UpdateAnnouncementRes = AnnouncementItem

type DeleteAnnouncementReq struct {
	g.Meta `path:"/announcements/{id}" method:"delete" tags:"Dinh dưỡng & Tương tác" summary:"Xóa thông báo" perm:"engagement.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type ListSurveysReq struct {
	g.Meta `path:"/surveys" method:"get" tags:"Dinh dưỡng & Tương tác" summary:"Danh sách khảo sát NPS" perm:"engagement.view,engagement.manage"`
	dto.PageReq
	Status string `json:"status" in:"query"`
}
type ListSurveysRes = dto.PageRes[*SurveyItem]

type CreateSurveyReq struct {
	g.Meta `path:"/surveys" method:"post" tags:"Dinh dưỡng & Tương tác" summary:"Tạo khảo sát mới" perm:"engagement.manage"`
	SurveyInput
}
type CreateSurveyRes = SurveyItem

type UpdateSurveyReq struct {
	g.Meta `path:"/surveys/{id}" method:"put" tags:"Dinh dưỡng & Tương tác" summary:"Cập nhật khảo sát" perm:"engagement.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	SurveyInput
}
type UpdateSurveyRes = SurveyItem

type DeleteSurveyReq struct {
	g.Meta `path:"/surveys/{id}" method:"delete" tags:"Dinh dưỡng & Tương tác" summary:"Xóa khảo sát" perm:"engagement.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type NPSReportReq struct {
	g.Meta   `path:"/reports/nps" method:"get" tags:"Dinh dưỡng & Tương tác" summary:"Báo cáo chỉ số hài lòng NPS" perm:"engagement.view,report.view"`
	SurveyID uint `json:"survey_id" in:"query"`
}
type NPSReportRes struct {
	NPSScore        float64               `json:"nps_score"`
	AverageScore    float64               `json:"average_score"`
	TotalResponses  int64                 `json:"total_responses"`
	Promoters       int64                 `json:"promoters"`
	Passives        int64                 `json:"passives"`
	Detractors      int64                 `json:"detractors"`
	PromoterPct     float64               `json:"promoter_pct"`
	PassivePct      float64               `json:"passive_pct"`
	DetractorPct    float64               `json:"detractor_pct"`
	RecentResponses []*SurveyResponseItem `json:"recent_responses"`
	Responses       []*SurveyResponseItem `json:"responses"`
}

// ---- Member (/v1/member) --------------------------------------------------------

type MemberNutritionOverviewReq struct {
	g.Meta `path:"/meal-plans" method:"get" tags:"Cổng hội viên" summary:"Tổng quan dinh dưỡng & thực đơn của tôi" role:"MEMBER"`
	Goal   string `json:"goal" in:"query" dc:"Mục tiêu để chọn goal_target: CUT (giảm mỡ) | MAINTAIN (duy trì) | BULK (tăng cơ). Bỏ trống = suy từ thực đơn đang áp dụng, mặc định MAINTAIN"`
}

// NutritionGoalTarget: gợi ý calo & macro/ngày theo một mục tiêu (NU-01).
type NutritionGoalTarget struct {
	Goal          string `json:"goal"           dc:"CUT | MAINTAIN | BULK"`
	Label         string `json:"label"          dc:"Tên mục tiêu tiếng Việt"`
	AdjustmentPct int    `json:"adjustment_pct" dc:"Điều chỉnh so với TDEE (%): CUT -20, MAINTAIN 0, BULK +10"`
	Calories      int    `json:"calories"       dc:"kcal/ngày"`
	ProteinG      int    `json:"protein_g"`
	CarbsG        int    `json:"carbs_g"`
	FatG          int    `json:"fat_g"`
}

type MemberNutritionOverviewRes struct {
	RecommendedCalories int               `json:"recommended_calories" dc:"Calo duy trì (TDEE) – giữ như cũ"`
	Plans               []*MealPlanDetail `json:"plans"`
	Foods               []*FoodItem       `json:"foods"`
	// NU-01: gợi ý theo mục tiêu (bổ sung, không đổi các trường cũ).
	BMR            int                    `json:"bmr"             dc:"BMR Mifflin–St Jeor (0 khi chưa có chỉ số cơ thể)"`
	TDEE           int                    `json:"tdee"            dc:"BMR × hệ số vận động (= recommended_calories)"`
	ActivityFactor float64                `json:"activity_factor" dc:"Hệ số vận động (1.55 – vận động vừa)"`
	HasBodyMetric  bool                   `json:"has_body_metric" dc:"false → TDEE là giá trị mặc định 2000 kcal"`
	Goal           string                 `json:"goal"            dc:"Mục tiêu đang áp dụng: CUT | MAINTAIN | BULK"`
	GoalSource     string                 `json:"goal_source"     dc:"QUERY (tham số goal) | MEAL_PLAN (suy từ thực đơn) | DEFAULT"`
	GoalTarget     *NutritionGoalTarget   `json:"goal_target"     dc:"Gợi ý cho mục tiêu đang áp dụng"`
	GoalTargets    []*NutritionGoalTarget `json:"goal_targets"    dc:"Gợi ý cho cả 3 mục tiêu (CUT, MAINTAIN, BULK)"`
}

type MemberFoodLogsReq struct {
	g.Meta `path:"/food-logs" method:"get" tags:"Cổng hội viên" summary:"Nhật ký bữa ăn trong ngày" role:"MEMBER"`
	Date   string `json:"date" in:"query"`
}
type MemberFoodLogsRes struct {
	Date          string         `json:"date"`
	TotalCalories float64        `json:"total_calories"`
	TotalProtein  float64        `json:"total_protein"`
	TotalCarbs    float64        `json:"total_carbs"`
	TotalFat      float64        `json:"total_fat"`
	Logs          []*FoodLogItem `json:"logs"`
	Items         []*FoodLogItem `json:"items"`
}

type CreateFoodLogReq struct {
	g.Meta  `path:"/food-logs" method:"post" tags:"Cổng hội viên" summary:"Ghi nhật ký ăn uống" role:"MEMBER"`
	FoodID  uint    `json:"food_id"  v:"required|min:1#Vui lòng chọn thực phẩm"`
	Qty     float64 `json:"qty"      d:"1" v:"required|min:0.0001|max:100#Vui lòng nhập khẩu phần|Khẩu phần phải lớn hơn 0|Khẩu phần tối đa 100 (VD: món 100g tối đa 10.000g)"`
	Meal    string  `json:"meal"     d:"BREAKFAST" v:"required|in:BREAKFAST,LUNCH,DINNER,SNACK#Bữa ăn không hợp lệ"`
	EatenAt string  `json:"eaten_at"`
}
type CreateFoodLogRes = FoodLogItem

type UpdateFoodLogReq struct {
	g.Meta `path:"/food-logs/{id}" method:"put" tags:"Cổng hội viên" summary:"Sửa bản ghi ăn uống" role:"MEMBER"`
	ID     uint    `json:"id"      in:"path" v:"required|min:1"`
	FoodID uint    `json:"food_id" v:"required|min:1#Vui lòng chọn thực phẩm"`
	Qty    float64 `json:"qty"     v:"required|min:0.0001|max:100#Vui lòng nhập khẩu phần|Khẩu phần phải lớn hơn 0|Khẩu phần tối đa 100 (VD: món 100g tối đa 10.000g)"`
	Meal   string  `json:"meal"    v:"required|in:BREAKFAST,LUNCH,DINNER,SNACK#Vui lòng chọn bữa ăn|Bữa ăn không hợp lệ"`
}
type UpdateFoodLogRes = FoodLogItem

type DeleteFoodLogReq struct {
	g.Meta `path:"/food-logs/{id}" method:"delete" tags:"Cổng hội viên" summary:"Xóa bản ghi ăn uống" role:"MEMBER"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type MemberAnnouncementsReq struct {
	g.Meta `path:"/announcements" method:"get" tags:"Cổng hội viên" summary:"Bảng tin phòng tập" role:"MEMBER"`
}
type MemberAnnouncementsRes []*AnnouncementItem

type MemberSurveysReq struct {
	g.Meta `path:"/surveys" method:"get" tags:"Cổng hội viên" summary:"Danh sách khảo sát đang mở" role:"MEMBER"`
}
type MemberSurveysRes []*SurveyItem

type SubmitSurveyResponseReq struct {
	g.Meta  `path:"/surveys/{id}/responses" method:"post" tags:"Cổng hội viên" summary:"Gửi phản hồi khảo sát NPS" role:"MEMBER"`
	ID      uint           `json:"id"      in:"path" v:"required|min:1"`
	Score   int            `json:"score"             v:"between:0,10#Điểm đánh giá từ 0 đến 10"`
	Comment string         `json:"comment"           v:"max-length:1000"`
	Answers map[string]any `json:"answers"`
}
type SubmitSurveyResponseRes = SurveyResponseItem
