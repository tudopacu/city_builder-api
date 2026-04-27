package requests

type IntersectionInput struct {
	X int `json:"x" binding:"required"`
	Y int `json:"y" binding:"required"`
}

type RoadTypeInput struct {
	ID uint `json:"id" binding:"required"`
}

type RoadData struct {
	StartIntersection IntersectionInput `json:"start_intersection" binding:"required"`
	EndIntersection   IntersectionInput `json:"end_intersection" binding:"required"`
	RoadType          RoadTypeInput     `json:"road_type" binding:"required"`
}

type AddRoadsRequest struct {
	Roads []RoadData `json:"roads" binding:"required,min=1"`
}
