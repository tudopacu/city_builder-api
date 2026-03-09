package dto

type PlayerInventoryItem struct {
	ID       uint `json:"id"`
	Item     Item `json:"item"`
	Quantity int  `json:"quantity"`
}
