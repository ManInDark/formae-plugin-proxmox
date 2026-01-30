// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/platform-engineering-labs/formae/pkg/plugin"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// https://pve.proxmox.com/pve-docs/api-viewer/
type TargetConfig struct {
	URL  string `json:"url"`
	NODE string `json:"node"`
}

type LXCProperties struct {
	VMID        int    `json:"vmid"`
	NAME        string `json:"name"`
	DESCRIPTION string `json:"description"`
	OSTEMPLATE  string `json:"ostemplate"`
}

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
	if props.VMID == 0 {
		return nil, fmt.Errorf("vmid missing")
	}
	if props.NAME == "" {
		return nil, fmt.Errorf("name missing")
	}
	if props.OSTEMPLATE == "" {
		return nil, fmt.Errorf("ostemplate missing")
	}
	return &props, nil
}

// ErrNotImplemented is returned by stub methods that need implementation.
var ErrNotImplemented = errors.New("not implemented")

// Plugin implements the Formae ResourcePlugin interface.
// The SDK automatically provides identity methods (Name, Version, Namespace)
// by reading formae-plugin.pkl at startup.
type Plugin struct{}

// Compile-time check: Plugin must satisfy ResourcePlugin interface.
var _ plugin.ResourcePlugin = &Plugin{}

// =============================================================================
// Configuration Methods
// =============================================================================

// RateLimit returns the rate limiting configuration for this plugin.
// Adjust MaxRequestsPerSecondForNamespace based on your provider's API limits.
func (p *Plugin) RateLimit() plugin.RateLimitConfig {
	return plugin.RateLimitConfig{
		Scope:                            plugin.RateLimitScopeNamespace,
		MaxRequestsPerSecondForNamespace: 10, // TODO: Adjust based on provider limits
	}
}

// DiscoveryFilters returns filters to exclude certain resources from discovery.
// Resources matching ALL conditions in a filter are excluded.
// Return nil if you want to discover all resources.
func (p *Plugin) DiscoveryFilters() []plugin.MatchFilter {
	// Example: exclude resources with a specific tag
	// return []plugin.MatchFilter{
	//     {
	//         ResourceTypes: []string{"PROXMOX::Service::Resource"},
	//         Conditions: []plugin.FilterCondition{
	//             {PropertyPath: "$.Tags[?(@.Key=='skip-discovery')].Value", PropertyValue: "true"},
	//         },
	//     },
	// }
	return nil
}

// LabelConfig returns the configuration for extracting human-readable labels
// from discovered resources.
func (p *Plugin) LabelConfig() plugin.LabelConfig {
	return plugin.LabelConfig{
		// Default JSONPath query to extract label from resources
		// Example for tagged resources: $.Tags[?(@.Key=='Name')].Value
		DefaultQuery: "$.name",

		// Override for specific resource types
		ResourceOverrides: map[string]string{
			// "PROXMOX::Service::SpecialResource": "$.DisplayName",
		},
	}
}

// =============================================================================
// CRUD Operations
// =============================================================================

// Create provisions a new resource.
func (p *Plugin) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {

	log.Println(req.Properties)

	props, err := parseLXCProperties(req.Properties)
	if err != nil {
		log.Println(err.Error())
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   err.Error(),
			},
		}, nil
	}

	log.Println("LXC Properties: ", props.VMID, props.NAME, props.OSTEMPLATE, props.DESCRIPTION)

	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, nil
	}

	username, token, err := getCredentials()
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, nil
	}

	client := &http.Client{}

	// data := map[string]any{"node": config.NODE, "ostemplate": props.OSTEMPLATE, "id": props.VMID, "hostname": props.NAME, "description": props.DESCRIPTION}
	// jsonBody, err := json.Marshal(data)

	arguments := "vmid=" + strconv.Itoa(props.VMID) + "&ostemplate=" + props.OSTEMPLATE + "&hostname=" + props.NAME

	request, err := http.NewRequest("POST", config.URL+"/api2/json/nodes/"+config.NODE+"/lxc", bytes.NewBuffer([]byte(arguments)))
	request.Header.Set("Authorization", "PVEAPIToken="+username+"="+token)

	resp, err := client.Do(request)

	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, nil
	}

	body, err := io.ReadAll(resp.Body)

	log.Println("Response StatusCode: ", resp.Status)
	log.Println("Response Body: ", string(body))

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        strconv.Itoa(props.VMID),
		},
	}, nil
}

// Read retrieves the current state of a resource.
func (p *Plugin) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	// TODO: Implement resource read
	//
	// 1. Use req.NativeID to identify the resource
	// 2. Parse req.TargetConfig for provider credentials
	// 3. Call your provider's API to get current state
	// 4. Return ReadResult with Properties as JSON string

	return &resource.ReadResult{
		ResourceType: req.ResourceType,
		ErrorCode:    resource.OperationErrorCodeInternalFailure,
	}, ErrNotImplemented
}

// Update modifies an existing resource.
func (p *Plugin) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	// TODO: Implement resource update
	//
	// 1. Use req.NativeID to identify the resource
	// 2. Use req.PatchDocument for changes (JSON Patch format)
	//    Or compare req.PriorProperties with req.DesiredProperties
	// 3. Call your provider's API to apply changes
	// 4. Return ProgressResult with status

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusFailure,
			ErrorCode:       resource.OperationErrorCodeInternalFailure,
			StatusMessage:   "Update not implemented",
		},
	}, ErrNotImplemented
}

// Delete removes a resource.
func (p *Plugin) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	// TODO: Implement resource deletion
	//
	// 1. Use req.NativeID to identify the resource
	// 2. Parse req.TargetConfig for provider credentials
	// 3. Call your provider's API to delete the resource
	// 4. Return ProgressResult with status

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusFailure,
			ErrorCode:       resource.OperationErrorCodeInternalFailure,
			StatusMessage:   "Delete not implemented",
		},
	}, ErrNotImplemented
}

// Status checks the progress of an async operation.
// Called when Create/Update/Delete return InProgress status.
func (p *Plugin) Status(ctx context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	// TODO: Implement status checking for async operations
	//
	// 1. Use req.RequestID to identify the operation
	// 2. Call your provider's API to check operation status
	// 3. Return ProgressResult with current status
	//
	// If your provider's operations are synchronous, return Success immediately.

	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusFailure,
			ErrorCode:       resource.OperationErrorCodeInternalFailure,
			StatusMessage:   "Status not implemented",
		},
	}, ErrNotImplemented
}

// List returns all resource identifiers of a given type.
// Called during discovery to find unmanaged resources.
func (p *Plugin) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	// TODO: Implement resource listing for discovery
	//
	// 1. Use req.ResourceType to determine what to list
	// 2. Parse req.TargetConfig for provider credentials
	// 3. Use req.PageToken/PageSize for pagination
	// 4. Call your provider's API to list resources
	// 5. Return NativeIDs and NextPageToken (if more pages)

	return &resource.ListResult{
		NativeIDs:     []string{},
		NextPageToken: nil,
	}, ErrNotImplemented
}
