package dto

type PlayerBuilding struct {
	ID            uint      `json:"id"`
	Building      *Building `json:"building"`
	BuildingLevel uint      `json:"level"`
	X             int       `json:"x"`
	Y             int       `json:"y"`
}
