package config

import (
	"flag"
	"os"
	"strings"
	"time"
)

type Config struct {
	// HTTP Server
	ListenAddress string
	MetricsPath   string

	// Maersk Vessels API
	MaerskConsumerKey string

	// AISStream.io
	AISStreamAPIKey string

	// Query Filters
	FilterVesselNames []string

	// Polling
	VesselsRefreshInterval time.Duration
	RequestTimeout         time.Duration
}

func ParseFlags() (Config, error) {
	listenAddress := flag.String("listen-address", ":9878", "Address the exporter listens on")
	metricsPath := flag.String("metrics-path", "/metrics", "Path to expose Prometheus metrics")
	maerskConsumerKey := flag.String("maersk-consumer-key", "", "Maersk Vessels API Consumer-Key (or set MAERSK_CONSUMER_KEY)")
	aisstreamAPIKey := flag.String("aisstream-api-key", "", "AISStream.io API Key (or set AISSTREAM_API_KEY)")
	vesselNameFilter := flag.String("vessel-name-filter", "MAERSK", "Comma separated list of vessel name prefixes to track (e.g., 'MAERSK,MSC')")
	vesselsRefreshInterval := flag.Duration("vessels-refresh-interval", 30*time.Minute, "How often to refresh vessel metadata from Maersk API")
	requestTimeout := flag.Duration("request-timeout", 30*time.Second, "Timeout for each API request")
	flag.Parse()

	// Use environment variables as fallback for secrets
	finalMaerskConsumerKey := *maerskConsumerKey
	if finalMaerskConsumerKey == "" {
		finalMaerskConsumerKey = os.Getenv("MAERSK_CONSUMER_KEY")
	}

	finalAISStreamAPIKey := *aisstreamAPIKey
	if finalAISStreamAPIKey == "" {
		finalAISStreamAPIKey = os.Getenv("AISSTREAM_API_KEY")
	}

	cfg := Config{
		ListenAddress:          *listenAddress,
		MetricsPath:            *metricsPath,
		MaerskConsumerKey:      finalMaerskConsumerKey,
		AISStreamAPIKey:        finalAISStreamAPIKey,
		FilterVesselNames:      ParseList(*vesselNameFilter),
		VesselsRefreshInterval: *vesselsRefreshInterval,
		RequestTimeout:         *requestTimeout,
	}

	return cfg, nil
}

// ParseList parses a comma-separated list into a slice of trimmed strings.
func ParseList(value string) []string {
	if value == "" {
		return nil
	}
	var out []string
	for _, item := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
