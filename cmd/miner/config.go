package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	MinerID     string `json:"miner_id"`
	Difficulty  int    `json:"difficulty"`
	DataDir     string `json:"data_dir"`
	StatsFile   string `json:"stats_file"`
	AutoSave    bool   `json:"auto_save"`
	ShowDetails bool   `json:"show_details"`
}

const (
	ConfigFileName = "miner-config.json"
	DefaultDataDir = "miner-data"
)

func LoadConfig() (*Config, error) {
	config := &Config{
		MinerID:     DefaultMinerID,
		Difficulty:  DefaultDifficulty,
		DataDir:     DefaultDataDir,
		StatsFile:   "mining-stats.json",
		AutoSave:    true,
		ShowDetails: false,
	}

	// Try to load existing config
	configPath := getConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	return config, nil
}

func SaveConfig(config *Config) error {
	configPath := getConfigPath()
	
	// Ensure config directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func getConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".cryptochain", ConfigFileName)
}

func (c *Config) Validate() error {
	if c.MinerID == "" {
		return fmt.Errorf("miner ID cannot be empty")
	}
	
	if c.Difficulty < 1 || c.Difficulty > 8 {
		return fmt.Errorf("difficulty must be between 1 and 8")
	}
	
	if c.DataDir == "" {
		return fmt.Errorf("data directory cannot be empty")
	}
	
	return nil
}