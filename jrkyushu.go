package jrwatcher

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"

	"github.com/WJQSERVER-STUDIO/httpc"
)

const jrKyushuBase = "https://www.jrkyushu.co.jp/trains/info"

type jrKyushuScraper struct{ client *httpc.Client }

type jrKyushuXML struct {
	Time string       `xml:"time"`
	Info jrKyushuInfoData `xml:"info"`
}

type jrKyushuInfoData struct {
	Areas []jrKyushuArea `xml:"aif"`
}

type jrKyushuArea struct {
	Name string `xml:"nm"`
	Sts  int    `xml:"sts"`
}

var jrKyushuAreaNames = map[string]string{
	"Fukuoka-Kitakyushu":  "福岡・北九州エリア",
	"Oita":                "大分エリア",
	"Saga-Nagasaki":       "佐賀・長崎エリア",
	"Kumamoto":            "熊本エリア",
	"Miyazaki":            "宮崎エリア",
	"Kagoshima":           "鹿児島エリア",
	"Kyushu-Shinkansen":   "九州新幹線",
	"Nishi-Kyushu-Shinkansen": "西九州新幹線",
}

func newJRKyushuScraper() *jrKyushuScraper {
	return &jrKyushuScraper{client: httpc.New(httpc.WithTimeout(15 * time.Second))}
}

func (s *jrKyushuScraper) fetchXML(path string) ([]byte, error) {
	resp, err := s.client.GET(jrKyushuBase + path).
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36").
		SetHeader("Accept", "application/xml, text/xml, */*").
		Execute()
	if err != nil {
		return nil, fmt.Errorf("httpc get: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func FetchJRKyushuTrainInfo() (*TrainInfo, error) {
	s := newJRKyushuScraper()

	raw, err := s.fetchXML("/data/IDS2Web.xml")
	if err != nil {
		return nil, fmt.Errorf("fetch xml: %w", err)
	}

	var data jrKyushuXML
	if err := xml.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("xml parse: %w", err)
	}

	routes := make([]Route, 0, len(data.Info.Areas))
	for _, area := range data.Info.Areas {
		name := jrKyushuAreaNames[area.Name]
		if name == "" {
			name = area.Name
		}
		r := Route{
			Name:   name,
			LineID: fmt.Sprintf("jr-kyushu-%s", area.Name),
			Area:   "jr-kyushu",
		}
		if area.Sts >= 2 {
			r.Status = RouteStatusDelay
		} else if area.Sts > 0 {
			r.Status = RouteStatusInfo
		} else {
			r.Status = RouteStatusNormal
		}
		routes = append(routes, r)
	}

	return &TrainInfo{
		Area:      "jr-kyushu",
		UpdatedAt: time.Now(),
		Routes:    routes,
	}, nil
}
