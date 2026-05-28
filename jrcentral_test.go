package jrwatcher

import "testing"

func TestFetchJRCentral(t *testing.T) {
	info, err := FetchJRCentralTrainInfo()
	if err != nil {
		t.Fatalf("FetchJRCentralTrainInfo failed: %v", err)
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

func TestFetchJRCentralViaArea(t *testing.T) {
	info, err := FetchTrainInfoByArea(AreaJRCentral)
	if err != nil {
		t.Fatalf("FetchTrainInfoByArea(jr-central) failed: %v", err)
	}
	if len(info.Routes) == 0 {
		t.Error("no routes parsed")
	}
	if info.Area != AreaJRCentral {
		t.Errorf("expected area jr-central, got %s", info.Area)
	}
	t.Logf("JR Central via area API: %d routes", len(info.Routes))
}
