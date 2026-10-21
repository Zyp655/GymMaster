package entity

import (
	"time"

	"gorm.io/datatypes"
)

// Sprint 22: Điểm thưởng, huy hiệu, thử thách, giới thiệu.
const (
	LoyaltyReasonCheckin   = "CHECKIN"
	LoyaltyReasonRenew     = "RENEW"
	LoyaltyReasonReferral  = "REFERRAL"
	LoyaltyReasonReview    = "REVIEW"
	LoyaltyReasonRedeem    = "REDEEM"
	LoyaltyReasonChallenge = "CHALLENGE"
	LoyaltyReasonAdjust    = "ADJUST"

	RewardTypeVoucher = "VOUCHER"
	RewardTypeProduct = "PRODUCT"

	ChallengeMetricCheckins   = "CHECKINS"
	ChallengeMetricClasses    = "CLASSES"
	ChallengeMetricPTSessions = "PT_SESSIONS"
)

type LoyaltyLedger struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	MemberID  uint   `gorm:"not null"`
	Points    int    `gorm:"not null"`
	Reason    string `gorm:"type:varchar(30);not null"`
	RefType   string `gorm:"type:varchar(30);not null;default:''"`
	RefID     *uint
	Note      string `gorm:"type:varchar(255);not null;default:''"`
	CreatedBy *uint
	CreatedAt time.Time
}

func (LoyaltyLedger) TableName() string { return "loyalty_ledgers" }

type LoyaltyReward struct {
	ID          uint    `gorm:"primaryKey;autoIncrement"`
	Name        string  `gorm:"type:varchar(150);not null"`
	Description string  `gorm:"type:varchar(500);not null;default:''"`
	CostPoints  int     `gorm:"not null"`
	RewardType  string  `gorm:"type:varchar(20);not null"`
	RefID       *uint
	DiscountPct float64 `gorm:"type:numeric(5,2);not null;default:10"`
	Stock       int     `gorm:"not null;default:100"`
	Status      string  `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (LoyaltyReward) TableName() string { return "loyalty_rewards" }

type Badge struct {
	ID          uint           `gorm:"primaryKey;autoIncrement"`
	Code        string         `gorm:"type:varchar(50);not null;unique"`
	Name        string         `gorm:"type:varchar(150);not null"`
	Description string         `gorm:"type:varchar(255);not null;default:''"`
	Icon        string         `gorm:"type:varchar(100);not null;default:'trophy'"`
	Rule        datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt   time.Time
}

func (Badge) TableName() string { return "badges" }

type MemberBadge struct {
	MemberID uint      `gorm:"primaryKey"`
	BadgeID  uint      `gorm:"primaryKey"`
	EarnedAt time.Time `gorm:"not null"`
}

func (MemberBadge) TableName() string { return "member_badges" }

type Challenge struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	BranchID     *uint
	Name         string    `gorm:"type:varchar(150);not null"`
	Description  string    `gorm:"type:varchar(500);not null;default:''"`
	Metric       string    `gorm:"type:varchar(30);not null"`
	Target       int       `gorm:"not null"`
	StartDate    time.Time `gorm:"type:date;not null"`
	EndDate      time.Time `gorm:"type:date;not null"`
	RewardPoints int       `gorm:"not null;default:100"`
	Status       string    `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (Challenge) TableName() string { return "challenges" }

type ChallengeParticipant struct {
	ChallengeID uint `gorm:"primaryKey"`
	MemberID    uint `gorm:"primaryKey"`
	Progress    int  `gorm:"not null;default:0"`
	CompletedAt *time.Time
	JoinedAt    time.Time `gorm:"not null"`
}

func (ChallengeParticipant) TableName() string { return "challenge_participants" }

// Sprint 23: Dinh dưỡng, bảng tin & khảo sát NPS.
const (
	MealBreakfast = "BREAKFAST"
	MealLunch     = "LUNCH"
	MealDinner    = "DINNER"
	MealSnack     = "SNACK"

	SurveyTypeNPS    = "NPS"
	SurveyTypeCustom = "CUSTOM"
)

type Food struct {
	ID        uint    `gorm:"primaryKey;autoIncrement"`
	Name      string  `gorm:"type:varchar(150);not null;unique"`
	Unit      string  `gorm:"type:varchar(50);not null;default:'100g'"`
	Calories  float64 `gorm:"type:numeric(7,2);not null;default:0"`
	Protein   float64 `gorm:"type:numeric(6,2);not null;default:0"`
	Carbs     float64 `gorm:"type:numeric(6,2);not null;default:0"`
	Fat       float64 `gorm:"type:numeric(6,2);not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Food) TableName() string { return "foods" }

type MealPlan struct {
	ID             uint       `gorm:"primaryKey;autoIncrement"`
	MemberID       uint       `gorm:"not null"`
	TrainerID      *uint
	Name           string     `gorm:"type:varchar(150);not null"`
	Goal           string     `gorm:"type:varchar(255);not null;default:''"`
	TargetCalories int        `gorm:"not null;default:2000"`
	StartDate      *time.Time `gorm:"type:date"`
	EndDate        *time.Time `gorm:"type:date"`
	Status         string     `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (MealPlan) TableName() string { return "meal_plans" }

type MealPlanItem struct {
	ID       uint    `gorm:"primaryKey;autoIncrement"`
	PlanID   uint    `gorm:"not null"`
	DayIndex int     `gorm:"not null;default:1"`
	Meal     string  `gorm:"type:varchar(20);not null"`
	FoodID   uint    `gorm:"not null"`
	Qty      float64 `gorm:"type:numeric(6,2);not null;default:1"`
	Note     string  `gorm:"type:varchar(255);not null;default:''"`
}

func (MealPlanItem) TableName() string { return "meal_plan_items" }

type FoodLog struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	MemberID  uint      `gorm:"not null"`
	FoodID    uint      `gorm:"not null"`
	Qty       float64   `gorm:"type:numeric(10,4);not null;default:1"`
	Meal      string    `gorm:"type:varchar(20);not null"`
	EatenAt   time.Time `gorm:"type:date;not null"`
	CreatedAt time.Time
}

func (FoodLog) TableName() string { return "food_logs" }

type Announcement struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	BranchID  *uint
	Title     string    `gorm:"type:varchar(200);not null"`
	Body      string    `gorm:"type:text;not null"`
	ImageURL  string    `gorm:"column:image_url;type:varchar(500);not null;default:''"`
	PublishAt time.Time `gorm:"not null"`
	ExpireAt  *time.Time
	Pinned    bool `gorm:"not null;default:false"`
	CreatedBy *uint
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Announcement) TableName() string { return "announcements" }

type Survey struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	Title       string    `gorm:"type:varchar(200);not null"`
	Description string    `gorm:"type:varchar(500);not null;default:''"`
	Type        string    `gorm:"type:varchar(20);not null;default:'NPS'"`
	StartAt     time.Time `gorm:"not null"`
	EndAt       *time.Time
	Status      string `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Survey) TableName() string { return "surveys" }

type SurveyResponse struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	SurveyID  uint           `gorm:"not null"`
	MemberID  uint           `gorm:"not null"`
	Score     int            `gorm:"not null"`
	Comment   string         `gorm:"type:text;not null;default:''"`
	Answers   datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt time.Time
}

func (SurveyResponse) TableName() string { return "survey_responses" }
