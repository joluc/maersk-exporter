package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joluc/maersk-exporter/pkg/config"
	"github.com/joluc/maersk-exporter/pkg/models"
)

// AISMessage represents the structure of messages from AISStream.io.
type AISMessage struct {
	MetaData struct {
		MMSI      int     `json:"MMSI"`
		ShipName  string  `json:"ShipName"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		TimeUTC   string  `json:"time_utc"`
	} `json:"MetaData"`
	Message struct {
		PositionReport struct {
			Cog                float64 `json:"Cog"`
			Sog                float64 `json:"Sog"`
			TrueHeading        int     `json:"TrueHeading"`
			NavigationalStatus int     `json:"NavigationalStatus"`
		} `json:"PositionReport"`
	} `json:"Message"`
}

// AISStreamClient manages the WebSocket connection to AISStream.io.
type AISStreamClient struct {
	cfg       config.Config
	positions map[string]models.AISPosition
	mu        sync.RWMutex
}

// NewAISStreamClient creates a new AISStream client.
func NewAISStreamClient(cfg config.Config) *AISStreamClient {
	return &AISStreamClient{
		cfg:       cfg,
		positions: make(map[string]models.AISPosition),
	}
}

// Run starts the WebSocket client with auto-reconnect.
func (c *AISStreamClient) Run(ctx context.Context) {
	for {
		if err := c.connectAndListen(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("aisstream error, reconnecting", "error", err)
			time.Sleep(5 * time.Second)
		}
	}
}

// connectAndListen establishes WebSocket connection and processes messages.
func (c *AISStreamClient) connectAndListen(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 45 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, "wss://stream.aisstream.io/v0/stream", nil)
	if err != nil {
		return fmt.Errorf("dialing websocket: %w", err)
	}
	defer conn.Close()

	// Context for managing ping goroutine
	pingCtx, pingCancel := context.WithCancel(ctx)
	defer pingCancel()

	// Set up ping/pong to keep connection alive
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-pingCtx.Done():
				return
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					return
				}
			}
		}
	}()

	// Send subscription
	subscription := map[string]interface{}{
		"APIKey":             c.cfg.AISStreamAPIKey,
		"BoundingBoxes":      [][][]float64{{{-90, -180}, {90, 180}}},
		"FilterMessageTypes": []string{"PositionReport"},
	}

	if err := conn.WriteJSON(subscription); err != nil {
		return fmt.Errorf("sending subscription: %w", err)
	}

	slog.Info("connected to aisstream.io", "filters", c.cfg.FilterVesselNames)

	// Read loop
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("reading message: %w", err)
		}

		var aisMsg AISMessage
		if err := json.Unmarshal(message, &aisMsg); err != nil {
			slog.Warn("failed to decode AIS message", "error", err)
			continue
		}

		// Filter by vessel name
		if !c.matchesFilter(aisMsg.MetaData.ShipName) {
			continue
		}

		// Store position
		pos := c.convertToPosition(aisMsg)
		c.mu.Lock()
		c.positions[normalizeName(aisMsg.MetaData.ShipName)] = pos
		c.mu.Unlock()
	}
}

// matchesFilter checks if the ship name matches configured filters.
func (c *AISStreamClient) matchesFilter(shipName string) bool {
	if len(c.cfg.FilterVesselNames) == 0 {
		return true
	}

	normalized := strings.ToUpper(strings.TrimSpace(shipName))
	for _, filter := range c.cfg.FilterVesselNames {
		if strings.Contains(normalized, strings.ToUpper(filter)) {
			return true
		}
	}
	return false
}

// convertToPosition converts AIS message to internal position model.
func (c *AISStreamClient) convertToPosition(msg AISMessage) models.AISPosition {
	timestamp := time.Now()
	if msg.MetaData.TimeUTC != "" {
		if t, err := time.Parse(time.RFC3339, msg.MetaData.TimeUTC); err == nil {
			timestamp = t
		}
	}

	return models.AISPosition{
		MMSI:               msg.MetaData.MMSI,
		ShipName:           strings.TrimSpace(msg.MetaData.ShipName),
		Latitude:           msg.MetaData.Latitude,
		Longitude:          msg.MetaData.Longitude,
		COG:                msg.Message.PositionReport.Cog,
		SOG:                msg.Message.PositionReport.Sog,
		TrueHeading:        msg.Message.PositionReport.TrueHeading,
		NavigationalStatus: msg.Message.PositionReport.NavigationalStatus,
		Timestamp:          timestamp,
	}
}

// GetPositions returns a copy of all current positions.
func (c *AISStreamClient) GetPositions() map[string]models.AISPosition {
	c.mu.RLock()
	defer c.mu.RUnlock()

	positions := make(map[string]models.AISPosition, len(c.positions))
	for k, v := range c.positions {
		positions[k] = v
	}
	return positions
}
