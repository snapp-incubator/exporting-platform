package exporters

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/prometheus/client_golang/prometheus"
)

type Metric struct {
	Name   string
	Labels []string
	Metric *prometheus.Desc
}

type KeyStoneCollector struct {
	metrics map[string]Metric
}

func authenticate() (*gophercloud.ServiceClient, error) {
	opts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		fmt.Println("Error happened", err)
	}

	providerClient, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		fmt.Println("Error happened", err)
	}

	computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		fmt.Println("Error happened", err)
	}

	return computeClient, nil
}

func NewKeystoneCollector() *KeyStoneCollector {
	projectMetrics := []Metric{
		{Name: "projects"},
		{Name: "project_info", Labels: []string{
			"is_domain", "description", "domain_id", "enabled",
			"id", "name", "parent_id", "tags", "team",
		}},
	}

	metrics := make(map[string]Metric)
	for _, m := range projectMetrics {
		metrics[m.Name] = Metric{
			Name:   m.Name,
			Labels: m.Labels,
			Metric: prometheus.NewDesc(
				prometheus.BuildFQName("openstack", "keystone", m.Name),
				fmt.Sprintf("Description of %s", m.Name),
				m.Labels,
				nil,
			),
		}
	}

	return &KeyStoneCollector{
		metrics: metrics,
	}
}

func (kc *KeyStoneCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range kc.metrics {
		ch <- m.Metric
	}
}

func (kc *KeyStoneCollector) Collect(ch chan<- prometheus.Metric) {
	client, err := authenticate()
	if err != nil {
		fmt.Printf("Authentication error: %v\n", err)
		return
	}

	allPages, err := projects.List(client, projects.ListOpts{}).AllPages()
	if err != nil {
		fmt.Printf("Error listing projects: %v\n", err)
		return
	}

	allProjects, err := projects.ExtractProjects(allPages)
	if err != nil {
		fmt.Printf("Error extracting projects: %v\n", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		kc.metrics["projects"].Metric,
		prometheus.GaugeValue,
		float64(len(allProjects)),
	)

	for _, p := range allProjects {
		tagString := ""
		if len(p.Tags) > 0 {
			tagString = strings.Join(p.Tags, ",")
		}

		team := "kia"

		ch <- prometheus.MustNewConstMetric(
			kc.metrics["project_info"].Metric,
			prometheus.GaugeValue,
			1.0,
			strconv.FormatBool(p.IsDomain),
			p.Description,
			p.DomainID,
			strconv.FormatBool(p.Enabled),
			p.ID,
			p.Name,
			p.ParentID,
			tagString,
			team,
		)
		fmt.Println(p.Name, p.ID)
	}
}
