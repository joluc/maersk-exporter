package exporter

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joluc/maersk-exporter/pkg/config"
	"github.com/joluc/maersk-exporter/pkg/models"
)

type Exporter struct {
	client    *http.Client
	aisClient *AISStreamClient
	cfg       config.Config

	mu          sync.RWMutex
	snapshot    models.Snapshot
	scrapeCount atomic.Uint64
}

func New(cfg config.Config) *Exporter {
	return &Exporter{
		client:    &http.Client{Timeout: cfg.RequestTimeout},
		aisClient: NewAISStreamClient(cfg),
		cfg:       cfg,
	}
}

// Start begins the background processes (WebSocket + periodic refresh).
func (e *Exporter) Start(ctx context.Context) {
	go e.aisClient.Run(ctx)
	go e.RefreshLoop(ctx)
}

func (e *Exporter) RootHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "Maersk Fleet & Position Monitoring Exporter\nMetrics: %s\nHealth: /healthz\n", e.cfg.MetricsPath)
	}
}

func (e *Exporter) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	e.mu.RLock()
	lastRefresh := e.snapshot.LastRefresh
	hasError := e.snapshot.Error != nil
	e.mu.RUnlock()

	if lastRefresh.IsZero() || hasError {
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, "not ready\n")
		return
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "ok\n")
}

func (e *Exporter) MetricsHandler(w http.ResponseWriter, _ *http.Request) {
	e.scrapeCount.Add(1)

	e.mu.RLock()
	s := e.snapshot
	e.mu.RUnlock()

	var b strings.Builder
	b.Grow(8192)

	// Meta metrics
	up := 1.0
	if s.Error != nil {
		up = 0
	}
	writeMetric(&b, "maersk_up", "Whether the latest Maersk API refresh succeeded", "gauge",
		fmt.Sprintf("%.0f", up))
	writeMetric(&b, "maersk_last_refresh_timestamp_seconds", "Unix timestamp of last refresh", "gauge",
		fmt.Sprintf("%.0f", float64(s.LastRefresh.Unix())))
	writeMetric(&b, "maersk_refresh_duration_seconds", "Duration of latest refresh operation", "gauge",
		fmt.Sprintf("%.6f", s.RefreshDuration.Seconds()))
	writeMetric(&b, "maersk_scrapes_total", "Total number of /metrics scrapes", "counter",
		fmt.Sprintf("%d", e.scrapeCount.Load()))
	writeMetric(&b, "maersk_vessels_total", "Number of vessels tracked", "gauge",
		fmt.Sprintf("%d", len(s.Vessels)))
	writeMetric(&b, "maersk_positions_total", "Number of vessels with AIS positions", "gauge",
		fmt.Sprintf("%d", len(s.Positions)))

	// Per-vessel metadata metrics
	if len(s.Vessels) > 0 {
		writeMetricHeader(&b, "maersk_vessel_capacity_teu", "Vessel capacity in TEU", "gauge")
		writeMetricHeader(&b, "maersk_vessel_age_years", "Vessel age in years", "gauge")

		currentYear := time.Now().Year()
		for _, vessel := range s.Vessels {
			labels := fmt.Sprintf(`vessel_name="%s",imo="%d",flag="%s"`,
				escapeLabel(vessel.VesselLongName),
				vessel.VesselIMONumber,
				escapeLabel(vessel.VesselFlagCode))

			if vessel.VesselCapacityTEU > 0 {
				fmt.Fprintf(&b, "maersk_vessel_capacity_teu{%s} %d\n", labels, vessel.VesselCapacityTEU)
			}
			if vessel.VesselBuiltYear > 0 {
				fmt.Fprintf(&b, "maersk_vessel_age_years{%s} %d\n", labels, currentYear-vessel.VesselBuiltYear)
			}
		}
	}

	// Per-position metrics
	if len(s.Positions) > 0 {
		writeMetricHeader(&b, "maersk_vessel_latitude_degrees", "Vessel latitude in degrees", "gauge")
		writeMetricHeader(&b, "maersk_vessel_longitude_degrees", "Vessel longitude in degrees", "gauge")
		writeMetricHeader(&b, "maersk_vessel_speed_over_ground_knots", "Vessel speed over ground in knots", "gauge")
		writeMetricHeader(&b, "maersk_vessel_course_over_ground_degrees", "Vessel course over ground in degrees", "gauge")
		writeMetricHeader(&b, "maersk_vessel_heading_degrees", "Vessel true heading in degrees", "gauge")
		writeMetricHeader(&b, "maersk_vessel_position_age_seconds", "Age of position data in seconds", "gauge")

		now := time.Now()
		for _, pos := range s.Positions {
			labels := fmt.Sprintf(`vessel_name="%s",mmsi="%d",nav_status="%s"`,
				escapeLabel(pos.ShipName),
				pos.MMSI,
				navStatusLabel(pos.NavigationalStatus))

			fmt.Fprintf(&b, "maersk_vessel_latitude_degrees{%s} %.6f\n", labels, pos.Latitude)
			fmt.Fprintf(&b, "maersk_vessel_longitude_degrees{%s} %.6f\n", labels, pos.Longitude)
			fmt.Fprintf(&b, "maersk_vessel_speed_over_ground_knots{%s} %.2f\n", labels, pos.SOG)

			if pos.COG >= 0 {
				fmt.Fprintf(&b, "maersk_vessel_course_over_ground_degrees{%s} %.1f\n", labels, pos.COG)
			}
			if pos.TrueHeading >= 0 && pos.TrueHeading <= 359 {
				fmt.Fprintf(&b, "maersk_vessel_heading_degrees{%s} %d\n", labels, pos.TrueHeading)
			}

			age := now.Sub(pos.Timestamp).Seconds()
			if age >= 0 {
				fmt.Fprintf(&b, "maersk_vessel_position_age_seconds{%s} %.0f\n", labels, age)
			}
		}
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	io.WriteString(w, b.String())
}

func writeMetric(b *strings.Builder, metric, help, metricType, value string) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s %s\n%s %s\n", metric, help, metric, metricType, metric, value)
}

func writeMetricHeader(b *strings.Builder, metric, help, metricType string) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s %s\n", metric, help, metric, metricType)
}

func escapeLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

var navStatusLabels = [16]string{
	"under_way_using_engine", "at_anchor", "not_under_command", "restricted_manoeuvrability",
	"constrained_by_draught", "moored", "aground", "engaged_in_fishing",
	"under_way_sailing", "reserved_hsc", "reserved_wing", "reserved_future_1",
	"reserved_future_2", "reserved_future_3", "ais_sart", "not_defined",
}

func navStatusLabel(status int) string {
	if status >= 0 && status < len(navStatusLabels) {
		return navStatusLabels[status]
	}
	return "unknown"
}

func (e *Exporter) RefreshLoop(ctx context.Context) {
	e.Refresh(ctx)

	ticker := time.NewTicker(e.cfg.VesselsRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.Refresh(ctx)
		}
	}
}

func (e *Exporter) Refresh(ctx context.Context) {
	start := time.Now()

	vessels, err := fetchVessels(ctx, e.client, e.cfg)

	e.mu.Lock()
	defer e.mu.Unlock()

	if err != nil {
		slog.Error("refresh failed", "error", err)
		e.snapshot.LastRefresh = time.Now()
		e.snapshot.RefreshDuration = time.Since(start)
		e.snapshot.Error = err
		return
	}

	positions := e.aisClient.GetPositions()
	duration := time.Since(start)

	e.snapshot = models.Snapshot{
		LastRefresh:     time.Now(),
		RefreshDuration: duration,
		Vessels:         vessels,
		Positions:       positions,
		VesselByName:    indexByName(vessels),
	}

	slog.Info("refreshed vessels",
		"vessels_count", len(vessels),
		"positions_count", len(positions),
		"duration_ms", duration.Milliseconds())
}
