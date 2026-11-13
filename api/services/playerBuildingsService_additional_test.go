package services

import (
	"API/api/dto"
	"API/models"
	"net/http"
	"reflect"
	"testing"
	"time"

	"gorm.io/gorm"
)

func newPlayerBuildingLifecycleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newBuildingTestDB(t)
	execBuildingTestSQL(t, db,
		`CREATE TABLE building_productions (id INTEGER PRIMARY KEY, building_id INTEGER, item_id INTEGER,
			production_time_seconds INTEGER, quantity INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE player_building_productions (id INTEGER PRIMARY KEY, player_id INTEGER, player_building_id INTEGER,
			building_production_id INTEGER, end_time DATETIME, status TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT, type TEXT)`,
	)
	return db
}

func TestGetPlayerBuildingsScopesBuildingsAndCompletesExpiredProductions(t *testing.T) {
	db := newPlayerBuildingLifecycleTestDB(t)
	execBuildingTestSQL(t, db,
		`INSERT INTO player_buildings (id, player_id, building_id, map_id, building_level_id, x, y) VALUES
			(10, 1, 30, 1, 31, 2, 3), (20, 1, 30, 1, 31, 4, 5), (30, 1, 30, 1, 31, 6, 7),
			(40, 2, 30, 1, 31, 2, 3), (50, 1, 30, 2, 31, 2, 3)`,
		`INSERT INTO items (id, name, type) VALUES (7, 'Wood', 'resource')`,
		`INSERT INTO building_productions (id, building_id, item_id, quantity) VALUES (60, 30, 7, 5)`,
	)
	past, future := time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
	productions := []models.PlayerBuildingProduction{
		{ID: 100, PlayerID: 1, PlayerBuildingID: 10, BuildingProductionID: 60, EndTime: past, Status: "PENDING"},
		{ID: 200, PlayerID: 1, PlayerBuildingID: 20, BuildingProductionID: 60, EndTime: future, Status: "PENDING"},
		{ID: 300, PlayerID: 1, PlayerBuildingID: 30, BuildingProductionID: 60, EndTime: past, Status: "COLLECTED"},
		{ID: 400, PlayerID: 2, PlayerBuildingID: 40, BuildingProductionID: 60, EndTime: past, Status: "PENDING"},
		{ID: 500, PlayerID: 1, PlayerBuildingID: 50, BuildingProductionID: 60, EndTime: past, Status: "PENDING"},
	}
	if err := db.Create(&productions).Error; err != nil {
		t.Fatal(err)
	}
	buildings, err := GetPlayerBuildings(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(buildings) != 3 {
		t.Fatalf("buildings = %+v, want three for player 1 on map 1", buildings)
	}
	byID := make(map[uint]dto.PlayerBuilding)
	for _, building := range buildings {
		byID[building.ID] = building
		if building.Building == nil || building.Building.ID != 30 || building.Building.Name != "Sawmill" ||
			building.Building.BuildingCategory != "Production" || building.BuildingLevel != 1 {
			t.Fatalf("building associations missing: %+v", building)
		}
	}
	if len(byID) != 3 || byID[10].X != 2 || byID[10].Y != 3 || byID[20].X != 4 || byID[30].X != 6 {
		t.Fatalf("unexpected building IDs or positions: %+v", byID)
	}
	finished := byID[10].BuildingCurrentProduction
	if finished == nil || finished.ID != 100 || finished.BuildingProductionID != 60 ||
		finished.ItemName != "Wood" || finished.Quantity != 5 || finished.Status != "DONE" || !finished.EndTime.Equal(past) {
		t.Fatalf("expired production = %+v", finished)
	}
	pending := byID[20].BuildingCurrentProduction
	if pending == nil || pending.Status != "PENDING" || !pending.EndTime.Equal(future) {
		t.Fatalf("unfinished production = %+v", pending)
	}
	if byID[30].BuildingCurrentProduction != nil {
		t.Fatal("collected production was returned")
	}
	var saved []models.PlayerBuildingProduction
	if err := db.Order("id").Find(&saved).Error; err != nil {
		t.Fatal(err)
	}
	wantStatuses := []string{"DONE", "PENDING", "COLLECTED", "PENDING", "PENDING"}
	if len(saved) != len(wantStatuses) {
		t.Fatalf("saved productions = %+v", saved)
	}
	for i, production := range saved {
		if production.Status != wantStatuses[i] {
			t.Errorf("production %d status = %s, want %s", production.ID, production.Status, wantStatuses[i])
		}
	}
}

func TestGetPlayerBuildingsReturnsEmptySlice(t *testing.T) {
	newPlayerBuildingLifecycleTestDB(t)
	buildings, err := GetPlayerBuildings(1, 1)
	if err != nil || buildings == nil || len(buildings) != 0 {
		t.Fatalf("buildings = %#v, err = %v", buildings, err)
	}
}

func TestGetPlayerBuildingsReportsDatabaseFailures(t *testing.T) {
	for _, tc := range []struct{ name, setup, message string }{
		{"production update", "DROP TABLE player_building_productions", "failed to update expired productions"},
		{"building association", "DROP TABLE building_levels", "failed to fetch player buildings for player_id 1 on map_id 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newPlayerBuildingLifecycleTestDB(t)
			execBuildingTestSQL(t, db,
				`INSERT INTO player_buildings (id, player_id, building_id, map_id, building_level_id, x, y) VALUES (10, 1, 30, 1, 31, 2, 3)`,
				tc.setup,
			)
			buildings, err := GetPlayerBuildings(1, 1)
			if err == nil || err.Error() != tc.message || len(buildings) != 0 {
				t.Fatalf("buildings = %+v, err = %v; want %q", buildings, err, tc.message)
			}
		})
	}
}

func TestAddPlayerBuildingAcceptsOriginAndCreatesStorageInventory(t *testing.T) {
	db := newBuildingTestDB(t)
	execBuildingTestSQL(t, db,
		`UPDATE building_categories SET name = 'Storage'`,
		`UPDATE building_levels SET capacity = 75`,
		`UPDATE terrains SET x = 0, y = 0`,
	)
	request := buildingTestRequest()
	request.X, request.Y = 0, 0
	status, response := AddPlayerBuilding(1, request)
	if status != http.StatusCreated || response.Error != "" || response.PlayerBuilding == nil {
		t.Fatalf("status = %d, response = %+v", status, response)
	}
	if response.PlayerBuilding.X != 0 || response.PlayerBuilding.Y != 0 || response.PlayerBuilding.Building.BuildingCategory != "Storage" {
		t.Fatalf("storage building = %+v", response.PlayerBuilding)
	}
	after := snapshotBuildingTest(t, db)
	if len(after.Buildings) != 1 || len(after.Inventories) != 1 || len(after.Items) != 0 {
		t.Fatalf("stored state = %+v", after)
	}
	inventory := after.Inventories[0]
	if inventory.PlayerBuildingID != response.PlayerBuilding.ID || inventory.PlayerID != 1 ||
		inventory.MapID != request.MapID || inventory.Capacity != 75 {
		t.Fatalf("storage inventory = %+v", inventory)
	}
}

func TestAddPlayerBuildingRejectsNegativeCoordinates(t *testing.T) {
	for _, coordinate := range []string{"x", "y"} {
		t.Run(coordinate, func(t *testing.T) {
			request := buildingTestRequest()
			if coordinate == "x" {
				request.X = -1
			} else {
				request.Y = -1
			}
			status, response := AddPlayerBuilding(1, request)
			if status != http.StatusBadRequest || response.Error != "coordinates must be non-negative" || response.PlayerBuilding != nil {
				t.Fatalf("status = %d, response = %+v", status, response)
			}
		})
	}
}

func TestAddPlayerBuildingValidatesRectangularFootprint(t *testing.T) {
	for _, tc := range []struct {
		name, terrainChange, wantError string
		existing                       bool
		existingX, existingY           int
		existingMap                    uint
	}{
		{name: "all six grass and dirt tiles"},
		{name: "missing far corner", terrainChange: "DELETE FROM terrains WHERE map_id = 1 AND x = 3 AND y = 5", wantError: "some terrain tiles are missing"},
		{name: "invalid middle tile", terrainChange: "UPDATE terrains SET tile_id = 3 WHERE map_id = 1 AND x = 3 AND y = 4", wantError: "building can only be placed on grass or dirt tiles, found water at position (3, 4)"},
		{name: "overlap at new far corner", existing: true, existingX: 3, existingY: 5, existingMap: 1, wantError: "building overlaps with an existing building"},
		{name: "overlap at existing far corner", existing: true, existingX: 1, existingY: 1, existingMap: 1, wantError: "building overlaps with an existing building"},
		{name: "adjacent right", existing: true, existingX: 4, existingY: 3, existingMap: 1},
		{name: "adjacent below", existing: true, existingX: 2, existingY: 6, existingMap: 1},
		{name: "adjacent left", existing: true, existingX: 0, existingY: 3, existingMap: 1},
		{name: "adjacent above", existing: true, existingX: 2, existingY: 0, existingMap: 1},
		{name: "same coordinates on another map", existing: true, existingX: 2, existingY: 3, existingMap: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newBuildingTestDB(t)
			execBuildingTestSQL(t, db,
				`UPDATE buildings SET width = 2, length = 3 WHERE id = 30`,
				`INSERT INTO tiles (id, type) VALUES (2, 'dirt'), (3, 'water')`,
				`INSERT INTO terrains (id, map_id, tile_id, x, y) VALUES
					(2, 1, 2, 2, 4), (3, 1, 1, 2, 5), (4, 1, 2, 3, 3), (5, 1, 1, 3, 4), (6, 1, 2, 3, 5),
					(7, 1, 3, 4, 6), (8, 2, 3, 2, 3)`,
				`INSERT INTO building_construction_costs (building_id, item_id, quantity) VALUES (30, 7, 3)`,
				`INSERT INTO player_inventories (id, player_id, capacity) VALUES (10, 1, 100)`,
				`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity) VALUES (1, 10, 7, 8)`,
			)
			if tc.terrainChange != "" {
				execBuildingTestSQL(t, db, tc.terrainChange)
			}
			if tc.existing {
				if err := db.Exec(`INSERT INTO player_buildings (id, player_id, building_id, map_id, building_level_id, x, y)
					VALUES (10, 2, 30, ?, 31, ?, ?)`, tc.existingMap, tc.existingX, tc.existingY).Error; err != nil {
					t.Fatal(err)
				}
			}
			before := snapshotBuildingTest(t, db)
			status, response := AddPlayerBuilding(1, buildingTestRequest())
			after := snapshotBuildingTest(t, db)
			if tc.wantError != "" {
				if status != http.StatusBadRequest || response.Error != tc.wantError || response.PlayerBuilding != nil {
					t.Fatalf("status = %d, response = %+v; want 400, %q", status, response, tc.wantError)
				}
				if !reflect.DeepEqual(after, before) {
					t.Fatal("rejected footprint changed buildings or inventory")
				}
				return
			}
			if status != http.StatusCreated || response.Error != "" || response.PlayerBuilding == nil || response.PlayerBuilding.Building == nil {
				t.Fatalf("status = %d, response = %+v", status, response)
			}
			building := response.PlayerBuilding
			if building.X != 2 || building.Y != 3 || building.Building.Width != 2 || building.Building.Length != 3 {
				t.Fatalf("placed footprint = %+v", building)
			}
			if len(after.Buildings) != len(before.Buildings)+1 || len(after.Items) != 1 || after.Items[0].Quantity != 5 {
				t.Fatalf("placement did not create one building and consume three items: %+v", after)
			}
			if !reflect.DeepEqual(after.Inventories, before.Inventories) ||
				(tc.existing && !reflect.DeepEqual(after.Buildings[0], before.Buildings[0])) {
				t.Fatal("placement changed an existing building or storage inventory")
			}
		})
	}
}

func TestAddPlayerBuildingReportsMissingRecordsAndReadFailures(t *testing.T) {
	for _, tc := range []struct {
		name, setup, message string
		status               int
	}{
		{"missing building", "DELETE FROM buildings", "building not found", http.StatusNotFound},
		{"missing level", "DELETE FROM building_levels", "building level not found", http.StatusNotFound},
		{"building query", "DROP TABLE buildings", "failed to place building", http.StatusInternalServerError},
		{"level query", "DROP TABLE building_levels", "failed to place building", http.StatusInternalServerError},
		{"terrain query", "DROP TABLE terrains", "failed to place building", http.StatusInternalServerError},
		{"existing buildings query", "DROP TABLE player_buildings", "failed to place building", http.StatusInternalServerError},
		{"construction costs query", "DROP TABLE building_construction_costs", "failed to place building", http.StatusInternalServerError},
		{"inventory query", "DROP TABLE player_inventories", "failed to place building", http.StatusInternalServerError},
		{"zero cost", "UPDATE building_construction_costs SET quantity = 0", "failed to place building", http.StatusInternalServerError},
		{"negative cost", "UPDATE building_construction_costs SET quantity = -1", "failed to place building", http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newBuildingTestDB(t)
			execBuildingTestSQL(t, db,
				`INSERT INTO building_construction_costs (building_id, item_id, quantity) VALUES (30, 7, 3)`,
				tc.setup,
			)
			status, response := AddPlayerBuilding(1, buildingTestRequest())
			if status != tc.status || response.Error != tc.message || response.PlayerBuilding != nil {
				t.Fatalf("status = %d, response = %+v; want %d, %q", status, response, tc.status, tc.message)
			}
		})
	}
}

func TestAddPlayerBuildingRollsBackStorageAndReloadFailures(t *testing.T) {
	for _, tc := range []struct{ name, setup string }{
		{"storage inventory insert", `CREATE TRIGGER reject_storage BEFORE INSERT ON player_inventories
			WHEN NEW.player_building_id IS NOT NULL BEGIN SELECT RAISE(ABORT, 'storage insert failed'); END`},
		{"created building reload", `CREATE TRIGGER remove_created_building AFTER INSERT ON player_buildings
			BEGIN DELETE FROM player_buildings WHERE id = NEW.id; END`},
		{"deduction lost update", `CREATE TRIGGER ignore_deduction BEFORE UPDATE OF quantity ON player_inventory_items
			BEGIN SELECT RAISE(IGNORE); END`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newBuildingTestDB(t)
			execBuildingTestSQL(t, db,
				`UPDATE building_categories SET name = 'Storage'`,
				`INSERT INTO building_construction_costs (building_id, item_id, quantity) VALUES (30, 7, 3)`,
				`INSERT INTO player_inventories (id, player_id, capacity) VALUES (10, 1, 100)`,
				`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity) VALUES (1, 10, 7, 8)`,
				tc.setup,
			)
			before := snapshotBuildingTest(t, db)
			status, response := AddPlayerBuilding(1, buildingTestRequest())
			if status != http.StatusInternalServerError || response.Error != "failed to place building" || response.PlayerBuilding != nil {
				t.Fatalf("status = %d, response = %+v", status, response)
			}
			if !reflect.DeepEqual(snapshotBuildingTest(t, db), before) {
				t.Fatal("failed placement changed buildings or inventory")
			}
		})
	}
}

func seedDeleteBuildingTest(t *testing.T, db *gorm.DB, category string) {
	t.Helper()
	if err := db.Exec("UPDATE building_categories SET name = ?", category).Error; err != nil {
		t.Fatal(err)
	}
	execBuildingTestSQL(t, db,
		`INSERT INTO player_buildings (id, player_id, building_id, map_id, building_level_id, x, y) VALUES
			(10, 1, 30, 1, 31, 2, 3), (20, 2, 30, 2, 31, 4, 5)`,
	)
	if category == "Storage" {
		execBuildingTestSQL(t, db,
			`INSERT INTO player_inventories (id, player_id, player_building_id, capacity) VALUES
				(1, 1, 10, 25), (2, 1, 10, 25), (3, 2, 20, 25)`,
			`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity) VALUES
				(1, 1, 7, 3), (2, 2, 7, 4), (3, 3, 7, 5)`,
		)
	}
}

func TestDeletePlayerBuildingRemovesOnlySelectedBuildingAndStorage(t *testing.T) {
	for _, category := range []string{"Production", "Storage"} {
		t.Run(category, func(t *testing.T) {
			db := newBuildingTestDB(t)
			seedDeleteBuildingTest(t, db, category)
			status, err := DeletePlayerBuilding(1, 10)
			if status != http.StatusOK || err != nil {
				t.Fatalf("status = %d, err = %v", status, err)
			}
			after := snapshotBuildingTest(t, db)
			if len(after.Buildings) != 1 || after.Buildings[0].ID != 20 {
				t.Fatalf("remaining buildings = %+v", after.Buildings)
			}
			if category == "Storage" {
				if len(after.Inventories) != 1 || after.Inventories[0].ID != 3 ||
					len(after.Items) != 1 || after.Items[0].ID != 3 || after.Items[0].Quantity != 5 {
					t.Fatalf("remaining storage = %+v", after)
				}
			}
			status, err = DeletePlayerBuilding(1, 10)
			if status != http.StatusNotFound || err == nil || err.Error() != "player building not found" {
				t.Fatalf("repeat deletion: status = %d, err = %v", status, err)
			}
			if !reflect.DeepEqual(snapshotBuildingTest(t, db), after) {
				t.Fatal("repeat deletion changed unrelated state")
			}
		})
	}
}

func TestDeletePlayerBuildingRejectsAnotherPlayersBuilding(t *testing.T) {
	for _, category := range []string{"Production", "Storage"} {
		t.Run(category, func(t *testing.T) {
			db := newBuildingTestDB(t)
			seedDeleteBuildingTest(t, db, category)
			before := snapshotBuildingTest(t, db)
			// Building 20 belongs to player 2.
			status, err := DeletePlayerBuilding(1, 20)
			if status != http.StatusNotFound || err == nil || err.Error() != "player building not found" {
				t.Fatalf("status = %d, err = %v; want 404 player building not found", status, err)
			}
			if !reflect.DeepEqual(snapshotBuildingTest(t, db), before) {
				t.Fatal("refused deletion changed buildings or storage")
			}
		})
	}
}

func TestDeletePlayerBuildingRollsBackDeleteFailures(t *testing.T) {
	for _, tc := range []struct{ name, trigger string }{
		{"second inventory contents", `CREATE TRIGGER reject_delete BEFORE DELETE ON player_inventory_items
			WHEN OLD.player_inventory_id = 2 BEGIN SELECT RAISE(ABORT, 'item delete failed'); END`},
		{"inventories", `CREATE TRIGGER reject_delete BEFORE DELETE ON player_inventories
			BEGIN SELECT RAISE(ABORT, 'inventory delete failed'); END`},
		{"building", `CREATE TRIGGER reject_delete BEFORE DELETE ON player_buildings
			BEGIN SELECT RAISE(ABORT, 'building delete failed'); END`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newBuildingTestDB(t)
			seedDeleteBuildingTest(t, db, "Storage")
			execBuildingTestSQL(t, db, tc.trigger)
			before := snapshotBuildingTest(t, db)
			status, err := DeletePlayerBuilding(1, 10)
			if status != http.StatusInternalServerError || err == nil || err.Error() != "failed to delete player building" {
				t.Fatalf("status = %d, err = %v", status, err)
			}
			if !reflect.DeepEqual(snapshotBuildingTest(t, db), before) {
				t.Fatal("failed deletion changed buildings or storage")
			}
		})
	}
}

func TestDeletePlayerBuildingReportsReadFailures(t *testing.T) {
	for _, table := range []string{"building_categories", "player_inventories"} {
		t.Run(table, func(t *testing.T) {
			db := newBuildingTestDB(t)
			seedDeleteBuildingTest(t, db, "Storage")
			execBuildingTestSQL(t, db, "DROP TABLE "+table)
			status, err := DeletePlayerBuilding(1, 10)
			if status != http.StatusInternalServerError || err == nil || err.Error() != "failed to delete player building" {
				t.Fatalf("status = %d, err = %v", status, err)
			}
			var count int64
			if err := db.Model(&models.PlayerBuilding{}).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 2 {
				t.Fatalf("failed read removed buildings: count = %d", count)
			}
		})
	}
}
