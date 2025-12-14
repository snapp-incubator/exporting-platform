package exporters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Tenant struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type BaremetalDevice struct {
	ID   int    `json:"id"`
	Name string `json:"name"`

	Site struct {
		Name string `json:"name"`
	} `json:"site"`

	Tenant struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"tenant"`

	Role struct {
		Name string `json:"name"`
	} `json:"device_role"`

	DeviceType struct {
		Model string `json:"model"`
	} `json:"device_type"`
}

type InventoryItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var ramGBRegex = regexp.MustCompile(`(?i)(\d+)\s*gb`)

var ssdGBRegex = regexp.MustCompile(`(?i)(\d+)\s*gb`)

var generationPatterns = map[int][]string{
	9:  {"g9", "gen9"},
	10: {"g10", "gen10"},
	11: {"g11", "gen11"},
}

func detectGeneration(model string) int {
	m := strings.ToLower(model)

	for gen, patterns := range generationPatterns {
		for _, p := range patterns {
			if strings.Contains(m, p) {
				return gen
			}
		}
	}
	return 0
}

type NetboxFetcher struct {
	Address       string
	Token         string
	UseTLS        bool
	IgnoreTenants []string

	SnapshotPath string
	Interval     time.Duration

	httpClient *http.Client
	logf       func(format string, args ...interface{})
}

var lastSnapshotMemory []byte
var lastSnapshotMu sync.RWMutex

func StartNetboxFetcher(address, token string, useTLS bool, ignore []string) {
	f := &NetboxFetcher{
		Address:       address,
		Token:         token,
		UseTLS:        useTLS,
		IgnoreTenants: ignore,

		SnapshotPath: "/tmp/netbox.prom",
		Interval:     5 * time.Minute,

		httpClient: &http.Client{Timeout: 20 * time.Second},
		logf: func(format string, args ...interface{}) {
			fmt.Printf("[netbox-fetcher] "+format+"\n", args...)
		},
	}

	go f.loop()
}

func (f *NetboxFetcher) loop() {
	f.logf("Starting NetBox fetcher (interval=%s)", f.Interval)

	ticker := time.NewTicker(f.Interval)
	defer ticker.Stop()

	f.safeRun()

	for range ticker.C {
		f.safeRun()
	}
}

func (f *NetboxFetcher) safeRun() {
	defer func() {
		if r := recover(); r != nil {
			f.logf("panic recovered: %v", r)
		}
	}()
	f.run()
}

func (f *NetboxFetcher) run() {
	f.logf("fetching latest NetBox snapshot...")

	metrics, err := f.buildSnapshotMetrics()
	if err != nil {
		f.logf("ERROR building snapshot: %v", err)
		return
	}

	tmp := f.SnapshotPath + ".tmp"

	if err := os.WriteFile(tmp, metrics, 0644); err != nil {
		f.logf("ERROR writing tmp file: %v", err)
		return
	}
	if err := os.Rename(tmp, f.SnapshotPath); err != nil {
		f.logf("ERROR renaming snapshot file: %v", err)
		return
	}

	lastSnapshotMu.Lock()
	lastSnapshotMemory = metrics
	lastSnapshotMu.Unlock()

	f.logf("Snapshot updated (size=%d bytes)", len(metrics))
}

func (f *NetboxFetcher) buildSnapshotMetrics() ([]byte, error) {

	tenants, err := f.fetchTenants()
	if err != nil {
		return nil, err
	}
	tenants = filterTenants(tenants, f.IgnoreTenants)

	buf := &bytes.Buffer{}
	fmt.Fprintln(buf, "# NetBox Snapshot Exporter")

	for _, t := range tenants {

		devices, _ := f.fetchDevicesForTenant(t.Slug)

		fmt.Fprintf(buf,
			"netbox_tenant_baremetal_count{tenant=%q} %d\n",
			t.Slug, len(devices),
		)

		for _, d := range devices {

			gen := detectGeneration(d.DeviceType.Model)

			fmt.Fprintf(buf,
				"netbox_baremetal_info{id=%q,name=%q,site=%q,tenant=%q} %d\n",
				fmt.Sprint(d.ID), d.Name, d.Site.Name, d.Tenant.Slug, gen,
			)

			items, _ := f.fetchInventory(d.ID)

			ramTotal := 0.0
			ramModules := []int{}

			ssdTotal := 0.0
			ssdModules := []int{}

			for _, it := range items {
				text := strings.ToLower(it.Name + " " + it.Description)

				if strings.Contains(text, "ram") || strings.Contains(text, "ddr") {
					if m := ramGBRegex.FindStringSubmatch(text); len(m) == 2 {
						gb, _ := strconv.Atoi(m[1])
						ramModules = append(ramModules, gb)
						ramTotal += float64(gb)
					}
				}

				if strings.Contains(text, "ssd") {
					if m := ssdGBRegex.FindStringSubmatch(text); len(m) == 2 {
						gb, _ := strconv.Atoi(m[1])
						ssdModules = append(ssdModules, gb)
						ssdTotal += float64(gb)
					}
				}
			}

			labels := fmt.Sprintf("id=%q,name=%q,site=%q,tenant=%q",
				fmt.Sprint(d.ID), d.Name, d.Site.Name, d.Tenant.Slug)

			fmt.Fprintf(buf, "netbox_baremetal_ram_total_gb{%s} %.0f\n", labels, ramTotal)
			fmt.Fprintf(buf, "netbox_baremetal_ram_module_count{%s} %d\n", labels, len(ramModules))

			for i, size := range ramModules {
				fmt.Fprintf(buf,
					"netbox_baremetal_ram_module_size_gb{%s,index=%q} %d\n",
					labels, fmt.Sprint(i+1), size,
				)
			}

			fmt.Fprintf(buf, "netbox_baremetal_ssd_total_gb{%s} %.0f\n", labels, ssdTotal)
			fmt.Fprintf(buf, "netbox_baremetal_ssd_count{%s} %d\n", labels, len(ssdModules))

			for i, size := range ssdModules {
				fmt.Fprintf(buf,
					"netbox_baremetal_ssd_module_size_gb{%s,index=%q} %d\n",
					labels, fmt.Sprint(i+1), size,
				)
			}
		}
	}

	return buf.Bytes(), nil
}

func (f *NetboxFetcher) fetchJSON(path string, dst interface{}) error {
	schema := "http://"
	if f.UseTLS {
		schema = "https://"
	}

	url := schema + f.Address + path
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Token "+f.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("netbox http %d from %s: %s",
			resp.StatusCode, path, string(body))
	}

	return json.Unmarshal(body, dst)
}

func (f *NetboxFetcher) fetchTenants() ([]Tenant, error) {
	var obj struct {
		Results []Tenant `json:"results"`
	}
	err := f.fetchJSON("/api/tenancy/tenants/?limit=2000", &obj)
	return obj.Results, err
}

func (f *NetboxFetcher) fetchDevicesForTenant(slug string) ([]BaremetalDevice, error) {

	var obj struct {
		Results []BaremetalDevice `json:"results"`
	}
	err := f.fetchJSON("/api/dcim/devices/?role=server&tenant="+slug+"&limit=2000&expand=device_type", &obj)
	return obj.Results, err
}

func (f *NetboxFetcher) fetchInventory(id int) ([]InventoryItem, error) {
	var obj struct {
		Results []InventoryItem `json:"results"`
	}
	err := f.fetchJSON(fmt.Sprintf("/api/dcim/inventory-items/?device_id=%d&limit=500", id), &obj)
	return obj.Results, err
}

func filterTenants(all []Tenant, ignored []string) []Tenant {
	out := []Tenant{}
	skip := map[string]bool{}
	for _, n := range ignored {
		skip[strings.ToLower(n)] = true
	}
	for _, t := range all {
		if skip[strings.ToLower(t.Slug)] || skip[strings.ToLower(t.Name)] {
			continue
		}
		out = append(out, t)
	}
	return out
}
