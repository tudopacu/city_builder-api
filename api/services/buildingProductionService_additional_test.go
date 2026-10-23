package services

import (
	"API/models"
	"net/http"
	"reflect"
	"testing"
	"time"

	"gorm.io/gorm"
)

func newStartProductionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newCollectionTestDB(t)
	execBuildingTestSQL(t, db,
		`CREATE TABLE player_buildings (id INTEGER PRIMARY KEY, player_id INTEGER, building_id INTEGER, map_id INTEGER,
			building_level_id INTEGER, x INTEGER, y INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT, type TEXT)`,
		`INSERT INTO player_buildings (id, player_id, building_id) VALUES (10, 1, 20)`,
		`INSERT INTO items (id, name, type) VALUES (7, 'Wood', 'resource')`,
		`INSERT INTO building_productions (id, building_id, item_id, production_time_seconds, quantity) VALUES (30, 20, 7, 60, 5)`,
	)
	return db
}

func TestStartProductionPersistsPendingProductionAndReturnsDetails(t *testing.T) {
	for _, history := range []string{"none", "collected", "different building"} {
		t.Run(history, func(t *testing.T) {
			db := newStartProductionTestDB(t)
			if history != "none" {
				previous := models.PlayerBuildingProduction{
					ID: 100, PlayerID: 1, PlayerBuildingID: 10, BuildingProductionID: 30,
					EndTime: time.Now().Add(-time.Hour), Status: "COLLECTED",
				}
				if history == "different building" {
					previous.PlayerBuildingID, previous.Status = 11, "PENDING"
				}
				if err := db.Create(&previous).Error; err != nil {
					t.Fatal(err)
				}
			}
			before := time.Now()
			status, response, err := StartProduction(1, 10, 30)
			after := time.Now()
			if status != http.StatusCreated || err != nil || response == nil || response.BuildingCurrentProduction == nil {
				t.Fatalf("status = %d, response = %+v, err = %v", status, response, err)
			}
			production := response.BuildingCurrentProduction
			if production.ID == 0 || production.Status != "PENDING" || production.BuildingProductionID != 30 ||
				production.ItemName != "Wood" || production.Quantity != 5 {
				t.Fatalf("production response = %+v", production)
			}
			if response.CurrentTimestamp.Before(before) || response.CurrentTimestamp.After(after) ||
				production.EndTime.Sub(response.CurrentTimestamp) != time.Minute {
				t.Fatalf("incorrect production timing: %+v", response)
			}
			var saved models.PlayerBuildingProduction
			if err := db.First(&saved, production.ID).Error; err != nil {
				t.Fatal(err)
			}
			if saved.PlayerID != 1 || saved.PlayerBuildingID != 10 || saved.BuildingProductionID != 30 ||
				saved.Status != "PENDING" || !saved.EndTime.Equal(production.EndTime) {
				t.Fatalf("saved production = %+v", saved)
			}
			var count int64
			if err := db.Model(&models.PlayerBuildingProduction{}).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			wantCount := int64(1)
			if history != "none" {
				wantCount = 2
			}
			if count != wantCount {
				t.Fatalf("production count = %d, want %d", count, wantCount)
			}
		})
	}
}

func TestStartProductionRejectsMissingMismatchedAndActiveProductions(t *testing.T) {
	for _, tc := range []struct {
		name, setup, message string
		status               int
	}{
		{"missing player building", "DELETE FROM player_buildings", "player building not found", http.StatusNotFound},
		{"missing recipe", "DELETE FROM building_productions", "building production not found", http.StatusNotFound},
		{"recipe belongs to different building", "UPDATE building_productions SET building_id = 21", "building production does not belong to this building", http.StatusBadRequest},
		{"pending", `INSERT INTO player_building_productions (player_id, player_building_id, building_production_id, status)
			VALUES (1, 10, 30, 'PENDING')`, "production already in progress", http.StatusConflict},
		{"done but uncollected", `INSERT INTO player_building_productions (player_id, player_building_id, building_production_id, status)
			VALUES (1, 10, 30, 'DONE')`, "production already in progress", http.StatusConflict},
		{"insert failure", `CREATE TRIGGER reject_production BEFORE INSERT ON player_building_productions
			BEGIN SELECT RAISE(ABORT, 'production insert failed'); END`, "failed to start production", http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newStartProductionTestDB(t)
			execBuildingTestSQL(t, db, tc.setup)
			before := snapshotCollection(t, db)
			status, response, err := StartProduction(1, 10, 30)
			if status != tc.status || response != nil || err == nil || err.Error() != tc.message {
				t.Fatalf("status = %d, response = %+v, err = %v; want %d, %q", status, response, err, tc.status, tc.message)
			}
			if !reflect.DeepEqual(snapshotCollection(t, db), before) {
				t.Fatal("rejected start changed production or inventory")
			}
		})
	}
}

func TestCollectProductionRejectsInvalidRecipeQuantity(t *testing.T) {
	for _, quantity := range []int{0, -5} {
		t.Run(map[int]string{0: "zero", -5: "negative"}[quantity], func(t *testing.T) {
			db := newCollectionTestDB(t)
			seedCollection(t, db, quantity, "DONE", time.Now().Add(-time.Hour),
				[]models.PlayerInventory{{ID: 10, PlayerID: 1, Capacity: 100}}, nil)
			// GORM applies the model's default of 1 to zero quantities on insert.
			if err := db.Model(&models.BuildingProduction{}).Where("id = ?", 30).Update("quantity", quantity).Error; err != nil {
				t.Fatal(err)
			}
			before := snapshotCollection(t, db)
			status, err := CollectProduction(1, 10, 30)
			if status != http.StatusInternalServerError || err == nil || err.Error() != "failed to collect production" {
				t.Fatalf("status = %d, err = %v", status, err)
			}
			if !reflect.DeepEqual(snapshotCollection(t, db), before) {
				t.Fatal("invalid recipe quantity changed production or inventory")
			}
		})
	}
}

func TestCollectProductionReportsReadFailuresWithoutCompletingProduction(t *testing.T) {
	for _, table := range []string{"building_productions", "player_inventories", "player_inventory_items"} {
		t.Run(table, func(t *testing.T) {
			db := newCollectionTestDB(t)
			seedCollection(t, db, 5, "DONE", time.Now().Add(-time.Hour),
				[]models.PlayerInventory{{ID: 10, PlayerID: 1, Capacity: 100}}, nil)
			execBuildingTestSQL(t, db, "DROP TABLE "+table)
			status, err := CollectProduction(1, 10, 30)
			if status != http.StatusInternalServerError || err == nil || err.Error() != "failed to collect production" {
				t.Fatalf("status = %d, err = %v", status, err)
			}
			var entry models.PlayerBuildingProduction
			if err := db.First(&entry, 100).Error; err != nil {
				t.Fatal(err)
			}
			if entry.Status != "DONE" {
				t.Fatalf("failed read changed production status to %s", entry.Status)
			}
		})
	}
}

func seedProductionCleanup(t *testing.T, db *gorm.DB) {
	t.Helper()
	execBuildingTestSQL(t, db,
		`INSERT INTO player_building_productions (id, player_id, player_building_id, building_production_id, status)
			VALUES (1, 1, 10, 30, 'COLLECTED'), (2, 1, 10, 30, 'PENDING'),
			(3, 1, 20, 30, 'DONE'), (4, 2, 30, 40, 'COLLECTED')`,
	)
}

func TestDeleteCollectedProductionsRemovesOnlyCollectedEntries(t *testing.T) {
	db := newCollectionTestDB(t)
	seedProductionCleanup(t, db)
	DeleteCollectedProductions()
	after := snapshotCollection(t, db)
	if len(after.Productions) != 2 || after.Productions[0].ID != 2 || after.Productions[0].Status != "PENDING" ||
		after.Productions[1].ID != 3 || after.Productions[1].Status != "DONE" {
		t.Fatalf("remaining productions = %+v", after.Productions)
	}
	DeleteCollectedProductions()
	if !reflect.DeepEqual(snapshotCollection(t, db), after) {
		t.Fatal("cleanup without collected productions changed pending or done entries")
	}
}

func TestDeleteCollectedProductionsPreservesEntriesOnDeleteFailure(t *testing.T) {
	db := newCollectionTestDB(t)
	seedProductionCleanup(t, db)
	execBuildingTestSQL(t, db, `CREATE TRIGGER reject_cleanup BEFORE DELETE ON player_building_productions
		WHEN OLD.id = 4 BEGIN SELECT RAISE(ABORT, 'cleanup failed'); END`)
	before := snapshotCollection(t, db)
	DeleteCollectedProductions()
	if !reflect.DeepEqual(snapshotCollection(t, db), before) {
		t.Fatal("failed cleanup partially deleted production entries")
	}
}
