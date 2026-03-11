package models

import "time"

// Vessel represents a Maersk vessel from the Vessels API.
type Vessel struct {
	VesselIMONumber   int
	CarrierVesselCode string
	VesselLongName    string
	VesselFlagCode    string
	VesselBuiltYear   int
	VesselCallSign    string
	VesselCapacityTEU int
}

// AISPosition represents real-time AIS position data from AISStream.io.
type AISPosition struct {
	MMSI               int
	ShipName           string
	Latitude           float64
	Longitude          float64
	COG                float64 // Course over ground (degrees)
	SOG                float64 // Speed over ground (knots)
	TrueHeading        int     // True heading (degrees)
	NavigationalStatus int     // AIS nav status (0-15)
	Timestamp          time.Time
}

// Snapshot holds the latest fetched data from both APIs.
type Snapshot struct {
	LastRefresh     time.Time
	RefreshDuration time.Duration
	Error           error
	Vessels         []Vessel
	Positions       map[string]AISPosition // Key: normalized ship name
	VesselByName    map[string]Vessel      // Key: normalized ship name
}
