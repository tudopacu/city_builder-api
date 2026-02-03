package models

import "time"

type ItemRecipeInput struct {
	ID          uint      `gorm:"primaryKey"`
	RecipeID    uint      `gorm:"not null;index:idx-item_recipe_inputs-recipe_id"`
	InputItemID uint      `gorm:"not null;index:idx-item_recipe_inputs-input_item_id"`
	Quantity    int       `gorm:"not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   *time.Time

	Recipe    ItemRecipe `gorm:"foreignKey:RecipeID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	InputItem Item       `gorm:"foreignKey:InputItemID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (ItemRecipeInput) TableName() string {
	return "item_recipe_inputs"
}
