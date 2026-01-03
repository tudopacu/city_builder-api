package models

import (
	"API/api/dto"
	"time"
)

type Building struct {
	ID                 uint      `gorm:"primaryKey"`
	Name               string    `gorm:"not null"`
	ImageURL           *string   `gorm:"type:varchar(255)" json:"image_url,omitempty"`
	Description        string    `gorm:"type:text"`
	Width              int       `gorm:"not null"`
	Length             int       `gorm:"not null"`
	BuildingCategoryID uint      `gorm:"not null"`
	CreatedAt          time.Time `gorm:"autoCreateTime"`
	UpdatedAt          *time.Time

	Category BuildingCategory `gorm:"foreignKey:BuildingCategoryID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Levels   []BuildingLevel  `gorm:"foreignKey:BuildingID"`
}

func (Building) TableName() string {
	return "buildings"
}

func (b Building) ToDTO() dto.Building {
	return dto.Building{
		ID:               b.ID,
		Name:             b.Name,
		ImageURL:         b.ImageURL,
		Description:      b.Description,
		Width:            b.Width,
		Length:           b.Length,
		BuildingCategory: b.Category.Name,
	}
}
