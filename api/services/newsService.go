package services

import (
	"API/api/dto"
	"API/database"
	"API/models"
	"fmt"
	"log"
)

func GetNews() ([]dto.News, error) {
	var newsModels []models.News

	if err := database.DB.Find(&newsModels).Error; err != nil {
		log.Default().Println("failed to fetch news", err)
		return nil, fmt.Errorf("failed to fetch news")
	}

	var dtoNews []dto.News
	for _, news := range newsModels {
		dtoNews = append(dtoNews, news.ToDTO())
	}

	return dtoNews, nil
}
