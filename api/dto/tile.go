package dto

type Tile struct {
	ID       uint    `json:"id"`
	Type     string  `json:"type"`
	Walkable bool    `json:"walkable"`
	ImageURL *string `json:"image_url,omitempty"`
}
