package dto

type PlayerInventoryItem struct {
	ItemID   uint   `json:"item_id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
}

type PlayerInventory struct {
	PlayerID      uint                  `json:"player_id"`
	Items         []PlayerInventoryItem `json:"items"`
	TotalCapacity int                   `json:"total_capacity"`
	TotalItems    int                   `json:"total_items"`
}
