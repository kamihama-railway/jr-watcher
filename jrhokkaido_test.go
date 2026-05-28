package jrwatcher

import "testing"

func TestFetchJRHokkaido(t *testing.T) {
	info, err := FetchJRHokkaidoTrainInfo()
	if err != nil {
		t.Fatalf("FetchJRHokkaidoTrainInfo failed: %v", err)
	}

	t.Logf("Area: %s", info.Area)
	t.Logf("Routes: %d", len(info.Routes))

	for _, r := range info.Routes {
		t.Logf("  [%s] %s (id=%s)", r.Status, r.Name, r.LineID)
	}

	if len(info.Routes) == 0 {
		t.Error("no routes parsed")
	}
}
