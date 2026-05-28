package jrwatcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/WJQSERVER-STUDIO/httpc"
)

const jrHokkaidoBase = "https://www3.jrhokkaido.co.jp/webunkou"

type jrHokkaidoScraper struct{ client *httpc.Client }

// Line name master
type jrHokkaidoSenkuMaster []jrHokkaidoSenkuEntry

type jrHokkaidoSenkuEntry struct {
	Code     string             `json:"senku"`
	Name     jrHokkaidoLangText `json:"name"`
	FullName jrHokkaidoLangText `json:"fullName"`
}

type jrHokkaidoLangText struct {
	JA string `json:"ja"`
}

// Area data
type jrHokkaidoAreaData struct {
	Time   string                `json:"time"`
	Today  jrHokkaidoAreaDay     `json:"today"`
}

type jrHokkaidoAreaDay struct {
	SenkuStatus map[string]int `json:"senkuStatus"`
}

// Top data
type jrHokkaidoTopData struct {
	Time  string             `json:"time"`
	Today jrHokkaidoTopDay   `json:"today"`
}

type jrHokkaidoTopDay struct {
	Status map[string]int `json:"status"`
}

// Built-in per-area senku codes and their line names (fallback if master not available)
var jrHokkaidoAreas = []struct {
	ID   string
	Name string
	File string // area_XX.json number
	Code string // area short code
}{
	{ID: "01", Name: "札幌近郊", File: "01", Code: "spo"},
	{ID: "02", Name: "道央エリア", File: "02", Code: "doo"},
	{ID: "03", Name: "道南エリア", File: "03", Code: "donan"},
	{ID: "04", Name: "道北エリア", File: "04", Code: "dohoku"},
	{ID: "05", Name: "道東エリア", File: "05", Code: "doto"},
}

func newJRHokkaidoScraper() *jrHokkaidoScraper {
	return &jrHokkaidoScraper{client: httpc.New(httpc.WithTimeout(15 * time.Second))}
}

func (s *jrHokkaidoScraper) fetchJSON(path string, dest any) error {
	resp, err := s.client.GET(jrHokkaidoBase + path).
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36").
		SetHeader("Accept", "application/json").
		Execute()
	if err != nil {
		return fmt.Errorf("httpc get: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})

	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("json decode: %w", err)
	}
	return nil
}

func FetchJRHokkaidoTrainInfo() (*TrainInfo, error) {
	s := newJRHokkaidoScraper()

	// Fetch line name master
	var master jrHokkaidoSenkuMaster
	_ = s.fetchJSON("/json/master/senku_name_master.json", &master)

	// Build line name lookup
	nameByCode := buildJRHokkaidoLineNames(&master)

	// Fetch all areas
	var allRoutes []Route
	var allAnns []Announcement

	for _, area := range jrHokkaidoAreas {
		var data jrHokkaidoAreaData
		if err := s.fetchJSON("/json/area/area_"+area.File+".json", &data); err != nil {
			continue
		}

		routes := buildJRHokkaidoRoutes(&data, area, nameByCode)
		allRoutes = append(allRoutes, routes...)
	}

	// Fetch Shinkansen separately
	var shinkansenTop jrHokkaidoTopData
	if err := s.fetchJSON("/json/top/top.json", &shinkansenTop); err == nil {
		if status, ok := shinkansenTop.Today.Status["shin"]; ok {
			r := Route{
				Name:   "北海道新幹線",
				LineID: "jr-hokkaido-shinkansen",
				Area:   "jr-hokkaido",
			}
			if status >= 2 {
				r.Status = RouteStatusDelay
			} else {
				r.Status = RouteStatusNormal
			}
			allRoutes = append(allRoutes, r)
		}
	}

	return &TrainInfo{
		Area:          "jr-hokkaido",
		UpdatedAt:     time.Now(),
		Routes:        allRoutes,
		Announcements: allAnns,
	}, nil
}

func buildJRHokkaidoLineNames(master *jrHokkaidoSenkuMaster) map[string]string {
	m := map[string]string{
		"express":       "特急列車",
		"airport":       "エアポート",
		"hakochise":     "函館・千歳線",
		"hakodateLiner": "はこだてライナー",
		"hakodate":      "函館線",
		"gakuen":        "学園都市線",
		"sekisho":       "石勝線",
		"muroran":       "室蘭線",
		"nemuro":        "根室線",
		"hidaka":        "日高線",
		"soya":          "宗谷線",
		"sekihoku":      "石北線",
		"furano":        "富良野線",
		"rumoi":         "留萌線",
		"senmo":         "釧網線",
		"hanasaki":      "花咲線",
		"sassho":        "札沼線",
	}

	if master != nil {
		for _, entry := range *master {
			if entry.Name.JA != "" {
				// Map by area+code pattern
				_ = entry.Code
			}
		}
	}

	return m
}

func buildJRHokkaidoRoutes(data *jrHokkaidoAreaData, area struct {
	ID   string
	Name string
	File string
	Code string
}, nameByCode map[string]string) []Route {
	var routes []Route

	for code, status := range data.Today.SenkuStatus {
		r := Route{
			Area: "jr-hokkaido",
		}

		if name, ok := nameByCode[code]; ok {
			r.Name = fmt.Sprintf("%s %s", area.Name, name)
			r.LineID = fmt.Sprintf("jr-hokkaido-%s", code)
		} else {
			r.Name = fmt.Sprintf("%s %s", area.Name, code)
			r.LineID = fmt.Sprintf("jr-hokkaido-%s", code)
		}

		if status >= 2 {
			r.Status = RouteStatusDelay
		} else {
			r.Status = RouteStatusNormal
		}

		routes = append(routes, r)
	}

	return routes
}
