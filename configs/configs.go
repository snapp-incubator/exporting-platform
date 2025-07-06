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
		Address string `json:"address" yaml:"address"`
		Path    string `json:"path"    yaml:"path"`
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
}

func Load(filePath string) (*Config, error) {
	cfg := Default()
	if err := cleanenv.ReadConfig(filePath, cfg); err != nil {
		if envErr := cleanenv.ReadEnv(cfg); envErr != nil {
			return nil, multierr.Combine(err, envErr)
		}
	}
	cfg.setTokenFromFile()
	return cfg, nil
}

func Default() *Config {
	cfg := &Config{}
	cfg.Exporter.Address = "0.0.0.0:9090"
	cfg.Exporter.Path = "/metrics"

	cfg.Harbor.Enabled = true
	cfg.Harbor.UseTLS = true

	cfg.Keystone.Enabled = true

	return cfg
}

func (c *Config) setTokenFromFile() {
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
