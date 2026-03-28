package requests

type AddInventoryItemRequest struct {
	PlayerID    uint `json:"player_id" binding:"required"`
	ItemID      uint `json:"item_id" binding:"required"`
	InventoryID uint `json:"inventory_id" binding:"required"`
	Quantity    int  `json:"quantity" binding:"required,min=1"`
}
