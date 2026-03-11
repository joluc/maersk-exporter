# Maersk Fleet Exporter Helm Chart

Helm chart for deploying the Maersk Fleet & Position Monitoring Exporter on Kubernetes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- API credentials from [Maersk Developer Portal](https://developer.maersk.com/) and [AISStream.io](https://aisstream.io/)

## Installation

### 1. Create Secret (Recommended)

```bash
kubectl create secret generic maersk-exporter-credentials \
  --from-literal=maersk-consumer-key='your-maersk-key' \
  --from-literal=aisstream-api-key='your-aisstream-key' \
  -n monitoring
```

### 2. Install Chart

```bash
helm install maersk-exporter ./chart/maersk-exporter \
  --namespace monitoring \
  --create-namespace \
  --set credentials.existingSecret=maersk-exporter-credentials
```

## Configuration

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `credentials.existingSecret` | Name of existing secret (recommended) | `""` |
| `config.vesselNameFilter` | Vessel name prefixes to track | `"MAERSK"` |
| `config.vesselsRefreshInterval` | Metadata refresh interval | `"30m"` |
| `resources.limits.cpu` | CPU limit | `200m` |
| `resources.limits.memory` | Memory limit | `256Mi` |
| `serviceMonitor.enabled` | Enable Prometheus ServiceMonitor | `false` |
| `serviceMonitor.labels` | Labels for ServiceMonitor (e.g., `release: prometheus`) | `{}` |

### Example: Track Multiple Shipping Lines

```bash
helm install maersk-exporter ./chart/maersk-exporter \
  --set credentials.existingSecret=maersk-exporter-credentials \
  --set config.vesselNameFilter="MAERSK,MSC,CMA"
```

### Example: Enable Prometheus Operator Integration

```bash
helm install maersk-exporter ./chart/maersk-exporter \
  --set credentials.existingSecret=maersk-exporter-credentials \
  --set serviceMonitor.enabled=true \
  --set serviceMonitor.labels.release=prometheus
```

## Accessing Metrics

```bash
# Port forward
kubectl port-forward -n monitoring svc/maersk-exporter 9878:9878

# View metrics
curl http://localhost:9878/metrics

# Health check
curl http://localhost:9878/healthz
```

## Security

The chart uses secure defaults:
- Non-root user (UID 65534)
- Read-only root filesystem
- Dropped capabilities
- Credentials via Kubernetes Secrets

## Uninstall

```bash
helm uninstall maersk-exporter -n monitoring
kubectl delete secret maersk-exporter-credentials -n monitoring
```

## Values

See [values.yaml](values.yaml) for all configuration options.
