package services

import (
	"API/api/dto"
	"reflect"
	"testing"

	"gorm.io/gorm"
)

func seedRoadsServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newServiceTestDB(t,
		`CREATE TABLE roads (
			id INTEGER PRIMARY KEY, start_intersection_id INTEGER, end_intersection_id INTEGER,
			road_type_id INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE intersections (id INTEGER PRIMARY KEY, player_id INTEGER, map_id INTEGER, x INTEGER, y INTEGER)`,
		`CREATE TABLE road_types (id INTEGER PRIMARY KEY, type TEXT, image_url TEXT)`,
		`INSERT INTO road_types (id, type, image_url) VALUES (3, 'Stone', '/images/stone-road.png'), (4, 'Dirt', NULL)`,
		`INSERT INTO intersections (id, player_id, map_id, x, y) VALUES
			(11, 2, 9, 4, 5), (12, 2, 9, 7, 8), (13, 2, 9, -1, 0),
			(21, 3, 9, 1, 2), (22, 3, 9, 3, 4),
			(31, 2, 10, 2, 3), (32, 2, 10, 4, 5)`,
		`INSERT INTO roads (id, start_intersection_id, end_intersection_id, road_type_id) VALUES
			(5, 11, 12, 3), (6, 21, 22, 3), (7, 31, 32, 3), (8, 12, 13, 4), (9, NULL, 13, 4)`,
	)
}

func TestGetRoadsByPlayerAndMapReturnsScopedDetails(t *testing.T) {
	seedRoadsServiceDB(t)
	imageURL := "/images/stone-road.png"
	want := []dto.Road{
		{
			ID: 5, StartIntersection: dto.Intersection{ID: 11, X: 4, Y: 5}, EndIntersection: dto.Intersection{ID: 12, X: 7, Y: 8},
			RoadType: dto.RoadType{ID: 3, Type: "Stone", ImageURL: &imageURL},
		},
		{
			ID: 8, StartIntersection: dto.Intersection{ID: 12, X: 7, Y: 8}, EndIntersection: dto.Intersection{ID: 13, X: -1, Y: 0},
			RoadType: dto.RoadType{ID: 4, Type: "Dirt"},
		},
	}

	got, err := GetRoadsByPlayerAndMap(2, 9)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("roads = %#v, error = %v, want %#v", got, err, want)
	}
}

func TestGetRoadsByPlayerAndMapReturnsEmptyNonNilSlice(t *testing.T) {
	seedRoadsServiceDB(t)
	for _, test := range []struct {
		name     string
		playerID uint
		mapID    uint
	}{
		{"unknown player", 99, 9},
		{"unknown map", 2, 99},
		{"player and map combination", 3, 10},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := GetRoadsByPlayerAndMap(test.playerID, test.mapID)
			if err != nil || got == nil || len(got) != 0 {
				t.Fatalf("roads = %#v, error = %v, want a non-nil empty slice", got, err)
			}
		})
	}
}

func TestGetRoadsByPlayerAndMapReturnsDatabaseErrors(t *testing.T) {
	for _, table := range []string{"roads", "intersections", "road_types"} {
		t.Run(table, func(t *testing.T) {
			db := seedRoadsServiceDB(t)
			execServiceTestSQL(t, db, "DROP TABLE "+table)

			got, err := GetRoadsByPlayerAndMap(2, 9)
			if err == nil || err.Error() != "failed to fetch roads for player_id 2 on map_id 9" || got != nil {
				t.Fatalf("roads = %#v, error = %v, want player/map-specific fetch error", got, err)
			}
		})
	}
}
