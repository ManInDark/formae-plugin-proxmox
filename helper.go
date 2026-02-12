package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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

func setupLogging() {
	programLevel := new(slog.LevelVar)
	env := os.Getenv("PROXMOX_LOG_LEVEL")
	programLevel.UnmarshalText([]byte(env))
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel}))
	slog.Info("Set log level", "level", programLevel)
	slog.SetDefault(logger)
}

func createAuthorizationString(username, token string) string {
	return "PVEAPIToken=" + username + "=" + token
}

func authenticatedRequest(method, url, authorization string, urlparams url.Values) ([]byte, error) {
	client := &http.Client{}
	body := &bytes.Buffer{}
	if urlparams != nil {
		body = bytes.NewBuffer([]byte(urlparams.Encode()))
	}

	request, err := http.NewRequest(method, url, body)
	if err != nil {
		slog.Error("Error creating request", "err", err)
		return nil, err
	}
	request.Header.Set("Authorization", authorization)

	resp, err := client.Do(request)
	if err != nil {
		slog.Error("Error executing request", "err", err)
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Error reading response", "err", err)
		return nil, err
	}

	slog.Debug("Executed Request", "url", method, "status code", resp.Status, "body", string(data))

	return data, nil
}
