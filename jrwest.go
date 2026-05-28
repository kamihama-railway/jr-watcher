package jrwatcher

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/WJQSERVER-STUDIO/httpc"
)

const jrWestAPI = "https://trafficinfo.westjr.co.jp/api/v1/trafficinfo.json"

type jrWestScraper struct{ client *httpc.Client }

type jrWestResponse struct {
	CreatedAt   time.Time         `json:"createdAt"`
	MasterData  jrWestMaster      `json:"masterData"`
	AreaInfos   []jrWestAreaInfo  `json:"areaTrafficInfos"`
}

type jrWestMaster struct {
	Areas []jrWestMasterArea `json:"areas"`
}

type jrWestMasterArea struct {
	ID     int               `json:"id"`
	Name   string            `json:"name"`
	Places []jrWestPlace     `json:"places"`
}

type jrWestPlace struct {
	ID    int             `json:"id"`
	Name  string          `json:"name"`
	Lines []jrWestLineRef `json:"lines"`
}

type jrWestLineRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type jrWestAreaInfo struct {
	ID              int                  `json:"id"`
	DailyData       []jrWestDaily        `json:"dailyData"`
}

type jrWestDaily struct {
	Date    string            `json:"date"`
	Infos   []jrWestPlaceInfo `json:"placeTrafficInfos"`
}

type jrWestPlaceInfo struct {
	ID     int              `json:"id"`
	Trains []jrWestLineInfo `json:"conventionalLineTrafficInfos"`
}

type jrWestLineInfo struct {
	ID       int                     `json:"id"`
	LineName string                  `json:"lineName"`
	IconType string                  `json:"iconType"`
	Details  []jrWestTrafficDetail   `json:"conventionalLineTrafficInfoDetails"`
	Versions []jrWestTrafficVersion  `json:"trafficVersions"`
	Sections []jrWestSection         `json:"sections"`
}

type jrWestTrafficDetail struct {
	ConditionName string `json:"conditionName"`
	CauseText     string `json:"causeText"`
	CauseDetail   string `json:"causeDetail"`
}

type jrWestTrafficVersion struct {
	Title     string `json:"title"`
	UpdatedAt string `json:"updatedAt"`
}

type jrWestSection struct {
	ConditionName string `json:"conditionName"`
	StationName   string `json:"stationName"`
	ToStationName string `json:"toStationName"`
}

func newJRWestScraper() *jrWestScraper {
	return &jrWestScraper{client: httpc.New(httpc.WithTimeout(15 * time.Second))}
}

func (s *jrWestScraper) fetch() (*jrWestResponse, error) {
	resp, err := s.client.GET(jrWestAPI).
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36").
		SetHeader("Accept", "application/json").
		Execute()
	if err != nil {
		return nil, fmt.Errorf("httpc get: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var data jrWestResponse
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	return &data, nil
}

func FetchJRWestTrainInfo() (*TrainInfo, error) {
	s := newJRWestScraper()
	data, err := s.fetch()
	if err != nil {
		return nil, err
	}

	routes := buildJRWestRoutes(data)
	announcements := buildJRWestAnnouncements(data)

	return &TrainInfo{
		Area:          AreaJRWest,
		UpdatedAt:     data.CreatedAt,
		Routes:        routes,
		Announcements: announcements,
	}, nil
}

func buildJRWestRoutes(data *jrWestResponse) []Route {
	// Collect all non-normal line statuses from traffic data
	statusMap := make(map[int]jrWestLineInfo)
	for _, area := range data.AreaInfos {
		if len(area.DailyData) == 0 {
			continue
		}
		for _, place := range area.DailyData[0].Infos {
			for _, line := range place.Trains {
				statusMap[line.ID] = line
			}
		}
	}

	// Build full line list from master data, cross-referencing status
	var routes []Route
	seen := make(map[string]bool)
	for _, area := range data.MasterData.Areas {
		for _, place := range area.Places {
			for _, line := range place.Lines {
				id := fmt.Sprintf("jrwest-%d", line.ID)
				if seen[id] {
					continue
				}
				seen[id] = true

				r := Route{
					Name:   line.Name,
					LineID: id,
					Area:   AreaJRWest,
				}

				if ti, ok := statusMap[line.ID]; ok {
					r.Status = resolveJRWestStatus(ti)
					r.Note = resolveJRWestNote(ti)
				} else {
					r.Status = RouteStatusNormal
				}

				routes = append(routes, r)
			}
		}
	}

	return routes
}

func resolveJRWestStatus(line jrWestLineInfo) RouteStatus {
	if len(line.Details) > 0 {
		for _, d := range line.Details {
			cn := d.ConditionName
			if strings.Contains(cn, "運転見合わせ") || strings.Contains(cn, "運転を見合わせ") {
				return RouteStatusAdjust
			}
			if strings.Contains(cn, "運休") || strings.Contains(cn, "遅れ") || strings.Contains(cn, "取り止め") {
				return RouteStatusDelay
			}
		}
	}

	if len(line.Sections) > 0 {
		for _, sec := range line.Sections {
			cn := sec.ConditionName
			if strings.Contains(cn, "運転見合わせ") {
				return RouteStatusAdjust
			}
			if strings.Contains(cn, "運休") || strings.Contains(cn, "遅れ") || strings.Contains(cn, "取り止め") {
				return RouteStatusDelay
			}
		}
	}

	if len(line.Details) > 0 || len(line.Sections) > 0 || line.IconType != "0000" {
		return RouteStatusInfo
	}

	return RouteStatusNormal
}

func resolveJRWestNote(line jrWestLineInfo) string {
	var parts []string

	for _, d := range line.Details {
		if d.ConditionName != "" {
			parts = append(parts, d.ConditionName)
		}
		if d.CauseText != "" {
			parts = append(parts, d.CauseText)
		}
		if d.CauseDetail != "" {
			parts = append(parts, d.CauseDetail)
		}
	}

	for _, sec := range line.Sections {
		if sec.StationName != "" || sec.ToStationName != "" {
			parts = append(parts, fmt.Sprintf("%s-%s", sec.StationName, sec.ToStationName))
		}
	}

	if len(line.Versions) > 0 && parts == nil {
		parts = append(parts, line.Versions[0].Title)
	}

	if len(parts) > 0 {
		return strings.Join(parts, " / ")
	}
	return ""
}

func buildJRWestAnnouncements(data *jrWestResponse) []Announcement {
	var anns []Announcement
	seen := make(map[string]bool)

	for _, area := range data.AreaInfos {
		if len(area.DailyData) == 0 {
			continue
		}
		today := area.DailyData[0]

		for _, place := range today.Infos {
			for _, line := range place.Trains {
				for _, v := range line.Versions {
					if v.Title == "" || seen[v.Title] {
						continue
					}
					seen[v.Title] = true
					a := Announcement{
						Date:    truncateDate(v.UpdatedAt),
						Message: fmt.Sprintf("[%s] %s", line.LineName, v.Title),
					}
					anns = append(anns, a)
				}
			}
		}
	}

	return anns
}

func truncateDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
