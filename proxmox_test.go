package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
	"github.com/stretchr/testify/require"
)

func testTargetConfig() json.RawMessage {
	return json.RawMessage(`{"url": "https://proxmox.mid:8006", "node": "proxmox"}`)
}

type LXC struct {
	VMID int    `json:"vmid"`
	NAME string `json:"name"`
}

type RESP_DATA struct {
	DATA []LXC `json:"data"`
}

func TestCreate(t *testing.T) {
	username := os.Getenv("PROXMOX_USERNAME")
	token := os.Getenv("PROXMOX_TOKEN")
	if username == "" {
		t.Skip("PROXMOX_USERNAME not set")
	}
	if token == "" {
		t.Skip("PROXMOX_TOKEN not set")
	}

	plugin := &Plugin{}
	ctx := context.Background()

	properties := map[string]any{
		"vmid":        200,
		"name":        "testlxc",
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

		var props RESP_DATA

		request, err := http.NewRequest("GET", config.URL+"/api2/json/nodes/"+config.NODE+"/lxc", nil)
		if err != nil {
			t.Logf("Something unexpected happened")
			return false
		}
		request.Header.Set("Authorization", "PVEAPIToken="+username+"="+token)

		resp, err := client.Do(request)

		data, err := io.ReadAll(resp.Body)
		json.Unmarshal(data, &props)

		for i := 0; i < len(props.DATA); i++ {
			lxccontainer := props.DATA[i]
			t.Logf("Found container: %s", lxccontainer.NAME)
			if lxccontainer.VMID == 200 {
				t.Logf("Created Successfully: %s", lxccontainer.NAME)
				return true
			}
		}

		return false
	}, 10*time.Second, time.Second, "Create operation should complete successfully")
}
