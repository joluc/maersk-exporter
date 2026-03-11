package models

import (
	"testing"
	"time"
)

func TestVessel(t *testing.T) {
	v := Vessel{
		VesselIMONumber:   9123456,
		CarrierVesselCode: "TEST123",
		VesselLongName:    "TEST VESSEL",
		VesselFlagCode:    "DK",
		VesselBuiltYear:   2020,
		VesselCallSign:    "ABCD",
		VesselCapacityTEU: 10000,
	}

	if v.VesselIMONumber != 9123456 {
		t.Errorf("expected IMO 9123456, got %d", v.VesselIMONumber)
	}
	if v.VesselLongName != "TEST VESSEL" {
		t.Errorf("expected name 'TEST VESSEL', got %s", v.VesselLongName)
	}
}

func TestAISPosition(t *testing.T) {
	now := time.Now()
	pos := AISPosition{
		MMSI:               123456789,
		ShipName:           "TEST SHIP",
		Latitude:           55.6761,
		Longitude:          12.5683,
		COG:                180.5,
		SOG:                15.2,
		TrueHeading:        180,
		NavigationalStatus: 0,
		Timestamp:          now,
	}

	if pos.MMSI != 123456789 {
		t.Errorf("expected MMSI 123456789, got %d", pos.MMSI)
	}
	if pos.Latitude != 55.6761 {
		t.Errorf("expected latitude 55.6761, got %.4f", pos.Latitude)
	}
	if pos.Longitude != 12.5683 {
		t.Errorf("expected longitude 12.5683, got %.4f", pos.Longitude)
	}
}

func TestSnapshot(t *testing.T) {
	now := time.Now()
	s := Snapshot{
		LastRefresh:     now,
		RefreshDuration: 500 * time.Millisecond,
		Vessels: []Vessel{
			{VesselIMONumber: 123, VesselLongName: "VESSEL 1"},
			{VesselIMONumber: 456, VesselLongName: "VESSEL 2"},
		},
		Positions: map[string]AISPosition{
			"VESSEL 1": {MMSI: 111, ShipName: "VESSEL 1"},
		},
		VesselByName: map[string]Vessel{
			"VESSEL 1": {VesselIMONumber: 123, VesselLongName: "VESSEL 1"},
		},
	}

	if len(s.Vessels) != 2 {
		t.Errorf("expected 2 vessels, got %d", len(s.Vessels))
	}
	if len(s.Positions) != 1 {
		t.Errorf("expected 1 position, got %d", len(s.Positions))
	}
	if s.RefreshDuration != 500*time.Millisecond {
		t.Errorf("expected duration 500ms, got %v", s.RefreshDuration)
	}
}
