package jrwatcher

import (
	"sync"
	"time"
)

type cacheEntry struct {
	info      *TrainInfo
	expiresAt time.Time
	staleAt   time.Time
	mu        sync.Mutex
	refreshing bool
}

var (
	cacheStore sync.Map
)

type ttlConfig struct {
	ttl     time.Duration
	staleTTL time.Duration
}

var areaTTL = map[Area]ttlConfig{
	AreaJREastKanto:      {ttl: 60 * time.Second, staleTTL: 120 * time.Second},
	AreaJREastTohoku:     {ttl: 60 * time.Second, staleTTL: 120 * time.Second},
	AreaJREastShinetsu:   {ttl: 60 * time.Second, staleTTL: 120 * time.Second},
	AreaJREastExpress:    {ttl: 60 * time.Second, staleTTL: 120 * time.Second},
	AreaJREastShinkansen: {ttl: 60 * time.Second, staleTTL: 120 * time.Second},
	AreaJRCentral:        {ttl: 120 * time.Second, staleTTL: 300 * time.Second},
	AreaJRCentralShinkan: {ttl: 120 * time.Second, staleTTL: 300 * time.Second},
	AreaJRWest:           {ttl: 120 * time.Second, staleTTL: 300 * time.Second},
	"jr-hokkaido":        {ttl: 120 * time.Second, staleTTL: 300 * time.Second},
	"jr-kyushu":          {ttl: 120 * time.Second, staleTTL: 300 * time.Second},
	"jr-shikoku":         {ttl: 300 * time.Second, staleTTL: 600 * time.Second},
}

func defaultTTL() ttlConfig {
	return ttlConfig{ttl: 60 * time.Second, staleTTL: 120 * time.Second}
}

func getTTL(area Area) ttlConfig {
	if t, ok := areaTTL[area]; ok {
		return t
	}
	return defaultTTL()
}

func cachedFetch(area Area, fetchFn func() (*TrainInfo, error)) (*TrainInfo, error) {
	cfg := getTTL(area)
	now := time.Now()

	raw, _ := cacheStore.LoadOrStore(area, &cacheEntry{})
	entry := raw.(*cacheEntry)

	entry.mu.Lock()
	cached := entry.info
	expiresAt := entry.expiresAt
	staleAt := entry.staleAt
	inFlight := entry.refreshing
	entry.mu.Unlock()

	if cached != nil && now.Before(expiresAt) {
		return cached, nil
	}

	if cached != nil && now.Before(staleAt) && !inFlight {
		entry.mu.Lock()
		entry.refreshing = true
		entry.mu.Unlock()
		go func() {
			refreshCache(area, entry, fetchFn, cfg)
			entry.mu.Lock()
			entry.refreshing = false
			entry.mu.Unlock()
		}()
		return cached, nil
	}

	return refreshCache(area, entry, fetchFn, cfg)
}

func refreshCache(area Area, entry *cacheEntry, fetchFn func() (*TrainInfo, error), cfg ttlConfig) (*TrainInfo, error) {
	entry.mu.Lock()
	defer entry.mu.Unlock()

	info, err := fetchFn()
	if err != nil {
		if entry.info != nil {
			return entry.info, nil
		}
		return nil, err
	}

	now := time.Now()
	entry.info = info
	entry.expiresAt = now.Add(cfg.ttl)
	entry.staleAt = now.Add(cfg.staleTTL)

	return info, nil
}

func invalidateCache(area Area) {
	cacheStore.Delete(area)
}

func clearCache() {
	cacheStore.Range(func(key, _ any) bool {
		cacheStore.Delete(key)
		return true
	})
}
