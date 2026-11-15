package config

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config encapsulates configuration for the proxy, studio server, and upstream routes.
type Config struct {
	Server struct {
		ProxyPort  int    `yaml:"proxy_port"`
		StudioPort int    `yaml:"studio_port"`
		Host       string `yaml:"host"`
	} `yaml:"server"`
	Upstream struct {
		URL            string `yaml:"url"`
		APIKey         string `yaml:"api_key"`
		TimeoutSeconds int    `yaml:"timeout_seconds"`
		MockUpstream   bool   `yaml:"mock_upstream"`
	} `yaml:"upstream"`
	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`
	Schemas struct {
		Directory string `yaml:"directory"`
	} `yaml:"schemas"`
}

// DefaultConfig returns reasonable default configurations.
func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.Server.ProxyPort = 8080
	cfg.Server.StudioPort = 4141
	cfg.Server.Host = "0.0.0.0"

	cfg.Upstream.URL = "https://api.openai.com"
	cfg.Upstream.TimeoutSeconds = 60
	cfg.Upstream.MockUpstream = false

	cfg.Database.Path = "data/schemadoctor.db"
	cfg.Schemas.Directory = "schemas"
	return cfg
}

// LoadFromFile parses a YAML configuration file and applies environment overrides.
func LoadFromFile(filePath string) (*Config, error) {
	cfg := DefaultConfig()

	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
	}

	cfg.applyEnvOverrides()
	return cfg, nil
}

func (c *Config) applyEnvOverrides() {
	if val := os.Getenv("SCHEMADOCTOR_PORT"); val != "" {
		if p, err := strconv.Atoi(val); err == nil {
			c.Server.ProxyPort = p
		}
	}
	if val := os.Getenv("SCHEMADOCTOR_STUDIO_PORT"); val != "" {
		if p, err := strconv.Atoi(val); err == nil {
			c.Server.StudioPort = p
		}
	}
	if val := os.Getenv("SCHEMADOCTOR_HOST"); val != "" {
		c.Server.Host = val
	}
	if val := os.Getenv("SCHEMADOCTOR_UPSTREAM_URL"); val != "" {
		c.Upstream.URL = val
	}
	if val := os.Getenv("SCHEMADOCTOR_MOCK_UPSTREAM"); val == "true" || val == "1" {
		c.Upstream.MockUpstream = true
	}
	if val := os.Getenv("SCHEMADOCTOR_DB_PATH"); val != "" {
		c.Database.Path = val
	}
	if val := os.Getenv("SCHEMADOCTOR_SCHEMAS_DIR"); val != "" {
		c.Schemas.Directory = val
	}
}
