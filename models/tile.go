package models

import (
	"API/api/dto"
	"time"
)

type Tile struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Type      string    `gorm:"type:varchar(255);not null;index:idx_tiles_type" json:"type"`
	ImageURL  *string   `gorm:"type:varchar(255)" json:"image_url,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Terrains []Terrain `gorm:"foreignKey:TileID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"terrains,omitempty"`
}

func (t *Tile) ToDTO() dto.Tile {
	return dto.Tile{
		ID:       t.ID,
		Type:     t.Type,
		ImageURL: t.ImageURL,
	}
}
