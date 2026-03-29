package dto

type PlayerInventory struct {
	ID             uint                  `json:"id"`
	PlayerBuilding PlayerBuilding        `json:"player_building"`
	Items          []PlayerInventoryItem `json:"items"`
	Capacity       int                   `json:"capacity"`
}
