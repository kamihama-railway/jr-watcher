package jrwatcher

import "testing"

func TestFetchJRWest(t *testing.T) {
	info, err := FetchJRWestTrainInfo()
	if err != nil {
		t.Fatalf("FetchJRWestTrainInfo failed: %v", err)
	}

	t.Logf("Area: %s", info.Area)
	t.Logf("UpdatedAt: %s", info.UpdatedAt.Format("2006-01-02 15:04"))
	t.Logf("Routes: %d", len(info.Routes))
	t.Logf("Announcements: %d", len(info.Announcements))

	for _, r := range info.Routes {
		t.Logf("  [%s] %s (id=%s)", r.Status, r.Name, r.LineID)
		if r.Note != "" {
			t.Logf("    note: %s", r.Note)
		}
	}

	for _, a := range info.Announcements {
		t.Logf("  [%s] %s", a.Date, a.Message)
	}

	if len(info.Routes) == 0 {
		t.Error("no routes parsed")
	}
}

func TestFetchJRWestViaArea(t *testing.T) {
	info, err := FetchTrainInfoByArea(AreaJRWest)
	if err != nil {
		t.Fatalf("FetchTrainInfoByArea(jr-west) failed: %v", err)
	}
	if len(info.Routes) == 0 {
		t.Error("no routes parsed")
	}
	t.Logf("JR West via area API: %d routes", len(info.Routes))
}
