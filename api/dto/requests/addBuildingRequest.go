package requests

type AddBuildingRequest struct {
	BuildingID uint `json:"building_id" binding:"required"`
	MapID      uint `json:"map_id" binding:"required"`
	X          int  `json:"x" binding:"required"`
	Y          int  `json:"y" binding:"required"`
}
