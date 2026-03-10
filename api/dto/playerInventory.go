package dto

type PlayerInventory struct {
	ID             uint                  `json:"id"`
	PlayerBuilding PlayerBuilding        `json:"player_building"`
	Capacity       int                   `json:"capacity"`
	Items          []PlayerInventoryItem `json:"items"`
}
