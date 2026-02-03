package models

import "time"

type ItemRecipe struct {
	ID                    uint      `gorm:"primaryKey"`
	ItemID                uint      `gorm:"not null;index:idx-item_recipes-item_id"`
	ProductionTimeSeconds int       `gorm:"not null"`
	CreatedAt             time.Time `gorm:"autoCreateTime"`
	UpdatedAt             *time.Time

	Item   Item              `gorm:"foreignKey:ItemID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Inputs []ItemRecipeInput `gorm:"foreignKey:RecipeID"`
}

func (ItemRecipe) TableName() string {
	return "item_recipes"
}
