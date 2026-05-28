package jrwatcher

import "time"

type RouteStatus string

const (
	RouteStatusNormal  RouteStatus = "normal"
	RouteStatusDelay   RouteStatus = "delay"
	RouteStatusInfo    RouteStatus = "info"
	RouteStatusAdjust  RouteStatus = "adjust"
	RouteStatusUnknown RouteStatus = "unknown"
)

type Area string

const (
	AreaJREastKanto      Area = "jr-east/kanto"
	AreaJREastTohoku     Area = "jr-east/tohoku"
	AreaJREastShinetsu   Area = "jr-east/shinetsu"
	AreaJREastExpress    Area = "jr-east/express"
	AreaJREastShinkansen Area = "jr-east/shinkansen"
	AreaJRCentral        Area = "jr-central/zairaisen"
	AreaJRCentralShinkan Area = "jr-central/shinkansen"
	AreaJRWest           Area = "jr-west"
)

type Route struct {
	Name   string      `json:"name"`
	LineID string      `json:"lineid"`
	Status RouteStatus `json:"status"`
	Note   string      `json:"note,omitempty"`
	Area   Area        `json:"area"`
}

type Announcement struct {
	Date    string `json:"date"`
	Message string `json:"message"`
	PDFURL  string `json:"pdf_url,omitempty"`
}

type TrainInfo struct {
	Area          Area           `json:"area"`
	UpdatedAt     time.Time      `json:"updated_at"`
	Routes        []Route        `json:"routes"`
	Announcements []Announcement `json:"announcements"`
}
