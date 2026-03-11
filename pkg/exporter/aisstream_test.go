package exporter

import (
	"testing"

	"github.com/joluc/maersk-exporter/pkg/config"
	"github.com/joluc/maersk-exporter/pkg/models"
)

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		name     string
		filters  []string
		shipName string
		expected bool
	}{
		{"empty filter matches all", []string{}, "ANY SHIP", true},
		{"exact match", []string{"MAERSK"}, "MAERSK DENVER", true},
		{"case insensitive", []string{"MAERSK"}, "maersk denver", true},
		{"with spaces", []string{"MAERSK"}, "  MAERSK DENVER  ", true},
		{"no match", []string{"MAERSK"}, "MSC VESSEL", false},
		{"multiple filters", []string{"MAERSK", "MSC"}, "MSC VESSEL", true},
		{"partial match", []string{"MAERSK"}, "THE MAERSK DENVER", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &AISStreamClient{
				cfg: config.Config{FilterVesselNames: tt.filters},
			}
			result := client.matchesFilter(tt.shipName)
			if result != tt.expected {
				t.Errorf("matchesFilter(%q) with filters %v: expected %v, got %v",
					tt.shipName, tt.filters, tt.expected, result)
			}
		})
	}
}

func TestConvertToPosition(t *testing.T) {
	client := &AISStreamClient{}

	msg := AISMessage{}
	msg.MetaData.MMSI = 123456789
	msg.MetaData.ShipName = "  TEST VESSEL  "
	msg.MetaData.Latitude = 55.6761
	msg.MetaData.Longitude = 12.5683
	msg.MetaData.TimeUTC = "2026-03-11T12:00:00Z"
	msg.Message.PositionReport.Cog = 180.5
	msg.Message.PositionReport.Sog = 15.2
	msg.Message.PositionReport.TrueHeading = 180
	msg.Message.PositionReport.NavigationalStatus = 0

	pos := client.convertToPosition(msg)

	if pos.MMSI != 123456789 {
		t.Errorf("expected MMSI 123456789, got %d", pos.MMSI)
	}
	if pos.ShipName != "TEST VESSEL" {
		t.Errorf("expected trimmed ship name 'TEST VESSEL', got %q", pos.ShipName)
	}
	if pos.Latitude != 55.6761 {
		t.Errorf("expected latitude 55.6761, got %.4f", pos.Latitude)
	}
	if pos.Longitude != 12.5683 {
		t.Errorf("expected longitude 12.5683, got %.4f", pos.Longitude)
	}
	if pos.COG != 180.5 {
		t.Errorf("expected COG 180.5, got %.1f", pos.COG)
	}
	if pos.SOG != 15.2 {
		t.Errorf("expected SOG 15.2, got %.1f", pos.SOG)
	}
	if pos.TrueHeading != 180 {
		t.Errorf("expected heading 180, got %d", pos.TrueHeading)
	}
	if pos.NavigationalStatus != 0 {
		t.Errorf("expected nav status 0, got %d", pos.NavigationalStatus)
	}
}

func TestGetPositions(t *testing.T) {
	client := NewAISStreamClient(config.Config{})

	// Add some positions
	client.mu.Lock()
	client.positions["VESSEL1"] = models.AISPosition{MMSI: 111, ShipName: "VESSEL 1"}
	client.positions["VESSEL2"] = models.AISPosition{MMSI: 222, ShipName: "VESSEL 2"}
	client.mu.Unlock()

	// Get positions
	positions := client.GetPositions()

	if len(positions) != 2 {
		t.Errorf("expected 2 positions, got %d", len(positions))
	}

	if pos, ok := positions["VESSEL1"]; !ok || pos.MMSI != 111 {
		t.Error("expected VESSEL1 with MMSI 111")
	}

	// Verify it's a copy (modifying returned map shouldn't affect original)
	positions["VESSEL3"] = models.AISPosition{MMSI: 333}

	client.mu.RLock()
	if len(client.positions) != 2 {
		t.Error("modifying returned positions affected original")
	}
	client.mu.RUnlock()
}
