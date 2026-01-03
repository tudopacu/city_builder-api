package models

import "time"

type BuildingCategory struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"not null"`
	Description *string   `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   *time.Time

	Buildings []Building `gorm:"foreignKey:BuildingCategoryID"`
}

func (BuildingCategory) TableName() string {
	return "building_categories"
}
