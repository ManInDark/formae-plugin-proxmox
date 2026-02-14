package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

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

func (p *Plugin) CreateLXC(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	props, err := parseLXCProperties(req.Properties)
	if err != nil {
		slog.Error(err.Error())
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
		slog.Error(err.Error())
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
		slog.Error(err.Error())
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
		"vmid":            {props.VMID},
		"ostemplate":      {props.OSTemplate},
		"password":        {props.Password},
		"hostname":        {props.Hostname},
		"cores":           {strconv.Itoa(props.Cores)},
		"memory":          {strconv.Itoa(props.Memory)},
		"ssh-public-keys": {strings.Join(props.SSHKeys, "\n")},
	}
	if props.Description != "" {
		urlparams.Add("description", props.Description)
	}
	if props.OnBoot != 0 {
		urlparams.Add("onboot", strconv.Itoa(props.OnBoot))
	}

	data, err := authenticatedRequest(http.MethodPost, config.URL+"/api2/json/nodes/"+config.NODE+"/lxc", createAuthorizationString(username, token), urlparams)

	if err != nil {
		slog.Error(err.Error())
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, err
	}

	var taskData ProxmoxDataResponse

	err = json.Unmarshal(data, &taskData)

	if err != nil {
		slog.Error(err.Error())
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
			RequestID:       taskData.Data,
			NativeID:        props.VMID,
		},
	}, nil
}

func (p *Plugin) ReadLXC(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	username, token, err := getCredentials()
	if err != nil {
		slog.Error(err.Error())
		return &resource.ReadResult{
			ErrorCode: resource.OperationErrorCodeInvalidRequest,
		}, err
	}

	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		slog.Error(err.Error())
		return &resource.ReadResult{}, nil
	}

	data, err := authenticatedRequest(http.MethodGet, config.URL+"/api2/json/nodes/"+config.NODE+"/lxc/"+req.NativeID+"/config", createAuthorizationString(username, token), nil)
	if err != nil {
		slog.Error(err.Error())
		return &resource.ReadResult{
			ErrorCode: resource.OperationErrorCodeNetworkFailure,
		}, err
	}

	var props StatusLXCConfigResponse

	err = json.Unmarshal(data, &props)
	if err != nil {
		slog.Error("Error unmarshalling json", "data", data, "err", err.Error())
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
		OnBoot:      lxcdata.OnBoot,
	}

	propsJSON, err := json.Marshal(properties)
	if err != nil {
		slog.Error(err.Error())
		return &resource.ReadResult{
			ErrorCode: resource.OperationErrorCodeInternalFailure,
		}, err
	}

	return &resource.ReadResult{
		ResourceType: req.ResourceType,
		Properties:   string(propsJSON),
	}, nil
}

func (p *Plugin) UpdateLXC(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	prior, err := parseLXCProperties(req.PriorProperties)
	if err != nil {
		slog.Error(err.Error())
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
		slog.Error(err.Error())
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

	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		slog.Error(err.Error())
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
		slog.Error(err.Error())
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
	if prior.OnBoot != desir.OnBoot {
		urlparams.Add("onboot", strconv.Itoa(desir.OnBoot))
	}

	_, err = authenticatedRequest("PUT", config.URL+"/api2/json/nodes/"+config.NODE+"/lxc/"+desir.VMID+"/config", createAuthorizationString(username, token), urlparams)

	if err != nil {
		slog.Error(err.Error())
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, err
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

func (p *Plugin) DeleteLXC(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		slog.Error(err.Error())
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
		slog.Error(err.Error())
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, err
	}

	_, err = authenticatedRequest(http.MethodDelete, config.URL+"/api2/json/nodes/"+config.NODE+"/lxc/"+req.NativeID+"?force=1&purge=1", createAuthorizationString(username, token), nil)

	if err != nil {
		slog.Error(err.Error())
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

type RequestStatusProxmoxResponse struct {
	PId        int    `json:"pid"`
	UpId       string `json:"upid"`
	Node       string `json:"node"`
	PStart     int    `json:"pstart"`
	Status     string `json:"status"`
	Id         string `json:"id"`
	StartTime  int    `json:"starttime"`
	ExitStatus string `json:"exitstatus"`
	User       string `json:"user"`
	Type       string `json:"type"`
}

func (p *Plugin) StatusLXC(ctx context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		slog.Error(err.Error())
		return &resource.StatusResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCheckStatus,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, err
	}

	username, token, err := getCredentials()
	if err != nil {
		slog.Error(err.Error())
		return &resource.StatusResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCheckStatus,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, err
	}

	var proxmoxResponse RequestStatusProxmoxResponse

	data, err := authenticatedRequest(http.MethodDelete, config.URL+"/api2/json/nodes/"+config.NODE+"/tasks/"+req.RequestID+"/status", createAuthorizationString(username, token), nil)

	err = json.Unmarshal(data, &proxmoxResponse)
	if err != nil {
		slog.Error(err.Error())
		return &resource.StatusResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCheckStatus,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   err.Error(),
			},
		}, err
	}

	var status resource.OperationStatus

	switch proxmoxResponse.Status {
	case "running":
		status = resource.OperationStatusInProgress
	case "stopped":
		status = resource.OperationStatusSuccess
	}

	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: status,
		},
	}, nil
}

func (p *Plugin) ListLXC(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {

	username, token, err := getCredentials()
	if err != nil {
		slog.Error(err.Error())
		return &resource.ListResult{
			NativeIDs: []string{},
		}, err
	}

	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		slog.Error(err.Error())
		return &resource.ListResult{
			NativeIDs: []string{},
		}, err
	}

	var props StatusGeneralResponse

	data, err := authenticatedRequest(http.MethodGet, config.URL+"/api2/json/nodes/"+config.NODE+"/lxc", createAuthorizationString(username, token), nil)
	if err != nil {
		slog.Error(err.Error())
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
