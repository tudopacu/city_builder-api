package controllers

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetBuildingsReturnsCategoryCostsAndProductions(t *testing.T) {
	newControllerTestCache(t)
	newControllerTestDB(t,
		`CREATE TABLE buildings (
			id INTEGER PRIMARY KEY, name TEXT, image_url TEXT, description TEXT,
			width INTEGER, length INTEGER, building_category_id INTEGER)`,
		`CREATE TABLE building_categories (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE building_construction_costs (
			id INTEGER PRIMARY KEY, building_id INTEGER, item_id INTEGER, quantity INTEGER)`,
		`CREATE TABLE building_productions (
			id INTEGER PRIMARY KEY, building_id INTEGER, item_id INTEGER,
			quantity INTEGER, production_time_seconds INTEGER)`,
		`INSERT INTO building_categories (id, name) VALUES (2, 'Production')`,
		`INSERT INTO buildings (id, name, image_url, description, width, length, building_category_id)
			VALUES (5, 'Sawmill', '/images/sawmill.png', 'Cuts timber', 3, 4, 2)`,
		`INSERT INTO items (id, name) VALUES (7, 'Stone'), (8, 'Wood')`,
		`INSERT INTO building_construction_costs (id, building_id, item_id, quantity)
			VALUES (11, 5, 7, 20)`,
		`INSERT INTO building_productions (id, building_id, item_id, quantity, production_time_seconds)
			VALUES (12, 5, 8, 6, 120)`,
	)

	response := performControllerRequest(http.MethodGet, "/game/buildings", "/game/buildings", "", GetBuildings)
	assertControllerJSON(t, response, http.StatusOK, `{"buildings":[{
		"id":5,"name":"Sawmill","image_url":"/images/sawmill.png","description":"Cuts timber",
		"width":3,"length":4,"building_category":"Production",
		"costs":[{"item_id":7,"item_name":"Stone","quantity":20}],
		"productions":[{"building_production_id":12,"item_id":8,"item_name":"Wood","quantity":6,"production_time_seconds":120}]
	}]}`)
}

func TestGetBuildingsReturnsEmptyCatalog(t *testing.T) {
	newControllerTestCache(t)
	newControllerTestDB(t, `CREATE TABLE buildings (id INTEGER PRIMARY KEY)`)

	response := performControllerRequest(http.MethodGet, "/game/buildings", "/game/buildings", "", GetBuildings)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}
	var payload map[string][]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	buildings, exists := payload["buildings"]
	if !exists || len(payload) != 1 || len(buildings) != 0 {
		t.Fatalf("expected an empty buildings catalog, got %s", response.Body.String())
	}
}

func TestGetBuildingsReturnsCachedCatalog(t *testing.T) {
	cache := newControllerTestCache(t)
	cache.Set("buildings:all", `[{
		"id":9,"name":"Warehouse","description":"Stores goods","width":2,"length":3,
		"building_category":"Storage","costs":[],"productions":[]
	}]`)
	newControllerTestDB(t)

	response := performControllerRequest(http.MethodGet, "/game/buildings", "/game/buildings", "", GetBuildings)
	assertControllerJSON(t, response, http.StatusOK, `{"buildings":[{
		"id":9,"name":"Warehouse","description":"Stores goods","width":2,"length":3,
		"building_category":"Storage","costs":[],"productions":[]
	}]}`)
}

func TestGetBuildingsReturnsDatabaseError(t *testing.T) {
	newControllerTestCache(t)
	newControllerTestDB(t)

	response := performControllerRequest(http.MethodGet, "/game/buildings", "/game/buildings", "", GetBuildings)
	assertControllerError(t, response, http.StatusInternalServerError, "failed to fetch buildings")
}
