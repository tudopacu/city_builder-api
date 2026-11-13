package controllers

import (
	"API/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func newBuildingControllersDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newControllerTestDB(t,
		`CREATE TABLE buildings (id INTEGER PRIMARY KEY, name TEXT, width INTEGER, length INTEGER, building_category_id INTEGER)`,
		`CREATE TABLE building_categories (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE building_levels (id INTEGER PRIMARY KEY, building_id INTEGER, level INTEGER, capacity INTEGER DEFAULT 0,
			population_capacity INTEGER DEFAULT 0, population_cost INTEGER DEFAULT 0)`,
		`CREATE TABLE players (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE building_construction_costs (id INTEGER PRIMARY KEY, building_id INTEGER, item_id INTEGER, quantity INTEGER)`,
		`CREATE TABLE player_buildings (id INTEGER PRIMARY KEY, player_id INTEGER, building_id INTEGER, map_id INTEGER,
			building_level_id INTEGER, x INTEGER, y INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE player_inventories (id INTEGER PRIMARY KEY, player_id INTEGER, map_id INTEGER, player_building_id INTEGER,
			capacity INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE player_inventory_items (id INTEGER PRIMARY KEY, player_inventory_id INTEGER, item_id INTEGER,
			quantity INTEGER, created_at DATETIME, updated_at DATETIME, UNIQUE (player_inventory_id, item_id))`,
		`CREATE TABLE building_productions (id INTEGER PRIMARY KEY, building_id INTEGER, item_id INTEGER,
			production_time_seconds INTEGER, quantity INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE player_building_productions (id INTEGER PRIMARY KEY, player_id INTEGER, player_building_id INTEGER,
			building_production_id INTEGER, end_time DATETIME, status TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT, type TEXT)`,
		`CREATE TABLE tiles (id INTEGER PRIMARY KEY, type TEXT)`,
		`CREATE TABLE terrains (id INTEGER PRIMARY KEY, map_id INTEGER, tile_id INTEGER, x INTEGER, y INTEGER)`,
		`INSERT INTO building_categories (id, name) VALUES (1, 'Production')`,
		`INSERT INTO buildings (id, name, width, length, building_category_id) VALUES (30, 'Sawmill', 1, 1, 1)`,
		`INSERT INTO building_levels (id, building_id, level, capacity) VALUES (31, 30, 1, 25)`,
		`INSERT INTO tiles (id, type) VALUES (1, 'grass')`,
		`INSERT INTO terrains (id, map_id, tile_id, x, y) VALUES (1, 9, 1, 2, 3)`,
	)
}

func TestGetPlayerBuildingsReturnsScopedBuildingsAndProduction(t *testing.T) {
	db := newBuildingControllersDB(t)
	execControllerSQL(t, db,
		`INSERT INTO player_buildings (id, player_id, building_id, map_id, building_level_id, x, y) VALUES
			(10, 1, 30, 9, 31, 2, 3), (20, 1, 30, 9, 31, 4, 5),
			(30, 2, 30, 9, 31, 6, 7), (40, 1, 30, 8, 31, 8, 9)`,
		`INSERT INTO items (id, name, type) VALUES (7, 'Wood', 'resource')`,
		`INSERT INTO building_productions (id, building_id, item_id, quantity) VALUES (50, 30, 7, 5)`,
		`INSERT INTO player_building_productions (id, player_id, player_building_id, building_production_id, end_time, status) VALUES
			(100, 1, 10, 50, '2020-01-01 00:00:00', 'PENDING'),
			(200, 1, 20, 50, '2020-01-01 00:00:00', 'COLLECTED'),
			(300, 2, 30, 50, '2020-01-01 00:00:00', 'PENDING'),
			(400, 1, 40, 50, '2020-01-01 00:00:00', 'PENDING')`,
	)
	response := performPlayerRequest(t, 1, http.MethodGet, "/game/get_player_buildings/:map_id",
		"/game/get_player_buildings/9", "", GetPlayerBuildings)
	assertControllerJSON(t, response, http.StatusOK, `{"player_buildings":[
		{"id":10,"building":{"id":30,"name":"Sawmill","description":"","width":1,"length":1,"building_category":"Production"},
		"level":1,"x":2,"y":3,"building_current_production":{"id":100,"end_time":"2020-01-01T00:00:00Z","status":"DONE","building_production_id":50,"item_name":"Wood","quantity":5}},
		{"id":20,"building":{"id":30,"name":"Sawmill","description":"","width":1,"length":1,"building_category":"Production"},
		"level":1,"x":4,"y":5,"building_current_production":null}]}`)
	var productions []models.PlayerBuildingProduction
	if err := db.Order("id").Find(&productions).Error; err != nil {
		t.Fatal(err)
	}
	want := []string{"DONE", "COLLECTED", "PENDING", "PENDING"}
	if len(productions) != len(want) {
		t.Fatalf("productions = %+v", productions)
	}
	for i, production := range productions {
		if production.Status != want[i] {
			t.Errorf("production %d status = %q, want %q", production.ID, production.Status, want[i])
		}
	}
}

func TestGetPlayerBuildingsReturnsEmptyArray(t *testing.T) {
	newBuildingControllersDB(t)
	response := performPlayerRequest(t, 1, http.MethodGet, "/game/get_player_buildings/:map_id",
		"/game/get_player_buildings/9", "", GetPlayerBuildings)
	assertControllerJSON(t, response, http.StatusOK, `{"player_buildings":[]}`)
}

func TestGetPlayerBuildingsRejectsInvalidMapID(t *testing.T) {
	for _, value := range []string{"abc", "-1", "1.5", "4294967296"} {
		t.Run(value, func(t *testing.T) {
			response := performPlayerRequest(t, 1, http.MethodGet, "/game/get_player_buildings/:map_id",
				"/game/get_player_buildings/"+value, "", GetPlayerBuildings)
			assertControllerError(t, response, http.StatusBadRequest, "invalid map_id")
		})
	}
}

func TestGetPlayerBuildingsReturnsDatabaseError(t *testing.T) {
	db := newBuildingControllersDB(t)
	execControllerSQL(t, db, "DROP TABLE player_building_productions")
	response := performPlayerRequest(t, 1, http.MethodGet, "/game/get_player_buildings/:map_id",
		"/game/get_player_buildings/9", "", GetPlayerBuildings)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 1 || payload["error"] == nil {
		t.Fatalf("expected error envelope, got %s", response.Body.String())
	}
}

const addBuildingControllerBody = `{"building_id":30,"map_id":9,"x":2,"y":3}`

func TestAddPlayerBuildingRejectsInvalidRequest(t *testing.T) {
	for _, body := range []string{
		``, `{`, `[]`, `{}`, `{"building_id":"30","map_id":9,"x":2,"y":3}`,
		`{"building_id":-1,"map_id":9,"x":2,"y":3}`,
		`{"map_id":9,"x":2,"y":3}`,
		`{"building_id":30,"x":2,"y":3}`,
		`{"building_id":30,"map_id":9,"y":3}`,
		`{"building_id":30,"map_id":9,"x":2}`,
	} {
		t.Run(body, func(t *testing.T) {
			response := performPlayerRequest(t, 1, http.MethodPost, "/game/add_building", "/game/add_building", body, AddPlayerBuilding)
			assertControllerError(t, response, http.StatusBadRequest, "invalid request")
		})
	}
	for _, coordinate := range []string{"x", "y"} {
		t.Run("negative "+coordinate, func(t *testing.T) {
			body := strings.Replace(addBuildingControllerBody, `"`+coordinate+`":`+map[string]string{"x": "2", "y": "3"}[coordinate], `"`+coordinate+`":-1`, 1)
			response := performPlayerRequest(t, 1, http.MethodPost, "/game/add_building", "/game/add_building", body, AddPlayerBuilding)
			assertControllerError(t, response, http.StatusBadRequest, "coordinates must be non-negative")
		})
	}
}

func TestAddPlayerBuildingCreatesBuildingAndConsumesItems(t *testing.T) {
	for _, category := range []string{"Production", "Storage"} {
		t.Run(category, func(t *testing.T) {
			db := newBuildingControllersDB(t)
			if err := db.Exec("UPDATE building_categories SET name = ? WHERE id = 1", category).Error; err != nil {
				t.Fatal(err)
			}
			execControllerSQL(t, db,
				`INSERT INTO building_construction_costs (building_id, item_id, quantity) VALUES (30, 7, 3)`,
				`INSERT INTO player_inventories (id, player_id, map_id, capacity) VALUES (10, 1, 9, 100)`,
				`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity) VALUES (1, 10, 7, 8)`,
			)
			response := performPlayerRequest(t, 1, http.MethodPost, "/game/add_building", "/game/add_building", addBuildingControllerBody, AddPlayerBuilding)
			assertControllerJSON(t, response, http.StatusCreated, fmt.Sprintf(`{"player_building":{"id":1,
				"building":{"id":30,"name":"Sawmill","description":"","width":1,"length":1,"building_category":%q},
				"level":1,"x":2,"y":3,"building_current_production":null}}`, category))
			var building models.PlayerBuilding
			if err := db.First(&building, 1).Error; err != nil {
				t.Fatal(err)
			}
			if building.PlayerID != 1 || building.MapID != 9 || building.BuildingID != 30 || building.BuildingLevelID != 31 || building.X != 2 || building.Y != 3 {
				t.Fatalf("persisted building = %+v", building)
			}
			var item models.PlayerInventoryItem
			if err := db.First(&item, 1).Error; err != nil {
				t.Fatal(err)
			}
			if item.Quantity != 5 {
				t.Fatalf("construction stock = %d, want 5", item.Quantity)
			}
			var inventories []models.PlayerInventory
			if err := db.Where("player_building_id = ?", building.ID).Find(&inventories).Error; err != nil {
				t.Fatal(err)
			}
			if category == "Storage" {
				if len(inventories) != 1 || inventories[0].Capacity != 25 || inventories[0].PlayerID != 1 || inventories[0].MapID != 9 {
					t.Fatalf("storage inventory = %+v", inventories)
				}
			} else if len(inventories) != 0 {
				t.Fatalf("production building created inventories: %+v", inventories)
			}
		})
	}
}

func TestAddPlayerBuildingReturnsServiceErrors(t *testing.T) {
	for _, tc := range []struct {
		name, setup, message string
		status               int
	}{
		{"missing building", "DELETE FROM buildings", "building not found", http.StatusNotFound},
		{"missing level", "DELETE FROM building_levels", "building level not found", http.StatusNotFound},
		{"missing terrain", "DELETE FROM terrains", "some terrain tiles are missing", http.StatusBadRequest},
		{"invalid terrain", "UPDATE tiles SET type = 'water'", "building can only be placed on grass or dirt tiles, found water at position (2, 3)", http.StatusBadRequest},
		{"overlap", `INSERT INTO player_buildings (id, player_id, building_id, map_id, building_level_id, x, y) VALUES (10, 1, 30, 9, 31, 2, 3)`, "building overlaps with an existing building", http.StatusBadRequest},
		{"insufficient items", `INSERT INTO building_construction_costs (building_id, item_id, quantity) VALUES (30, 7, 3)`, "not enough items for construction: item 7, required 3, available 0", http.StatusBadRequest},
		{"database failure", "DROP TABLE terrains", "failed to place building", http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newBuildingControllersDB(t)
			execControllerSQL(t, db, tc.setup)
			var before int64
			if err := db.Model(&models.PlayerBuilding{}).Count(&before).Error; err != nil {
				t.Fatal(err)
			}
			response := performPlayerRequest(t, 1, http.MethodPost, "/game/add_building", "/game/add_building", addBuildingControllerBody, AddPlayerBuilding)
			assertControllerError(t, response, tc.status, tc.message)
			var after int64
			if err := db.Model(&models.PlayerBuilding{}).Count(&after).Error; err != nil {
				t.Fatal(err)
			}
			if after != before {
				t.Fatalf("rejected placement changed building count from %d to %d", before, after)
			}
		})
	}
}

// A player_id in the body used to decide whose building it was and whose items paid for it.
func TestAddPlayerBuildingIgnoresPlayerIDInBody(t *testing.T) {
	db := newBuildingControllersDB(t)
	execControllerSQL(t, db,
		`INSERT INTO building_construction_costs (building_id, item_id, quantity) VALUES (30, 7, 3)`,
		`INSERT INTO player_inventories (id, player_id, map_id, capacity) VALUES (10, 1, 9, 100), (20, 2, 9, 100)`,
		`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity) VALUES (1, 10, 7, 8), (2, 20, 7, 8)`,
	)
	response := performPlayerRequest(t, 1, http.MethodPost, "/game/add_building", "/game/add_building",
		`{"player_id":2,"building_id":30,"map_id":9,"x":2,"y":3}`, AddPlayerBuilding)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var building models.PlayerBuilding
	if err := db.First(&building, 1).Error; err != nil {
		t.Fatal(err)
	}
	if building.PlayerID != 1 {
		t.Fatalf("building belongs to player %d, want the authenticated player 1", building.PlayerID)
	}
	var items []models.PlayerInventoryItem
	if err := db.Order("id").Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Quantity != 5 || items[1].Quantity != 8 {
		t.Fatalf("stock = %+v, want player 1 to pay 3 and player 2 to keep 8", items)
	}
}

func TestDeletePlayerBuildingDeletesStorageAndContents(t *testing.T) {
	db := newBuildingControllersDB(t)
	execControllerSQL(t, db,
		`UPDATE building_categories SET name = 'Storage'`,
		`INSERT INTO player_buildings (id, player_id, building_id, map_id, building_level_id, x, y) VALUES
			(10, 1, 30, 9, 31, 2, 3), (20, 1, 30, 9, 31, 4, 5)`,
		`INSERT INTO player_inventories (id, player_id, player_building_id, capacity) VALUES (1, 1, 10, 25), (2, 1, 20, 25)`,
		`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity) VALUES (1, 1, 7, 3), (2, 2, 7, 4)`,
	)
	response := performPlayerRequest(t, 1, http.MethodDelete, "/game/remove_player_building/:player_building_id",
		"/game/remove_player_building/10", "", DeletePlayerBuilding)
	assertControllerJSON(t, response, http.StatusOK, `{"message":"player building deleted successfully"}`)
	for _, check := range []struct {
		table string
		id    int
	}{{"player_buildings", 20}, {"player_inventories", 2}, {"player_inventory_items", 2}} {
		var ids []int
		if err := db.Table(check.table).Pluck("id", &ids).Error; err != nil {
			t.Fatal(err)
		}
		if len(ids) != 1 || ids[0] != check.id {
			t.Errorf("remaining %s IDs = %v, want [%d]", check.table, ids, check.id)
		}
	}
}

func TestDeletePlayerBuildingRejectsAnotherPlayersBuilding(t *testing.T) {
	db := newBuildingControllersDB(t)
	execControllerSQL(t, db,
		`INSERT INTO player_buildings (id, player_id, building_id, map_id, building_level_id, x, y) VALUES (10, 1, 30, 9, 31, 2, 3)`,
	)
	response := performPlayerRequest(t, 2, http.MethodDelete, "/game/remove_player_building/:player_building_id",
		"/game/remove_player_building/10", "", DeletePlayerBuilding)
	assertControllerError(t, response, http.StatusNotFound, "player building not found")
	var remaining int64
	if err := db.Model(&models.PlayerBuilding{}).Count(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Fatalf("player buildings = %d, want player 1's building to remain", remaining)
	}
}

func TestGetPlayerPopulationReturnsTotalsForPlayerAndMap(t *testing.T) {
	db := newBuildingControllersDB(t)
	execControllerSQL(t, db,
		`INSERT INTO building_levels (id, building_id, level, population_capacity, population_cost) VALUES
			(41, 40, 1, 10, 0), (42, 30, 2, 0, 3)`,
		// Only player 1 on map 9 counts: one house and one sawmill. The sawmill on map 8
		// and the house of player 2 belong elsewhere.
		`INSERT INTO player_buildings (id, player_id, building_id, map_id, building_level_id, x, y) VALUES
			(10, 1, 40, 9, 41, 2, 3), (20, 1, 30, 9, 42, 4, 5), (30, 1, 30, 8, 42, 6, 7), (40, 2, 40, 9, 41, 8, 9)`,
	)

	response := performPlayerRequest(t, 1, http.MethodGet, "/game/get_player_population/:map_id",
		"/game/get_player_population/9", "", GetPlayerPopulation)

	assertControllerJSON(t, response, http.StatusOK, `{"population":{"total":10,"used":3,"available":7}}`)
}

func TestGetPlayerPopulationRejectsInvalidMapID(t *testing.T) {
	response := performPlayerRequest(t, 1, http.MethodGet, "/game/get_player_population/:map_id",
		"/game/get_player_population/abc", "", GetPlayerPopulation)
	assertControllerError(t, response, http.StatusBadRequest, "invalid map_id")
}

func TestDeletePlayerBuildingRejectsHousingStillInUse(t *testing.T) {
	db := newBuildingControllersDB(t)
	execControllerSQL(t, db,
		`INSERT INTO buildings (id, name, width, length, building_category_id) VALUES (40, 'House', 1, 1, 1)`,
		`INSERT INTO building_levels (id, building_id, level, population_capacity, population_cost) VALUES (41, 40, 1, 2, 0)`,
		`UPDATE building_levels SET population_cost = 3 WHERE id = 31`,
		`INSERT INTO player_buildings (id, player_id, building_id, map_id, building_level_id, x, y) VALUES
			(10, 1, 40, 9, 41, 2, 3), (20, 1, 30, 9, 31, 4, 5)`,
	)

	response := performPlayerRequest(t, 1, http.MethodDelete, "/game/remove_player_building/:player_building_id",
		"/game/remove_player_building/10", "", DeletePlayerBuilding)

	assertControllerError(t, response, http.StatusConflict,
		"cannot remove housing: 3 population is in use but only 0 would remain")
	var remaining int64
	if err := db.Model(&models.PlayerBuilding{}).Count(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if remaining != 2 {
		t.Fatalf("player buildings = %d, want 2 after a refused removal", remaining)
	}
}

func TestDeletePlayerBuildingRejectsInvalidID(t *testing.T) {
	for _, id := range []string{"abc", "-1", "1.5", "4294967296"} {
		t.Run(id, func(t *testing.T) {
			response := performPlayerRequest(t, 1, http.MethodDelete, "/game/remove_player_building/:player_building_id",
				"/game/remove_player_building/"+id, "", DeletePlayerBuilding)
			assertControllerError(t, response, http.StatusBadRequest, "invalid player_building_id")
		})
	}
}

func TestDeletePlayerBuildingReturnsServiceErrors(t *testing.T) {
	for _, tc := range []struct {
		name, setup, message string
		status               int
	}{
		{"not found", "", "player building not found", http.StatusNotFound},
		{"database error", "DROP TABLE player_buildings", "failed to delete player building", http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newBuildingControllersDB(t)
			if tc.setup != "" {
				execControllerSQL(t, db, tc.setup)
			}
			response := performPlayerRequest(t, 1, http.MethodDelete, "/game/remove_player_building/:player_building_id",
				"/game/remove_player_building/10", "", DeletePlayerBuilding)
			assertControllerError(t, response, tc.status, tc.message)
		})
	}
}
