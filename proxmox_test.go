package main

import (
	"context"
	"encoding/json"
	"io"
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
	}

	propertiesJSON, err := json.Marshal(properties)

	require.NoError(t, err, "failed to marshal properties")

	req := &resource.CreateRequest{
		ResourceType: "PROXMOX::Service::LXC",
		Label:        "test-create",
		Properties:   propertiesJSON,
		TargetConfig: testTargetConfig(),
	}

	config, err := parseTargetConfig(testTargetConfig())

	result, err := plugin.Create(ctx, req)

	require.NoError(t, err, "Create should not return error")
	require.NotNil(t, result.ProgressResult, "Create should return ProgressResult")

	require.Eventually(t, func() bool {
		client := &http.Client{}

		var props StatusGeneralResponse

		request, err := http.NewRequest("GET", config.URL+"/api2/json/nodes/"+config.NODE+"/lxc", nil)
		if err != nil {
			t.Logf("Something unexpected happened")
			return false
		}
		request.Header.Set("Authorization", "PVEAPIToken="+username+"="+token)

		resp, err := client.Do(request)

		data, err := io.ReadAll(resp.Body)

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
		NativeID:     strconv.Itoa(120),
		ResourceType: "PROXMOX::Service::LXC",
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
	require.Equal(t, "ntfy", props["hostname"], "hostname should match")
	require.Equal(t, strconv.Itoa(120), props["vmid"], "vmid should match")
}
