package entity

import "time"

const (
	EquipmentCategoryCardio     = "CARDIO"
	EquipmentCategoryStrength   = "STRENGTH"
	EquipmentCategoryFreeWeight = "FREE_WEIGHT"
	EquipmentCategoryFunctional = "FUNCTIONAL"
	EquipmentCategoryAccessory  = "ACCESSORY"
	EquipmentCategoryOther      = "OTHER"

	EquipmentStatusActive      = "ACTIVE"
	EquipmentStatusBroken      = "BROKEN"
	EquipmentStatusMaintenance = "MAINTENANCE"
	EquipmentStatusRetired     = "RETIRED"

	MaintenanceTypePeriodic = "PERIODIC"
	MaintenanceTypeRepair   = "REPAIR"

	MaintenanceStatusOpen       = "OPEN"
	MaintenanceStatusInProgress = "IN_PROGRESS"
	MaintenanceStatusDone       = "DONE"
)

// Equipment là thiết bị phòng tập.
type Equipment struct {
	BaseEntity
	BranchID                uint `gorm:"not null"`
	RoomID                  *uint
	Code                    string     `gorm:"type:varchar(30);not null"`
	Name                    string     `gorm:"type:varchar(150);not null"`
	Category                string     `gorm:"type:varchar(50);not null;default:'CARDIO'"`
	PurchaseDate            *time.Time `gorm:"type:date"`
	Price                   float64    `gorm:"type:numeric(14,2);not null;default:0"`
	WarrantyUntil           *time.Time `gorm:"type:date"`
	Status                  string     `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	MaintenanceIntervalDays int        `gorm:"not null;default:90"`
	NextMaintenanceDate     *time.Time `gorm:"type:date"`
	ImageURL                string     `gorm:"column:image_url;type:varchar(500);not null;default:''"`
	Note                    string     `gorm:"type:text;not null;default:''"`
}

func (Equipment) TableName() string { return "equipments" }

// MaintenanceLog là nhật ký bảo trì / sửa chữa thiết bị.
type MaintenanceLog struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	EquipmentID uint   `gorm:"not null"`
	Type        string `gorm:"type:varchar(20);not null"`
	ReportedBy  *uint
	Description string  `gorm:"type:text;not null;default:''"`
	Cost        float64 `gorm:"type:numeric(14,2);not null;default:0"`
	Status      string  `gorm:"type:varchar(20);not null;default:'OPEN'"`
	StartedAt   *time.Time
	FinishedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (MaintenanceLog) TableName() string { return "maintenance_logs" }
