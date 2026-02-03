// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/platform-engineering-labs/formae/pkg/plugin"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// https://pve.proxmox.com/pve-docs/api-viewer/

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
		DefaultQuery: "$.hostname",

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
		}, err
	}

	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		log.Println(err.Error())
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, err
	}

	username, token, err := getCredentials()
	if err != nil {
		log.Println(err.Error())
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, err
	}

	urlparams := url.Values{
		"vmid":       {props.VMID},
		"ostemplate": {props.OSTemplate},
		"hostname":   {props.Hostname},
		"cores":      {strconv.Itoa(props.Cores)},
		"memory":     {strconv.Itoa(props.Memory)},
	}
	if props.Description != "" {
		urlparams.Add("description", props.Description)
	}

	_, err = authenticatedRequest(http.MethodPost, config.URL+"/api2/json/nodes/"+config.NODE+"/lxc", createAuthorizationString(username, token), urlparams)

	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, err
	}

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        props.VMID,
		},
	}, nil
}

func (p *Plugin) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	username, token, err := getCredentials()
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: resource.OperationErrorCodeInvalidRequest,
		}, err
	}

	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		return &resource.ReadResult{}, nil
	}

	data, err := authenticatedRequest(http.MethodGet, config.URL+"/api2/json/nodes/"+config.NODE+"/lxc/"+req.NativeID+"/config", createAuthorizationString(username, token), nil)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: resource.OperationErrorCodeNetworkFailure,
		}, err
	}

	var props StatusLXCConfigResponse

	err = json.Unmarshal(data, &props)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: resource.OperationErrorCodeInvalidRequest,
		}, err
	}

	lxcdata := props.Data

	properties := LXCProperties{
		VMID:        req.NativeID,
		Hostname:    lxcdata.Hostname,
		Description: lxcdata.Description,
		Cores:       lxcdata.Cores,
		Memory:      lxcdata.Memory,
	}

	propsJSON, err := json.Marshal(properties)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: resource.OperationErrorCodeInternalFailure,
		}, err
	}

	return &resource.ReadResult{
		ResourceType: req.ResourceType,
		Properties:   string(propsJSON),
	}, nil
}

// Update modifies an existing resource.
func (p *Plugin) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {

	prior, err := parseLXCProperties(req.PriorProperties)
	if err != nil {
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   err.Error(),
			},
		}, err
	}

	desir, err := parseLXCProperties(req.DesiredProperties)
	if err != nil {
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   err.Error(),
			},
		}, err
	}

	if prior == nil {
		p.Create(ctx, &resource.CreateRequest{
			ResourceType: req.ResourceType,
			Properties:   req.DesiredProperties,
			TargetConfig: req.TargetConfig,
		})
	}

	if prior.VMID != desir.VMID {
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   "can't change vmid",
			},
		}, fmt.Errorf("can't change vmid")
	}

	if prior.Hostname != desir.Hostname || prior.Description != desir.Description {
		config, err := parseTargetConfig(req.TargetConfig)
		if err != nil {
			log.Println(err.Error())
			return &resource.UpdateResult{
				ProgressResult: &resource.ProgressResult{
					Operation:       resource.OperationCreate,
					OperationStatus: resource.OperationStatusFailure,
					ErrorCode:       resource.OperationErrorCodeInternalFailure,
					StatusMessage:   err.Error(),
				},
			}, err
		}

		username, token, err := getCredentials()
		if err != nil {
			return &resource.UpdateResult{
				ProgressResult: &resource.ProgressResult{
					Operation:       resource.OperationUpdate,
					OperationStatus: resource.OperationStatusFailure,
					ErrorCode:       resource.OperationErrorCodeAccessDenied,
					StatusMessage:   err.Error(),
				},
			}, err
		}

		urlparams := url.Values{
			"vmid":        {desir.VMID},
			"hostname":    {desir.Hostname},
			"cores":       {strconv.Itoa(desir.Cores)},
			"memory":      {strconv.Itoa(desir.Memory)},
			"description": {desir.Description},
		}

		_, err = authenticatedRequest(http.MethodPut, config.URL+"/api2/json/nodes/"+config.NODE+"/lxc/"+desir.VMID+"/config", createAuthorizationString(username, token), urlparams)

		if err != nil {
			return &resource.UpdateResult{
				ProgressResult: &resource.ProgressResult{
					Operation:       resource.OperationCreate,
					OperationStatus: resource.OperationStatusFailure,
					ErrorCode:       resource.OperationErrorCodeInternalFailure,
					StatusMessage:   err.Error(),
				},
			}, err
		}
	}

	result, err := p.Read(ctx, &resource.ReadRequest{
		NativeID:     req.NativeID,
		ResourceType: req.ResourceType,
		TargetConfig: req.TargetConfig,
	})

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           req.NativeID,
			ResourceProperties: json.RawMessage(result.Properties),
		},
	}, nil
}

// Delete removes a resource.
func (p *Plugin) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		log.Println(err.Error())
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, err
	}

	username, token, err := getCredentials()
	if err != nil {
		log.Println(err.Error())
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, err
	}

	_, err = authenticatedRequest(http.MethodDelete, config.URL+"/api2/json/nodes/"+config.NODE+"/lxc/"+req.NativeID, createAuthorizationString(username, token), nil)

	if err != nil {
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, err
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        req.NativeID,
		},
	}, nil

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

	username, token, err := getCredentials()
	if err != nil {
		return &resource.ListResult{
			NativeIDs: []string{},
		}, err
	}

	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		return &resource.ListResult{
			NativeIDs: []string{},
		}, err
	}

	var props StatusGeneralResponse

	data, err := authenticatedRequest(http.MethodGet, config.URL+"/api2/json/nodes/"+config.NODE+"/lxc", createAuthorizationString(username, token), nil)
	if err != nil {
		return &resource.ListResult{
			NativeIDs: []string{},
		}, err
	}

	json.Unmarshal(data, &props)

	nativeIds := make([]string, 0, len(props.Data))

	for _, value := range props.Data {
		nativeIds = append(nativeIds, strconv.Itoa(value.VMID))
	}

	return &resource.ListResult{
		NativeIDs:     nativeIds,
		NextPageToken: nil,
	}, nil
}
