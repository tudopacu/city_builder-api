package models

import (
	"API/api/dto"
	"time"
)

type Item struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"not null"`
	Description *string   `gorm:"type:text"`
	Type        string    `gorm:"size:50;not null;index:idx_items_type"`
	IconURL     *string   `gorm:"type:varchar(255)"`
	IsTradeable bool      `gorm:"not null;default:true"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   *time.Time

	Recipes                   []ItemRecipe               `gorm:"foreignKey:ItemID"`
	RecipeInputs              []ItemRecipeInput          `gorm:"foreignKey:InputItemID"`
	BuildingConstructionCosts []BuildingConstructionCost `gorm:"foreignKey:ItemID"`
	BuildingProductions       []BuildingProduction       `gorm:"foreignKey:ItemID"`
	PlayerInventoryItems      []PlayerInventoryItem      `gorm:"foreignKey:ItemID"`
}

func (Item) TableName() string {
	return "items"
}

func (i Item) ToDTO() dto.Item {
	return dto.Item{
		ID:          i.ID,
		Name:        i.Name,
		Description: i.Description,
		Type:        i.Type,
		IconURL:     i.IconURL,
		IsTradeable: i.IsTradeable,
	}
}
