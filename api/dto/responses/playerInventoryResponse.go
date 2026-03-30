package responses

import "API/api/dto"

type PlayerInventoryResponse struct {
	PlayerInventories []dto.PlayerInventory `json:"player_inventories"`
	TotalQuantity     int                   `json:"total_quantity"`
	TotalCapacity     int                   `json:"total_capacity"`
}
