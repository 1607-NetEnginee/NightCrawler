package core

import (
	"runtime"
	"strings"
	"testing"
)

func TestScanRequest_Validate_NoTargets(t *testing.T) {
	r := ScanRequest{}
	err := r.Validate()
	if err == nil || !strings.Contains(err.Error(), "no targets") {
		t.Fatalf("expected 'no targets' error, got: %v", err)
	}
}

func TestScanRequest_Validate_DefaultsApplied(t *testing.T) {
	r := ScanRequest{Targets: []string{"example.com"}}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Profile != "default" {
		t.Errorf("profile not defaulted: got %q", r.Profile)
	}
	if r.FailOn != "high" {
		t.Errorf("fail-on not defaulted: got %q", r.FailOn)
	}
	if r.Concurrency != runtime.NumCPU() {
		t.Errorf("concurrency not defaulted to NumCPU: got %d", r.Concurrency)
	}
	if len(r.OutputFormats) == 0 {
		t.Errorf("output formats not defaulted")
	}
}

func TestScanRequest_Validate_BadProfile(t *testing.T) {
	r := ScanRequest{Targets: []string{"example.com"}, Profile: "nope"}
	err := r.Validate()
	if err == nil || !strings.Contains(err.Error(), "unknown profile") {
		t.Fatalf("expected unknown profile error, got: %v", err)
	}
}

func TestScanRequest_Validate_BadFailOn(t *testing.T) {
	r := ScanRequest{Targets: []string{"example.com"}, FailOn: "wat"}
	err := r.Validate()
	if err == nil || !strings.Contains(err.Error(), "fail-on") {
		t.Fatalf("expected fail-on error, got: %v", err)
	}
}

func TestScanRequest_Validate_NegativeConcurrencyRejected(t *testing.T) {
	r := ScanRequest{Targets: []string{"example.com"}, Concurrency: -3}
	err := r.Validate()
	if err == nil {
		t.Fatal("expected validation error for negative concurrency")
	}
}

func TestSeverityCounts_AddAndTotal(t *testing.T) {
	var c SeverityCounts
	c.Add("low")
	c.Add("low")
	c.Add("high")
	c.Add("critical")
	c.Add("info")
	if c.Total() != 5 {
		t.Fatalf("total = %d, want 5", c.Total())
	}
	if c.Low != 2 {
		t.Errorf("low = %d, want 2", c.Low)
	}
	if c.Critical != 1 {
		t.Errorf("critical = %d, want 1", c.Critical)
	}
}
