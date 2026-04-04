package dto

type PlayerInventoryItem struct {
	ID       uint `json:"id"`
	ItemID   uint `json:"item_id"`
	Quantity int  `json:"quantity"`
}
