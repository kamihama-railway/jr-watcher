package jrwatcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/WJQSERVER-STUDIO/httpc"
)

// JR Central base
const jrCentralBase = "https://traininfo.jr-central.co.jp/zairaisen/"

// JR Central account status
type jrCentralAccountStatus int

const (
	jrCentralStatusService jrCentralAccountStatus = 0 // 0=normal
)

type jrCentralScraper struct {
	client *httpc.Client
}

type jrLineMaster struct {
	Lst []jrLine `json:"lst"`
}

type jrLine struct {
	Code        string `json:"ryokakuSenkuCd"`
	Name        string `json:"ryokakuSenkuMei"`
	Section     string `json:"ryokakuSenkuKaishiShuryoEki"`
	Color       string `json:"ryokakuSenkuColorCdchi"`
	DisplaySort string `json:"hyojiJun"`
}

type jrOperationStatus struct {
	Check      string `json:"check"`
	CreateTime string `json:"create_time"`
	Ono        string `json:"ono"`
	Message    []jrOpMessage `json:"message_info"`
}

type jrOpMessage struct {
	Trainline []jrLangName   `json:"trainline"`
	Delivery  []jrLangMsg    `json:"delivery_msg"`
}

type jrLangName struct {
	Lang string `json:"lang"`
	Name string `json:"name"`
}

type jrLangMsg struct {
	Lang    string `json:"lang"`
	Message string `json:"message"`
}

type jrNotice struct {
	Check      string `json:"check"`
	CreateTime string `json:"create_time"`
	NoticeInfo []jrNoticeItem `json:"notice_info"`
}

type jrNoticeItem struct {
	LineID     string        `json:"trainlineid"`
	Trainline  []jrLangName  `json:"trainline"`
	InfoLine   []jrNoticeMsg `json:"notice_infoline"`
}

type jrNoticeMsg struct {
	No            string           `json:"no"`
	From          string           `json:"publication_from"`
	To            string           `json:"publication_to"`
	NoticeMessage []jrNoticeDetail `json:"notice_message"`
}

type jrNoticeDetail struct {
	Lang  string `json:"lang"`
	Title string `json:"title"`
}

func newJRCentralScraper() *jrCentralScraper {
	return &jrCentralScraper{
		client: httpc.New(httpc.WithTimeout(15 * time.Second)),
	}
}

func (s *jrCentralScraper) fetchJSON(path string, dest any) error {
	resp, err := s.client.GET(jrCentralBase + path).
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
		return fmt.Errorf("json decode %s: %w", path, err)
	}
	return nil
}

func FetchJRCentralTrainInfo() (*TrainInfo, error) {
	s := newJRCentralScraper()

	var lineMaster jrLineMaster
	if err := s.fetchJSON("data/hp_senku_master_ja.json", &lineMaster); err != nil {
		return nil, fmt.Errorf("line master: %w", err)
	}

	var opStatus jrOperationStatus
	if err := s.fetchJSON("data/trainInfo/json/unkou.json", &opStatus); err != nil {
		return nil, fmt.Errorf("operation status: %w", err)
	}

	var notices jrNotice
	if err := s.fetchJSON("data/notice/json/oshirase.json", &notices); err != nil {
		_ = notices
	}

	now := time.Now()
	updatedAt := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.FixedZone("JST", 9*60*60))

	routes := buildJRRoutes(&lineMaster, &opStatus)
	announcements := buildJRAnnouncements(&notices)

	return &TrainInfo{
		Area:          AreaJRCentral,
		UpdatedAt:     updatedAt,
		Routes:        routes,
		Announcements: announcements,
	}, nil
}

func buildJRRoutes(lines *jrLineMaster, op *jrOperationStatus) []Route {
	lineStatus := buildOpLineMap(op)

	routes := make([]Route, 0, len(lines.Lst))
	for _, line := range lines.Lst {
		if line.Code == "350010" {
			continue
		}

		r := Route{
			Name:   strings.TrimSpace(line.Name),
			LineID: line.Code,
			Area:   AreaJRCentral,
		}

		if msg, ok := lineStatus[line.Name]; ok {
			if strings.Contains(msg, "運転見合わせ") || strings.Contains(msg, "運転を見合わせ") {
				r.Status = RouteStatusAdjust
			} else {
				r.Status = RouteStatusDelay
			}
			r.Note = truncateJA(msg, 200)
		} else {
			r.Status = RouteStatusNormal
		}

		routes = append(routes, r)
	}
	return routes
}

func buildOpLineMap(op *jrOperationStatus) map[string]string {
	m := make(map[string]string)
	for _, msg := range op.Message {
		var jaName string
		var jaMsg string
		for _, t := range msg.Trainline {
			if t.Lang == "ja" {
				jaName = t.Name
				break
			}
		}
		for _, d := range msg.Delivery {
			if d.Lang == "ja" {
				jaMsg = d.Message
				break
			}
		}
		if jaName != "" && strings.TrimSpace(jaMsg) != "" {
			m[jaName] = jaMsg
		}
	}
	return m
}

func buildJRAnnouncements(n *jrNotice) []Announcement {
	var anns []Announcement
	for _, item := range n.NoticeInfo {
		var jaName string
		for _, t := range item.Trainline {
			if t.Lang == "ja" {
				jaName = t.Name
				break
			}
		}
		for _, info := range item.InfoLine {
			var title string
			for _, m := range info.NoticeMessage {
				if m.Lang == "ja" && m.Title != "" {
					title = m.Title
					break
				}
			}
			if title != "" {
				a := Announcement{
					Date:    formatDate(info.From),
					Message: fmt.Sprintf("[%s] %s", jaName, title),
				}
				anns = append(anns, a)
			}
		}
	}
	return anns
}

func truncateJA(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

func formatDate(s string) string {
	if len(s) >= 8 {
		return s[:4] + "-" + s[4:6] + "-" + s[6:8]
	}
	return s
}
