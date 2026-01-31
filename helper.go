package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func parseTargetConfig(data json.RawMessage) (*TargetConfig, error) {
	var cfg TargetConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid target config: %w", err)
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("target config missing 'url'")
	}
	if cfg.NODE == "" {
		return nil, fmt.Errorf("target config missing 'node'")
	}
	return &cfg, nil
}

func getCredentials() (username, token string, err error) {
	username = os.Getenv("PROXMOX_USERNAME")
	token = os.Getenv("PROXMOX_TOKEN")
	if username == "" {
		return "", "", fmt.Errorf("PROXMOX_USERNAME not set")
	}
	if token == "" {
		return "", "", fmt.Errorf("PROXMOX_TOKEN not set")
	}
	return username, token, nil
}

func parseLXCProperties(data json.RawMessage) (*LXCProperties, error) {
	var props LXCProperties
	if err := json.Unmarshal(data, &props); err != nil {
		return nil, fmt.Errorf("invalid file properties: %w", err)
	}
	if props.VMID == "" {
		return nil, fmt.Errorf("vmid missing")
	}
	if props.Hostname == "" {
		return nil, fmt.Errorf("name missing")
	}
	if props.OSTemplate == "" {
		return nil, fmt.Errorf("ostemplate missing")
	}
	return &props, nil
}
