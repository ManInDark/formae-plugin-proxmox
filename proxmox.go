// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"

	"github.com/platform-engineering-labs/formae/pkg/plugin"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// https://pve.proxmox.com/pve-docs/api-viewer/

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

func (p *Plugin) RateLimit() plugin.RateLimitConfig {
	return plugin.RateLimitConfig{
		Scope:                            plugin.RateLimitScopeNamespace,
		MaxRequestsPerSecondForNamespace: 10,
	}
}

func (p *Plugin) DiscoveryFilters() []plugin.MatchFilter {
	return nil
}

// LabelConfig returns the configuration for extracting human-readable labels
// from discovered resources.
func (p *Plugin) LabelConfig() plugin.LabelConfig {
	return plugin.LabelConfig{
		DefaultQuery:      "$.hostname",
		ResourceOverrides: map[string]string{
			// "PROXMOX::Service::SpecialResource": "$.DisplayName",
		},
	}
}

// =============================================================================
// CRUD Operations
// =============================================================================

func (p *Plugin) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	return p.CreateLXC(ctx, req)
}

func (p *Plugin) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	return p.ReadLXC(ctx, req)
}

func (p *Plugin) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	return p.UpdateLXC(ctx, req)
}

func (p *Plugin) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	return p.DeleteLXC(ctx, req)
}

func (p *Plugin) Status(ctx context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return p.StatusLXC(ctx, req)
}

// Called during discovery to find unmanaged resources.
func (p *Plugin) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	return p.ListLXC(ctx, req)
}
