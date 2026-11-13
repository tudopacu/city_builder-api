package controllers

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"gorm.io/gorm"
)

type controllerInventoryItemRow struct {
	ID                uint
	PlayerInventoryID uint
	ItemID            uint
	Quantity          int
}

func newAddInventoryItemControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newPlayerInventoriesTestDB(t)
	execControllerSQL(t, db,
		`ALTER TABLE items ADD COLUMN description TEXT`,
		`ALTER TABLE items ADD COLUMN icon_url TEXT`,
		`ALTER TABLE items ADD COLUMN is_tradeable BOOLEAN`,
		`ALTER TABLE items ADD COLUMN created_at DATETIME`,
		`ALTER TABLE items ADD COLUMN updated_at DATETIME`,
		`INSERT INTO player_inventories (id, player_id, map_id, player_building_id, capacity)
			VALUES (10, 1, 9, 0, 20), (20, 2, 9, 0, 100)`,
		`INSERT INTO items (id, name, type) VALUES
			(7, 'Wood', 'resource'), (8, 'Stone', 'resource'), (9, 'Iron', 'resource')`,
		`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity)
			VALUES (100, 10, 7, 5), (101, 10, 8, 4), (200, 20, 7, 12)`,
	)
	return db
}

func readControllerInventoryItems(t *testing.T, db *gorm.DB) []controllerInventoryItemRow {
	t.Helper()
	var rows []controllerInventoryItemRow
	if err := db.Table("player_inventory_items").Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestAddInventoryItemCreatesEntryAndReturnsUpdatedInventory(t *testing.T) {
	db := newAddInventoryItemControllerTestDB(t)
	response := performPlayerRequest(t, 1, http.MethodPost, "/game/add_inventory_item", "/game/add_inventory_item",
		`{"inventory_id":10,"item_id":9,"quantity":11}`, AddInventoryItem)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}

	rows := readControllerInventoryItems(t, db)
	if len(rows) != 4 {
		t.Fatalf("inventory item rows = %v, want four entries", rows)
	}
	newID := rows[3].ID
	if newID == 0 {
		t.Fatal("new inventory item has no ID")
	}
	want := []controllerInventoryItemRow{
		{100, 10, 7, 5}, {101, 10, 8, 4}, {200, 20, 7, 12}, {newID, 10, 9, 11},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("persisted inventory items = %v, want %v", rows, want)
	}
	assertControllerJSON(t, response, http.StatusOK, fmt.Sprintf(`{"player_inventory":{
		"id":10,"player_building_id":0,"capacity":20,"items":[
			{"id":100,"item_id":7,"quantity":5},
			{"id":101,"item_id":8,"quantity":4},
			{"id":%d,"item_id":9,"quantity":11}
		]}}`, newID))
}

func TestAddInventoryItemIncrementsExistingEntryUpToCapacity(t *testing.T) {
	db := newAddInventoryItemControllerTestDB(t)
	response := performPlayerRequest(t, 1, http.MethodPost, "/game/add_inventory_item", "/game/add_inventory_item",
		`{"inventory_id":10,"item_id":7,"quantity":11}`, AddInventoryItem)
	assertControllerJSON(t, response, http.StatusOK, `{"player_inventory":{
		"id":10,"player_building_id":0,"capacity":20,"items":[
			{"id":100,"item_id":7,"quantity":16},
			{"id":101,"item_id":8,"quantity":4}
		]}}`)
	want := []controllerInventoryItemRow{{100, 10, 7, 16}, {101, 10, 8, 4}, {200, 20, 7, 12}}
	if rows := readControllerInventoryItems(t, db); !reflect.DeepEqual(rows, want) {
		t.Fatalf("persisted inventory items = %v, want %v", rows, want)
	}
}

func TestAddInventoryItemRejectsInvalidRequests(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"malformed JSON", `{"inventory_id":`},
		{"array body", `[]`},
		{"null body", `null`},
		{"empty object", `{}`},
		{"missing inventory", `{"item_id":7,"quantity":1}`},
		{"missing item", `{"inventory_id":10,"quantity":1}`},
		{"missing quantity", `{"inventory_id":10,"item_id":7}`},
		{"zero inventory", `{"inventory_id":0,"item_id":7,"quantity":1}`},
		{"zero item", `{"inventory_id":10,"item_id":0,"quantity":1}`},
		{"zero quantity", `{"inventory_id":10,"item_id":7,"quantity":0}`},
		{"negative quantity", `{"inventory_id":10,"item_id":7,"quantity":-1}`},
		{"negative inventory", `{"inventory_id":-1,"item_id":7,"quantity":1}`},
		{"string inventory", `{"inventory_id":"10","item_id":7,"quantity":1}`},
		{"fractional item", `{"inventory_id":10,"item_id":7.5,"quantity":1}`},
		{"string quantity", `{"inventory_id":10,"item_id":7,"quantity":"1"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := performPlayerRequest(t, 1, http.MethodPost, "/game/add_inventory_item", "/game/add_inventory_item", test.body, AddInventoryItem)
			assertControllerError(t, response, http.StatusBadRequest, "invalid request")
		})
	}
}

func TestAddInventoryItemReturnsServiceErrorsWithoutChangingInventory(t *testing.T) {
	for _, test := range []struct {
		name    string
		body    string
		status  int
		message string
	}{
		{"missing inventory", `{"inventory_id":99,"item_id":7,"quantity":1}`,
			http.StatusNotFound, "inventory not found"},
		// Inventory 20 belongs to player 2, and these requests come from player 1.
		{"another player's inventory", `{"inventory_id":20,"item_id":7,"quantity":1}`,
			http.StatusForbidden, "inventory does not belong to player"},
		{"player_id in body is ignored", `{"player_id":2,"inventory_id":20,"item_id":7,"quantity":1}`,
			http.StatusForbidden, "inventory does not belong to player"},
		{"missing item", `{"inventory_id":10,"item_id":99,"quantity":1}`,
			http.StatusNotFound, "item not found"},
		{"existing item exceeds capacity", `{"inventory_id":10,"item_id":7,"quantity":12}`,
			http.StatusBadRequest, "not enough capacity: available 11, requested 12"},
		{"new item exceeds capacity", `{"inventory_id":10,"item_id":9,"quantity":12}`,
			http.StatusBadRequest, "not enough capacity: available 11, requested 12"},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := newAddInventoryItemControllerTestDB(t)
			before := readControllerInventoryItems(t, db)
			response := performPlayerRequest(t, 1, http.MethodPost, "/game/add_inventory_item", "/game/add_inventory_item", test.body, AddInventoryItem)
			assertControllerError(t, response, test.status, test.message)
			if after := readControllerInventoryItems(t, db); !reflect.DeepEqual(after, before) {
				t.Fatalf("inventory items changed on rejected request: before %v, after %v", before, after)
			}
		})
	}
}

func TestAddInventoryItemReturnsWriteErrorsWithoutChangingInventory(t *testing.T) {
	for _, test := range []struct {
		name      string
		operation string
		itemID    uint
		message   string
	}{
		{"create", "INSERT", 9, "failed to add inventory item"},
		{"update", "UPDATE", 7, "failed to update inventory item"},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := newAddInventoryItemControllerTestDB(t)
			before := readControllerInventoryItems(t, db)
			execControllerSQL(t, db, fmt.Sprintf(`CREATE TRIGGER reject_inventory_item_write
				BEFORE %s ON player_inventory_items
				BEGIN SELECT RAISE(ABORT, 'write rejected by test'); END`, test.operation))

			body := fmt.Sprintf(`{"inventory_id":10,"item_id":%d,"quantity":1}`, test.itemID)
			response := performPlayerRequest(t, 1, http.MethodPost, "/game/add_inventory_item", "/game/add_inventory_item", body, AddInventoryItem)
			assertControllerError(t, response, http.StatusInternalServerError, test.message)
			if after := readControllerInventoryItems(t, db); !reflect.DeepEqual(after, before) {
				t.Fatalf("inventory items changed after a failed write: before %v, after %v", before, after)
			}
		})
	}
}
