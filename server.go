package jrwatcher

import (
	"net/http"

	"github.com/infinite-iroha/touka"
)

type Server struct {
	Engine *touka.Engine
	Addr   string
}

func NewServer() *Server {
	r := touka.New()

	s := &Server{
		Engine: r,
		Addr:   ":8080",
	}

	r.GET("/api/v1/traininfo", s.handleTrainInfo)
	r.GET("/api/v1/traininfo/:area", s.handleTrainInfoByArea)
	r.GET("/api/v1/routes/:area", s.handleRoutes)
	r.GET("/api/v1/routes/abnormal", s.handleAbnormalRoutes)
	r.GET("/api/v1/announcements/:area", s.handleAnnouncements)
	r.GET("/api/v1/areas", s.handleAllAreas)

	return s
}

func (s *Server) SetAddr(addr string) {
	s.Addr = addr
}

func (s *Server) Start() error {
	return s.Engine.Run(touka.WithAddr(s.Addr))
}

func (s *Server) handleTrainInfo(c *touka.Context) {
	info, err := FetchTrainInfo()
	if err != nil {
		c.GetLogger().Errorf("fetch kanto train info: %v", err)
		c.JSON(http.StatusInternalServerError, touka.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (s *Server) handleTrainInfoByArea(c *touka.Context) {
	area := Area(c.Param("area"))
	info, err := FetchTrainInfoByArea(area)
	if err != nil {
		c.GetLogger().Errorf("fetch %s train info: %v", area, err)
		c.JSON(http.StatusInternalServerError, touka.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (s *Server) handleRoutes(c *touka.Context) {
	area := Area(c.Param("area"))
	routes, err := FetchRoutes(area)
	if err != nil {
		c.GetLogger().Errorf("fetch %s routes: %v", area, err)
		c.JSON(http.StatusInternalServerError, touka.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, routes)
}

func (s *Server) handleAnnouncements(c *touka.Context) {
	area := Area(c.Param("area"))
	anns, err := FetchAnnouncements(area)
	if err != nil {
		c.GetLogger().Errorf("fetch %s announcements: %v", area, err)
		c.JSON(http.StatusInternalServerError, touka.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, anns)
}

func (s *Server) handleAllAreas(c *touka.Context) {
	all, err := FetchAllAreas()
	if err != nil {
		c.GetLogger().Errorf("fetch all areas: %v", err)
	}
	c.JSON(http.StatusOK, all)
}

func (s *Server) handleAbnormalRoutes(c *touka.Context) {
	all, err := FetchAllAreas()
	if err != nil {
		c.GetLogger().Errorf("fetch all areas: %v", err)
	}

	type abnormalEntry struct {
		Area  Area   `json:"area"`
		Route Route  `json:"route"`
	}

	var result []abnormalEntry
	for area, info := range all {
		for _, r := range info.Routes {
			if r.Status != RouteStatusNormal && r.Status != RouteStatusUnknown {
				result = append(result, abnormalEntry{Area: area, Route: r})
			}
		}
	}

	c.JSON(http.StatusOK, result)
}
