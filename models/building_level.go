package models

import "time"

type BuildingLevel struct {
	ID               uint `gorm:"primaryKey"`
	BuildingID       uint `gorm:"not null"`
	Level            uint `gorm:"not null"`
	BuildTimeSeconds uint `gorm:"not null"`
	Capacity         int  `gorm:"not null;default:0"`
	// PopulationCapacity is the population a building adds while it exists (housing).
	PopulationCapacity int `gorm:"not null;default:0"`
	// PopulationCost is the population a building uses while it exists.
	PopulationCost int `gorm:"not null;default:0"`
	ImageURL       *string
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      *time.Time

	Building Building `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (BuildingLevel) TableName() string {
	return "building_levels"
}
