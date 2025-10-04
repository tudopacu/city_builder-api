package models

import (
	"time"
)

type Terrain struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	MapID  uint `gorm:"not null;index:idx_terrains_map_id" json:"map_id"`
	TileID uint `gorm:"not null;index:idx_terrains_tile_id" json:"tile_id"`

	X int `gorm:"not null" json:"x"`
	Y int `gorm:"not null" json:"y"`

	// Unique index on (map_id, x, y)
	// GORM syntax: give the same index name to group fields together
	// and mark as unique
	_ struct{} `gorm:"uniqueIndex:idx_terrains_coordinates"` // phantom field for unique composite index
	// Alternatively, you can apply tags directly:
	// MapID uint `gorm:"not null;index:idx_terrains_map_id;uniqueIndex:idx_terrains_coordinates"`
	// X     int  `gorm:"not null;uniqueIndex:idx_terrains_coordinates"`
	// Y     int  `gorm:"not null;uniqueIndex:idx_terrains_coordinates"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Associations
	//Map  Map  `gorm:"foreignKey:MapID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"map"`
	Tile Tile `gorm:"foreignKey:TileID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"tile"`
}
