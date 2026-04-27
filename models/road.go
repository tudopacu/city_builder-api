package models

import (
	"API/api/dto"
	"time"
)

type Road struct {
	ID          uint       `gorm:"primaryKey"`
	Name        string     `gorm:"not null"`
	ImageURL    *string    `gorm:"type:varchar(255)" json:"image_url,omitempty"`
	Description *string    `gorm:"type:text"`
	Width       int        `gorm:"not null"`
	Length      int        `gorm:"not null"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   *time.Time
}

func (Road) TableName() string {
	return "roads"
}

func (r Road) ToDTO() dto.Road {
	return dto.Road{
		ID:          r.ID,
		Name:        r.Name,
		ImageURL:    r.ImageURL,
		Description: r.Description,
		Width:       r.Width,
		Length:      r.Length,
	}
}
