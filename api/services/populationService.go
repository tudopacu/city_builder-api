package services

import (
	"API/api/dto"
	"API/database"
	"API/models"
	"fmt"
	"log"
	"net/http"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// populationConflictError is returned when an action would leave a player with
// more population in use than their housing provides.
type populationConflictError struct {
	message string
}

func (e populationConflictError) Error() string {
	return e.message
}

// loadPopulation derives a player's population on a map from the buildings that
// currently exist: housing buildings add to the total and every building uses
// its population cost. Nothing is stored, so building or removing a building
// changes the population immediately and the numbers cannot drift.
func loadPopulation(tx *gorm.DB, playerID, mapID uint) (dto.Population, error) {
	var totals struct {
		Total int
		Used  int
	}

	err := tx.Table("player_buildings").
		Select("COALESCE(SUM(building_levels.population_capacity), 0) AS total, "+
			"COALESCE(SUM(building_levels.population_cost), 0) AS used").
		Joins("JOIN building_levels ON building_levels.id = player_buildings.building_level_id").
		Where("player_buildings.player_id = ? AND player_buildings.map_id = ?", playerID, mapID).
		Scan(&totals).Error
	if err != nil {
		log.Default().Printf("failed to load population for player_id %d on map_id %d: %s", playerID, mapID, err)
		return dto.Population{}, fmt.Errorf("failed to load population")
	}

	return dto.Population{
		Total:     totals.Total,
		Used:      totals.Used,
		Available: totals.Total - totals.Used,
	}, nil
}

// lockPlayerForPopulation serializes population changes for one player, so two
// requests cannot both spend the same free population.
func lockPlayerForPopulation(tx *gorm.DB, playerID uint) error {
	var ids []uint
	if err := tx.Model(&models.Player{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", playerID).
		Pluck("id", &ids).Error; err != nil {
		log.Default().Printf("failed to lock player_id %d for population change: %s", playerID, err)
		return fmt.Errorf("failed to check population")
	}
	return nil
}

// ensurePopulationAvailable fails with a 400 when the player does not have
// enough free population for a building that needs `required`.
func ensurePopulationAvailable(tx *gorm.DB, playerID, mapID uint, required int) (int, error) {
	if required <= 0 {
		return http.StatusOK, nil
	}

	if err := lockPlayerForPopulation(tx, playerID); err != nil {
		return http.StatusInternalServerError, err
	}

	population, err := loadPopulation(tx, playerID, mapID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if population.Available < required {
		return http.StatusBadRequest, fmt.Errorf(
			"not enough population: required %d, available %d", required, population.Available,
		)
	}

	return http.StatusOK, nil
}

// ensureRemovalKeepsPopulationValid blocks removing a building when the
// population it provides is still in use. A building that provides `capacity`
// and uses `cost` leaves the player with total-capacity housing and used-cost
// population in use, and that must still fit.
func ensureRemovalKeepsPopulationValid(tx *gorm.DB, playerID, mapID uint, capacity, cost int) error {
	if capacity <= 0 {
		// Removing a building that provides no housing only frees population.
		return nil
	}

	if err := lockPlayerForPopulation(tx, playerID); err != nil {
		return err
	}

	population, err := loadPopulation(tx, playerID, mapID)
	if err != nil {
		return err
	}

	remainingTotal := population.Total - capacity
	remainingUsed := population.Used - cost
	if remainingTotal < remainingUsed {
		return populationConflictError{message: fmt.Sprintf(
			"cannot remove housing: %d population is in use but only %d would remain",
			remainingUsed, remainingTotal,
		)}
	}

	return nil
}

// GetPlayerPopulation returns the player's total, used and available population on a map.
func GetPlayerPopulation(playerID, mapID uint) (dto.Population, error) {
	return loadPopulation(database.DB, playerID, mapID)
}
