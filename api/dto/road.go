package dto

type Intersection struct {
	ID uint `json:"id"`
	X  int  `json:"x"`
	Y  int  `json:"y"`
}

type RoadType struct {
	ID       uint    `json:"id"`
	Type     string  `json:"type"`
	ImageURL *string `json:"image_url,omitempty"`
}

type Road struct {
	ID                uint         `json:"id"`
	StartIntersection Intersection `json:"start_intersection"`
	EndIntersection   Intersection `json:"end_intersection"`
	RoadType          RoadType     `json:"road_type"`
}
