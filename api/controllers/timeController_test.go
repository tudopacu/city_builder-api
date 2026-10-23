package controllers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGetCurrentTimestampReturnsCurrentTimeAsJSON(t *testing.T) {
	before := time.Now()
	response := performControllerRequest(http.MethodGet, "/game/current_timestamp", "/game/current_timestamp", "", GetCurrentTimestamp)
	after := time.Now()
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", got)
	}
	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 1 {
		t.Fatalf("unexpected timestamp response: %s", response.Body.String())
	}
	timestamp, err := time.Parse(time.RFC3339Nano, payload["timestamp"])
	if err != nil {
		t.Fatalf("timestamp is not an RFC3339 time: %v", err)
	}
	if timestamp.Before(before) || timestamp.After(after) {
		t.Fatalf("timestamp = %v, want between %v and %v", timestamp, before, after)
	}
}
