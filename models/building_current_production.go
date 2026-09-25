package models

import "time"

type BuildingCurrentProduction struct {
	ID                   int64      `gorm:"primaryKey"`
	PlayerID             uint       `gorm:"not null"`
	PlayerBuildingID     uint       `gorm:"not null"`
	BuildingProductionID uint       `gorm:"not null"`
	EndTime              time.Time  `gorm:"not null"`
	Status               string     `gorm:"type:ENUM('PENDING','DONE','COLLECTED');not null;default:'PENDING'"`
	CreatedAt            time.Time  `gorm:"autoCreateTime"`
	UpdatedAt            *time.Time `gorm:"autoUpdateTime"`

	BuildingProduction BuildingProduction `gorm:"foreignKey:BuildingProductionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (BuildingCurrentProduction) TableName() string {
	return "building_current_productions"
}
