package requests

type RoadData struct {
	StartX     int  `json:"start_x" binding:"required"`
	StartY     int  `json:"start_y" binding:"required"`
	EndX       int  `json:"end_x" binding:"required"`
	EndY       int  `json:"end_y" binding:"required"`
	RoadTypeID uint `json:"road_type_id" binding:"required"`
}

type AddRoadsRequest struct {
	PlayerID uint       `json:"player_id" binding:"required"`
	MapID    uint       `json:"map_id" binding:"required"`
	Roads    []RoadData `json:"roads" binding:"required,min=1"`
}
