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
//
// With lock set, the player's buildings are read with a locking read. That read
// sees the latest committed buildings instead of the transaction's older
// snapshot, which is what makes the check-then-build sequence safe when two
// requests run at once. Callers that lock must hold the player lock first.
func loadPopulation(tx *gorm.DB, playerID, mapID uint, lock bool) (dto.Population, error) {
	query := tx.Table("player_buildings").
		Where("player_id = ? AND map_id = ?", playerID, mapID)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var levelIDs []uint
	if err := query.Pluck("building_level_id", &levelIDs).Error; err != nil {
		log.Default().Printf("failed to load buildings for population of player_id %d on map_id %d: %s", playerID, mapID, err)
		return dto.Population{}, fmt.Errorf("failed to load population")
	}
	if len(levelIDs) == 0 {
		return dto.Population{}, nil
	}

	// Count how many buildings sit on each level, then look the levels up once.
	buildingsPerLevel := make(map[uint]int)
	for _, levelID := range levelIDs {
		buildingsPerLevel[levelID]++
	}
	uniqueLevelIDs := make([]uint, 0, len(buildingsPerLevel))
	for levelID := range buildingsPerLevel {
		uniqueLevelIDs = append(uniqueLevelIDs, levelID)
	}

	var levels []models.BuildingLevel
	if err := tx.Where("id IN ?", uniqueLevelIDs).Find(&levels).Error; err != nil {
		log.Default().Printf("failed to load building levels for population of player_id %d on map_id %d: %s", playerID, mapID, err)
		return dto.Population{}, fmt.Errorf("failed to load population")
	}

	total, used := 0, 0
	for _, level := range levels {
		count := buildingsPerLevel[level.ID]
		total += count * level.PopulationCapacity
		used += count * level.PopulationCost
	}

	return dto.Population{Total: total, Used: used, Available: total - used}, nil
}

// lockPlayerForPopulation serializes building changes for one player, so two
// requests cannot both spend the same free population. Every transaction that
// adds or removes a building takes this lock first, which also keeps the lock
// order the same everywhere.
func lockPlayerForPopulation(tx *gorm.DB, playerID uint) error {
	var ids []uint
	if err := tx.Model(&models.Player{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", playerID).
		Pluck("id", &ids).Error; err != nil {
		log.Default().Printf("failed to lock player_id %d for building change: %s", playerID, err)
		return fmt.Errorf("failed to check population")
	}
	return nil
}

// ensurePopulationAvailable fails with a 400 when the player does not have
// enough free population for a building that needs `required`. The caller must
// already hold the player lock.
func ensurePopulationAvailable(tx *gorm.DB, playerID, mapID uint, required int) (int, error) {
	if required <= 0 {
		return http.StatusOK, nil
	}

	population, err := loadPopulation(tx, playerID, mapID, true)
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
// population in use, and that must still fit. The caller must already hold the
// player lock.
func ensureRemovalKeepsPopulationValid(tx *gorm.DB, playerID, mapID uint, capacity, cost int) error {
	if capacity <= 0 {
		// Removing a building that provides no housing only frees population.
		return nil
	}

	population, err := loadPopulation(tx, playerID, mapID, true)
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
	return loadPopulation(database.DB, playerID, mapID, false)
}
