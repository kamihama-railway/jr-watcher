package jrwatcher

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/WJQSERVER-STUDIO/httpc"
)

const (
	jrCentralShinkansenBase = "https://traininfo.jr-central.co.jp/shinkansen"
	jrCentralShinkansenVar  = jrCentralShinkansenBase + "/var/train_info"
	jrCentralShinkansenData = jrCentralShinkansenBase + "/common/data"
)

type jrShinkansenScraper struct{ client *httpc.Client }

type jrShinkansenStatus struct {
	Screen struct {
		Message     *jrShinkansenMsg `json:"message"`
		NoticeList  []jrShinkansenNotice `json:"noticeList"`
	} `json:"screen"`
}

type jrShinkansenMsg struct {
	AreaText string `json:"area_text"`
	Statuses []jrShinkansenStatusItem `json:"statuses"`
}

type jrShinkansenStatusItem struct {
	Direction string `json:"direction"`
	Area      string `json:"area"`
	StatusID  string `json:"status_id"`
	CauseID   string `json:"cause_id"`
}

type jrShinkansenNotice struct {
	Title string `json:"title"`
}

type jrShinkansenServiceStatus struct {
	SuspensionEnabled bool                       `json:"suspensionInfoIsEnabled"`
	ServiceInfo       jrShinkansenServiceInfo    `json:"serviceStatusInfo"`
}

type jrShinkansenServiceInfo struct {
	DateTime  int64                          `json:"datetime"`
	Data      []jrShinkansenServiceData      `json:"data"`
}

type jrShinkansenServiceData struct {
	Time     string `json:"time"`
	Contents []jrShinkansenServiceContent `json:"contents"`
}

type jrShinkansenServiceContent struct {
	Title    string `json:"title"`
	Messages []jrShinkansenServiceMsg `json:"messages"`
}

type jrShinkansenServiceMsg struct {
	Message string `json:"message"`
}

func newJRShinkansenScraper() *jrShinkansenScraper {
	return &jrShinkansenScraper{client: httpc.New(httpc.WithTimeout(15 * time.Second))}
}

func (s *jrShinkansenScraper) fetchJSON(url string, dest any) error {
	resp, err := s.client.GET(url).
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

	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("json decode: %w", err)
	}
	return nil
}

func FetchJRCentralShinkansenTrainInfo() (*TrainInfo, error) {
	s := newJRShinkansenScraper()

	var status jrShinkansenStatus
	if err := s.fetchJSON(jrCentralShinkansenVar+"/ti01_ja.json", &status); err != nil {
		return nil, fmt.Errorf("status: %w", err)
	}

	var svc jrShinkansenServiceStatus
	if err := s.fetchJSON(jrCentralShinkansenVar+"/service_status.json", &svc); err != nil {
		return nil, fmt.Errorf("service status: %w", err)
	}

	routes := buildJRShinkansenRoutes(&status, &svc)
	announcements := buildJRShinkansenAnnouncements(&status)

	return &TrainInfo{
		Area:          AreaJRCentralShinkan,
		UpdatedAt:     time.Now(),
		Routes:        routes,
		Announcements: announcements,
	}, nil
}

func buildJRShinkansenRoutes(status *jrShinkansenStatus, svc *jrShinkansenServiceStatus) []Route {
	r := Route{
		Name:   "東海道・山陽新幹線",
		LineID: "jr-central-shinkansen",
		Area:   AreaJRCentralShinkan,
	}

	if status.Screen.Message != nil && len(status.Screen.Message.Statuses) > 0 {
		var parts []string
		for _, s := range status.Screen.Message.Statuses {
			if s.StatusID == "2" {
				r.Status = RouteStatusAdjust
			} else if s.StatusID == "1" {
				r.Status = RouteStatusDelay
			} else {
				r.Status = RouteStatusInfo
			}
			if s.Area != "" {
				parts = append(parts, fmt.Sprintf("%s: %s", s.Area, s.StatusID))
			}
		}
		if len(parts) > 0 {
			r.Note = strings.Join(parts, " / ")
		}
	} else if svc.ServiceInfo.Data != nil && len(svc.ServiceInfo.Data) > 0 {
		r.Status = RouteStatusDelay
		var notes []string
		for _, d := range svc.ServiceInfo.Data {
			for _, c := range d.Contents {
				for _, m := range c.Messages {
					notes = append(notes, m.Message)
				}
			}
		}
		if len(notes) > 0 {
			r.Note = strings.Join(notes, " ")
		}
	} else {
		r.Status = RouteStatusNormal
	}

	return []Route{r}
}

func buildJRShinkansenAnnouncements(status *jrShinkansenStatus) []Announcement {
	var anns []Announcement
	for _, n := range status.Screen.NoticeList {
		if n.Title != "" {
			anns = append(anns, Announcement{
				Date:    time.Now().Format("2006-01-02"),
				Message: n.Title,
			})
		}
	}
	return anns
}
