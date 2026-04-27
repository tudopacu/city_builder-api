package services

import (
	"API/api/dto"
	"API/api/dto/requests"
	"API/api/dto/responses"
	"API/database"
	"API/models"
	"fmt"
	"log"
	"net/http"

	"gorm.io/gorm"
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

func AddRoads(playerID uint, mapID uint, request requests.AddRoadsRequest) (int, responses.AddRoadsResponse) {
	createdRoadDTOs := make([]dto.Road, 0, len(request.Roads))

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, roadData := range request.Roads {
			startIntersection := models.Intersection{
				PlayerID: &playerID,
				MapID:    &mapID,
				X:        roadData.StartIntersection.X,
				Y:        roadData.StartIntersection.Y,
			}
			if err := tx.Create(&startIntersection).Error; err != nil {
				log.Default().Printf("failed to create start intersection for player_id %d: %s", playerID, err)
				return fmt.Errorf("failed to create start intersection at (%d, %d)", roadData.StartIntersection.X, roadData.StartIntersection.Y)
			}

			endIntersection := models.Intersection{
				PlayerID: &playerID,
				MapID:    &mapID,
				X:        roadData.EndIntersection.X,
				Y:        roadData.EndIntersection.Y,
			}
			if err := tx.Create(&endIntersection).Error; err != nil {
				log.Default().Printf("failed to create end intersection for player_id %d: %s", playerID, err)
				return fmt.Errorf("failed to create end intersection at (%d, %d)", roadData.EndIntersection.X, roadData.EndIntersection.Y)
			}

			road := models.Road{
				StartIntersectionID: &startIntersection.ID,
				EndIntersectionID:   &endIntersection.ID,
				RoadTypeID:          &roadData.RoadType.ID,
			}
			if err := tx.Create(&road).Error; err != nil {
				log.Default().Printf("failed to create road for player_id %d: %s", playerID, err)
				return fmt.Errorf("failed to create road")
			}

			if err := tx.
				Preload("StartIntersection").
				Preload("EndIntersection").
				Preload("RoadType").
				First(&road, road.ID).Error; err != nil {
				log.Default().Printf("failed to load created road with ID %d: %s", road.ID, err)
				return fmt.Errorf("failed to load road details")
			}

			createdRoadDTOs = append(createdRoadDTOs, road.ToDTO())
		}
		return nil
	})

	if err != nil {
		return http.StatusInternalServerError, responses.AddRoadsResponse{Error: err.Error()}
	}

	return http.StatusCreated, responses.AddRoadsResponse{Roads: createdRoadDTOs}
}
