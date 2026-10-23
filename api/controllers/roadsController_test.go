package controllers

import (
	"net/http"
	"testing"

	"gorm.io/gorm"
)

func newRoadsControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newControllerTestDB(t,
		`CREATE TABLE roads (
			id INTEGER PRIMARY KEY, start_intersection_id INTEGER, end_intersection_id INTEGER,
			road_type_id INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE intersections (id INTEGER PRIMARY KEY, player_id INTEGER, map_id INTEGER, x INTEGER, y INTEGER)`,
		`CREATE TABLE road_types (id INTEGER PRIMARY KEY, type TEXT, image_url TEXT)`,
	)
}

func TestGetRoadsReturnsDetailsForRequestedPlayerAndMap(t *testing.T) {
	db := newRoadsControllerTestDB(t)
	execControllerSQL(t, db,
		`INSERT INTO road_types (id, type, image_url) VALUES (3, 'Stone', '/images/stone-road.png')`,
		`INSERT INTO intersections (id, player_id, map_id, x, y) VALUES
			(11, 2, 9, 4, 5), (12, 2, 9, 7, 8),
			(21, 3, 9, 1, 2), (22, 3, 9, 3, 4),
			(31, 2, 10, 2, 3), (32, 2, 10, 4, 5)`,
		`INSERT INTO roads (id, start_intersection_id, end_intersection_id, road_type_id) VALUES
			(5, 11, 12, 3), (6, 21, 22, 3), (7, 31, 32, 3)`,
	)

	response := performControllerRequest(http.MethodGet, "/game/roads/:player_id/:map_id", "/game/roads/2/9", "", GetRoads)
	assertControllerJSON(t, response, http.StatusOK, `{"roads":[{
		"id":5,"start_intersection":{"id":11,"x":4,"y":5},"end_intersection":{"id":12,"x":7,"y":8},
		"road_type":{"id":3,"type":"Stone","image_url":"/images/stone-road.png"}
	}]}`)
}

func TestGetRoadsReturnsEmptyArray(t *testing.T) {
	db := newRoadsControllerTestDB(t)
	execControllerSQL(t, db,
		`INSERT INTO intersections (id, player_id, map_id, x, y) VALUES (1, 2, 10, 0, 0), (2, 3, 9, 1, 1)`,
		`INSERT INTO roads (id, start_intersection_id, end_intersection_id) VALUES (1, 1, 1), (2, 2, 2)`,
	)

	response := performControllerRequest(http.MethodGet, "/game/roads/:player_id/:map_id", "/game/roads/2/9", "", GetRoads)
	assertControllerJSON(t, response, http.StatusOK, `{"roads":[]}`)
}

func TestGetRoadsRejectsInvalidIDs(t *testing.T) {
	for _, parameter := range []string{"player_id", "map_id"} {
		for _, value := range []string{"abc", "-1", "1.5", "4294967296"} {
			t.Run(parameter+"/"+value, func(t *testing.T) {
				playerID, mapID := "2", "9"
				if parameter == "player_id" {
					playerID = value
				} else {
					mapID = value
				}
				response := performControllerRequest(http.MethodGet, "/game/roads/:player_id/:map_id",
					"/game/roads/"+playerID+"/"+mapID, "", GetRoads)
				assertControllerError(t, response, http.StatusBadRequest, "invalid "+parameter)
			})
		}
	}
}

func TestGetRoadsReturnsDatabaseError(t *testing.T) {
	newControllerTestDB(t)

	response := performControllerRequest(http.MethodGet, "/game/roads/:player_id/:map_id", "/game/roads/2/9", "", GetRoads)
	assertControllerError(t, response, http.StatusInternalServerError, "failed to fetch roads for player_id 2 on map_id 9")
}
