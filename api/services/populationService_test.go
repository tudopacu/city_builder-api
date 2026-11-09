package services

import (
	"API/api/dto"
	"API/api/dto/requests"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"gorm.io/gorm"
)

// seedPopulationFixture adds a housing building (building 40, level 41) next to
// the Sawmill (building 30, level 31) from the shared building fixture.
func seedPopulationFixture(t *testing.T, sawmillPopulationCost, housingCapacity int) *gorm.DB {
	t.Helper()
	db := newBuildingTestDB(t)
	execBuildingTestSQL(t, db,
		`INSERT INTO building_categories (id, name) VALUES (2, 'Housing')`,
		`INSERT INTO buildings (id, name, width, length, building_category_id) VALUES (40, 'House', 1, 1, 2)`,
		`INSERT INTO players (id) VALUES (1)`,
		fmt.Sprintf(`UPDATE building_levels SET population_cost = %d WHERE id = 31`, sawmillPopulationCost),
		fmt.Sprintf(
			`INSERT INTO building_levels (id, building_id, level, population_capacity) VALUES (41, 40, 1, %d)`,
			housingCapacity,
		),
	)
	return db
}

func insertPopulationBuilding(t *testing.T, db *gorm.DB, id, buildingID, levelID, mapID, x, y int) {
	t.Helper()
	execBuildingTestSQL(t, db, fmt.Sprintf(
		`INSERT INTO player_buildings (id, player_id, building_id, map_id, building_level_id, x, y)
			VALUES (%d, 1, %d, %d, %d, %d, %d)`,
		id, buildingID, mapID, levelID, x, y,
	))
}

func requirePopulation(t *testing.T, playerID, mapID uint, want dto.Population) {
	t.Helper()
	got, err := GetPlayerPopulation(playerID, mapID)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("population = %+v, want %+v", got, want)
	}
}

func countPlayerBuildings(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := db.Table("player_buildings").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func TestPopulationIsZeroWithoutBuildings(t *testing.T) {
	seedPopulationFixture(t, 2, 5)
	requirePopulation(t, 1, 1, dto.Population{})
}

func TestAddPlayerBuildingRequiresFreePopulation(t *testing.T) {
	db := seedPopulationFixture(t, 2, 5)

	status, response := AddPlayerBuilding(buildingTestRequest())

	if status != http.StatusBadRequest || !strings.Contains(response.Error, "not enough population") {
		t.Fatalf("status = %d, response = %+v, want 400 with a population error", status, response)
	}
	if got := countPlayerBuildings(t, db); got != 0 {
		t.Fatalf("player_buildings = %d, want 0 after a rejected build", got)
	}
}

func TestHousingProvidesPopulationThatBuildingsUseAndRelease(t *testing.T) {
	db := seedPopulationFixture(t, 2, 5)
	insertPopulationBuilding(t, db, 100, 40, 41, 1, 8, 8)
	requirePopulation(t, 1, 1, dto.Population{Total: 5, Used: 0, Available: 5})

	status, response := AddPlayerBuilding(buildingTestRequest())
	if status != http.StatusCreated || response.PlayerBuilding == nil {
		t.Fatalf("status = %d, response = %+v, want 201", status, response)
	}
	requirePopulation(t, 1, 1, dto.Population{Total: 5, Used: 2, Available: 3})

	deleteStatus, err := DeletePlayerBuilding(response.PlayerBuilding.ID)
	if deleteStatus != http.StatusOK || err != nil {
		t.Fatalf("status = %d, err = %v, want 200", deleteStatus, err)
	}
	requirePopulation(t, 1, 1, dto.Population{Total: 5, Used: 0, Available: 5})
}

func TestAddPlayerBuildingStopsWhenPopulationRunsOut(t *testing.T) {
	// Housing for 3 and a sawmill that uses 2: the first sawmill fits, a second does not.
	db := seedPopulationFixture(t, 2, 3)
	insertPopulationBuilding(t, db, 100, 40, 41, 1, 8, 8)
	insertPopulationBuilding(t, db, 101, 30, 31, 1, 9, 9)
	requirePopulation(t, 1, 1, dto.Population{Total: 3, Used: 2, Available: 1})

	status, response := AddPlayerBuilding(buildingTestRequest())

	if status != http.StatusBadRequest || !strings.Contains(response.Error, "required 2, available 1") {
		t.Fatalf("status = %d, response = %+v, want 400 with required 2, available 1", status, response)
	}
	if got := countPlayerBuildings(t, db); got != 2 {
		t.Fatalf("player_buildings = %d, want 2", got)
	}
}

func TestHousingBuildsWithoutPopulation(t *testing.T) {
	db := seedPopulationFixture(t, 2, 5)
	execBuildingTestSQL(t, db, `INSERT INTO terrains (id, map_id, tile_id, x, y) VALUES (2, 1, 1, 6, 6)`)

	status, response := AddPlayerBuilding(requests.AddBuildingRequest{PlayerID: 1, BuildingID: 40, MapID: 1, X: 6, Y: 6})

	if status != http.StatusCreated || response.PlayerBuilding == nil {
		t.Fatalf("status = %d, response = %+v, want 201", status, response)
	}
	requirePopulation(t, 1, 1, dto.Population{Total: 5, Used: 0, Available: 5})
}

func TestRemovingHousingIsBlockedWhilePopulationIsInUse(t *testing.T) {
	db := seedPopulationFixture(t, 2, 5)
	insertPopulationBuilding(t, db, 100, 40, 41, 1, 8, 8)
	insertPopulationBuilding(t, db, 101, 30, 31, 1, 9, 9)
	insertPopulationBuilding(t, db, 102, 30, 31, 1, 10, 10)
	requirePopulation(t, 1, 1, dto.Population{Total: 5, Used: 4, Available: 1})

	status, err := DeletePlayerBuilding(100)

	if status != http.StatusConflict || err == nil || !strings.Contains(err.Error(), "cannot remove housing") {
		t.Fatalf("status = %d, err = %v, want 409 with a housing error", status, err)
	}
	if got := countPlayerBuildings(t, db); got != 3 {
		t.Fatalf("player_buildings = %d, want 3 after a blocked removal", got)
	}

	// Once the sawmills are gone the housing can be removed too.
	for _, id := range []uint{101, 102, 100} {
		if deleteStatus, deleteErr := DeletePlayerBuilding(id); deleteStatus != http.StatusOK || deleteErr != nil {
			t.Fatalf("delete %d: status = %d, err = %v, want 200", id, deleteStatus, deleteErr)
		}
	}
	requirePopulation(t, 1, 1, dto.Population{})
}

func TestRemovingHousingIsAllowedWhenOtherHousingCoversThePopulation(t *testing.T) {
	db := seedPopulationFixture(t, 2, 5)
	insertPopulationBuilding(t, db, 100, 40, 41, 1, 8, 8)
	insertPopulationBuilding(t, db, 101, 40, 41, 1, 7, 7)
	insertPopulationBuilding(t, db, 102, 30, 31, 1, 9, 9)
	requirePopulation(t, 1, 1, dto.Population{Total: 10, Used: 2, Available: 8})

	status, err := DeletePlayerBuilding(100)

	if status != http.StatusOK || err != nil {
		t.Fatalf("status = %d, err = %v, want 200", status, err)
	}
	requirePopulation(t, 1, 1, dto.Population{Total: 5, Used: 2, Available: 3})
}

func TestPopulationIsTrackedPerMap(t *testing.T) {
	db := seedPopulationFixture(t, 2, 5)
	insertPopulationBuilding(t, db, 100, 40, 41, 1, 8, 8)
	insertPopulationBuilding(t, db, 101, 30, 31, 2, 8, 8)

	requirePopulation(t, 1, 1, dto.Population{Total: 5, Used: 0, Available: 5})
	requirePopulation(t, 1, 2, dto.Population{Total: 0, Used: 2, Available: -2})
}
