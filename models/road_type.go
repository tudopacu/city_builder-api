package models

import "time"

type RoadType struct {
	ID        uint       `gorm:"primaryKey"`
	Type      string     `gorm:"not null"`
	ImageURL  *string    `gorm:"type:varchar(255)"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt *time.Time
}

func (RoadType) TableName() string {
	return "road_types"
}
