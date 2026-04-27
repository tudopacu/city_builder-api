package dto

type Road struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	ImageURL    *string `json:"image_url,omitempty"`
	Description *string `json:"description,omitempty"`
	Width       int     `json:"width"`
	Length      int     `json:"length"`
}
