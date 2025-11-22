package dto

type Map struct {
	ID       uint      `json:"id"`
	Name     string    `json:"name"`
	Width    int       `json:"width"`
	Length   int       `json:"length"`
	Terrains []Terrain `json:"terrains"`
}
