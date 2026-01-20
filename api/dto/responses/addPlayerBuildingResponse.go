package responses

import "API/api/dto"

type AddPlayerBuildingResponse struct {
	PlayerBuilding *dto.PlayerBuilding `json:"player_building,omitempty"`
	Error          string              `json:"error,omitempty"`
}
