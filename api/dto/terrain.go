package dto

type Terrain struct {
	ID       uint    `json:"id"`
	X        int     `json:"x"`
	Y        int     `json:"y"`
	Type     string  `json:"type"`
	Walkable bool    `json:"walkable"`
	ImageURL *string `json:"image_url,omitempty"`
}
