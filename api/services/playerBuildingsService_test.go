package services

import (
	"API/api/dto/requests"
	"API/database"
	"API/models"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func execBuildingTestSQL(t *testing.T, db *gorm.DB, statements ...string) {
	t.Helper()
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func newBuildingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previousDB
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	// Minimal tables avoid unrelated MySQL-specific models and associations.
	execBuildingTestSQL(t, db,
		`CREATE TABLE buildings (id INTEGER PRIMARY KEY, name TEXT, width INTEGER, length INTEGER, building_category_id INTEGER)`,
		`CREATE TABLE building_categories (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE building_levels (id INTEGER PRIMARY KEY, building_id INTEGER, level INTEGER, capacity INTEGER DEFAULT 0)`,
		`CREATE TABLE building_construction_costs (id INTEGER PRIMARY KEY, building_id INTEGER, item_id INTEGER, quantity INTEGER)`,
		`CREATE TABLE player_buildings (id INTEGER PRIMARY KEY, player_id INTEGER, building_id INTEGER, map_id INTEGER,
			building_level_id INTEGER, x INTEGER, y INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE player_inventories (id INTEGER PRIMARY KEY, player_id INTEGER, map_id INTEGER, player_building_id INTEGER,
			capacity INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE player_inventory_items (id INTEGER PRIMARY KEY, player_inventory_id INTEGER, item_id INTEGER,
			quantity INTEGER, created_at DATETIME, updated_at DATETIME, UNIQUE (player_inventory_id, item_id))`,
		`CREATE TABLE tiles (id INTEGER PRIMARY KEY, type TEXT)`,
		`CREATE TABLE terrains (id INTEGER PRIMARY KEY, map_id INTEGER, tile_id INTEGER, x INTEGER, y INTEGER)`,
		`INSERT INTO building_categories (id, name) VALUES (1, 'Production')`,
		`INSERT INTO buildings (id, name, width, length, building_category_id) VALUES (30, 'Sawmill', 1, 1, 1)`,
		`INSERT INTO building_levels (id, building_id, level) VALUES (31, 30, 1)`,
		`INSERT INTO tiles (id, type) VALUES (1, 'grass')`,
		`INSERT INTO terrains (id, map_id, tile_id, x, y) VALUES (1, 1, 1, 2, 3)`,
	)
	return db
}

func buildingTestRequest() requests.AddBuildingRequest {
	return requests.AddBuildingRequest{PlayerID: 1, BuildingID: 30, MapID: 1, X: 2, Y: 3}
}

type buildingTestSnapshot struct {
	Buildings   []models.PlayerBuilding
	Inventories []models.PlayerInventory
	Items       []models.PlayerInventoryItem
}

func snapshotBuildingTest(t *testing.T, db *gorm.DB) buildingTestSnapshot {
	t.Helper()
	var snapshot buildingTestSnapshot
	for _, value := range []any{&snapshot.Buildings, &snapshot.Inventories, &snapshot.Items} {
		if err := db.Order("id ASC").Find(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	return snapshot
}

func TestAddPlayerBuildingConsumesConstructionCostsAcrossInventories(t *testing.T) {
	db := newBuildingTestDB(t)
	execBuildingTestSQL(t, db,
		`INSERT INTO building_construction_costs (id, building_id, item_id, quantity) VALUES
			(1, 30, 7, 4), (2, 30, 8, 3), (3, 30, 7, 5), (4, 99, 9, 100)`,
		`INSERT INTO player_inventories (id, player_id, capacity) VALUES (20, 1, 100), (10, 1, 100), (1, 2, 1000)`,
		`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity) VALUES
			(1, 20, 7, 10), (2, 10, 7, 4), (3, 10, 8, 2), (4, 20, 8, 8), (5, 10, 9, 6), (6, 1, 7, 200)`,
	)
	status, response := AddPlayerBuilding(buildingTestRequest())
	if status != http.StatusCreated || response.Error != "" || response.PlayerBuilding == nil {
		t.Fatalf("status = %d, response = %+v", status, response)
	}
	building := response.PlayerBuilding
	if building.ID == 0 || building.Building == nil || building.Building.ID != 30 ||
		building.Building.BuildingCategory != "Production" || building.BuildingLevel != 1 || building.X != 2 || building.Y != 3 {
		t.Fatalf("unexpected building response: %+v", building)
	}
	after := snapshotBuildingTest(t, db)
	if len(after.Buildings) != 1 || after.Buildings[0].PlayerID != 1 || after.Buildings[0].MapID != 1 || after.Buildings[0].BuildingLevelID != 31 {
		t.Fatalf("unexpected saved buildings: %+v", after.Buildings)
	}
	quantities := make(map[uint]int)
	for _, item := range after.Items {
		quantities[item.ID] = item.Quantity
	}
	want := map[uint]int{1: 5, 2: 0, 3: 0, 4: 7, 5: 6, 6: 200}
	if !reflect.DeepEqual(quantities, want) {
		t.Fatalf("remaining quantities = %v, want %v", quantities, want)
	}
}

func TestAddPlayerBuildingWithoutCostsNeedsNoInventories(t *testing.T) {
	db := newBuildingTestDB(t)
	status, response := AddPlayerBuilding(buildingTestRequest())
	if status != http.StatusCreated || response.Error != "" || response.PlayerBuilding == nil {
		t.Fatalf("status = %d, response = %+v", status, response)
	}
	after := snapshotBuildingTest(t, db)
	if len(after.Buildings) != 1 || len(after.Inventories) != 0 || len(after.Items) != 0 {
		t.Fatalf("unexpected database contents: %+v", after)
	}
}

func TestAddPlayerBuildingRejectsInsufficientConstructionItemsWithoutWrites(t *testing.T) {
	for _, tc := range []struct {
		name      string
		costs     string
		itemID    uint
		required  int
		available int
	}{
		{name: "missing item", costs: "(30, 7, 1), (30, 8, 4)", itemID: 8, required: 4, available: 0},
		{name: "insufficient quantity", costs: "(30, 7, 7)", itemID: 7, required: 7, available: 6},
		{name: "duplicate costs are combined", costs: "(30, 7, 4), (30, 7, 5)", itemID: 7, required: 9, available: 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newBuildingTestDB(t)
			execBuildingTestSQL(t, db,
				"INSERT INTO building_construction_costs (building_id, item_id, quantity) VALUES "+tc.costs,
				`INSERT INTO player_inventories (id, player_id, capacity) VALUES (10, 1, 100), (20, 2, 1000)`,
				`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity) VALUES
					(1, 10, 7, 6), (2, 20, 7, 100), (3, 20, 8, 100)`,
				`CREATE TRIGGER reject_deduction BEFORE UPDATE ON player_inventory_items BEGIN SELECT RAISE(ABORT, 'unexpected deduction'); END`,
				`CREATE TRIGGER reject_placement BEFORE INSERT ON player_buildings BEGIN SELECT RAISE(ABORT, 'unexpected placement'); END`,
			)
			before := snapshotBuildingTest(t, db)
			status, response := AddPlayerBuilding(buildingTestRequest())
			if status != http.StatusBadRequest || response.PlayerBuilding != nil {
				t.Fatalf("status = %d, response = %+v", status, response)
			}
			itemPattern := regexp.MustCompile(fmt.Sprintf(`item(?:_id)?[^0-9]*%d\b`, tc.itemID))
			if !itemPattern.MatchString(response.Error) ||
				!strings.Contains(response.Error, fmt.Sprintf("required %d", tc.required)) ||
				!strings.Contains(response.Error, fmt.Sprintf("available %d", tc.available)) {
				t.Fatalf("error does not identify missing item and amounts: %q", response.Error)
			}
			if !reflect.DeepEqual(snapshotBuildingTest(t, db), before) {
				t.Fatal("insufficient resources changed buildings or inventory")
			}
		})
	}
}

func TestAddPlayerBuildingRollsBackOnFailure(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup string
	}{
		{
			name: "later deduction failure",
			setup: `CREATE TRIGGER reject_later_deduction BEFORE UPDATE OF quantity ON player_inventory_items
				WHEN OLD.id = 2 AND (SELECT quantity FROM player_inventory_items WHERE id = 1) = 0
				BEGIN SELECT RAISE(ABORT, 'later deduction failed'); END`,
		},
		{
			name:  "building insert failure",
			setup: `CREATE TRIGGER reject_placement BEFORE INSERT ON player_buildings BEGIN SELECT RAISE(ABORT, 'placement failed'); END`,
		},
		{name: "association reload failure", setup: "DROP TABLE building_categories"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newBuildingTestDB(t)
			execBuildingTestSQL(t, db,
				`INSERT INTO building_construction_costs (building_id, item_id, quantity) VALUES (30, 7, 6)`,
				`INSERT INTO player_inventories (id, player_id, capacity) VALUES (10, 1, 100), (20, 1, 100)`,
				`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity) VALUES (1, 10, 7, 3), (2, 20, 7, 5)`,
				tc.setup,
			)
			before := snapshotBuildingTest(t, db)
			status, response := AddPlayerBuilding(buildingTestRequest())
			if status != http.StatusInternalServerError || response.Error == "" || response.PlayerBuilding != nil {
				t.Fatalf("status = %d, response = %+v", status, response)
			}
			if !reflect.DeepEqual(snapshotBuildingTest(t, db), before) {
				t.Fatal("failed placement did not roll back all building and inventory changes")
			}
		})
	}
}

func TestAddPlayerBuildingPreservesPlacementValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup string
	}{
		{name: "invalid terrain", setup: "UPDATE tiles SET type = 'water' WHERE id = 1"},
		{name: "missing terrain", setup: "DELETE FROM terrains"},
		{
			name: "overlapping building",
			setup: `INSERT INTO player_buildings (id, player_id, building_id, map_id, building_level_id, x, y)
				VALUES (1, 2, 30, 1, 31, 2, 3)`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newBuildingTestDB(t)
			execBuildingTestSQL(t, db,
				`INSERT INTO building_construction_costs (building_id, item_id, quantity) VALUES (30, 7, 5)`,
				`INSERT INTO player_inventories (id, player_id, capacity) VALUES (10, 1, 100)`,
				`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity) VALUES (1, 10, 7, 10)`,
				tc.setup,
			)
			before := snapshotBuildingTest(t, db)
			status, response := AddPlayerBuilding(buildingTestRequest())
			if status != http.StatusBadRequest || response.Error == "" || response.PlayerBuilding != nil {
				t.Fatalf("status = %d, response = %+v", status, response)
			}
			if !reflect.DeepEqual(snapshotBuildingTest(t, db), before) {
				t.Fatal("rejected placement changed buildings or inventory")
			}
		})
	}
}
