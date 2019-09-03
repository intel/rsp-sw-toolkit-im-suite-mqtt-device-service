/*
 * INTEL CONFIDENTIAL
 * Copyright (2019) Intel Corporation.
 *
 * The source code contained or described herein and all documents related to the source code ("Material")
 * are owned by Intel Corporation or its suppliers or licensors. Title to the Material remains with
 * Intel Corporation or its suppliers and licensors. The Material may contain trade secrets and proprietary
 * and confidential information of Intel Corporation and its suppliers and licensors, and is protected by
 * worldwide copyright and trade secret laws and treaty provisions. No part of the Material may be used,
 * copied, reproduced, modified, published, uploaded, posted, transmitted, distributed, or disclosed in
 * any way without Intel/'s prior express written permission.
 * No license under any patent, copyright, trade secret or other intellectual property right is granted
 * to or conferred upon you by disclosure or delivery of the Materials, either expressly, by implication,
 * inducement, estoppel or otherwise. Any license under such intellectual property rights must be express
 * and approved by Intel in writing.
 * Unless otherwise agreed by Intel in writing, you may not remove or alter this notice or any other
 * notice embedded in Materials by Intel or Intel's suppliers or licensors in any way.
 */

package driver

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/jsonrpc"
	"strings"
	"time"

	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
)

const (
	RSPPrefix = "RSP"
)

// HandleReadCommands handles CommandRequests to read data via MQTT.
//
// It satisfies them by creating a new MQTT client with the protocol, sending the
// requests as JSON RPC messages on all configured topics, then waiting for a
// response on any of the response topics; once a response comes in, it returns
// that result.
func (driver *Driver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModel.CommandRequest) ([]*sdkModel.CommandValue, error) {
	var responses = make([]*sdkModel.CommandValue, len(reqs))
	var err error

	for i, req := range reqs {
		res, err := driver.handleReadCommandRequest(deviceName, req)
		if err != nil {
			driver.Logger.Warn("Handle read commands failed", "cause", err)
			return responses, err
		}

		responses[i] = res
	}

	return responses, err
}

// handleReadCommandRequest takes care of the JSON RPC command/response portion
// of the HandleReadCommands.
//
// The command request is published on all of the incoming connection info topics.
func (driver *Driver) handleReadCommandRequest(deviceName string, req sdkModel.CommandRequest) (*sdkModel.CommandValue, error) {
	method := req.DeviceResourceName
	var request jsonrpc.Message
	var requestId string

	// Sensor devices start with "RSP", this will not be needed in near future as Edgex is going to support GET requests with query parameters
	// If the device is sensor add the device_id as params to the command request
	if strings.HasPrefix(deviceName, RSPPrefix) {
		req := jsonrpc.NewRSPCommandRequest(method, deviceName)
		request, requestId = req, req.Id
	} else {
		req := jsonrpc.NewRequest(method)
		request, requestId = req, req.Id
	}

	responseChan := make(chan *jsonrpc.Response)
	driver.responseMap.Store(requestId, responseChan)
	defer func() {
		driver.responseMap.Delete(requestId)
		close(responseChan)
	}()

	if err := driver.publishCommand(request); err != nil {
		return nil, err
	}

	timeout := time.NewTimer(time.Duration(driver.Config.MaxWaitTimeForReq) * time.Second)

	// wait for either the response or a timeout
	for {
		select {
		case response := <-responseChan:
			driver.Logger.Info(fmt.Sprintf("received on response channel: %+v", response))
			if response.Id == requestId {
				driver.Logger.Info(fmt.Sprintf("response id matches!  %+v", response))
				// if these are the droids we are looking for, format a response object for sending back to EdgeX
				return driver.createEdgeXResponse(req.DeviceResourceName, response)
			}
			driver.Logger.Info(fmt.Sprintf("not the droids we are looking for :( %+v", response))
		case <-timeout.C:
			return nil, fmt.Errorf("timed out waiting for command response for request: %+v", request)
		}
	}
}

func (driver *Driver) publishCommand(request jsonrpc.Message) error {
	// marshal request to jsonrpc format
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return errors.Wrap(err, "marshalling of command request failed")
	}

	// Publish the command request
	driver.Logger.Info("Publish command", "command", string(requestBytes))
	driver.Client.Publish(driver.Config.CommandTopic, driver.Config.CommandQos, notRetained, requestBytes)
	return nil
}

// HandleWriteCommands ignores all requests; write commands (PUT requests) are not currently supported.
func (driver *Driver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModel.CommandRequest, params []*sdkModel.CommandValue) error {
	return nil
}
