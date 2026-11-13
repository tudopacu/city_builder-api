package services

import (
	"API/api/dto"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
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
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		var loads atomic.Int32
		load := func(context.Context) ([]int, error) {
			loads.Add(1)
			<-release
			return []int{1, 2}, nil
		}

		const callers = 5
		var done sync.WaitGroup
		results := make([][]int, callers)
		for i := range callers {
			done.Go(func() {
				results[i], _ = cached(context.Background(), "shared:test", time.Minute, load)
			})
		}
		// Returns once every caller is blocked: one in the load, the rest
		// waiting on it.
		synctest.Wait()
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
		// Each caller owns its result.
		results[0][0] = 99
		if results[1][0] != 1 {
			t.Fatalf("callers share a result: %v", results[1])
		}
	})
	if !reflect.DeepEqual(cache.sets, []string{"v2:shared:test"}) {
		t.Fatalf("cache writes = %v, want one", cache.sets)
	}
}

func TestCachedReturnsBeforeCacheWrite(t *testing.T) {
	cache := newServiceTestCache(t)
	var pending []func()
	startCacheWrite = func(write func()) { pending = append(pending, write) }

	got, err := cached(context.Background(), "async:test", time.Minute, func(context.Context) ([]int, error) {
		return []int{1}, nil
	})
	if err != nil || !reflect.DeepEqual(got, []int{1}) {
		t.Fatalf("result = %v, error = %v", got, err)
	}
	if len(pending) != 1 || len(cache.sets) != 0 {
		t.Fatalf("pending writes = %d, cache writes = %v; want the write deferred", len(pending), cache.sets)
	}
	pending[0]()
	if cache.values["v2:async:test"] != "[1]" {
		t.Fatalf("cache entry = %q", cache.values["v2:async:test"])
	}
}

func TestCachedLoadContextOutlivesCallerButHasDeadline(t *testing.T) {
	newServiceTestCache(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := cached(ctx, "ctx:test", time.Minute, func(loadCtx context.Context) (int, error) {
		if loadCtx.Err() != nil {
			t.Errorf("load context cancelled with the caller: %v", loadCtx.Err())
		}
		if _, ok := loadCtx.Deadline(); !ok {
			t.Error("load context has no deadline")
		}
		return 1, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadCacheLogsOnlyCacheFailures(t *testing.T) {
	var logs bytes.Buffer
	previousOutput := log.Default().Writer()
	log.Default().SetOutput(&logs)
	t.Cleanup(func() { log.Default().SetOutput(previousOutput) })

	cache := newServiceTestCache(t)
	cache.getErr = context.Canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := readCache[int](ctx, "k"); ok || logs.Len() != 0 {
		t.Fatalf("cancelled request: ok = %v, logs = %q", ok, logs.String())
	}

	cache.getErr = errors.New("connection refused")
	if _, ok := readCache[int](context.Background(), "k"); ok || !strings.Contains(logs.String(), "connection refused") {
		t.Fatalf("cache failure: ok = %v, logs = %q", ok, logs.String())
	}
}

// When this fails, a cached DTO changed shape: bump cacheKeyVersion and record
// the new fingerprint, so entries from the previous release are not decoded
// into the new shape.
func TestCachedDTOShapesMatchKeyVersion(t *testing.T) {
	recorded := map[string]string{
		"v2": "72daff780bf588ff",
	}
	shapes := []reflect.Type{
		reflect.TypeFor[[]dto.BuildingWithDetails](),
		reflect.TypeFor[[]dto.Item](),
		reflect.TypeFor[[]dto.Map](),
		reflect.TypeFor[*dto.Map](),
		reflect.TypeFor[[]dto.Tile](),
		reflect.TypeFor[newsCacheEntry](),
	}
	var b strings.Builder
	for _, shape := range shapes {
		writeTypeShape(&b, shape, map[reflect.Type]bool{})
		b.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(b.String()))
	got := hex.EncodeToString(sum[:8])
	if recorded[cacheKeyVersion] != got {
		t.Fatalf("cached DTO fingerprint = %s, recorded for %s = %q; bump cacheKeyVersion and record it",
			got, cacheKeyVersion, recorded[cacheKeyVersion])
	}
}

func writeTypeShape(b *strings.Builder, typ reflect.Type, seen map[reflect.Type]bool) {
	switch typ.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		b.WriteString(typ.Kind().String() + "(")
		writeTypeShape(b, typ.Elem(), seen)
		b.WriteString(")")
	case reflect.Map:
		b.WriteString("map(")
		writeTypeShape(b, typ.Key(), seen)
		b.WriteString(",")
		writeTypeShape(b, typ.Elem(), seen)
		b.WriteString(")")
	case reflect.Struct:
		// Types with their own JSON encoding (time.Time) are opaque here.
		if seen[typ] || typ.PkgPath() == "time" {
			b.WriteString(typ.String())
			return
		}
		seen[typ] = true
		b.WriteString("{")
		for i := range typ.NumField() {
			field := typ.Field(i)
			fmt.Fprintf(b, "%s %q:", field.Name, field.Tag.Get("json"))
			writeTypeShape(b, field.Type, seen)
			b.WriteString(";")
		}
		b.WriteString("}")
	default:
		b.WriteString(typ.Kind().String())
	}
}

func TestCachedDoesNotCacheLoadErrors(t *testing.T) {
	cache := newServiceTestCache(t)
	loadErr := errors.New("database down")

	_, err := cached(context.Background(), "failing:test", time.Minute, func(context.Context) ([]int, error) {
		return nil, loadErr
	})
	if !errors.Is(err, loadErr) || len(cache.sets) != 0 {
		t.Fatalf("error = %v, cache writes = %v", err, cache.sets)
	}
}
