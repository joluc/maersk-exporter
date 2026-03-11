package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/joluc/maersk-exporter/pkg/config"
	"github.com/joluc/maersk-exporter/pkg/models"
)

// VesselAPIVessel represents a single vessel from the API response.
type VesselAPIVessel struct {
	VesselIMONumber   int    `json:"vesselIMONumber"`
	CarrierVesselCode string `json:"carrierVesselCode"`
	VesselLongName    string `json:"vesselLongName"`
	VesselFlagCode    string `json:"vesselFlagCode"`
	VesselBuiltYear   int    `json:"vesselBuiltYear"`
	VesselCallSign    string `json:"vesselCallSign"`
	VesselCapacityTEU int    `json:"vesselCapacityTEU"`
}

// fetchVessels retrieves vessel metadata from Maersk Vessels API.
func fetchVessels(ctx context.Context, client *http.Client, cfg config.Config) ([]models.Vessel, error) {
	if len(cfg.FilterVesselNames) == 0 {
		return nil, fmt.Errorf("no vessel names configured for filtering")
	}

	baseURL := "https://api.maersk.com/reference-data/vessels"
	params := url.Values{}
	params.Add("vesselNames", strings.Join(cfg.FilterVesselNames, ","))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Consumer-Key", cfg.MaerskConsumerKey)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiVessels []VesselAPIVessel
	if err := json.NewDecoder(resp.Body).Decode(&apiVessels); err != nil {
		return nil, fmt.Errorf("decoding JSON response: %w", err)
	}

	vessels := make([]models.Vessel, len(apiVessels))
	for i, v := range apiVessels {
		vessels[i] = models.Vessel{
			VesselIMONumber:   v.VesselIMONumber,
			CarrierVesselCode: v.CarrierVesselCode,
			VesselLongName:    v.VesselLongName,
			VesselFlagCode:    v.VesselFlagCode,
			VesselBuiltYear:   v.VesselBuiltYear,
			VesselCallSign:    v.VesselCallSign,
			VesselCapacityTEU: v.VesselCapacityTEU,
		}
	}

	return vessels, nil
}

// normalizeName normalizes a vessel name for matching between APIs.
func normalizeName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}

// indexByName creates a map of vessels indexed by normalized name.
func indexByName(vessels []models.Vessel) map[string]models.Vessel {
	index := make(map[string]models.Vessel, len(vessels))
	for _, v := range vessels {
		index[normalizeName(v.VesselLongName)] = v
	}
	return index
}
