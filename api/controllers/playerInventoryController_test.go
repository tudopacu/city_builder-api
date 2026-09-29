package controllers

import (
	"API/database"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newPlayerInventoriesTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
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

	for _, statement := range []string{
		`CREATE TABLE player_inventories (
			id INTEGER PRIMARY KEY, player_id INTEGER, map_id INTEGER, player_building_id INTEGER,
			capacity INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE player_inventory_items (
			id INTEGER PRIMARY KEY, player_inventory_id INTEGER, item_id INTEGER,
			quantity INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT, type TEXT)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func requestPlayerInventories(playerID, mapID string) *httptest.ResponseRecorder {
	router := gin.New()
	router.GET("/game/get_player_inventories/:player_id/:map_id", GetPlayerInventories)
	response := httptest.NewRecorder()
	path := "/game/get_player_inventories/" + playerID
	if mapID != "" {
		path += "/" + mapID
	}
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	return response
}

func TestGetPlayerInventoriesReturnsContentsAndTotalsForPlayerAndMap(t *testing.T) {
	db := newPlayerInventoriesTestDB(t)
	for _, statement := range []string{
		`INSERT INTO player_inventories (id, player_id, map_id, player_building_id, capacity) VALUES
			(10, 1, 9, 100, 50), (20, 1, 9, 200, 100), (30, 1, 9, 300, 25),
			(40, 2, 9, 400, 500), (50, 1, 10, 500, 600)`,
		`INSERT INTO items (id, name, type) VALUES (7, 'Wood', 'resource'), (8, 'Stone', 'resource')`,
		`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity) VALUES
			(1, 10, 7, 5), (2, 10, 8, 12), (3, 20, 8, 30), (4, 40, 7, 400), (5, 50, 7, 500)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	response := requestPlayerInventories("1", "9")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 1 || payload["player_inventory"] == nil {
		t.Fatalf("unexpected response envelope: %s", response.Body.String())
	}
	var inventoryResponse struct {
		Inventories   []map[string]any `json:"player_inventories"`
		TotalQuantity int              `json:"total_quantity"`
		TotalCapacity int              `json:"total_capacity"`
	}
	if err := json.Unmarshal(payload["player_inventory"], &inventoryResponse); err != nil {
		t.Fatal(err)
	}
	if inventoryResponse.TotalQuantity != 47 || inventoryResponse.TotalCapacity != 175 {
		t.Fatalf("unexpected inventory totals: %s", response.Body.String())
	}
	inventories := inventoryResponse.Inventories
	if len(inventories) != 3 {
		t.Fatalf("inventory count = %d, want 3; body = %s", len(inventories), response.Body.String())
	}

	wantByID := map[float64]string{
		10: `{"id":10,"capacity":50,"player_building_id":100,"items":[{"id":1,"item_id":7,"quantity":5},{"id":2,"item_id":8,"quantity":12}]}`,
		20: `{"id":20,"capacity":100,"player_building_id":200,"items":[{"id":3,"item_id":8,"quantity":30}]}`,
		30: `{"id":30,"capacity":25,"player_building_id":300,"items":[]}`,
	}
	for _, inventory := range inventories {
		id, ok := inventory["id"].(float64)
		if !ok {
			t.Fatalf("invalid inventory ID: %v", inventory["id"])
		}
		wantJSON, exists := wantByID[id]
		if !exists {
			t.Fatalf("unexpected or duplicate inventory: %v", inventory)
		}
		var want map[string]any
		if err := json.Unmarshal([]byte(wantJSON), &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(inventory, want) {
			t.Errorf("inventory %v = %v, want %v", id, inventory, want)
		}
		delete(wantByID, id)
	}
}

func TestGetPlayerInventoriesReturnsEmptyArray(t *testing.T) {
	db := newPlayerInventoriesTestDB(t)
	if err := db.Exec(`INSERT INTO player_inventories (id, player_id, map_id, capacity)
		VALUES (10, 1, 10, 100)`).Error; err != nil {
		t.Fatal(err)
	}
	response := requestPlayerInventories("1", "9")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"player_inventory": map[string]any{
		"player_inventories": []any{},
		"total_quantity":     float64(0),
		"total_capacity":     float64(0),
	}}
	if !reflect.DeepEqual(payload, want) {
		t.Fatalf("response = %s, want an empty player_inventories array", response.Body.String())
	}
}

func TestGetPlayerInventoriesRejectsInvalidPlayerID(t *testing.T) {
	for _, playerID := range []string{"abc", "-1", "1.5", "4294967296"} {
		t.Run(playerID, func(t *testing.T) {
			response := requestPlayerInventories(playerID, "9")
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", response.Code, response.Body.String())
			}
			var payload map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload["error"] != "invalid player_id" {
				t.Fatalf("error = %q, want invalid player_id", payload["error"])
			}
		})
	}
}

func TestGetPlayerInventoriesRejectsMissingMapID(t *testing.T) {
	response := requestPlayerInventories("1", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", response.Code, response.Body.String())
	}
}

func TestGetPlayerInventoriesRejectsInvalidMapID(t *testing.T) {
	for _, mapID := range []string{"abc", "-1", "1.5", "4294967296"} {
		t.Run(mapID, func(t *testing.T) {
			response := requestPlayerInventories("1", mapID)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", response.Code, response.Body.String())
			}
			var payload map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload["error"] != "invalid map_id" {
				t.Fatalf("error = %q, want invalid map_id", payload["error"])
			}
		})
	}
}

func TestGetPlayerInventoriesReturnsDatabaseError(t *testing.T) {
	db := newPlayerInventoriesTestDB(t)
	if err := db.Exec("DROP TABLE player_inventories").Error; err != nil {
		t.Fatal(err)
	}
	response := requestPlayerInventories("1", "9")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", response.Code, response.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] == "" {
		t.Fatalf("missing error message: %s", response.Body.String())
	}
}
