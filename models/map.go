package models

import (
	"API/api/dto"
	"time"
)

type Map struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Width     int       `gorm:"not null" json:"width"`
	Length    int       `gorm:"not null" json:"length"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Terrains []Terrain `gorm:"foreignKey:MapID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"terrains"`
}

func (m *Map) ToDTO() dto.Map {
	terrainsDTO := make([]dto.Terrain, len(m.Terrains))
	for i, terrain := range m.Terrains {
		terrainsDTO[i] = terrain.ToDTO()
	}

	return dto.Map{
		ID:       m.ID,
		Name:     m.Name,
		Width:    m.Width,
		Length:   m.Length,
		Terrains: terrainsDTO,
	}
}
