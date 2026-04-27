package models

import "time"

type Intersection struct {
	ID        uint       `gorm:"primaryKey"`
	MapID     *uint      `gorm:"index"`
	PlayerID  *uint      `gorm:"index"`
	X         int        `gorm:"not null"`
	Y         int        `gorm:"not null"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt *time.Time
}

func (Intersection) TableName() string {
	return "intersections"
}
