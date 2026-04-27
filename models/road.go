package models

import (
	"API/api/dto"
	"time"
)

type Road struct {
	ID                  uint       `gorm:"primaryKey"`
	StartIntersectionID *uint
	EndIntersectionID   *uint
	RoadTypeID          *uint
	CreatedAt           time.Time  `gorm:"autoCreateTime"`
	UpdatedAt           *time.Time

	StartIntersection Intersection `gorm:"foreignKey:StartIntersectionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	EndIntersection   Intersection `gorm:"foreignKey:EndIntersectionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	RoadType          RoadType     `gorm:"foreignKey:RoadTypeID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (Road) TableName() string {
	return "roads"
}

func (r Road) ToDTO() dto.Road {
	return dto.Road{
		ID: r.ID,
		StartIntersection: dto.Intersection{
			ID: r.StartIntersection.ID,
			X:  r.StartIntersection.X,
			Y:  r.StartIntersection.Y,
		},
		EndIntersection: dto.Intersection{
			ID: r.EndIntersection.ID,
			X:  r.EndIntersection.X,
			Y:  r.EndIntersection.Y,
		},
		RoadType: dto.RoadType{
			ID:       r.RoadType.ID,
			Type:     r.RoadType.Type,
			ImageURL: r.RoadType.ImageURL,
		},
	}
}
