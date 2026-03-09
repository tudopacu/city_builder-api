package models

import (
	"API/api/dto"
	"time"
)

type PlayerBuilding struct {
	ID              uint      `gorm:"primaryKey"`
	PlayerID        uint      `gorm:"not null;index:idx_pb_player_coordinates,unique"`
	BuildingID      uint      `gorm:"not null"`
	MapID           uint      `gorm:"not null;index:idx_pb_player_coordinates,unique"`
	BuildingLevelID uint      `gorm:"not null"`
	X               int       `gorm:"not null;index:idx_pb_player_coordinates,unique"`
	Y               int       `gorm:"not null;index:idx_pb_player_coordinates,unique"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       *time.Time

	Building      Building      `gorm:"foreignKey:BuildingID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	BuildingLevel BuildingLevel `gorm:"foreignKey:BuildingLevelID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Map           Map           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	PlayerInventories []PlayerInventory `gorm:"foreignKey:PlayerBuildingID"`
}

func (PlayerBuilding) TableName() string {
	return "player_buildings"
}

func (b PlayerBuilding) ToDTO() dto.PlayerBuilding {
	buildingDTO := b.Building.ToDTO()
	return dto.PlayerBuilding{
		ID:            b.ID,
		Building:      &buildingDTO,
		BuildingLevel: b.BuildingLevel.Level,
		X:             b.X,
		Y:             b.Y,
	}
}
