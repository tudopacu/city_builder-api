package models

import (
	"API/api/dto"
	"time"
)

type PlayerItem struct {
	ID        uint      `gorm:"primaryKey"`
	PlayerID  uint      `gorm:"not null;index:idx_player_item,unique"`
	ItemID    uint      `gorm:"not null;index:idx_player_item,unique"`
	Quantity  int       `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt *time.Time

	Player Player `gorm:"foreignKey:PlayerID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Item   Item   `gorm:"foreignKey:ItemID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (PlayerItem) TableName() string {
	return "player_items"
}

func (pi PlayerItem) ToDTO() dto.PlayerInventoryItem {
	return dto.PlayerInventoryItem{
		ItemID:   pi.ItemID,
		Name:     pi.Item.Name,
		Type:     pi.Item.Type,
		Quantity: pi.Quantity,
	}
}
