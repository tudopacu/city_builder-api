package models

import "time"

type BuildingConstructionCost struct {
	ID         uint      `gorm:"primaryKey"`
	BuildingID uint      `gorm:"not null;index:idx-building_construction_costs-building_id"`
	ItemID     uint      `gorm:"not null;index:idx-building_construction_costs-item_id"`
	Quantity   int       `gorm:"not null"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  *time.Time

	Building Building `gorm:"foreignKey:BuildingID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Item     Item     `gorm:"foreignKey:ItemID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (BuildingConstructionCost) TableName() string {
	return "building_construction_costs"
}
