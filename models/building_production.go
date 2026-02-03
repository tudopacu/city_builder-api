package models

import "time"

type BuildingProduction struct {
	ID                    uint      `gorm:"primaryKey"`
	BuildingID            uint      `gorm:"not null;index:idx-building_productions-building_id"`
	ItemID                uint      `gorm:"not null;index:idx-building_productions-item_id"`
	ProductionTimeSeconds int       `gorm:"not null"`
	Quantity              int       `gorm:"not null;default:1"`
	CreatedAt             time.Time `gorm:"autoCreateTime"`
	UpdatedAt             *time.Time

	Building Building `gorm:"foreignKey:BuildingID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Item     Item     `gorm:"foreignKey:ItemID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (BuildingProduction) TableName() string {
	return "building_productions"
}
