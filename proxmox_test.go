package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/stretchr/testify/require"
)

func testTargetConfig() json.RawMessage {
	return json.RawMessage(`{"url": "https://proxmox.mid:8006", "node": "proxmox"}`)
}

func TestCreate(t *testing.T) {
	username, token, err := getCredentials()
	if err != nil {
		t.Skip(err)
	}

	plugin := &Plugin{}
	ctx := context.Background()

	properties := map[string]any{
		"vmid":        "200",
		"hostname":    "testlxc",
		"description": "none",
		"ostemplate":  "local:vztmpl/alpine-3.22-default_20250617_amd64.tar.xz",
		"password":    "password",
		"cores":       1,
		"memory":      512,
	}

	propertiesJSON, err := json.Marshal(properties)

	require.NoError(t, err, "failed to marshal properties")

	req := &resource.CreateRequest{
		ResourceType: "PROXMOX::Compute::LXC",
		Properties:   propertiesJSON,
		TargetConfig: testTargetConfig(),
	}

	config, err := parseTargetConfig(testTargetConfig())

	result, err := plugin.Create(ctx, req)

	require.NoError(t, err, "Create should not return error")
	require.NotNil(t, result.ProgressResult, "Create should return ProgressResult")

	require.Eventually(t, func() bool {
		var props StatusGeneralResponse

		data, _ := authenticatedRequest(http.MethodGet, config.URL+"/api2/json/nodes/"+config.NODE+"/lxc", createAuthorizationString(username, token), nil)

		json.Unmarshal(data, &props)

		for i := 0; i < len(props.Data); i++ {
			lxccontainer := props.Data[i]
			if lxccontainer.VMID == 200 {
				t.Logf("Created Successfully: %s", lxccontainer.Name)
				return true
			}
		}

		return false
	}, 10*time.Second, time.Second, "Create operation should complete successfully")
}

func TestRead(t *testing.T) {
	ctx := context.Background()

	plugin := &Plugin{}

	req := &resource.ReadRequest{
		NativeID:     strconv.Itoa(200),
		ResourceType: "PROXMOX::Compute::LXC",
		TargetConfig: testTargetConfig(),
	}

	result, err := plugin.Read(ctx, req)

	require.NoError(t, err, "Read should not return error")
	require.Empty(t, result.ErrorCode, "Read should not return error code")
	require.NotEmpty(t, result.Properties, "Read should return properties")

	var props map[string]any

	err = json.Unmarshal([]byte(result.Properties), &props)
	require.NoError(t, err, "json should be parsable")

	require.NoError(t, err, "Properties should be valid JSON")
	require.Equal(t, "testlxc", props["hostname"], "hostname should match")
	require.Equal(t, strconv.Itoa(200), props["vmid"], "vmid should match")
	const core_num float64 = 1
	require.Equal(t, core_num, props["cores"], "cores should match")
	const mem_num float64 = 512
	require.Equal(t, mem_num, props["memory"], "memory should match")
	const onboot float64 = 0
	require.Equal(t, onboot, props["onboot"], "memory should match")
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	plugin := &Plugin{}

	priorProperties, _ := json.Marshal(map[string]any{
		"vmid":        "200",
		"hostname":    "testlxc",
		"description": "none",
		"ostemplate":  "local:vztmpl/alpine-3.22-default_20250617_amd64.tar.xz",
		"cores":       1,
		"memory":      512,
	})

	desiredProperties, _ := json.Marshal(map[string]any{
		"vmid":        "200",
		"hostname":    "testlxc-updated",
		"description": "none",
		"ostemplate":  "local:vztmpl/alpine-3.22-default_20250617_amd64.tar.xz",
		"cores":       2,
		"memory":      1024,
		"onboot":      1,
	})

	req := &resource.UpdateRequest{
		NativeID:          "200",
		ResourceType:      "PROXMOX::Compute::LXC",
		PriorProperties:   priorProperties,
		DesiredProperties: desiredProperties,
		TargetConfig:      testTargetConfig(),
	}

	result, err := plugin.Update(ctx, req)
	require.NoError(t, err, "Update should not return error")
	require.NotNil(t, result.ProgressResult, "Update should return ProgressResult")
	require.Equal(t, resource.OperationStatusSuccess, result.ProgressResult.OperationStatus, "Update should return Success status")

	readReq := &resource.ReadRequest{
		NativeID:     strconv.Itoa(200),
		ResourceType: "PROXMOX::Compute::LXC",
		TargetConfig: testTargetConfig(),
	}

	readResult, err := plugin.Read(ctx, readReq)
	var props map[string]any

	err = json.Unmarshal([]byte(readResult.Properties), &props)
	require.Equal(t, "testlxc-updated", props["hostname"], "hostname should have changed")
	const core_num float64 = 2
	require.Equal(t, core_num, props["cores"], "cores should have changed")
	const mem_num float64 = 1024
	require.Equal(t, mem_num, props["memory"], "memory should have changed")
	const onboot float64 = 1
	require.Equal(t, onboot, props["onboot"], "onboot should have changed")
}

func TestList(t *testing.T) {
	ctx := context.Background()
	plugin := &Plugin{}

	result, err := plugin.List(ctx, &resource.ListRequest{
		ResourceType: "PROXMOX::Compute::LXC",
		TargetConfig: testTargetConfig(),
	})
	require.NoError(t, err, "ListRequest should not return an error")

	slog.Info("Received Ids", "ids", result.NativeIDs)

	require.Contains(t, result.NativeIDs, "200", "List should include created LXC")
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	plugin := &Plugin{}

	username, token, err := getCredentials()
	if err != nil {
		t.Skip(err)
	}

	config, err := parseTargetConfig(testTargetConfig())
	if err != nil {
		t.Skip(err)
	}

	result, err := plugin.Delete(ctx, &resource.DeleteRequest{
		ResourceType: "PROXMOX::Compute::LXC",
		TargetConfig: testTargetConfig(),
		NativeID:     "200",
	})

	require.NoError(t, err, "Create should not return error")
	require.NotNil(t, result.ProgressResult, "Create should return ProgressResult")

	require.Eventually(t, func() bool {
		var props StatusGeneralResponse

		data, _ := authenticatedRequest(http.MethodGet, config.URL+"/api2/json/nodes/"+config.NODE+"/lxc", createAuthorizationString(username, token), nil)

		json.Unmarshal(data, &props)

		for i := 0; i < len(props.Data); i++ {
			lxccontainer := props.Data[i]
			if lxccontainer.VMID == 200 {
				return false
			}
		}

		return true
	}, 10*time.Second, time.Second, "Create operation should complete successfully")
}
