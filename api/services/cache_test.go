package services

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetAllBuildingsIgnoresUnversionedCacheEntry(t *testing.T) {
	cache := newServiceTestCache(t)
	// Written by a release that predates the population fields.
	cache.Set("buildings:all", `[{"id":5,"name":"Sawmill","costs":[],"productions":[]}]`)
	_, want := seedBuildingsServiceCatalog(t)

	got, err := GetAllBuildings(context.Background())
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("buildings = %#v, error = %v, want %#v", got, err, want)
	}
	if !reflect.DeepEqual(cache.gets, []string{"v2:buildings:all"}) {
		t.Fatalf("cache reads = %v, want only the versioned key", cache.gets)
	}
}

func TestCachedSharesConcurrentLoads(t *testing.T) {
	cache := newServiceTestCache(t)
	release := make(chan struct{})
	var loads atomic.Int32
	load := func() ([]int, error) {
		loads.Add(1)
		<-release
		return []int{1, 2}, nil
	}

	const callers = 5
	var started, done sync.WaitGroup
	started.Add(callers)
	done.Add(callers)
	results := make([][]int, callers)
	for i := range callers {
		go func() {
			defer done.Done()
			started.Done()
			results[i], _ = cached(context.Background(), "shared:test", time.Minute, load)
		}()
	}
	started.Wait()
	// Give every caller time to miss the cache and join the in-flight load.
	time.Sleep(50 * time.Millisecond)
	close(release)
	done.Wait()

	if n := loads.Load(); n != 1 {
		t.Fatalf("loads = %d, want 1", n)
	}
	for i, result := range results {
		if !reflect.DeepEqual(result, []int{1, 2}) {
			t.Fatalf("caller %d got %v", i, result)
		}
	}
	if !reflect.DeepEqual(cache.sets, []string{"v2:shared:test"}) {
		t.Fatalf("cache writes = %v, want one", cache.sets)
	}
}

func TestCachedDoesNotCacheLoadErrors(t *testing.T) {
	cache := newServiceTestCache(t)
	loadErr := errors.New("database down")

	_, err := cached(context.Background(), "failing:test", time.Minute, func() ([]int, error) {
		return nil, loadErr
	})
	if !errors.Is(err, loadErr) || len(cache.sets) != 0 {
		t.Fatalf("error = %v, cache writes = %v", err, cache.sets)
	}
}
