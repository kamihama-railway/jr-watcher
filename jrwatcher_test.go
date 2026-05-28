package jrwatcher

import "testing"

func TestFetchKanto(t *testing.T) {
	info, err := FetchTrainInfo()
	if err != nil {
		t.Fatalf("FetchTrainInfo failed: %v", err)
	}
	verifyTrainInfo(t, info)
}

func TestFetchKantoByArea(t *testing.T) {
	info, err := FetchTrainInfoByArea(AreaJREastKanto)
	if err != nil {
		t.Fatalf("FetchTrainInfoByArea(jr-east/kanto) failed: %v", err)
	}
	verifyTrainInfo(t, info)
	if info.Area != AreaJREastKanto {
		t.Errorf("expected area %s, got %s", AreaJREastKanto, info.Area)
	}
}

func TestFetchTohoku(t *testing.T) {
	info, err := FetchTrainInfoByArea(AreaJREastTohoku)
	if err != nil {
		t.Fatalf("FetchTrainInfoByArea(jr-east/tohoku) failed: %v", err)
	}
	verifyTrainInfo(t, info)
	if info.Area != AreaJREastTohoku {
		t.Errorf("expected area %s, got %s", AreaJREastTohoku, info.Area)
	}
}

func TestFetchShinetsu(t *testing.T) {
	info, err := FetchTrainInfoByArea(AreaJREastShinetsu)
	if err != nil {
		t.Fatalf("FetchTrainInfoByArea(jr-east/shinetsu) failed: %v", err)
	}
	verifyTrainInfo(t, info)
}

func TestFetchExpress(t *testing.T) {
	info, err := FetchTrainInfoByArea(AreaJREastExpress)
	if err != nil {
		t.Fatalf("FetchTrainInfoByArea(jr-east/express) failed: %v", err)
	}
	verifyTrainInfo(t, info)
}

func verifyTrainInfo(t *testing.T, info *TrainInfo) {
	t.Helper()
	t.Logf("Area: %s", info.Area)
	t.Logf("UpdatedAt: %s", info.UpdatedAt.Format("2006-01-02 15:04"))
	t.Logf("Routes: %d", len(info.Routes))
	t.Logf("Announcements: %d", len(info.Announcements))

	for _, r := range info.Routes {
		t.Logf("  [%s] %s (id=%s) area=%s", r.Status, r.Name, r.LineID, r.Area)
		if r.Note != "" {
			t.Logf("    note: %s", r.Note)
		}
	}

	for _, a := range info.Announcements {
		t.Logf("  [%s] %s", a.Date, a.Message)
		if a.PDFURL != "" {
			t.Logf("    pdf: %s", a.PDFURL)
		}
	}

	if len(info.Routes) == 0 {
		t.Error("no routes parsed")
	}
	if info.UpdatedAt.IsZero() {
		t.Error("updated at is zero")
	}
}

func TestFetchRoutes(t *testing.T) {
	routes, err := FetchRoutes(AreaJREastKanto)
	if err != nil {
		t.Fatalf("FetchRoutes failed: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("no routes returned")
	}
}

func TestFetchAnnouncements(t *testing.T) {
	anns, err := FetchAnnouncements(AreaJREastKanto)
	if err != nil {
		t.Fatalf("FetchAnnouncements failed: %v", err)
	}
	t.Logf("Announcements: %d", len(anns))
}
