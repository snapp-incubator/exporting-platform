package configs

import (
	"exporting_platform/internal/exporters"
	"fmt"
	"os"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
	"go.uber.org/multierr"
)

type Config struct {
	Exporter struct {
		Address  string `json:"address" yaml:"address"`
		Path     string `json:"path"    yaml:"path"`
		LogLevel string `json:"log_level" yaml:"log_level"`
	} `json:"exporter" yaml:"exporter"`
	Harbor struct {
		Address   string `json:"address" yaml:"address"`
		Enabled   bool   `json:"enabled" yaml:"enabled"`
		Token     string `json:"token"   yaml:"token"`
		TokenPath string `json:"token_path" yaml:"token_path"`
		UseTLS    bool   `json:"use_tls" yaml:"use_tls"`
	} `json:"harbor" yaml:"harbor"`
	Keystone struct {
		Enabled bool              `json:"enabled" yaml:"enabled"`
		Clouds  []exporters.Cloud `json:"clouds" yaml:"clouds"`
	}
	Netbox struct {
		Enabled       bool     `json:"enabled" yaml:"enabled"`
		Address       string   `json:"address" yaml:"address"`
		Token         string   `json:"token" yaml:"token"`
		TokenPath     string   `json:"token_path" yaml:"token_path"`
		UseTLS        bool     `json:"use_tls" yaml:"use_tls"`
		IgnoreTenants []string `json:"ignore_tenants" yaml:"ignore_tenants"`
	} `json:"netbox" yaml:"netbox"`
}

func Load(filePath string) (*Config, error) {
	cfg := Default()
	if err := cleanenv.ReadConfig(filePath, cfg); err != nil {
		if envErr := cleanenv.ReadEnv(cfg); envErr != nil {
			return nil, multierr.Combine(err, envErr)
		}
	}
	cfg.setHarborTokenFromFile()
	cfg.setNetboxTokenFromFile()

	return cfg, nil
}

func Default() *Config {
	cfg := &Config{}
	cfg.Exporter.Address = "0.0.0.0:9090"
	cfg.Exporter.Path = "/metrics"
	cfg.Exporter.LogLevel = "Info"

	cfg.Harbor.Enabled = true
	cfg.Harbor.UseTLS = true

	cfg.Keystone.Enabled = false

	cfg.Netbox.Enabled = true
	cfg.Netbox.UseTLS = true

	return cfg
}

func (c *Config) setHarborTokenFromFile() {
	if !c.Harbor.Enabled {
		return
	}

	if c.Harbor.Token != "" {
		return
	}

	if c.Harbor.TokenPath != "" {
		data, err := os.ReadFile(c.Harbor.TokenPath)
		if err != nil {
			fmt.Println("could not read file", c.Harbor.TokenPath, "Error:", err)
			os.Exit(4)
		}
		c.Harbor.Token = strings.TrimSpace(string(data))
	}
}
func (c *Config) setNetboxTokenFromFile() {
	if !c.Netbox.Enabled {
		return
	}

	if c.Netbox.Token != "" {
		return
	}

	if c.Netbox.TokenPath != "" {
		data, err := os.ReadFile(c.Netbox.TokenPath)
		if err != nil {
			fmt.Println("could not read netbox token file:", err)
			os.Exit(4)
		}
		c.Netbox.Token = strings.TrimSpace(string(data))
	}
}
