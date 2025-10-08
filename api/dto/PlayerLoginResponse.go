package dto

type PlayerLoginResponse struct {
	Player *Player `json:"player,omitempty"`
	Error  string  `json:"error,omitempty"`
}
