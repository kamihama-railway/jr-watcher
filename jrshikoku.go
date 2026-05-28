package jrwatcher

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/WJQSERVER-STUDIO/httpc"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const jrShikokuURL = "https://www.jr-shikoku.co.jp/info/"

type jrShikokuScraper struct{ client *httpc.Client }

func newJRShikokuScraper() *jrShikokuScraper {
	return &jrShikokuScraper{client: httpc.New(httpc.WithTimeout(15 * time.Second))}
}

func (s *jrShikokuScraper) fetchPage() ([]byte, error) {
	resp, err := s.client.GET(jrShikokuURL).
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36").
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8").
		SetHeader("Accept-Language", "ja,en-US;q=0.9,en;q=0.8").
		Execute()
	if err != nil {
		return nil, fmt.Errorf("httpc get: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

var jrShikokuLines = []struct {
	Name   string
	LineID string
}{
	{"予讃線（松山～宇和島）", "yosan-uwajima"},
	{"予讃線（高松～松山）", "yosan-takamatsu"},
	{"予讃線（向井原～内子）", "yosan-mukaihara"},
	{"瀬戸大橋線", "seto-ohashi"},
	{"土讃線（多度津～窪川）", "dosan-tadotsu"},
	{"土讃線（高松～多度津）", "dosan-takamatsu"},
	{"高徳線", "kotoku"},
	{"徳島線", "tokushima"},
	{"牟岐線", "mugi"},
	{"鳴門線", "naruto"},
	{"予土線", "yodo"},
}

func FetchJRShikokuTrainInfo() (*TrainInfo, error) {
	s := newJRShikokuScraper()

	raw, err := s.fetchPage()
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}

	content := string(raw)

	// Check if there's delay info
	hasDelay := !strings.Contains(content, "現在、遅れ等の情報はありません。")

	routes := make([]Route, 0, len(jrShikokuLines))

	if hasDelay {
		// Parse delay info from HTML
		delays := parseJRShikokuDelayInfo(content)

		for _, line := range jrShikokuLines {
			r := Route{
				Name:   line.Name,
				LineID: line.LineID,
				Area:   "jr-shikoku",
			}
			if note, ok := delays[line.Name]; ok {
				r.Status = RouteStatusDelay
				r.Note = note
			} else {
				r.Status = RouteStatusNormal
			}
			routes = append(routes, r)
		}
	} else {
		for _, line := range jrShikokuLines {
			routes = append(routes, Route{
				Name:   line.Name,
				LineID: line.LineID,
				Area:   "jr-shikoku",
				Status: RouteStatusNormal,
			})
		}
	}

	return &TrainInfo{
		Area:      "jr-shikoku",
		UpdatedAt: time.Now(),
		Routes:    routes,
	}, nil
}

func parseJRShikokuDelayInfo(htmlContent string) map[string]string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil
	}

	// Find dl inside delay_info div
	dl := findFirst(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.DataAtom == atom.Dl &&
			hasClassWithPrefix(n, "delay_info") == false
	})

	// Look for the dl within the delay_info class
	delayInfo := findFirst(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.DataAtom == atom.Div && hasClass(n, "delay_info")
	})
	if delayInfo == nil {
		return nil
	}

	dl = findFirst(delayInfo, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.DataAtom == atom.Dl
	})
	if dl == nil {
		return nil
	}

	result := make(map[string]string)

	// Parse dt/dd pairs
	current := dl.FirstChild
	for current != nil {
		if current.Type == html.ElementNode && current.DataAtom == atom.Dt {
			lineName := collectText(current)
			dd := current.NextSibling
			for dd != nil && !(dd.Type == html.ElementNode && dd.DataAtom == atom.Dd) {
				dd = dd.NextSibling
			}
			if dd != nil {
				note := collectText(dd)
				result[lineName] = note
				current = dd.NextSibling
			} else {
				current = current.NextSibling
			}
		} else {
			current = current.NextSibling
		}
	}

	return result
}
