package responses

import "API/api/dto"

type PlayerRegistrationResponse struct {
	Player *dto.Player `json:"player,omitempty"`
	Error  string      `json:"error,omitempty"`
}
