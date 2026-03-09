package models

import "time"

type PlayerInventoryItem struct {
	ID                uint      `gorm:"primaryKey"`
	PlayerInventoryID uint      `gorm:"not null;index:idx_pii_player_inventory_item,unique"`
	ItemID            uint      `gorm:"not null;index:idx_pii_player_inventory_item,unique"`
	Quantity          int       `gorm:"not null;default:0"`
	CreatedAt         time.Time `gorm:"autoCreateTime"`
	UpdatedAt         *time.Time

	PlayerInventory PlayerInventory `gorm:"foreignKey:PlayerInventoryID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Item            Item            `gorm:"foreignKey:ItemID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (PlayerInventoryItem) TableName() string {
	return "player_inventory_items"
}
