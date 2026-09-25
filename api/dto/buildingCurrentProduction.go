package dto

import "time"

type BuildingCurrentProduction struct {
	ID                   int64     `json:"id"`
	EndTime              time.Time `json:"end_time"`
	Status               string    `json:"status"`
	BuildingProductionID uint      `json:"building_production_id"`
	ItemName             string    `json:"item_name"`
	Quantity             int       `json:"quantity"`
}
