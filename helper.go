package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	return &props, nil
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
		log.Println("Error: ", err)
		return nil, err
	}
	request.Header.Set("Authorization", authorization)

	resp, err := client.Do(request)
	if err != nil {
		log.Println("Error: ", err)
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error: ", err)
		return nil, err
	}

	log.Println("URL: ", method, "Status Code:", resp.Status, "Body: ", string(data))

	return data, nil
}
