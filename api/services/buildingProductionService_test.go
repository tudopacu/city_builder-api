package services

import (
	"API/database"
	"API/models"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newCollectionTestDB(t *testing.T) *gorm.DB {
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

	// Define only the tables used by collection; the production MySQL ENUM and
	// unrelated model associations are not needed by this temporary SQLite DB.
	for _, statement := range []string{
		`CREATE TABLE building_productions (
			id INTEGER PRIMARY KEY, building_id INTEGER, item_id INTEGER,
			production_time_seconds INTEGER, quantity INTEGER,
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE player_building_productions (
			id INTEGER PRIMARY KEY, player_id INTEGER, player_building_id INTEGER,
			building_production_id INTEGER, end_time DATETIME, status TEXT,
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE player_inventories (
			id INTEGER PRIMARY KEY, player_id INTEGER, player_building_id INTEGER,
			capacity INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE player_inventory_items (
			id INTEGER PRIMARY KEY, player_inventory_id INTEGER, item_id INTEGER,
			quantity INTEGER, created_at DATETIME, updated_at DATETIME,
			UNIQUE (player_inventory_id, item_id))`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func seedCollection(t *testing.T, db *gorm.DB, quantity int, status string, endTime time.Time,
	inventories []models.PlayerInventory, items []models.PlayerInventoryItem) {
	t.Helper()
	production := models.BuildingProduction{ID: 30, ItemID: 7, Quantity: quantity}
	entry := models.PlayerBuildingProduction{
		ID: 100, PlayerID: 1, PlayerBuildingID: 10, BuildingProductionID: 30,
		EndTime: endTime, Status: status,
	}
	for _, value := range []any{&production, &entry} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	if len(inventories) > 0 {
		if err := db.Create(&inventories).Error; err != nil {
			t.Fatal(err)
		}
	}
	if len(items) > 0 {
		if err := db.Create(&items).Error; err != nil {
			t.Fatal(err)
		}
	}
}

type collectionSnapshot struct {
	Productions []models.PlayerBuildingProduction
	Inventories []models.PlayerInventory
	Items       []models.PlayerInventoryItem
}

func snapshotCollection(t *testing.T, db *gorm.DB) collectionSnapshot {
	t.Helper()
	var snapshot collectionSnapshot
	for _, value := range []any{&snapshot.Productions, &snapshot.Inventories, &snapshot.Items} {
		if err := db.Order("id ASC").Find(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	return snapshot
}

func TestCollectProductionFillsInventoriesInOrder(t *testing.T) {
	db := newCollectionTestDB(t)
	seedCollection(t, db, 10, "DONE", time.Now().Add(-time.Minute), []models.PlayerInventory{
		{ID: 40, PlayerID: 1, Capacity: 10},
		{ID: 10, PlayerID: 1, Capacity: 10},
		{ID: 30, PlayerID: 1, Capacity: 10},
		{ID: 20, PlayerID: 1, Capacity: 5},
		{ID: 50, PlayerID: 1, Capacity: 10},
		{ID: 1, PlayerID: 2, Capacity: 100},
	}, []models.PlayerInventoryItem{
		{PlayerInventoryID: 10, ItemID: 7, Quantity: 8},
		{PlayerInventoryID: 10, ItemID: 8, Quantity: 2},
		{PlayerInventoryID: 20, ItemID: 8, Quantity: 3},
		{PlayerInventoryID: 30, ItemID: 7, Quantity: 4},
		{PlayerInventoryID: 30, ItemID: 8, Quantity: 3},
		{PlayerInventoryID: 40, ItemID: 8, Quantity: 3},
		{PlayerInventoryID: 1, ItemID: 7, Quantity: 1},
	})

	// Reject the status update unless all output has already been stored.
	if err := db.Exec(`CREATE TRIGGER require_stored_items BEFORE UPDATE OF status ON player_building_productions
		WHEN NEW.status = 'COLLECTED' AND
			(SELECT SUM(quantity) FROM player_inventory_items WHERE item_id = 7 AND player_inventory_id IN (10, 20, 30, 40, 50)) <> 22
		BEGIN SELECT RAISE(ABORT, 'items must be stored first'); END`).Error; err != nil {
		t.Fatal(err)
	}

	status, err := CollectProduction(1, 10, 30)
	if err != nil || status != http.StatusOK {
		t.Fatalf("collect: status=%d, err=%v", status, err)
	}
	after := snapshotCollection(t, db)
	if after.Productions[0].Status != "COLLECTED" {
		t.Fatalf("production status = %s, want COLLECTED", after.Productions[0].Status)
	}
	want := map[uint]map[uint]int{
		1:  {7: 1},
		10: {7: 8, 8: 2},
		20: {7: 2, 8: 3},
		30: {7: 7, 8: 3},
		40: {7: 5, 8: 3},
	}
	got := make(map[uint]map[uint]int)
	for _, item := range after.Items {
		if got[item.PlayerInventoryID] == nil {
			got[item.PlayerInventoryID] = make(map[uint]int)
		}
		got[item.PlayerInventoryID][item.ItemID] = item.Quantity
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("inventory contents = %v, want %v", got, want)
	}

	status, err = CollectProduction(1, 10, 30)
	if status != http.StatusNotFound || err == nil {
		t.Fatalf("repeated collection: status=%d, err=%v", status, err)
	}
	if !reflect.DeepEqual(snapshotCollection(t, db), after) {
		t.Fatal("repeated collection changed the database")
	}
}

func TestCollectProductionFitsOneInventory(t *testing.T) {
	for _, status := range []string{"PENDING", "DONE"} {
		t.Run(status, func(t *testing.T) {
			db := newCollectionTestDB(t)
			seedCollection(t, db, 5, status, time.Now().Add(-time.Minute), []models.PlayerInventory{
				{ID: 1, PlayerID: 1, Capacity: 5},
			}, nil)
			code, err := CollectProduction(1, 10, 30)
			if err != nil || code != http.StatusOK {
				t.Fatalf("collect: status=%d, err=%v", code, err)
			}
			after := snapshotCollection(t, db)
			if len(after.Items) != 1 || after.Items[0].Quantity != 5 || after.Items[0].ItemID != 7 {
				t.Fatalf("unexpected stored items: %+v", after.Items)
			}
			if after.Productions[0].Status != "COLLECTED" {
				t.Fatalf("production status = %s, want COLLECTED", after.Productions[0].Status)
			}
		})
	}
}

func TestCollectProductionRejectsWithoutWrites(t *testing.T) {
	for _, tc := range []struct {
		name        string
		status      string
		unfinished  bool
		wrongPlayer bool
		inventories []models.PlayerInventory
		items       []models.PlayerInventoryItem
		wantCode    int
		wantError   string
	}{
		{name: "no inventories", status: "DONE", wantCode: http.StatusBadRequest, wantError: "not enough capacity"},
		{
			name: "all inventories full", status: "DONE", wantCode: http.StatusBadRequest, wantError: "not enough capacity",
			inventories: []models.PlayerInventory{{ID: 1, PlayerID: 1, Capacity: 5}},
			items:       []models.PlayerInventoryItem{{PlayerInventoryID: 1, ItemID: 8, Quantity: 5}},
		},
		{
			name: "insufficient combined capacity", status: "DONE", wantCode: http.StatusBadRequest,
			wantError: "not enough capacity: available 4, requested 5",
			inventories: []models.PlayerInventory{
				{ID: 1, PlayerID: 1, Capacity: 3}, {ID: 2, PlayerID: 1, Capacity: 1},
				{ID: 3, PlayerID: 2, Capacity: 100},
			},
		},
		{name: "unfinished", status: "PENDING", unfinished: true, wantCode: http.StatusBadRequest, wantError: "production is not finished yet"},
		{name: "already collected", status: "COLLECTED", wantCode: http.StatusNotFound, wantError: "production not found"},
		{name: "wrong player", status: "DONE", wrongPlayer: true, wantCode: http.StatusNotFound, wantError: "production not found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newCollectionTestDB(t)
			endTime := time.Now().Add(-time.Minute)
			if tc.unfinished {
				endTime = time.Now().Add(time.Hour)
			}
			seedCollection(t, db, 5, tc.status, endTime, tc.inventories, tc.items)
			before := snapshotCollection(t, db)

			// Any attempted write must fail with 500, so the expected 400/404
			// response also verifies that validation happens before any writes.
			for _, statement := range []string{
				`CREATE TRIGGER reject_insert BEFORE INSERT ON player_inventory_items BEGIN SELECT RAISE(ABORT, 'unexpected insert'); END`,
				`CREATE TRIGGER reject_update BEFORE UPDATE ON player_inventory_items BEGIN SELECT RAISE(ABORT, 'unexpected update'); END`,
				`CREATE TRIGGER reject_status BEFORE UPDATE ON player_building_productions BEGIN SELECT RAISE(ABORT, 'unexpected status update'); END`,
			} {
				if err := db.Exec(statement).Error; err != nil {
					t.Fatal(err)
				}
			}
			playerID := uint(1)
			if tc.wrongPlayer {
				playerID = 2
			}
			code, err := CollectProduction(playerID, 10, 30)
			if code != tc.wantCode || err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("collect: status=%d, err=%v; want status=%d, error=%q", code, err, tc.wantCode, tc.wantError)
			}
			if !reflect.DeepEqual(snapshotCollection(t, db), before) {
				t.Fatal("rejected collection changed the database")
			}
		})
	}
}

func TestCollectProductionRollsBackOnWriteFailure(t *testing.T) {
	for _, tc := range []struct {
		name             string
		secondItemExists bool
		trigger          string
	}{
		{
			name: "insert into second inventory fails",
			trigger: `CREATE TRIGGER fail_insert BEFORE INSERT ON player_inventory_items
				WHEN NEW.player_inventory_id = 20 BEGIN SELECT RAISE(ABORT, 'insert failed'); END`,
		},
		{
			name: "update second inventory fails", secondItemExists: true,
			trigger: `CREATE TRIGGER fail_update BEFORE UPDATE ON player_inventory_items
				WHEN NEW.player_inventory_id = 20 BEGIN SELECT RAISE(ABORT, 'update failed'); END`,
		},
		{
			name: "status update fails",
			trigger: `CREATE TRIGGER fail_status BEFORE UPDATE OF status ON player_building_productions
				BEGIN SELECT RAISE(ABORT, 'status update failed'); END`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newCollectionTestDB(t)
			items := []models.PlayerInventoryItem{{PlayerInventoryID: 10, ItemID: 7, Quantity: 3}}
			if tc.secondItemExists {
				items = append(items, models.PlayerInventoryItem{PlayerInventoryID: 20, ItemID: 7, Quantity: 1})
			}
			seedCollection(t, db, 5, "DONE", time.Now().Add(-time.Minute), []models.PlayerInventory{
				{ID: 10, PlayerID: 1, Capacity: 5}, {ID: 20, PlayerID: 1, Capacity: 5},
			}, items)
			before := snapshotCollection(t, db)
			if err := db.Exec(tc.trigger).Error; err != nil {
				t.Fatal(err)
			}
			code, err := CollectProduction(1, 10, 30)
			if code != http.StatusInternalServerError || err == nil {
				t.Fatalf("collect: status=%d, err=%v; want 500", code, err)
			}
			if !reflect.DeepEqual(snapshotCollection(t, db), before) {
				t.Fatal("failed collection did not roll back all changes")
			}
		})
	}
}
