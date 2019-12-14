// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018-2019 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package driver

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/intel/rsp-sw-toolkit-im-suite-mqtt-device-service/internal/jsonrpc"
	"github.com/pkg/errors"
	"time"

	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
)

// onCommandResponseReceived handles messages on the response topic and parses them as jsonrpc 2.0 Response messages
func (driver *Driver) onCommandResponseReceived(message mqtt.Message) {
	var response jsonrpc.Response

	if err := json.Unmarshal(message.Payload(), &response); err != nil {
		driver.Logger.Error("[Response listener] Unmarshalling of command response failed", "cause", err.Error())
		return
	}

	if response.Id != "" {
		driver.Logger.Info("[Response listener] Command response received", "topic", message.Topic(), "msg", string(message.Payload()))
		if responseChan, ok := driver.responseMap.Load(response.Id); ok {
			responseChan.(chan *jsonrpc.Response) <- &response
		}
	} else {
		driver.Logger.Debug("[Response listener] Command response ignored. No ID found in the message",
			"topic", message.Topic(), "msg", string(message.Payload()))
	}
}

func (driver *Driver) createEdgeXResponse(deviceResourceName string, response *jsonrpc.Response) (*sdkModel.CommandValue, error) {
	// Return just the result or error field from the jsonrpc response
	origin := time.Now().UnixNano() / int64(time.Millisecond)

	if len(response.Result) > 0 {
		if err := driver.validateResponse(deviceResourceName, response.Result); err != nil {
			return nil, errors.Wrapf(err, "Validation failed for %q: %+v", deviceResourceName, err)
		}
		driver.Logger.Info("Get command finished successfully", "response.result", string(response.Result))
		return sdkModel.NewStringValue(deviceResourceName, origin, string(response.Result)), nil

	} else if len(response.Error) > 0 {
		driver.Logger.Info("Get command finished with an error", "response.error", string(response.Error))
		return nil, fmt.Errorf(string(response.Error))
	}

	return nil, fmt.Errorf("response message missing both result and error field, unable to process. response: %+v", response)
}
