package main

import "encoding/json"

type TargetConfig struct {
	URL  string `json:"url"`
	NODE string `json:"node"`
}

type LXCProperties struct {
	VMID        string   `json:"vmid"`
	Hostname    string   `json:"hostname"`
	Description string   `json:"description,omitempty"`
	OSTemplate  string   `json:"ostemplate,omitempty"`
	Password    string   `json:"password,omitempty"`
	Cores       int      `json:"cores"`
	Memory      int      `json:"memory"`
	OnBoot      int      `json:"onboot"`
	SSHKeys     []string `json:"sshkeys"`
	Networks    []string `json:"networks"`
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

type DeleteRequest struct {
	NativeID     string
	ResourceType string
	TargetConfig json.RawMessage
}

type ListRequest struct {
	ResourceType         string
	TargetConfig         json.RawMessage
	PageSize             int32
	PageToken            *string
	AdditionalProperties map[string]string
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
	Cores       int    `json:"cores"`
	OSType      string `json:"ostype"`
	RootFS      string `json:"rootfs"`
	Hostname    string `json:"hostname"`
	Memory      int    `json:"memory"`
	Swap        int    `json:"swap"`
	Description string `json:"description"`
	Digest      string `json:"digest"`
	OnBoot      int    `json:"onboot"`
	Net0        string `json:"net0,omitempty"`
	Net1        string `json:"net1,omitempty"`
	Net2        string `json:"net2,omitempty"`
	Net3        string `json:"net3,omitempty"`
	Net4        string `json:"net4,omitempty"`
	Net5        string `json:"net5,omitempty"`
	Net6        string `json:"net6,omitempty"`
	Net7        string `json:"net7,omitempty"`
	Net8        string `json:"net8,omitempty"`
	Net9        string `json:"net9,omitempty"`
}

type StatusLXCConfigResponse struct {
	Data StatusLXCConfig `json:"data"`
}

type ProxmoxDataResponse struct {
	Data string `json:"data"`
}
