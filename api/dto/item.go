package dto

type Item struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Type        string  `json:"type"`
	IconURL     *string `json:"icon_url,omitempty"`
	IsTradeable bool    `json:"is_tradeable"`
}
