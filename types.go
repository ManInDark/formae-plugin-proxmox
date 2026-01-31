package main

import "encoding/json"

type TargetConfig struct {
	URL  string `json:"url"`
	NODE string `json:"node"`
}

type LXCProperties struct {
	VMID        string `json:"vmid"`
	Hostname    string `json:"hostname"`
	Description string `json:"description"`
	OSTemplate  string `json:"ostemplate"`
}

type ReadRequest struct {
	NativeID     string
	ResourceType string
	TargetConfig json.RawMessage
}

type UpdateRequest struct {
	NativeID          string
	ResourceType      string
	PriorProperties   json.RawMessage
	DesiredProperties json.RawMessage
	TargetConfig      json.RawMessage
}

type StatusLXCGeneral struct {
	Status  string `json:"status"`
	NetIn   int    `json:"netin"`
	NetOut  int    `json:"netout"`
	MaxDisk int    `json:"maxdisk"`
	Cpus    int    `json:"cpus"`
	Name    string `json:"name"`
	Memory  int    `json:"maxmem"`
	VMID    int    `json:"vmid"`
	Type    string `json:"type"`
	Swap    int    `json:"maxswap"`
}

type StatusGeneralResponse struct {
	Data []StatusLXCGeneral `json:"data"`
}

type StatusLXCConfig struct {
	Arch        string `json:"arch"`
	OSType      string `json:"ostype"`
	RootFS      string `json:"rootfs"`
	Hostname    string `json:"hostname"`
	Memory      int    `json:"memory"`
	Swap        int    `json:"swap"`
	Description string `json:"description"`
	Digest      string `json:"digest"`
}

type StatusLXCConfigResponse struct {
	Data StatusLXCConfig `json:"data"`
}
