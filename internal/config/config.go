package config

import (
	"os"

	"github.com/hesoyamTM/smart-calendar/internal/adapters/clients/claude"
	"github.com/hesoyamTM/smart-calendar/internal/adapters/clients/google"
	v1 "github.com/hesoyamTM/smart-calendar/internal/controller/restapi/v1"
	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTP   v1.Config     `yaml:"http"`
	Google google.Config `yaml:"google"`
	Claude claude.Config `yaml:"claude"`
}

func (c *Config) SetDefaults() {
	if c.HTTP.Port == 0 {
		c.HTTP.Port = 8080
	}
}

func LoadConfig(path string) (Config, error) {
	var cfg Config

	f, err := os.Open(path)
	if err != nil {
		return cfg, nil
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
