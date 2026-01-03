package models

import "time"

type BuildingLevel struct {
	ID               uint `gorm:"primaryKey"`
	BuildingID       uint `gorm:"not null"`
	Level            uint `gorm:"not null"`
	BuildTimeSeconds uint `gorm:"not null"`
	ImageURL         *string
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        *time.Time

	Building Building `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (BuildingLevel) TableName() string {
	return "building_levels"
}
