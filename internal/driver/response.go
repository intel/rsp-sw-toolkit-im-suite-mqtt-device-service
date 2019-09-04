// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018-2019 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

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
	"github.com/eclipse/paho.mqtt.golang"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/jsonrpc"
	"time"

	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
)

// onCommandResponseReceived handles messages on the response topic and parses them as jsonrpc 2.0 Response messages
func (driver *Driver) onCommandResponseReceived(message mqtt.Message) {
	var response jsonrpc.Response

	if err := json.Unmarshal(message.Payload(), &response); err != nil {
		driver.Logger.Error("[Response listener] Unmarshalling of command response failed", "cause", err)
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

	if response.Result != nil && len(response.Result) > 0 {
		driver.Logger.Info("Get command finished successfully", "response.result", string(response.Result))
		return sdkModel.NewStringValue(deviceResourceName, origin, string(response.Result)), nil

	} else if response.Error != nil && len(response.Error) > 0 {
		driver.Logger.Info("Get command finished with an error", "response.error", string(response.Error))
		return nil, fmt.Errorf(string(response.Error))
	}

	return nil, fmt.Errorf("response message missing both result and error field, unable to process. response: %+v", response)
}
