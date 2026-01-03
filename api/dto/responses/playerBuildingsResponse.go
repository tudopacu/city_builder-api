package responses

import "API/api/dto"

type PlayerBuildingsResponse struct {
	Buildings dto.PlayerBuilding `json:"buildings"`
}
