package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	Volume     float64 `json:"volume"`
	configFile string
	mu         sync.Mutex
}

func NewConfig(configDir string) *Config {
	c := &Config{
		configFile: filepath.Join(configDir, ".zoneout_config"),
		Volume:     0.5, // Default 50%
	}
	c.Load()
	return c
}

func (c *Config) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.configFile)
	if err != nil {
		// File doesn't exist yet, use defaults
		return nil
	}

	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(c.configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (c *Config) SetVolume(volume float64) error {
	c.mu.Lock()
	c.Volume = volume
	c.mu.Unlock()
	return c.Save()
}

func (c *Config) GetVolume() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Volume
}
