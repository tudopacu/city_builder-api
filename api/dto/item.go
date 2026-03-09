package dto

type Item struct {
	ID      uint    `json:"id"`
	Name    string  `json:"name"`
	Type    string  `json:"type"`
	IconURL *string `json:"icon_url,omitempty"`
}
