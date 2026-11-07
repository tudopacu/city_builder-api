package dto

// Population describes a player's population on one map.
// Total comes from housing buildings, Used from every other building, and
// Available is what is left for new buildings.
type Population struct {
	Total     int `json:"total"`
	Used      int `json:"used"`
	Available int `json:"available"`
}
