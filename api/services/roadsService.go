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

func AddRoads(request requests.AddRoadsRequest) (int, responses.AddRoadsResponse) {
	createdRoadDTOs := make([]dto.Road, 0, len(request.Roads))

	for _, roadData := range request.Roads {
		startIntersection := models.Intersection{
			PlayerID: &request.PlayerID,
			MapID:    &request.MapID,
			X:        roadData.StartX,
			Y:        roadData.StartY,
		}
		if err := database.DB.Create(&startIntersection).Error; err != nil {
			log.Default().Printf("failed to create start intersection for player_id %d: %s", request.PlayerID, err)
			return http.StatusInternalServerError, responses.AddRoadsResponse{Error: fmt.Sprintf("failed to create start intersection at (%d, %d)", roadData.StartX, roadData.StartY)}
		}

		endIntersection := models.Intersection{
			PlayerID: &request.PlayerID,
			MapID:    &request.MapID,
			X:        roadData.EndX,
			Y:        roadData.EndY,
		}
		if err := database.DB.Create(&endIntersection).Error; err != nil {
			log.Default().Printf("failed to create end intersection for player_id %d: %s", request.PlayerID, err)
			return http.StatusInternalServerError, responses.AddRoadsResponse{Error: fmt.Sprintf("failed to create end intersection at (%d, %d)", roadData.EndX, roadData.EndY)}
		}

		road := models.Road{
			StartIntersectionID: &startIntersection.ID,
			EndIntersectionID:   &endIntersection.ID,
			RoadTypeID:          &roadData.RoadTypeID,
		}
		if err := database.DB.Create(&road).Error; err != nil {
			log.Default().Printf("failed to create road for player_id %d: %s", request.PlayerID, err)
			return http.StatusInternalServerError, responses.AddRoadsResponse{Error: "failed to create road"}
		}

		if err := database.DB.
			Preload("StartIntersection").
			Preload("EndIntersection").
			Preload("RoadType").
			First(&road, road.ID).Error; err != nil {
			log.Default().Printf("failed to load created road with ID %d: %s", road.ID, err)
			return http.StatusInternalServerError, responses.AddRoadsResponse{Error: "failed to load road details"}
		}

		createdRoadDTOs = append(createdRoadDTOs, road.ToDTO())
	}

	return http.StatusCreated, responses.AddRoadsResponse{Roads: createdRoadDTOs}
}
