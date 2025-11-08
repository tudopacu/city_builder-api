package dto

type PlayerByCookieResponse struct {
	Player *Player `json:"player,omitempty"`
	Error  string  `json:"error,omitempty"`
}
