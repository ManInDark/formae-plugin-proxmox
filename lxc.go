package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

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
	if props.OnBoot != 0 {
		urlparams.Add("onboot", strconv.Itoa(props.OnBoot))
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

func (p *Plugin) ReadLXC(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
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
		log.Println("Error unmarshaling json: ", data)
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
	if prior.OnBoot != desir.OnBoot {
		urlparams.Add("onboot", strconv.Itoa(desir.OnBoot))
	}

	_, err = authenticatedRequest("PUT", config.URL+"/api2/json/nodes/"+config.NODE+"/lxc/"+desir.VMID+"/config", createAuthorizationString(username, token), urlparams)

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

func (p *Plugin) ListLXC(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {

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
