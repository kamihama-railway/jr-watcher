package jrwatcher

import (
	"fmt"
	"sync"
)

var defaultScraper = newScraper()

func FetchTrainInfo() (*TrainInfo, error) {
	return FetchTrainInfoByArea(AreaJREastKanto)
}

func FetchTrainInfoByArea(area Area) (*TrainInfo, error) {
	switch area {
	case AreaJRCentral:
		return cachedFetch(area, FetchJRCentralTrainInfo)
	case AreaJRCentralShinkan:
		return cachedFetch(area, FetchJRCentralShinkansenTrainInfo)
	case AreaJRWest:
		return cachedFetch(area, FetchJRWestTrainInfo)
	case "jr-hokkaido":
		return cachedFetch(area, FetchJRHokkaidoTrainInfo)
	case "jr-kyushu":
		return cachedFetch(area, FetchJRKyushuTrainInfo)
	case "jr-shikoku":
		return cachedFetch(area, FetchJRShikokuTrainInfo)
	}

	return cachedFetch(area, func() (*TrainInfo, error) {
		htmlBytes, err := defaultScraper.fetchPage(area)
		if err != nil {
			return nil, fmt.Errorf("fetch page %s: %w", area, err)
		}

		jsonBytes, err := defaultScraper.fetchJSON(area)
		if err != nil {
			return nil, fmt.Errorf("fetch json %s: %w", area, err)
		}

		htmlContent := string(htmlBytes)

		updatedAt, err := parseUpdatedAt(htmlContent)
		if err != nil {
			return nil, fmt.Errorf("parse updated at %s: %w", area, err)
		}

		routes, err := parseRoutes(htmlContent, area)
		if err != nil {
			return nil, fmt.Errorf("parse routes %s: %w", area, err)
		}

		announcements, err := parseAnnouncements(jsonBytes)
		if err != nil {
			return nil, fmt.Errorf("parse announcements %s: %w", area, err)
		}

		return &TrainInfo{
			Area:          area,
			UpdatedAt:     updatedAt,
			Routes:        routes,
			Announcements: announcements,
		}, nil
	})
}

func FetchAllAreas() (map[Area]*TrainInfo, error) {
	areas := []Area{
		AreaJREastKanto, AreaJREastTohoku, AreaJREastShinetsu,
		AreaJREastExpress, AreaJREastShinkansen,
		AreaJRCentral, AreaJRCentralShinkan, AreaJRWest,
		"jr-hokkaido", "jr-kyushu", "jr-shikoku",
	}
	result := make(map[Area]*TrainInfo, len(areas))
	var mu sync.Mutex
	var wg sync.WaitGroup
	errCh := make(chan error, len(areas))

	for _, area := range areas {
		wg.Add(1)
		area := area
		go func() {
			defer wg.Done()
			info, err := FetchTrainInfoByArea(area)
			if err != nil {
				errCh <- fmt.Errorf("%s: %w", area, err)
				return
			}
			mu.Lock()
			result[area] = info
			mu.Unlock()
		}()
	}

	wg.Wait()
	close(errCh)

	if err, ok := <-errCh; ok {
		return result, err
	}

	return result, nil
}

func FetchRoutes(area Area) ([]Route, error) {
	info, err := FetchTrainInfoByArea(area)
	if err != nil {
		return nil, err
	}
	return info.Routes, nil
}

func FetchAnnouncements(area Area) ([]Announcement, error) {
	info, err := FetchTrainInfoByArea(area)
	if err != nil {
		return nil, err
	}
	return info.Announcements, nil
}
