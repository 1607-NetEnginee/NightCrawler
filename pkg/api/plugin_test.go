package api

import "testing"

func TestSeverity_Score(t *testing.T) {
	cases := []struct {
		s    Severity
		want float64
	}{
		{SeverityCritical, 25},
		{SeverityHigh, 10},
		{SeverityMedium, 4},
		{SeverityLow, 1},
		{SeverityInfo, 0},
		{Severity("garbage"), 0},
	}
	for _, c := range cases {
		got := c.s.Score()
		if got != c.want {
			t.Errorf("Score(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}

func TestSeverity_OrderingIsStable(t *testing.T) {
	// Documenting the contract: these string values are the wire
	// format. Changing them is a breaking change. If this test is
	// ever updated, bump MAJOR.
	if SeverityCritical != "critical" {
		t.Errorf("SeverityCritical wire format changed")
	}
	if SeverityHigh != "high" {
		t.Errorf("SeverityHigh wire format changed")
	}
	if SeverityMedium != "medium" {
		t.Errorf("SeverityMedium wire format changed")
	}
	if SeverityLow != "low" {
		t.Errorf("SeverityLow wire format changed")
	}
	if SeverityInfo != "info" {
		t.Errorf("SeverityInfo wire format changed")
	}
}
