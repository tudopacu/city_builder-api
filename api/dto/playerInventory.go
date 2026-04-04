package dto

type PlayerInventory struct {
	ID               uint                  `json:"id"`
	PlayerBuildingID uint                  `json:"player_building_id"`
	Items            []PlayerInventoryItem `json:"items"`
	Capacity         int                   `json:"capacity"`
}
