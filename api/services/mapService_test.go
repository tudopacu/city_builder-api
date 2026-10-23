package services

import (
	"API/api/dto"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"gorm.io/gorm"
)

func newMapServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newServiceTestDB(t,
		`CREATE TABLE maps (id INTEGER PRIMARY KEY, name TEXT, image_url TEXT, width INTEGER, length INTEGER)`,
		`CREATE TABLE tiles (id INTEGER PRIMARY KEY, type TEXT, image_url TEXT)`,
		`CREATE TABLE terrains (id INTEGER PRIMARY KEY, map_id INTEGER, tile_id INTEGER,
			x INTEGER, y INTEGER, walkable BOOLEAN, set_x INTEGER, set_y INTEGER)`,
		`INSERT INTO maps (id, name, image_url, width, length) VALUES
			(3, 'Valley', '/valley.png', 10, 20), (9, 'Island', NULL, 30, 40)`,
		`INSERT INTO tiles (id, type, image_url) VALUES (7, 'grass', '/grass.png'), (8, 'water', NULL)`,
		`INSERT INTO terrains (id, map_id, tile_id, x, y, walkable, set_x, set_y) VALUES
			(1, 3, 7, 2, 4, 1, 5, 6), (2, 9, 8, 0, 1, 0, 2, 3)`,
	)
}

type mapServiceTestCase struct {
	name, cacheKey, errorMessage string
	load                         func() (any, error)
	want                         any
}

func mapServiceTestCases() []mapServiceTestCase {
	mapImage, tileImage := "/valley.png", "/grass.png"
	maps := []dto.Map{
		{ID: 3, Name: "Valley", ImageURL: &mapImage, Width: 10, Length: 20,
			Terrains: []dto.Terrain{{TileId: 7, X: 2, Y: 4, Type: "grass", Walkable: true, SetX: 5, SetY: 6}}},
		{ID: 9, Name: "Island", Width: 30, Length: 40,
			Terrains: []dto.Terrain{{TileId: 8, X: 0, Y: 1, Type: "water", Walkable: false, SetX: 2, SetY: 3}}},
	}
	return []mapServiceTestCase{
		{"all maps", "maps:all", "failed to fetch maps", func() (any, error) { return GetAllMaps() }, maps},
		{"one map", "map:9", "failed to fetch map, map_id 9", func() (any, error) { return GetMapByID(9) }, &maps[1]},
		{"tiles", "tiles:all", "failed to fetch tiles", func() (any, error) { return GetTiles() },
			[]dto.Tile{{ID: 7, Type: "grass", ImageURL: &tileImage}, {ID: 8, Type: "water"}}},
	}
}

func TestMapServicesLoadAndCacheDatabaseResults(t *testing.T) {
	for _, tc := range mapServiceTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			cache := newServiceTestCache(t)
			db := newMapServiceTestDB(t)
			got, err := tc.load()
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("result = %+v, error = %v; want %+v", got, err, tc.want)
			}
			if !reflect.DeepEqual(cache.gets, []string{tc.cacheKey}) || !reflect.DeepEqual(cache.sets, []string{tc.cacheKey}) {
				t.Fatalf("cache gets = %v, sets = %v; want %q", cache.gets, cache.sets, tc.cacheKey)
			}
			if cache.ttls[tc.cacheKey] != 10*time.Minute {
				t.Fatalf("cache TTL = %s, want 10m", cache.ttls[tc.cacheKey])
			}
			// A second call must work without the tables, proving it reads the
			// cached DTO and does not query the database again.
			execServiceTestSQL(t, db, `DROP TABLE maps`, `DROP TABLE tiles`, `DROP TABLE terrains`)
			cached, err := tc.load()
			if err != nil || !reflect.DeepEqual(cached, tc.want) {
				t.Fatalf("cached result = %+v, error = %v; want %+v", cached, err, tc.want)
			}
			if len(cache.sets) != 1 {
				t.Fatalf("cache hit unexpectedly wrote cache: %v", cache.sets)
			}
		})
	}
}

func TestMapServicesFallBackOnCacheProblems(t *testing.T) {
	for _, tc := range mapServiceTestCases() {
		for _, failure := range []string{"corrupt entry", "read failure", "write failure"} {
			t.Run(tc.name+"/"+failure, func(t *testing.T) {
				cache := newServiceTestCache(t)
				newMapServiceTestDB(t)
				switch failure {
				case "corrupt entry":
					cache.Set(tc.cacheKey, `{broken`)
				case "read failure":
					cache.getErr = errors.New("cache read unavailable")
				case "write failure":
					cache.setErr = errors.New("cache write unavailable")
				}
				got, err := tc.load()
				if err != nil || !reflect.DeepEqual(got, tc.want) {
					t.Fatalf("result = %+v, error = %v; want %+v", got, err, tc.want)
				}
				if !reflect.DeepEqual(cache.sets, []string{tc.cacheKey}) {
					t.Fatalf("cache writes = %v, want [%s]", cache.sets, tc.cacheKey)
				}
				if failure != "write failure" && !json.Valid([]byte(cache.values[tc.cacheKey])) {
					t.Fatalf("cache was not refreshed with JSON: %q", cache.values[tc.cacheKey])
				}
			})
		}
	}
}

func TestMapServicesReturnDatabaseErrorsWithoutCaching(t *testing.T) {
	for _, tc := range mapServiceTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			cache := newServiceTestCache(t)
			newServiceTestDB(t)
			got, err := tc.load()
			if err == nil || err.Error() != tc.errorMessage || !reflect.ValueOf(got).IsNil() {
				t.Fatalf("result = %+v, error = %v; want nil and %q", got, err, tc.errorMessage)
			}
			if len(cache.sets) != 0 {
				t.Fatalf("failed query wrote cache: %v", cache.sets)
			}
		})
	}
}

func TestMapServicesHandleEmptyResults(t *testing.T) {
	cache := newServiceTestCache(t)
	db := newMapServiceTestDB(t)
	execServiceTestSQL(t, db, `DELETE FROM terrains`, `DELETE FROM maps`, `DELETE FROM tiles`)
	maps, err := GetAllMaps()
	if err != nil || maps == nil || len(maps) != 0 {
		t.Fatalf("maps = %+v, error = %v; want empty slice", maps, err)
	}
	tiles, err := GetTiles()
	if err != nil || len(tiles) != 0 {
		t.Fatalf("tiles = %+v, error = %v; want no tiles", tiles, err)
	}
	m, err := GetMapByID(9)
	if m != nil || err == nil || err.Error() != "failed to fetch map, map_id 9" {
		t.Fatalf("map = %+v, error = %v; want missing map error", m, err)
	}
	if _, ok := cache.values["map:9"]; ok {
		t.Fatal("missing map was cached")
	}
}

func TestGetMapByIDKeepsCachedMapsSeparate(t *testing.T) {
	cache := newServiceTestCache(t)
	db := newMapServiceTestDB(t)
	first, err := GetMapByID(3)
	if err != nil {
		t.Fatal(err)
	}
	second, err := GetMapByID(9)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != 3 || second.ID != 9 || first.Name == second.Name {
		t.Fatalf("maps were mixed: first = %+v, second = %+v", first, second)
	}
	if !reflect.DeepEqual(cache.sets, []string{"map:3", "map:9"}) {
		t.Fatalf("cache writes = %v", cache.sets)
	}
	execServiceTestSQL(t, db, `DROP TABLE maps`)
	for _, want := range []*dto.Map{first, second} {
		got, err := GetMapByID(want.ID)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("cached map = %+v, error = %v; want %+v", got, err, want)
		}
	}
}
