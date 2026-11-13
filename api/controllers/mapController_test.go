package controllers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func newMapControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newControllerTestDB(t,
		`CREATE TABLE maps (id INTEGER PRIMARY KEY, name TEXT, image_url TEXT, width INTEGER, length INTEGER)`,
		`CREATE TABLE tiles (id INTEGER PRIMARY KEY, type TEXT, image_url TEXT)`,
		`CREATE TABLE terrains (id INTEGER PRIMARY KEY, map_id INTEGER, tile_id INTEGER,
			x INTEGER, y INTEGER, walkable BOOLEAN, set_x INTEGER, set_y INTEGER)`,
	)
}

func TestGetMapsReturnsMapsWithTerrain(t *testing.T) {
	newControllerTestCache(t)
	db := newMapControllerTestDB(t)
	execControllerSQL(t, db,
		`INSERT INTO maps (id, name, image_url, width, length) VALUES
			(3, 'Valley', '/valley.png', 10, 20), (8, 'Island', NULL, 30, 40)`,
		`INSERT INTO tiles (id, type) VALUES (7, 'grass')`,
		`INSERT INTO terrains (id, map_id, tile_id, x, y, walkable, set_x, set_y)
			VALUES (1, 3, 7, 2, 4, 1, 5, 6)`,
	)
	response := performControllerRequest(http.MethodGet, "/maps", "/maps", "", GetMaps)
	assertControllerJSON(t, response, http.StatusOK, `{"maps":[
		{"id":3,"name":"Valley","image_url":"/valley.png","width":10,"length":20,
		"terrains":[{"tile_id":7,"x":2,"y":4,"type":"grass","walkable":true,"set_x":5,"set_y":6}]},
		{"id":8,"name":"Island","width":30,"length":40,"terrains":[]}
	]}`)
}

func TestGetMapsReturnsEmptyArray(t *testing.T) {
	newControllerTestCache(t)
	newMapControllerTestDB(t)
	response := performControllerRequest(http.MethodGet, "/maps", "/maps", "", GetMaps)
	assertControllerJSON(t, response, http.StatusOK, `{"maps":[]}`)
}

func TestGetMapsReturnsServiceError(t *testing.T) {
	newControllerTestCache(t)
	newControllerTestDB(t)
	response := performControllerRequest(http.MethodGet, "/maps", "/maps", "", GetMaps)
	assertControllerError(t, response, http.StatusNotFound, "no maps found")
}

func TestGetMapReturnsRequestedMap(t *testing.T) {
	newControllerTestCache(t)
	db := newMapControllerTestDB(t)
	execControllerSQL(t, db,
		`INSERT INTO maps (id, name, width, length) VALUES (1, 'Other', 5, 5), (9, 'Island', 20, 30)`,
		`INSERT INTO tiles (id, type) VALUES (7, 'water'), (8, 'grass')`,
		`INSERT INTO terrains (id, map_id, tile_id, x, y, walkable, set_x, set_y) VALUES
			(1, 9, 7, 2, 3, 0, 4, 5), (2, 1, 8, 6, 7, 1, 0, 0)`,
	)
	response := performControllerRequest(http.MethodGet, "/game/map/:id", "/game/map/9", "", GetMap)
	assertControllerJSON(t, response, http.StatusOK, `{"map":{
		"id":9,"name":"Island","width":20,"length":30,
		"terrains":[{"tile_id":7,"x":2,"y":3,"type":"water","walkable":false,"set_x":4,"set_y":5}]
	}}`)
}

func TestGetMapRejectsInvalidID(t *testing.T) {
	for _, id := range []string{"abc", "-1", "1.5", "4294967296"} {
		t.Run(id, func(t *testing.T) {
			response := performControllerRequest(http.MethodGet, "/game/map/:id", "/game/map/"+id, "", GetMap)
			assertControllerError(t, response, http.StatusBadRequest, "invalid map_id")
		})
	}
}

func TestGetMapReturnsServiceErrors(t *testing.T) {
	for _, missingTable := range []bool{false, true} {
		name := "missing map"
		if missingTable {
			name = "database failure"
		}
		t.Run(name, func(t *testing.T) {
			newControllerTestCache(t)
			db := newMapControllerTestDB(t)
			if missingTable {
				execControllerSQL(t, db, `DROP TABLE maps`)
			}
			response := performControllerRequest(http.MethodGet, "/game/map/:id", "/game/map/99", "", GetMap)
			if response.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			// The controller currently serializes the error value directly.
			// Check the envelope without prescribing that representation.
			var payload map[string]json.RawMessage
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if len(payload) != 1 || payload["error"] == nil {
				t.Fatalf("unexpected error envelope: %s", response.Body.String())
			}
		})
	}
}

func TestGetTilesReturnsBareArray(t *testing.T) {
	newControllerTestCache(t)
	db := newMapControllerTestDB(t)
	execControllerSQL(t, db, `INSERT INTO tiles (id, type, image_url) VALUES
		(3, 'grass', '/grass.png'), (7, 'water', NULL)`)
	response := performControllerRequest(http.MethodGet, "/game/tiles", "/game/tiles", "", GetTiles)
	assertControllerJSON(t, response, http.StatusOK, `[
		{"id":3,"type":"grass","walkable":false,"image_url":"/grass.png"},
		{"id":7,"type":"water","walkable":false}
	]`)
}

func TestGetTilesReturnsServiceError(t *testing.T) {
	newControllerTestCache(t)
	newControllerTestDB(t)
	response := performControllerRequest(http.MethodGet, "/game/tiles", "/game/tiles", "", GetTiles)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 1 || payload["error"] == nil {
		t.Fatalf("unexpected error envelope: %s", response.Body.String())
	}
}

func TestMapControllersReturnCachedResponses(t *testing.T) {
	for _, tc := range []struct {
		name, key, value, route, target, want string
		handler                               gin.HandlerFunc
	}{
		{"maps", "v2:maps:all", `[{"id":9,"name":"Island","width":20,"length":30,"terrains":[]}]`,
			"/maps", "/maps", `{"maps":[{"id":9,"name":"Island","width":20,"length":30,"terrains":[]}]}`, GetMaps},
		{"map", "v2:map:9", `{"id":9,"name":"Island","width":20,"length":30,"terrains":[]}`,
			"/game/map/:id", "/game/map/9", `{"map":{"id":9,"name":"Island","width":20,"length":30,"terrains":[]}}`, GetMap},
		{"tiles", "v2:tiles:all", `[{"id":7,"type":"grass","walkable":true}]`,
			"/game/tiles", "/game/tiles", `[{"id":7,"type":"grass","walkable":true}]`, GetTiles},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cache := newControllerTestCache(t)
			cache.Set(tc.key, tc.value)
			newControllerTestDB(t)
			response := performControllerRequest(http.MethodGet, tc.route, tc.target, "", tc.handler)
			assertControllerJSON(t, response, http.StatusOK, tc.want)
		})
	}
}
