package services

import (
	"API/api/dto"
	"API/api/dto/requests"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"gorm.io/gorm"
)

func newPlayerInventoryServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newServiceTestDB(t,
		`CREATE TABLE player_inventories (id INTEGER PRIMARY KEY, player_id INTEGER,
			map_id INTEGER, player_building_id INTEGER, capacity INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE player_inventory_items (id INTEGER PRIMARY KEY, player_inventory_id INTEGER,
			item_id INTEGER, quantity INTEGER, created_at DATETIME, updated_at DATETIME,
			UNIQUE(player_inventory_id, item_id))`,
		`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT, type TEXT, description TEXT,
			icon_url TEXT, is_tradeable BOOLEAN, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE player_buildings (id INTEGER PRIMARY KEY, player_id INTEGER, map_id INTEGER,
			building_id INTEGER, building_level_id INTEGER, x INTEGER, y INTEGER)`,
		`CREATE TABLE buildings (id INTEGER PRIMARY KEY, name TEXT, building_category_id INTEGER)`,
		`CREATE TABLE building_categories (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE building_levels (id INTEGER PRIMARY KEY, building_id INTEGER, level INTEGER, capacity INTEGER)`,
	)
	execServiceTestSQL(t, db,
		`INSERT INTO building_categories (id, name) VALUES (4, 'Storage')`,
		`INSERT INTO buildings (id, name, building_category_id) VALUES (5, 'Warehouse', 4)`,
		`INSERT INTO building_levels (id, building_id, level, capacity) VALUES (6, 5, 2, 20)`,
		`INSERT INTO player_buildings (id, player_id, map_id, building_id, building_level_id)
			VALUES (100, 1, 9, 5, 6)`,
		`INSERT INTO player_inventories (id, player_id, map_id, player_building_id, capacity)
			VALUES (10, 1, 9, 100, 20), (20, 2, 9, 0, 100)`,
		`INSERT INTO items (id, name, type) VALUES (7, 'Wood', 'resource'), (8, 'Stone', 'resource'), (9, 'Iron', 'resource')`,
		`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity)
			VALUES (100, 10, 7, 5), (101, 10, 8, 4), (200, 20, 7, 12)`,
	)
	return db
}

type serviceInventoryItemRow struct {
	ID                uint
	PlayerInventoryID uint
	ItemID            uint
	Quantity          int
}

func readServiceInventoryItems(t *testing.T, db *gorm.DB) []serviceInventoryItemRow {
	t.Helper()
	var rows []serviceInventoryItemRow
	if err := db.Table("player_inventory_items").Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestGetPlayerInventoriesFiltersPlayerAndMapAndTotalsContents(t *testing.T) {
	db := newPlayerInventoryServiceTestDB(t)
	execServiceTestSQL(t, db,
		`INSERT INTO player_inventories (id, player_id, map_id, player_building_id, capacity)
			VALUES (30, 1, 9, 300, 50), (40, 1, 10, 400, 200)`,
		`INSERT INTO player_inventory_items (id, player_inventory_id, item_id, quantity)
			VALUES (400, 40, 9, 75)`,
	)
	inventories, quantity, capacity, err := GetPlayerInventories(1, 9)
	if err != nil {
		t.Fatal(err)
	}
	if quantity != 9 || capacity != 70 {
		t.Fatalf("totals = (%d, %d), want (9, 70)", quantity, capacity)
	}
	want := map[uint]dto.PlayerInventory{
		10: {ID: 10, PlayerBuildingID: 100, Capacity: 20, Items: []dto.PlayerInventoryItem{
			{ID: 100, ItemID: 7, Quantity: 5}, {ID: 101, ItemID: 8, Quantity: 4},
		}},
		30: {ID: 30, PlayerBuildingID: 300, Capacity: 50, Items: []dto.PlayerInventoryItem{}},
	}
	if len(inventories) != len(want) {
		t.Fatalf("inventories = %+v, want only player 1 on map 9", inventories)
	}
	for _, inventory := range inventories {
		expected, ok := want[inventory.ID]
		if !ok || !reflect.DeepEqual(inventory, expected) {
			t.Fatalf("unexpected inventory: %+v", inventory)
		}
		delete(want, inventory.ID)
	}
}

func TestGetPlayerInventoriesReturnsEmptyCollectionAndZeroTotals(t *testing.T) {
	newPlayerInventoryServiceTestDB(t)
	inventories, quantity, capacity, err := GetPlayerInventories(99, 9)
	if err != nil || inventories == nil || len(inventories) != 0 || quantity != 0 || capacity != 0 {
		t.Fatalf("GetPlayerInventories = (%+v, %d, %d, %v), want empty collection and zero totals", inventories, quantity, capacity, err)
	}
}

func TestGetPlayerInventoriesReturnsQueryAndPreloadErrorsWithoutPartialResults(t *testing.T) {
	for _, table := range []string{"player_inventories", "player_inventory_items", "items"} {
		t.Run(table, func(t *testing.T) {
			db := newPlayerInventoryServiceTestDB(t)
			execServiceTestSQL(t, db, "DROP TABLE "+table)
			inventories, quantity, capacity, err := GetPlayerInventories(1, 9)
			if err == nil || err.Error() != "failed to fetch player inventories for player_id 1" {
				t.Fatalf("error = %v, want player inventory fetch failure", err)
			}
			if inventories == nil || len(inventories) != 0 || quantity != 0 || capacity != 0 {
				t.Fatalf("failure returned partial results: (%+v, %d, %d)", inventories, quantity, capacity)
			}
		})
	}
}

func TestLoadInventoryWithAssociationsLoadsBuildingAndItemDetails(t *testing.T) {
	newPlayerInventoryServiceTestDB(t)
	inventory, err := loadInventoryWithAssociations(10)
	if err != nil {
		t.Fatal(err)
	}
	if inventory.ID != 10 || inventory.PlayerID != 1 || inventory.PlayerBuilding.ID != 100 ||
		inventory.PlayerBuilding.Building.Name != "Warehouse" || inventory.PlayerBuilding.Building.Category.Name != "Storage" ||
		inventory.PlayerBuilding.BuildingLevel.Level != 2 {
		t.Fatalf("inventory building associations not loaded: %+v", inventory)
	}
	if len(inventory.InventoryItems) != 2 {
		t.Fatalf("inventory items = %+v, want two entries", inventory.InventoryItems)
	}
	items := map[uint]string{7: "Wood", 8: "Stone"}
	for _, entry := range inventory.InventoryItems {
		if entry.Item.ID != entry.ItemID || entry.Item.Name != items[entry.ItemID] {
			t.Fatalf("item association not loaded: %+v", entry)
		}
	}
}

func TestAddInventoryItemCreatesOrIncrementsEntryUpToCapacity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		itemID uint
	}{
		{"new item", 9},
		{"existing item", 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newPlayerInventoryServiceTestDB(t)
			status, response := AddInventoryItem(1, requests.AddInventoryItemRequest{
				InventoryID: 10, ItemID: tc.itemID, Quantity: 11,
			})
			if status != http.StatusOK || response.Error != "" || response.PlayerInventory == nil {
				t.Fatalf("AddInventoryItem = (%d, %+v)", status, response)
			}
			wantRows := []serviceInventoryItemRow{{100, 10, 7, 5}, {101, 10, 8, 4}, {200, 20, 7, 12}}
			wantInventory := dto.PlayerInventory{ID: 10, PlayerBuildingID: 100, Capacity: 20,
				Items: []dto.PlayerInventoryItem{{ID: 100, ItemID: 7, Quantity: 5}, {ID: 101, ItemID: 8, Quantity: 4}}}
			if tc.itemID == 7 {
				wantRows[0].Quantity = 16
				wantInventory.Items[0].Quantity = 16
			} else {
				var added serviceInventoryItemRow
				if err := db.Table("player_inventory_items").Where("player_inventory_id = ? AND item_id = ?", 10, 9).First(&added).Error; err != nil {
					t.Fatal(err)
				}
				if added.ID == 0 {
					t.Fatal("new entry has no generated ID")
				}
				wantRows = append(wantRows, serviceInventoryItemRow{added.ID, 10, 9, 11})
				wantInventory.Items = append(wantInventory.Items, dto.PlayerInventoryItem{ID: added.ID, ItemID: 9, Quantity: 11})
			}
			if !reflect.DeepEqual(response.PlayerInventory, &wantInventory) {
				t.Fatalf("returned inventory = %+v, want %+v", response.PlayerInventory, wantInventory)
			}
			if rows := readServiceInventoryItems(t, db); !reflect.DeepEqual(rows, wantRows) {
				t.Fatalf("stored entries = %+v, want %+v", rows, wantRows)
			}
		})
	}
}

func TestAddInventoryItemRejectsMissingOwnershipAndCapacityViolationsWithoutWrites(t *testing.T) {
	for _, tc := range []struct {
		name     string
		playerID uint
		request  requests.AddInventoryItemRequest
		status   int
		error    string
	}{
		{"missing inventory", 1, requests.AddInventoryItemRequest{InventoryID: 99, ItemID: 7, Quantity: 1}, http.StatusNotFound, "inventory not found"},
		{"wrong owner", 2, requests.AddInventoryItemRequest{InventoryID: 10, ItemID: 7, Quantity: 1}, http.StatusForbidden, "inventory does not belong to player"},
		{"missing item", 1, requests.AddInventoryItemRequest{InventoryID: 10, ItemID: 99, Quantity: 1}, http.StatusNotFound, "item not found"},
		{"existing item over capacity", 1, requests.AddInventoryItemRequest{InventoryID: 10, ItemID: 7, Quantity: 12}, http.StatusBadRequest, "not enough capacity: available 11, requested 12"},
		{"new item over capacity", 1, requests.AddInventoryItemRequest{InventoryID: 10, ItemID: 9, Quantity: 12}, http.StatusBadRequest, "not enough capacity: available 11, requested 12"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newPlayerInventoryServiceTestDB(t)
			before := readServiceInventoryItems(t, db)
			status, response := AddInventoryItem(tc.playerID, tc.request)
			if status != tc.status || response.Error != tc.error || response.PlayerInventory != nil {
				t.Fatalf("AddInventoryItem = (%d, %+v), want (%d, %q)", status, response, tc.status, tc.error)
			}
			if after := readServiceInventoryItems(t, db); !reflect.DeepEqual(after, before) {
				t.Fatalf("rejected operation changed entries: before %+v, after %+v", before, after)
			}
		})
	}
}

func TestAddInventoryItemReturnsWriteErrorsWithoutChangingEntries(t *testing.T) {
	for _, tc := range []struct {
		name, operation string
		itemID          uint
		error           string
	}{
		{"create", "INSERT", 9, "failed to add inventory item"},
		{"update", "UPDATE", 7, "failed to update inventory item"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newPlayerInventoryServiceTestDB(t)
			before := readServiceInventoryItems(t, db)
			execServiceTestSQL(t, db, fmt.Sprintf(`CREATE TRIGGER reject_inventory_item_write
				BEFORE %s ON player_inventory_items
				BEGIN SELECT RAISE(ABORT, 'write rejected by test'); END`, tc.operation))
			status, response := AddInventoryItem(1, requests.AddInventoryItemRequest{
				InventoryID: 10, ItemID: tc.itemID, Quantity: 1,
			})
			if status != http.StatusInternalServerError || response.Error != tc.error || response.PlayerInventory != nil {
				t.Fatalf("AddInventoryItem = (%d, %+v), want write failure %q", status, response, tc.error)
			}
			if after := readServiceInventoryItems(t, db); !reflect.DeepEqual(after, before) {
				t.Fatalf("failed write changed entries: before %+v, after %+v", before, after)
			}
		})
	}
}

func TestAddInventoryItemReturnsReloadFailureWithoutSuccessPayload(t *testing.T) {
	db := newPlayerInventoryServiceTestDB(t)
	loads := 0
	if err := db.Callback().Query().Before("gorm:query").Register("test:fail_inventory_reload", func(tx *gorm.DB) {
		if tx.Statement.Table == "player_inventories" {
			loads++
			if loads == 2 {
				tx.AddError(errors.New("database read failed during reload"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	status, response := AddInventoryItem(1, requests.AddInventoryItemRequest{InventoryID: 10, ItemID: 9, Quantity: 1})
	if status != http.StatusInternalServerError || response.Error != "failed to reload inventory" || response.PlayerInventory != nil {
		t.Fatalf("AddInventoryItem = (%d, %+v), want reload failure", status, response)
	}
	if loads != 2 {
		t.Fatalf("inventory query count = %d, want initial load and reload", loads)
	}
}

func TestCalculateTotalInventoryStats(t *testing.T) {
	for _, tc := range []struct {
		name        string
		inventories []dto.PlayerInventory
		quantity    int
		capacity    int
	}{
		{"nil inventories", nil, 0, 0},
		{"empty inventories", []dto.PlayerInventory{}, 0, 0},
		{"empty inventory items", []dto.PlayerInventory{{Capacity: 25}, {Capacity: 75, Items: []dto.PlayerInventoryItem{}}}, 0, 100},
		{"several inventories and items", []dto.PlayerInventory{
			{Capacity: 20, Items: []dto.PlayerInventoryItem{{Quantity: 5}, {Quantity: 4}}},
			{Capacity: 100, Items: []dto.PlayerInventoryItem{{Quantity: 12}}},
			{Capacity: 50},
		}, 21, 170},
	} {
		t.Run(tc.name, func(t *testing.T) {
			quantity, capacity := CalculateTotalInventoryStats(tc.inventories)
			if quantity != tc.quantity || capacity != tc.capacity {
				t.Fatalf("totals = (%d, %d), want (%d, %d)", quantity, capacity, tc.quantity, tc.capacity)
			}
		})
	}
}
