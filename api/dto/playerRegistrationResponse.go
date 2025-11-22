package dto

type PlayerRegistrationResponse struct {
	Player *Player `json:"player,omitempty"`
	Error  string  `json:"error,omitempty"`
}
