package exporters

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"context"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/config"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/prometheus/client_golang/prometheus"
)

type Cloud struct {
	OpenstackName string `json:"openstack_name" yaml:"openstack_name"`
	MetricName    string `json:"metric_name" yaml:"metric_name"`
}
type Metric struct {
	Name   string
	Labels []string
	Metric *prometheus.Desc
}

type KeyStoneCollector struct {
	metrics map[string]Metric
	cloud   Cloud
}

func authenticate(cloud Cloud) (*gophercloud.ServiceClient, error) {
	ctx := context.Background()
	os.Setenv("OS_CLOUD", cloud.OpenstackName)
	authOptions, endpointOptions, tlsConfig, err := clouds.Parse()
	if err != nil {
		fmt.Printf("could not parse cloud.yaml: %s\n", err)
		return nil, err
	}

	providerClient, err := config.NewProviderClient(ctx, authOptions, config.WithTLSConfig(tlsConfig))
	if err != nil {
		fmt.Printf("could not create provider client: %s\n", err)
		return nil, err
	}
	identityClient, err := openstack.NewIdentityV3(providerClient, endpointOptions)
	if err != nil {
		fmt.Printf("could not create identity client: %s\n", err)
		return nil, err
	}
	return identityClient, nil
}

func NewKeystoneCollector(cloud Cloud) *KeyStoneCollector {
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
				prometheus.BuildFQName("apiexporter", cloud.MetricName, m.Name),
				fmt.Sprintf("Description of %s", m.Name),
				m.Labels,
				nil,
			),
		}
	}

	return &KeyStoneCollector{
		metrics: metrics,
		cloud:   cloud,
	}
}

func (kc *KeyStoneCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range kc.metrics {
		ch <- m.Metric
	}
}

func (kc *KeyStoneCollector) Collect(ch chan<- prometheus.Metric) {
	client, err := authenticate(kc.cloud)
	if err != nil {
		fmt.Printf("Authentication error: %v\n", err)
		return
	}

	ctx := context.Background()
	allPages, err := projects.List(client, projects.ListOpts{}).AllPages(ctx)
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
		var team string
		if p.Extra["team"] != nil {
			team = p.Extra["team"].(string)
		} else {
			team = ""
		}

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
	}
}
