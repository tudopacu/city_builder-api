package services

import (
	"API/api/dto"
	"API/database"
	"API/models"
	"fmt"
	"log"
)

func GetRoadsByPlayerAndMap(playerID uint, mapID uint) ([]dto.Road, error) {
	var roads []models.Road

	if err := database.DB.
		Preload("StartIntersection").
		Preload("EndIntersection").
		Preload("RoadType").
		Joins("JOIN intersections ON intersections.id = roads.start_intersection_id").
		Where("intersections.player_id = ? AND intersections.map_id = ?", playerID, mapID).
		Find(&roads).Error; err != nil {
		log.Default().Printf("failed to fetch roads for player_id %d on map_id %d: %s", playerID, mapID, err)
		return nil, fmt.Errorf("failed to fetch roads for player_id %d on map_id %d", playerID, mapID)
	}

	roadDTOs := make([]dto.Road, 0, len(roads))
	for _, road := range roads {
		roadDTOs = append(roadDTOs, road.ToDTO())
	}

	return roadDTOs, nil
}
