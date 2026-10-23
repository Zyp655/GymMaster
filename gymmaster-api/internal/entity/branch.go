package entity

type Branch struct {
	BaseEntity
	Code      string `gorm:"type:varchar(20);not null"`
	Name      string `gorm:"type:varchar(150);not null"`
	Address   string `gorm:"type:varchar(255);not null;default:''"`
	Phone     string `gorm:"type:varchar(20);not null;default:''"`
	Email     string `gorm:"type:varchar(150);not null;default:''"`
	OpenTime  string `gorm:"type:varchar(5);not null"`
	CloseTime string `gorm:"type:varchar(5);not null"`
	Capacity  int    `gorm:"not null;default:120"`
	Status    string `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
}

func (Branch) TableName() string { return "branches" }
