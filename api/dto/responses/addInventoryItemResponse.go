package responses

import "API/api/dto"

type AddInventoryItemResponse struct {
	PlayerInventory *dto.PlayerInventory `json:"player_inventory,omitempty"`
	Error           string               `json:"error,omitempty"`
}
