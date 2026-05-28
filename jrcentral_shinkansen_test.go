package jrwatcher

import "testing"

func TestFetchJRShinkansen(t *testing.T) {
	info, err := FetchJRCentralShinkansenTrainInfo()
	if err != nil {
		t.Fatalf("FetchJRCentralShinkansenTrainInfo failed: %v", err)
	}

	t.Logf("Area: %s", info.Area)
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
