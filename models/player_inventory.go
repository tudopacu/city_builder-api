package models

import (
	"API/api/dto"
	"time"
)

type PlayerInventory struct {
	ID               uint      `gorm:"primaryKey"`
	PlayerID         uint      `gorm:"not null"`
	PlayerBuildingID uint      `gorm:"not null"`
	Capacity         int       `gorm:"not null"`
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        *time.Time

	Player         Player                `gorm:"foreignKey:PlayerID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	PlayerBuilding PlayerBuilding        `gorm:"foreignKey:PlayerBuildingID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	InventoryItems []PlayerInventoryItem `gorm:"foreignKey:PlayerInventoryID"`
}

func (PlayerInventory) TableName() string {
	return "player_inventories"
}

func (pi PlayerInventory) ToDTO() dto.PlayerInventory {
	items := make([]dto.PlayerInventoryItem, 0, len(pi.InventoryItems))
	for _, item := range pi.InventoryItems {
		items = append(items, item.ToDTO())
	}
	return dto.PlayerInventory{
		ID:               pi.ID,
		PlayerBuildingID: pi.PlayerBuildingID,
		Capacity:         pi.Capacity,
		Items:            items,
	}
}
