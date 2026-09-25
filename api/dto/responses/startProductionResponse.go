package responses

import (
	"API/api/dto"
	"time"
)

type StartProductionResponse struct {
	BuildingCurrentProduction *dto.BuildingCurrentProduction `json:"building_current_production"`
	CurrentTimestamp          time.Time                      `json:"current_timestamp"`
}
