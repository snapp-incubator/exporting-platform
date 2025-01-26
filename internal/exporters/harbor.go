package exporters

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "custom_harbor"
	subsystem = "registries"
)

type registryEntry struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type HarborCollector struct {
	HarborAddress                 string
	RegistriesBackendHealthStatus *prometheus.Desc
	Token                         string
	UseTLS                        bool
}

func NewHarborCollector(harborAddr, token string, useTLS bool) *HarborCollector {
	healthDesc := prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "health"), "fetch harbor registry backend health status", []string{"registry"}, nil)
	return &HarborCollector{
		HarborAddress:                 harborAddr,
		RegistriesBackendHealthStatus: healthDesc,
		Token:                         token,
		UseTLS:                        useTLS,
	}
}

func (h *HarborCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- h.RegistriesBackendHealthStatus
}

func (h *HarborCollector) Collect(ch chan<- prometheus.Metric) {
	h.collectHarborRegistryBackendHealthStatus(ch)
}

func (h *HarborCollector) collectHarborRegistryBackendHealthStatus(ch chan<- prometheus.Metric) {
	var schema string
	if h.UseTLS {
		schema = "https://"
	} else {
		schema = "http://"
	}
	url := schema + h.HarborAddress + "/api/v2.0/registries?page=1&page_size=100"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Printf("client: could not create request: %s\n", err)
		os.Exit(1)
	}
	req.Header.Set("access", "application/json")
	req.Header.Set("Authorization", "Basic "+h.Token)

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		return
	}

	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error happened", err)
		return
	}

	var endpointData []registryEntry
	err = json.Unmarshal(responseBody, &endpointData)
	if err != nil {
		fmt.Println("Error happened while Unmarshalling data:", err, ". response body:", string(responseBody))
		return
	}

	var value float64
	for _, entry := range endpointData {
		if entry.Status == "healthy" {
			value = 1
		} else {
			value = 0
		}
		ch <- prometheus.MustNewConstMetric(h.RegistriesBackendHealthStatus, prometheus.GaugeValue, value, entry.Name)
	}
}
