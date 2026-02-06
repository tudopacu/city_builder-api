package dto

type BuildingCost struct {
	ItemID   uint   `json:"item_id"`
	ItemName string `json:"item_name"`
	Quantity int    `json:"quantity"`
}

type BuildingProductionItem struct {
	ItemID                uint   `json:"item_id"`
	ItemName              string `json:"item_name"`
	Quantity              int    `json:"quantity"`
	ProductionTimeSeconds int    `json:"production_time_seconds"`
}

type BuildingWithDetails struct {
	ID               uint                     `json:"id"`
	Name             string                   `json:"name"`
	ImageURL         *string                  `json:"image_url,omitempty"`
	Description      string                   `json:"description"`
	Width            int                      `json:"width"`
	Length           int                      `json:"length"`
	BuildingCategory string                   `json:"building_category"`
	Costs            []BuildingCost           `json:"costs"`
	Productions      []BuildingProductionItem `json:"productions"`
}
