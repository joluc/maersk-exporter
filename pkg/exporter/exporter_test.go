package exporter

import (
	"strings"
	"testing"

	"github.com/joluc/maersk-exporter/pkg/models"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"MAERSK DENVER", "MAERSK DENVER"},
		{"  MAERSK DENVER  ", "MAERSK DENVER"},
		{"maersk denver", "MAERSK DENVER"},
		{"  Maersk Denver  ", "MAERSK DENVER"},
		{"MAERSK\nDENVER", "MAERSK\nDENVER"},
	}

	for _, tt := range tests {
		result := normalizeName(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeName(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

func TestIndexByName(t *testing.T) {
	vessels := []models.Vessel{
		{VesselIMONumber: 123, VesselLongName: "MAERSK DENVER"},
		{VesselIMONumber: 456, VesselLongName: "MAERSK BOSTON"},
		{VesselIMONumber: 789, VesselLongName: "  MSC VESSEL  "},
	}

	index := indexByName(vessels)

	if len(index) != 3 {
		t.Errorf("expected index length 3, got %d", len(index))
	}

	if v, ok := index["MAERSK DENVER"]; !ok || v.VesselIMONumber != 123 {
		t.Errorf("expected MAERSK DENVER with IMO 123, got %v", v)
	}

	if v, ok := index["MSC VESSEL"]; !ok || v.VesselIMONumber != 789 {
		t.Errorf("expected MSC VESSEL (normalized) with IMO 789, got %v", v)
	}
}

func TestEscapeLabel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`simple`, `simple`},
		{`with"quote`, `with\"quote`},
		{"with\nline", `with\nline`},
		{`with\backslash`, `with\\backslash`},
		{`all"three\chars` + "\n", `all\"three\\chars\n`},
	}

	for _, tt := range tests {
		result := escapeLabel(tt.input)
		if result != tt.expected {
			t.Errorf("escapeLabel(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

func TestNavStatusLabel(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{0, "under_way_using_engine"},
		{1, "at_anchor"},
		{5, "moored"},
		{15, "not_defined"},
		{99, "unknown"},
		{-1, "unknown"},
	}

	for _, tt := range tests {
		result := navStatusLabel(tt.status)
		if result != tt.expected {
			t.Errorf("navStatusLabel(%d): expected %q, got %q", tt.status, tt.expected, result)
		}
	}
}

func TestWriteMetric(t *testing.T) {
	var b strings.Builder
	writeMetric(&b, "test_metric", "Test metric help", "gauge", "42.0")

	output := b.String()
	if !strings.Contains(output, "# HELP test_metric Test metric help") {
		t.Error("expected HELP line in output")
	}
	if !strings.Contains(output, "# TYPE test_metric gauge") {
		t.Error("expected TYPE line in output")
	}
	if !strings.Contains(output, "test_metric 42.0") {
		t.Error("expected metric value line in output")
	}
}

func TestWriteMetricHeader(t *testing.T) {
	var b strings.Builder
	writeMetricHeader(&b, "test_metric", "Test metric help", "counter")

	output := b.String()
	if !strings.Contains(output, "# HELP test_metric Test metric help") {
		t.Error("expected HELP line in output")
	}
	if !strings.Contains(output, "# TYPE test_metric counter") {
		t.Error("expected TYPE line in output")
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}
