package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Registry string    `yaml:"registry"`
	Transit  Transit   `yaml:"transit"`
	Packages []Package `yaml:"packages"`
}

type Transit struct {
	Host   string `yaml:"host"`
	Port   string `yaml:"port"`    // optional, defaults to "22"
	User   string `yaml:"user"`    // optional, defaults to $USER
	SSHKey string `yaml:"ssh_key"`
}

type Package struct {
	Name   string  `yaml:"name"`
	Images []Image `yaml:"images"`
	Charts []Chart `yaml:"charts"`
}

type Image struct {
	Source string `yaml:"source"`
	Dest   string `yaml:"dest"`
}

type Chart struct {
	Repo    string `yaml:"repo"`
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return &cfg, nil
}
