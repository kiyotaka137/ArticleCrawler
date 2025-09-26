package config

import (
    "os"
    "time"
    "gopkg.in/yaml.v3"
)

type ServerConfig struct {
    GRPCAddr string `yaml:"grpc_addr"`
    HTTPAddr string `yaml:"http_addr"`
}

type PipelineConfig struct {
    FetchWorkers  int `yaml:"fetch_workers"`
    ParseWorkers  int `yaml:"parse_workers"`
    EnrichWorkers int `yaml:"enrich_workers"`
    StoreWorkers  int `yaml:"store_workers"`
}

type RateLimitConfig struct {
    DefaultRPS int `yaml:"default_rps"`
    Burst      int `yaml:"burst"`
}

type DBConfig struct {
    URL string `yaml:"url"`
}

type BackoffConfig struct {
    BaseSeconds int `yaml:"base_seconds"`
    MaxRetries  int `yaml:"max_retries"`
}

type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Pipeline PipelineConfig `yaml:"pipeline"`
    RateLimit RateLimitConfig `yaml:"rate_limit"`
    Database DBConfig       `yaml:"database"`
    Backoff  BackoffConfig  `yaml:"backoff"`
}

func Load(path string) (*Config, error) {
    if path == "" {
        path = "config.yaml"
    }
    b, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    if err := yaml.Unmarshal(b, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}

func (c *Config) BackoffBase() time.Duration {
    return time.Duration(c.Backoff.BaseSeconds) * time.Second
}
