package jrwatcher

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/WJQSERVER-STUDIO/httpc"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var areaConfigs = map[Area]struct {
	PageURL string
	JSONURL string
}{
	AreaJREastKanto: {
		PageURL: "https://traininfo.jreast.co.jp/train_info/kanto.aspx",
		JSONURL: "https://www.jreast.co.jp/traininfojp/kanto.json",
	},
	AreaJREastTohoku: {
		PageURL: "https://traininfo.jreast.co.jp/train_info/tohoku.aspx",
		JSONURL: "https://www.jreast.co.jp/traininfojp/tohoku.json",
	},
	AreaJREastShinetsu: {
		PageURL: "https://traininfo.jreast.co.jp/train_info/shinetsu.aspx",
		JSONURL: "https://www.jreast.co.jp/traininfojp/shinetsu.json",
	},
	AreaJREastExpress: {
		PageURL: "https://traininfo.jreast.co.jp/train_info/chyokyori.aspx",
	},
	AreaJREastShinkansen: {
		PageURL: "https://traininfo.jreast.co.jp/train_info/shinkansen.aspx",
	},
}

var timeRe = regexp.MustCompile(`(\d+)年(\d+)月(\d+)日 (\d+)時(\d+)分`)

type rawAnnouncement struct {
	ID           string `json:"id"`
	ContentClass string `json:"contentclass"`
	Title        string `json:"title"`
	ServerFile   string `json:"server_filename"`
	MetaInfo     struct {
		Dummy       string `json:"dummy"`
		FileSize    string `json:"file_size"`
		InfoMessage string `json:"infomessage"`
		LinkName    string `json:"link_name"`
		LinkText    string `json:"link_text"`
		ReleaseDate string `json:"release_date"`
		Sort        string `json:"sort"`
	} `json:"metainfo"`
}

type scraper struct {
	client *httpc.Client
}

func newScraper() *scraper {
	return &scraper{client: httpc.New(httpc.WithTimeout(15 * time.Second))}
}

func (s *scraper) fetchBytes(url string) ([]byte, error) {
	resp, err := s.client.GET(url).
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36").
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8").
		SetHeader("Accept-Language", "ja,en-US;q=0.9,en;q=0.8").
		SetHeader("Referer", "https://www.jreast.co.jp/").
		Execute()
	if err != nil {
		return nil, fmt.Errorf("httpc get %s: %w", url, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body %s: %w", url, err)
	}
	return b, nil
}

func (s *scraper) fetchPage(area Area) ([]byte, error) {
	cfg, ok := areaConfigs[area]
	if !ok {
		return nil, fmt.Errorf("unknown area: %s", area)
	}
	return s.fetchBytes(cfg.PageURL)
}

func (s *scraper) fetchJSON(area Area) ([]byte, error) {
	cfg, ok := areaConfigs[area]
	if !ok {
		return nil, fmt.Errorf("unknown area: %s", area)
	}
	if cfg.JSONURL == "" {
		return nil, nil
	}
	b, err := s.fetchBytes(cfg.JSONURL)
	if err != nil {
		return nil, err
	}
	if len(b) < 100 && strings.Contains(string(b), "Access Denied") {
		return nil, nil
	}
	return b, nil
}

func parseUpdatedAt(htmlContent string) (time.Time, error) {
	m := timeRe.FindStringSubmatch(htmlContent)
	if m == nil {
		return time.Time{}, fmt.Errorf("updated time not found in page")
	}
	year := atoi(m[1])
	month := atoi(m[2])
	day := atoi(m[3])
	hour := atoi(m[4])
	min := atoi(m[5])
	return time.Date(year, time.Month(month), day, hour, min, 0, 0, time.FixedZone("JST", 9*60*60)), nil
}

func parseRoutes(htmlContent string, area Area) ([]Route, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("html parse: %w", err)
	}

	items := findAll(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.DataAtom == atom.Li && hasClass(n, "traininfo-routes__table__item")
	})

	routes := make([]Route, 0, len(items))
	for _, item := range items {
		r := Route{Area: area}

		if nameNode := findFirst(item, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.DataAtom == atom.Span &&
				(hasClass(n, "traininfo-routes__name") || hasClass(n, "name"))
		}); nameNode != nil {
			r.Name = collectText(nameNode)
		}

		if aNode := findFirst(item, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.DataAtom == atom.A
		}); aNode != nil {
			href := getAttr(aNode, "href")
			r.LineID = extractID(href)

			if statusNode := findFirst(aNode, func(n *html.Node) bool {
				return n.Type == html.ElementNode && n.DataAtom == atom.P && hasClassWithPrefix(n, "traininfo-routes__status")
			}); statusNode != nil {
				r.Status = parseRouteStatus(statusNode)
			}

			if noteNode := findFirst(aNode, func(n *html.Node) bool {
				return n.Type == html.ElementNode && n.DataAtom == atom.P && hasClass(n, "traininfo-routes__note")
			}); noteNode != nil {
				r.Note = collectText(noteNode)
			}
		}

		if r.Name != "" {
			routes = append(routes, r)
		}
	}

	return routes, nil
}

func parseRouteStatus(n *html.Node) RouteStatus {
	classes := getAttr(n, "class")
	for _, c := range strings.Fields(classes) {
		switch c {
		case "normal":
			return RouteStatusNormal
		case "delay":
			return RouteStatusDelay
		case "info":
			return RouteStatusInfo
		case "adjust":
			return RouteStatusAdjust
		}
	}
	return RouteStatusUnknown
}

func parseAnnouncements(data []byte) ([]Announcement, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var raw []rawAnnouncement
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	var anns []Announcement
	for _, r := range raw {
		if r.MetaInfo.Dummy == "1" {
			continue
		}
		a := Announcement{
			Date:    r.MetaInfo.ReleaseDate,
			Message: r.MetaInfo.InfoMessage,
		}
		if r.MetaInfo.LinkName != "" {
			a.PDFURL = "https://www.jreast.co.jp/" + r.MetaInfo.LinkName
		}
		anns = append(anns, a)
	}
	return anns, nil
}

func hasClass(n *html.Node, class string) bool {
	return hasClassWithPrefix(n, class)
}

func hasClassWithPrefix(n *html.Node, prefix string) bool {
	c := getAttr(n, "class")
	if c == "" {
		return false
	}
	for _, token := range strings.Fields(c) {
		if token == prefix {
			return true
		}
	}
	return false
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func collectText(n *html.Node) string {
	var b strings.Builder
	collectTextRec(n, &b)
	return strings.TrimSpace(b.String())
}

func collectTextRec(n *html.Node, b *strings.Builder) {
	if n.Type == html.TextNode {
		b.WriteString(strings.TrimSpace(n.Data))
		b.WriteRune(' ')
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectTextRec(c, b)
	}
}

func findFirst(root *html.Node, pred func(*html.Node) bool) *html.Node {
	if pred(root) {
		return root
	}
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirst(c, pred); found != nil {
			return found
		}
	}
	return nil
}

func findAll(root *html.Node, pred func(*html.Node) bool) []*html.Node {
	var result []*html.Node
	if pred(root) {
		result = append(result, root)
	}
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findAll(c, pred)...)
	}
	return result
}

func extractID(href string) string {
	for _, prefix := range []string{"lineid=", "group="} {
		if idx := strings.Index(href, prefix); idx >= 0 {
			v := href[idx+len(prefix):]
			if amp := strings.IndexByte(v, '&'); amp >= 0 {
				v = v[:amp]
			}
			return v
		}
	}
	return ""
}

func atoi(s string) int {
	v := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			v = v*10 + int(r-'0')
		}
	}
	return v
}
