package entity

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// StaffProfile – hồ sơ nhân viên (1-1 với users). Chi nhánh làm việc thể hiện qua user_roles.branch_id.
type StaffProfile struct {
	UserID         uint   `gorm:"primaryKey;autoIncrement:false"`
	StaffCode      string `gorm:"type:varchar(20);not null"`
	Position       string `gorm:"type:varchar(30);not null"`
	Gender         string `gorm:"type:varchar(10);not null;default:'OTHER'"`
	Dob            *time.Time
	IDCard         string    `gorm:"column:id_card;type:varchar(20);not null;default:''"`
	Address        string    `gorm:"type:varchar(255);not null;default:''"`
	HireDate       time.Time `gorm:"type:date;not null"`
	BaseSalary     float64   `gorm:"type:numeric(14,2);not null;default:0"`
	CommissionRate float64   `gorm:"type:numeric(5,2);not null;default:0"`
	Note           string    `gorm:"type:text;not null;default:''"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (StaffProfile) TableName() string { return "staff_profiles" }

// TrainerProfile – hồ sơ huấn luyện viên (chỉ cho nhân viên position = TRAINER).
type TrainerProfile struct {
	UserID          uint           `gorm:"primaryKey;autoIncrement:false"`
	Level           string         `gorm:"type:varchar(20);not null;default:'JUNIOR'"`
	Specialties     StaffTextArray `gorm:"type:text[];not null;default:'{}'"`
	Bio             string         `gorm:"type:text;not null;default:''"`
	Certifications  string         `gorm:"type:text;not null;default:''"`
	YearsExperience int            `gorm:"not null;default:0"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (TrainerProfile) TableName() string { return "trainer_profiles" }

// StaffAccount là tài khoản người dùng kèm hồ sơ nhân sự (đọc từ bảng users).
type StaffAccount struct {
	User
	StaffProfile   *StaffProfile   `gorm:"foreignKey:UserID;references:ID"`
	TrainerProfile *TrainerProfile `gorm:"foreignKey:UserID;references:ID"`
}

func (StaffAccount) TableName() string { return "users" }

// Room – phòng tập dùng cho lớp nhóm.
type Room struct {
	BaseEntity
	BranchID    uint   `gorm:"not null"`
	Name        string `gorm:"type:varchar(100);not null"`
	Capacity    int    `gorm:"not null"`
	Description string `gorm:"type:varchar(255);not null;default:''"`
	Status      string `gorm:"type:varchar(20);not null;default:'ACTIVE'"`

	Branch *Branch `gorm:"foreignKey:BranchID"`
}

func (Room) TableName() string { return "rooms" }

// Shift – ca làm việc của nhân viên (không xóa mềm).
type Shift struct {
	ID        uint `gorm:"primaryKey;autoIncrement"`
	BranchID  uint `gorm:"not null"`
	StaffID   uint `gorm:"not null"`
	StartAt   time.Time
	EndAt     time.Time
	Note      string `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy *uint
	UpdatedBy *uint
}

func (Shift) TableName() string { return "shifts" }

// ShiftDetail là ca làm kèm thông tin hiển thị (tên nhân viên, chi nhánh) lấy bằng JOIN.
type ShiftDetail struct {
	Shift
	StaffName  string
	StaffCode  string
	Position   string
	BranchName string
}

// StaffTextArray ánh xạ cột TEXT[] của PostgreSQL sang []string.
type StaffTextArray []string

// Value mã hóa thành literal mảng PostgreSQL: {"a","b"}.
func (a StaffTextArray) Value() (driver.Value, error) {
	if a == nil {
		return "{}", nil
	}
	parts := make([]string, len(a))
	for i, s := range a {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		parts[i] = `"` + s + `"`
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

// Scan giải mã literal mảng PostgreSQL một chiều.
func (a *StaffTextArray) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case nil:
		*a = StaffTextArray{}
		return nil
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("StaffTextArray: kiểu không hỗ trợ %T", src)
	}
	out, err := parsePgTextArray(s)
	if err != nil {
		return err
	}
	*a = out
	return nil
}

func parsePgTextArray(s string) ([]string, error) {
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return nil, fmt.Errorf("mảng không hợp lệ: %q", s)
	}
	body := s[1 : len(s)-1]
	out := []string{}
	if body == "" {
		return out, nil
	}
	var (
		cur     strings.Builder
		quoted  bool
		inQuote bool
		escaped bool
	)
	flush := func() {
		v := cur.String()
		if !quoted && strings.EqualFold(v, "NULL") {
			v = ""
		}
		out = append(out, v)
		cur.Reset()
		quoted = false
	}
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case escaped:
			cur.WriteByte(c)
			escaped = false
		case c == '\\':
			escaped = true
		case c == '"':
			inQuote = !inQuote
			quoted = true
		case c == ',' && !inQuote:
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return out, nil
}
