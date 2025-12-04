package dto

type Terrain struct {
	TileId   uint   `json:"tile_id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Type     string `json:"type"`
	Walkable bool   `json:"walkable"`
	SetX     int    `json:"set_x"`
	SetY     int    `json:"set_y"`
}
