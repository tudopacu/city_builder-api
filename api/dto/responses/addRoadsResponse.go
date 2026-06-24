package responses

import "API/api/dto"

type AddRoadsResponse struct {
	Roads []dto.Road `json:"roads,omitempty"`
	Error string     `json:"error,omitempty"`
}
