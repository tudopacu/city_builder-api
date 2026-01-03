package dto

type Building struct {
	ID               uint    `json:"id"`
	Name             string  `json:"name"`
	ImageURL         *string `json:"image_url,omitempty"`
	Description      string  `json:"description"`
	Width            int     `json:"width"`
	Length           int     `json:"length"`
	BuildingCategory string  `json:"building_category"`
}
