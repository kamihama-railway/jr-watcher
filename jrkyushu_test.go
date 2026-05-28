package jrwatcher

import "testing"

func TestFetchJRKyushu(t *testing.T) {
	info, err := FetchJRKyushuTrainInfo()
	if err != nil {
		t.Fatalf("FetchJRKyushuTrainInfo failed: %v", err)
	}
	t.Logf("Area: %s, Routes: %d", info.Area, len(info.Routes))
	for _, r := range info.Routes {
		t.Logf("  [%s] %s (id=%s)", r.Status, r.Name, r.LineID)
	}
	if len(info.Routes) == 0 {
		t.Error("no routes parsed")
	}
}
