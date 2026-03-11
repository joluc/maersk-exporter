# Maersk Fleet & Position Monitoring Exporter

[![Test](https://github.com/joluc/maersk-exporter/actions/workflows/test.yaml/badge.svg)](https://github.com/joluc/maersk-exporter/actions/workflows/test.yaml)
[![Security Scan](https://github.com/joluc/maersk-exporter/actions/workflows/security.yaml/badge.svg)](https://github.com/joluc/maersk-exporter/actions/workflows/security.yaml)
[![CodeQL](https://github.com/joluc/maersk-exporter/actions/workflows/codeql.yaml/badge.svg)](https://github.com/joluc/maersk-exporter/actions/workflows/codeql.yaml)

A Prometheus metrics exporter that combines Maersk Vessels API (metadata) with AISStream.io (real-time positions) to provide comprehensive vessel fleet monitoring. Track Maersk vessel positions, speed, course, and fleet metadata in Prometheus.

## Features

- Real-time AIS position tracking via WebSocket streaming (AISStream.io)
- Vessel metadata from Maersk Vessels API (capacity, age, flag, IMO)
- Automatic WebSocket reconnection with keep-alive
- Configurable vessel name filtering
- Thread-safe operation with graceful degradation on API errors
- Minimal resource usage with efficient data structures
- Kubernetes-ready with health checks and graceful shutdown

## Architecture

```
┌─────────────────┐
│ Maersk Vessels  │ → Periodic fetch (30m) → Vessel metadata
└─────────────────┘                           (IMO, capacity, year, flag)
                                                      ↓
┌─────────────────┐                           ┌─────────────┐
│  AISStream.io   │ → Real-time stream    →   │  Snapshot   │ → Prometheus
└─────────────────┘   (WebSocket)              │  (in memory)│    Metrics
                      Position updates          └─────────────┘
                      (lat/lon, speed, course)
```

## Exported Metrics

### Meta Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `maersk_up` | gauge | Whether the latest Maersk API refresh succeeded (1=success, 0=failure) |
| `maersk_last_refresh_timestamp_seconds` | gauge | Unix timestamp of last vessel metadata refresh |
| `maersk_refresh_duration_seconds` | gauge | Duration of latest refresh operation in seconds |
| `maersk_scrapes_total` | counter | Total number of /metrics endpoint scrapes |
| `maersk_vessels_total` | gauge | Number of vessels tracked from Maersk API |
| `maersk_positions_total` | gauge | Number of vessels with real-time AIS positions |

### Per-Vessel Metadata Metrics

Labels: `vessel_name`, `imo`, `flag`

| Metric Name | Type | Description |
|-------------|------|-------------|
| `maersk_vessel_capacity_teu` | gauge | Vessel capacity in TEU (Twenty-foot Equivalent Units) |
| `maersk_vessel_age_years` | gauge | Vessel age in years (calculated from build year) |

### Per-Position Metrics

Labels: `vessel_name`, `mmsi`, `nav_status`

| Metric Name | Type | Description |
|-------------|------|-------------|
| `maersk_vessel_latitude_degrees` | gauge | Vessel latitude in degrees (-90 to 90) |
| `maersk_vessel_longitude_degrees` | gauge | Vessel longitude in degrees (-180 to 180) |
| `maersk_vessel_speed_over_ground_knots` | gauge | Vessel speed over ground in knots |
| `maersk_vessel_course_over_ground_degrees` | gauge | Vessel course over ground in degrees (0-360) |
| `maersk_vessel_heading_degrees` | gauge | Vessel true heading in degrees (0-359) |
| `maersk_vessel_position_age_seconds` | gauge | Age of position data in seconds |

**Navigational Status Values:**
- `under_way_using_engine`, `at_anchor`, `not_under_command`, `restricted_manoeuvrability`, `constrained_by_draught`, `moored`, `aground`, `engaged_in_fishing`, `under_way_sailing`, `ais_sart`, `not_defined`

## Prerequisites

You need API credentials from two services:

1. **Maersk Vessels API**
   - Consumer Key for authentication
   - Access to vessel metadata (IMO, capacity, flag, etc.)

2. **AISStream.io**
   - API Key for WebSocket streaming
   - Real-time AIS position data

## Obtaining API Credentials

### Maersk Vessels API

1. Visit the [Maersk Developer Portal](https://developer.maersk.com/)
2. Create an account or sign in
3. Register a new application
4. Navigate to your application settings
5. Copy the **Consumer Key** (also called API Key)
6. Ensure your application has access to the "Vessels" API

### AISStream.io

1. Visit [AISStream.io](https://aisstream.io/)
2. Sign up for a free account
3. Navigate to your dashboard
4. Copy your **API Key**
5. Free tier includes global AIS coverage

## Getting Started

### Running Locally

```bash
# Clone the repository
git clone https://github.com/joluc/maersk-exporter.git
cd maersk-exporter

# Set environment variables
export MAERSK_CONSUMER_KEY="your-maersk-consumer-key"
export AISSTREAM_API_KEY="your-aisstream-api-key"

# Build and run
go build -o maersk-exporter ./cmd/maersk-exporter
./maersk-exporter -vessel-name-filter "MAERSK"

# Or run directly
go run ./cmd/maersk-exporter -vessel-name-filter "MAERSK"
```

The exporter will start on `http://localhost:9878` by default.

### Configuration

Configuration is done via command-line flags or environment variables.

#### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-listen-address` | `:9878` | Address the exporter listens on |
| `-metrics-path` | `/metrics` | Path to expose Prometheus metrics |
| `-maersk-consumer-key` | | Maersk Vessels API Consumer-Key (or `MAERSK_CONSUMER_KEY` env) |
| `-aisstream-api-key` | | AISStream.io API Key (or `AISSTREAM_API_KEY` env) |
| `-vessel-name-filter` | `MAERSK` | Comma-separated vessel name prefixes to track (e.g., `MAERSK,MSC`) |
| `-vessels-refresh-interval` | `30m` | How often to refresh vessel metadata from Maersk API |
| `-request-timeout` | `30s` | Timeout for HTTP API requests |

#### Environment Variables

For security, credentials should be set via environment variables:

```bash
export MAERSK_CONSUMER_KEY="your-maersk-consumer-key"
export AISSTREAM_API_KEY="your-aisstream-api-key"
```

#### Configuration File

Copy `.env.example` to `.env` and fill in your credentials:

```bash
cp .env.example .env
# Edit .env with your credentials
```

### Examples

#### Track Maersk vessels (default)

```bash
./maersk-exporter
```

#### Track multiple shipping lines

```bash
./maersk-exporter -vessel-name-filter "MAERSK,MSC,CMA"
```

#### Custom refresh interval

```bash
./maersk-exporter \
  -vessel-name-filter "MAERSK" \
  -vessels-refresh-interval 15m
```

## Docker Usage

### Build Docker Image

```bash
docker build -t maersk-exporter:latest .
```

### Run with Docker

```bash
docker run -p 9878:9878 \
  -e MAERSK_CONSUMER_KEY="your-maersk-consumer-key" \
  -e AISSTREAM_API_KEY="your-aisstream-api-key" \
  maersk-exporter:latest
```

### Run with custom flags

```bash
docker run -p 9878:9878 \
  -e MAERSK_CONSUMER_KEY="your-maersk-consumer-key" \
  -e AISSTREAM_API_KEY="your-aisstream-api-key" \
  maersk-exporter:latest \
  -vessel-name-filter "MAERSK,MSC" \
  -vessels-refresh-interval 15m
```

## Endpoints

- `http://localhost:9878/` - Root page with links
- `http://localhost:9878/metrics` - Prometheus metrics
- `http://localhost:9878/healthz` - Health check endpoint

## Example Metrics Output

```prometheus
# HELP maersk_up Whether the latest Maersk API refresh succeeded
# TYPE maersk_up gauge
maersk_up 1

# HELP maersk_vessels_total Number of vessels tracked
# TYPE maersk_vessels_total gauge
maersk_vessels_total 466

# HELP maersk_positions_total Number of vessels with AIS positions
# TYPE maersk_positions_total gauge
maersk_positions_total 23

# HELP maersk_vessel_capacity_teu Vessel capacity in TEU
# TYPE maersk_vessel_capacity_teu gauge
maersk_vessel_capacity_teu{vessel_name="MAERSK CHARLESTON",imo="9936379",flag="SG"} 15516

# HELP maersk_vessel_age_years Vessel age in years
# TYPE maersk_vessel_age_years gauge
maersk_vessel_age_years{vessel_name="MAERSK CHARLESTON",imo="9936379",flag="SG"} 3

# HELP maersk_vessel_latitude_degrees Vessel latitude in degrees
# TYPE maersk_vessel_latitude_degrees gauge
maersk_vessel_latitude_degrees{vessel_name="MAERSK CHARLESTON",mmsi="566798000",nav_status="under_way_using_engine"} 37.456789

# HELP maersk_vessel_longitude_degrees Vessel longitude in degrees
# TYPE maersk_vessel_longitude_degrees gauge
maersk_vessel_longitude_degrees{vessel_name="MAERSK CHARLESTON",mmsi="566798000",nav_status="under_way_using_engine"} -122.234567

# HELP maersk_vessel_speed_over_ground_knots Vessel speed over ground in knots
# TYPE maersk_vessel_speed_over_ground_knots gauge
maersk_vessel_speed_over_ground_knots{vessel_name="MAERSK CHARLESTON",mmsi="566798000",nav_status="under_way_using_engine"} 18.50

# HELP maersk_vessel_course_over_ground_degrees Vessel course over ground in degrees
# TYPE maersk_vessel_course_over_ground_degrees gauge
maersk_vessel_course_over_ground_degrees{vessel_name="MAERSK CHARLESTON",mmsi="566798000",nav_status="under_way_using_engine"} 245.3
```

## Helm Chart

Deploy on Kubernetes using the included Helm chart. See the [Helm Chart README](chart/maersk-exporter/README.md) for configuration options.

```bash
# Create secret with API credentials
kubectl create secret generic maersk-exporter-credentials \
  --from-literal=maersk-consumer-key='your-maersk-consumer-key' \
  --from-literal=aisstream-api-key='your-aisstream-api-key' \
  -n monitoring

# Install the chart
helm install maersk-exporter ./chart/maersk-exporter \
  --namespace monitoring \
  --create-namespace \
  --set credentials.existingSecret=maersk-exporter-credentials
```

## How It Works

### Data Flow

1. **Startup**: Exporter connects to both Maersk Vessels API (HTTP) and AISStream.io (WebSocket)
2. **Vessel Metadata**: Periodic refresh (default 30 minutes) fetches vessel metadata from Maersk API
3. **Position Updates**: Real-time AIS position reports stream continuously via WebSocket
4. **Name Matching**: Vessels are matched between sources using normalized ship names
5. **Metrics Export**: Combined data is exposed as Prometheus metrics on `/metrics`

### WebSocket Resilience

- Automatic reconnection on disconnect (5-second backoff)
- Keep-alive ping/pong every 30 seconds
- Read timeout of 60 seconds
- Graceful handling of network interruptions

### Name Matching

Ship names are normalized (uppercase, trimmed) to match vessels between APIs:
- Maersk API: `"MAERSK DENVER"`
- AIS Stream: `"MAERSK DENVER          "` (padded)
- Normalized: `"MAERSK DENVER"`

## API Documentation

- [Maersk Vessels API](https://developer.maersk.com/)
- [AISStream.io Documentation](https://aisstream.io/documentation)

## License

[MIT License](LICENSE)
