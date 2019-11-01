/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package driver

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/intel/rsp-sw-toolkit-im-suite-mqtt-device-service/internal/jsonrpc"
	"strings"
	"time"

	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
)

const (
	RSPPrefix = "RSP"
)

// HandleReadCommands is the entrypoint for a command from EdgeX command service
// The commands will be sent via mqtt to the rsp controller and response will be given
// back to EdgeX for returning to the caller
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

// handleReadCommandRequest is the internal code to send commands over mqtt to
// the rsp controller
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
	// cleanup
	defer func() {
		driver.responseMap.Delete(requestId)
		close(responseChan)
	}()

	if err := driver.publishCommand(request); err != nil {
		return nil, err
	}

	timeout := time.NewTimer(time.Duration(driver.Config.MaxWaitTimeForReq) * time.Second)
	defer timeout.Stop()

	// wait for either the response or a timeout
	for {
		select {
		case response := <-responseChan:
			if response.Id == requestId {
				// if these are the droids we are looking for, format a response object for sending back to EdgeX
				return driver.createEdgeXResponse(req.DeviceResourceName, response)
			}
		case <-timeout.C:
			return nil, fmt.Errorf("timed out waiting for command response for request: %+v", request)
		case <-driver.done:
			return nil, errors.New("done signaled. ignoring response")
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
