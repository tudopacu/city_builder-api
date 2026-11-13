package controllers

import (
	"API/api/dto/responses"
	"API/models"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func newProductionControllerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newBuildingControllersDB(t)
	execControllerSQL(t, db,
		`INSERT INTO player_buildings (id, player_id, building_id, map_id, building_level_id, x, y)
			VALUES (10, 1, 30, 9, 31, 2, 3)`,
		`INSERT INTO items (id, name, type) VALUES (7, 'Wood', 'resource')`,
		`INSERT INTO building_productions (id, building_id, item_id, production_time_seconds, quantity)
			VALUES (50, 30, 7, 60, 5)`,
	)
	return db
}

func TestBuildingProductionControllersRejectInvalidIDs(t *testing.T) {
	for _, endpoint := range []struct {
		name, method, path string
		handler            gin.HandlerFunc
	}{
		{"start", http.MethodPost, "/game/start_production", StartBuildingProduction},
		{"collect", http.MethodPut, "/game/collect_production", CollectBuildingProduction},
	} {
		for index, parameter := range []string{"player_building_id", "building_production_id"} {
			for _, value := range []string{"abc", "-1", "1.5", "4294967296"} {
				t.Run(endpoint.name+"/"+parameter+"/"+value, func(t *testing.T) {
					ids := []string{"10", "50"}
					ids[index] = value
					response := performPlayerRequest(t, 1, endpoint.method, endpoint.path+"/:player_building_id/:building_production_id",
						endpoint.path+"/"+strings.Join(ids, "/"), "", endpoint.handler)
					assertControllerError(t, response, http.StatusBadRequest, "invalid "+parameter)
				})
			}
		}
	}
}

func TestStartBuildingProductionCreatesPendingProduction(t *testing.T) {
	db := newProductionControllerDB(t)
	before := time.Now()
	response := performPlayerRequest(t, 1, http.MethodPost, "/game/start_production/:player_building_id/:building_production_id",
		"/game/start_production/10/50", "", StartBuildingProduction)
	after := time.Now()
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload responses.StartProductionResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	production := payload.BuildingCurrentProduction
	if production == nil || production.ID == 0 || production.BuildingProductionID != 50 ||
		production.ItemName != "Wood" || production.Quantity != 5 || production.Status != "PENDING" {
		t.Fatalf("unexpected production response: %s", response.Body.String())
	}
	if payload.CurrentTimestamp.Before(before) || payload.CurrentTimestamp.After(after) {
		t.Fatalf("current timestamp %v outside request interval [%v, %v]", payload.CurrentTimestamp, before, after)
	}
	if duration := production.EndTime.Sub(payload.CurrentTimestamp); duration != time.Minute {
		t.Fatalf("production duration = %s, want 1m", duration)
	}
	var saved models.PlayerBuildingProduction
	if err := db.First(&saved, production.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.PlayerID != 1 || saved.PlayerBuildingID != 10 || saved.BuildingProductionID != 50 ||
		saved.Status != "PENDING" || !saved.EndTime.Equal(production.EndTime) {
		t.Fatalf("persisted production = %+v", saved)
	}
}

func TestStartBuildingProductionReturnsServiceErrors(t *testing.T) {
	for _, tc := range []struct {
		name, setup, message string
		status               int
	}{
		{"missing player building", "DELETE FROM player_buildings", "player building not found", http.StatusNotFound},
		{"missing recipe", "DELETE FROM building_productions", "building production not found", http.StatusNotFound},
		{"wrong building recipe", "UPDATE building_productions SET building_id = 99", "building production does not belong to this building", http.StatusBadRequest},
		{"pending production", `INSERT INTO player_building_productions (player_id, player_building_id, building_production_id, status)
			VALUES (1, 10, 50, 'PENDING')`, "production already in progress", http.StatusConflict},
		{"uncollected production", `INSERT INTO player_building_productions (player_id, player_building_id, building_production_id, status)
			VALUES (1, 10, 50, 'DONE')`, "production already in progress", http.StatusConflict},
		{"write failure", `CREATE TRIGGER reject_production BEFORE INSERT ON player_building_productions
			BEGIN SELECT RAISE(ABORT, 'production insert failed'); END`, "failed to start production", http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newProductionControllerDB(t)
			execControllerSQL(t, db, tc.setup)
			var before int64
			if err := db.Model(&models.PlayerBuildingProduction{}).Count(&before).Error; err != nil {
				t.Fatal(err)
			}
			response := performPlayerRequest(t, 1, http.MethodPost, "/game/start_production/:player_building_id/:building_production_id",
				"/game/start_production/10/50", "", StartBuildingProduction)
			assertControllerError(t, response, tc.status, tc.message)
			var after int64
			if err := db.Model(&models.PlayerBuildingProduction{}).Count(&after).Error; err != nil {
				t.Fatal(err)
			}
			if after != before {
				t.Fatalf("rejected request changed production count from %d to %d", before, after)
			}
		})
	}
}

func TestCollectBuildingProductionStoresOutputOnce(t *testing.T) {
	for _, status := range []string{"PENDING", "DONE"} {
		t.Run(status, func(t *testing.T) {
			db := newProductionControllerDB(t)
			entry := models.PlayerBuildingProduction{
				ID: 100, PlayerID: 1, PlayerBuildingID: 10, BuildingProductionID: 50,
				EndTime: time.Now().Add(-time.Hour), Status: status,
			}
			if err := db.Create(&entry).Error; err != nil {
				t.Fatal(err)
			}
			execControllerSQL(t, db,
				`INSERT INTO player_inventories (id, player_id, map_id, capacity) VALUES (10, 1, 9, 10)`,
				`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity) VALUES (1, 10, 7, 3)`,
			)
			response := performPlayerRequest(t, 1, http.MethodPut, "/game/collect_production/:player_building_id/:building_production_id",
				"/game/collect_production/10/50", "", CollectBuildingProduction)
			assertControllerJSON(t, response, http.StatusOK, `{"message":"production collected successfully"}`)
			if err := db.First(&entry, 100).Error; err != nil {
				t.Fatal(err)
			}
			if entry.Status != "COLLECTED" {
				t.Fatalf("production status = %q, want COLLECTED", entry.Status)
			}
			var item models.PlayerInventoryItem
			if err := db.First(&item, 1).Error; err != nil {
				t.Fatal(err)
			}
			if item.Quantity != 8 {
				t.Fatalf("stored quantity = %d, want 8", item.Quantity)
			}
			response = performPlayerRequest(t, 1, http.MethodPut, "/game/collect_production/:player_building_id/:building_production_id",
				"/game/collect_production/10/50", "", CollectBuildingProduction)
			assertControllerError(t, response, http.StatusNotFound, "production not found")
			if err := db.First(&item, 1).Error; err != nil {
				t.Fatal(err)
			}
			if item.Quantity != 8 {
				t.Fatalf("repeat collection changed stored quantity to %d", item.Quantity)
			}
		})
	}
}

func TestCollectBuildingProductionReturnsServiceErrors(t *testing.T) {
	for _, tc := range []struct {
		name, setup, ids, message string
		playerID                  uint
		status                    int
	}{
		{"missing production", "DELETE FROM player_building_productions", "10/50", "production not found", 1, http.StatusNotFound},
		{"wrong player", "", "10/50", "production not found", 2, http.StatusNotFound},
		{"wrong player building", "", "20/50", "production not found", 1, http.StatusNotFound},
		{"wrong recipe", "", "10/60", "production not found", 1, http.StatusNotFound},
		{"unfinished", "UPDATE player_building_productions SET end_time = '2999-01-01 00:00:00'", "10/50", "production is not finished yet", 1, http.StatusBadRequest},
		{"insufficient capacity", "UPDATE player_inventories SET capacity = 4", "10/50", "not enough capacity: available 4, requested 5", 1, http.StatusBadRequest},
		{"database error", "DROP TABLE player_building_productions", "10/50", "failed to collect production", 1, http.StatusInternalServerError},
		{"write failure", `CREATE TRIGGER reject_collection BEFORE UPDATE OF status ON player_building_productions
			BEGIN SELECT RAISE(ABORT, 'status update failed'); END`, "10/50", "failed to collect production", 1, http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newProductionControllerDB(t)
			execControllerSQL(t, db,
				`INSERT INTO player_building_productions (id, player_id, player_building_id, building_production_id, end_time, status)
					VALUES (100, 1, 10, 50, '2020-01-01 00:00:00', 'PENDING')`,
				`INSERT INTO player_inventories (id, player_id, map_id, capacity) VALUES (10, 1, 9, 10)`,
			)
			if tc.setup != "" {
				execControllerSQL(t, db, tc.setup)
			}
			response := performPlayerRequest(t, tc.playerID, http.MethodPut, "/game/collect_production/:player_building_id/:building_production_id",
				"/game/collect_production/"+tc.ids, "", CollectBuildingProduction)
			assertControllerError(t, response, tc.status, tc.message)
			var itemCount int64
			if err := db.Model(&models.PlayerInventoryItem{}).Count(&itemCount).Error; err != nil {
				t.Fatal(err)
			}
			if itemCount != 0 {
				t.Fatalf("failed collection left %d inventory items", itemCount)
			}
		})
	}
}

// Player 2 used to be able to run production in player 1's building under their
// own player ID, then collect the output into their own storage.
func TestPlayerCannotProduceInAnotherPlayersBuilding(t *testing.T) {
	db := newProductionControllerDB(t) // building 10 belongs to player 1
	execControllerSQL(t, db,
		`INSERT INTO player_inventories (id, player_id, map_id, capacity) VALUES (77, 2, 9, 25)`,
	)

	response := performPlayerRequest(t, 2, http.MethodPost, "/game/start_production/:player_building_id/:building_production_id",
		"/game/start_production/10/50", "", StartBuildingProduction)
	assertControllerError(t, response, http.StatusNotFound, "player building not found")

	// A production the old code already recorded that way pays nothing out either.
	execControllerSQL(t, db,
		`INSERT INTO player_building_productions (id, player_id, player_building_id, building_production_id, end_time, status)
			VALUES (100, 2, 10, 50, '2020-01-01 00:00:00', 'DONE')`,
	)
	response = performPlayerRequest(t, 2, http.MethodPut, "/game/collect_production/:player_building_id/:building_production_id",
		"/game/collect_production/10/50", "", CollectBuildingProduction)
	assertControllerError(t, response, http.StatusNotFound, "production not found")

	var itemCount int64
	if err := db.Model(&models.PlayerInventoryItem{}).Count(&itemCount).Error; err != nil {
		t.Fatal(err)
	}
	if itemCount != 0 {
		t.Fatalf("player 2 received %d inventory items from player 1's building", itemCount)
	}

	// Nor does that production show up in player 1's game, where it would block the building.
	response = performPlayerRequest(t, 1, http.MethodGet, "/game/get_player_buildings/:map_id",
		"/game/get_player_buildings/9", "", GetPlayerBuildings)
	assertControllerJSON(t, response, http.StatusOK, `{"player_buildings":[
		{"id":10,"building":{"id":30,"name":"Sawmill","description":"","width":1,"length":1,"building_category":"Production"},
		"level":1,"x":2,"y":3,"building_current_production":null}]}`)
}
