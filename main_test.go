package main

import (
	"testing"
)

func TestRound2(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{2.628285, 2.63},
		{25.19574, 25.20},
		{21.44302, 21.44},
		{58.0, 58.0},
		{0.0, 0.0},
		{-1.2345, -1.23},
		{-1.2356, -1.24},
		{0.004, 0.0},
		{0.005, 0.01},
	}

	for _, tc := range tests {
		got := round2(tc.input)
		if got != tc.expected {
			t.Errorf("round2(%f) = %f; expected %f", tc.input, got, tc.expected)
		}
	}
}

func TestGetSystemStats(t *testing.T) {
	stats, err := getSystemStats()
	if err != nil {
		t.Fatalf("getSystemStats failed: %v", err)
	}

	// Verify CPU range
	if stats.CPUPercent < 0 || stats.CPUPercent > 100 {
		t.Errorf("expected CPU percent to be between 0 and 100, got %f", stats.CPUPercent)
	}

	// Verify Memory range
	if stats.MemoryPercent < 0 || stats.MemoryPercent > 100 {
		t.Errorf("expected Memory percent to be between 0 and 100, got %f", stats.MemoryPercent)
	}

	// Verify Disk range
	if stats.DiskPercent < 0 || stats.DiskPercent > 100 {
		t.Errorf("expected Disk percent to be between 0 and 100, got %f", stats.DiskPercent)
	}

	// Verify load averages are non-negative
	if stats.Load1 < 0 || stats.Load5 < 0 || stats.Load15 < 0 {
		t.Errorf("expected non-negative load averages, got Load1=%f, Load5=%f, Load15=%f", stats.Load1, stats.Load5, stats.Load15)
	}

	// Verify uptime
	if stats.Uptime <= 0 {
		t.Errorf("expected uptime to be greater than 0, got %d", stats.Uptime)
	}

	// Verify process count
	if stats.Procs <= 0 {
		t.Errorf("expected process count to be greater than 0, got %d", stats.Procs)
	}

	// Verify detailed memory and disk values
	if stats.MemoryUsedGB < 0 || stats.MemoryAvailGB <= 0 {
		t.Errorf("invalid memory metrics: used=%f, avail=%f", stats.MemoryUsedGB, stats.MemoryAvailGB)
	}
	if stats.DiskUsedGB < 0 || stats.DiskFreeGB <= 0 {
		t.Errorf("invalid disk metrics: used=%f, free=%f", stats.DiskUsedGB, stats.DiskFreeGB)
	}

	// Verify details
	if stats.Details.TotalMemoryGB <= 0 {
		t.Errorf("expected total memory details to be greater than 0, got %f", stats.Details.TotalMemoryGB)
	}

	if stats.Details.TotalDiskGB <= 0 {
		t.Errorf("expected total disk details to be greater than 0, got %f", stats.Details.TotalDiskGB)
	}

	if stats.Details.CPULogical <= 0 {
		t.Errorf("expected logical CPU cores to be greater than 0, got %d", stats.Details.CPULogical)
	}

	if stats.Details.Hostname == "" || stats.Details.OS == "" {
		t.Errorf("hostname or OS metadata is empty: hostname=%q, OS=%q", stats.Details.Hostname, stats.Details.OS)
	}

	t.Logf("Tested stats model: %+v", stats)
}
