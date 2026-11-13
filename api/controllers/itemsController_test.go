package controllers

import (
	"net/http"
	"testing"
)

func TestGetItemsReturnsCatalogInIDOrder(t *testing.T) {
	newControllerTestCache(t)
	newControllerTestDB(t,
		`CREATE TABLE items (
			id INTEGER PRIMARY KEY, name TEXT, description TEXT, type TEXT,
			icon_url TEXT, is_tradeable BOOLEAN)`,
		`INSERT INTO items (id, name, description, type, icon_url, is_tradeable) VALUES
			(8, 'Stone', NULL, 'resource', NULL, 0),
			(3, 'Wood', 'Building material', 'resource', '/images/wood.png', 1)`,
	)

	response := performControllerRequest(http.MethodGet, "/game/items", "/game/items", "", GetItems)
	assertControllerJSON(t, response, http.StatusOK, `{"items":[
		{"id":3,"name":"Wood","description":"Building material","type":"resource","icon_url":"/images/wood.png","is_tradeable":true},
		{"id":8,"name":"Stone","type":"resource","is_tradeable":false}
	]}`)
}

func TestGetItemsReturnsEmptyArray(t *testing.T) {
	newControllerTestCache(t)
	newControllerTestDB(t, `CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT, type TEXT)`)

	response := performControllerRequest(http.MethodGet, "/game/items", "/game/items", "", GetItems)
	assertControllerJSON(t, response, http.StatusOK, `{"items":[]}`)
}

func TestGetItemsReturnsCachedCatalog(t *testing.T) {
	cache := newControllerTestCache(t)
	cache.Set("v2:items:all", `[{"id":7,"name":"Iron","type":"resource","is_tradeable":true}]`)
	// A cache hit must work without querying the deliberately empty database.
	newControllerTestDB(t)

	response := performControllerRequest(http.MethodGet, "/game/items", "/game/items", "", GetItems)
	assertControllerJSON(t, response, http.StatusOK,
		`{"items":[{"id":7,"name":"Iron","type":"resource","is_tradeable":true}]}`)
}

func TestGetItemsReturnsDatabaseError(t *testing.T) {
	newControllerTestCache(t)
	newControllerTestDB(t)

	response := performControllerRequest(http.MethodGet, "/game/items", "/game/items", "", GetItems)
	assertControllerError(t, response, http.StatusInternalServerError, "failed to fetch items")
}
