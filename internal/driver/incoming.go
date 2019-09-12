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
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/jsonrpc"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	sdk "github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
)

const (
	sensorHeartbeat = "heartbeat"
	deviceIdKey     = "device_id"
	inventoryEvent  = "inventory_event"
	tagDataKey      = "epc"
	uriDataKey      = "uri"
)

func (driver *Driver) onIncomingDataReceived(message mqtt.Message) {
	outgoing := message.Payload()
	var incomingData jsonrpc.Notification
	if err := json.Unmarshal(message.Payload(), &incomingData); err != nil {
		driver.Logger.Error(fmt.Sprintf("Unmarshal failed. cause=%+v payload=%s messageObject=%+v",
			err, string(outgoing), message))
		return
	}

	if incomingData.Version != jsonRpcVersion {
		driver.Logger.Error(fmt.Sprintf("Invalid version: %s", incomingData.Version))
		return
	}

	// JsonRpc Responses do not contain a method field.
	// We also do not want to send these to core-data
	resourceName := incomingData.Method
	if resourceName == "" {
		driver.Logger.Warn("[Incoming listener] "+
			"Incoming reading ignored. "+
			"No method field in message.",
			"msg", string(outgoing))
		return
	}

	modified, err := driver.processResource(incomingData)
	if err != nil {
		driver.Logger.Error("Failed to handle %q: %+v", resourceName, err)
		return
	}
	if modified != nil {
		outgoing = modified
	}

	origin := time.Now().UnixNano() / int64(time.Millisecond)
	value := sdkModel.NewStringValue(resourceName, origin, string(outgoing))

	driver.Logger.Info("[Incoming listener] Incoming reading received",
		"topic", message.Topic(),
		"method", incomingData.Method,
		"msgLen", len(message.Payload()))

	driver.AsyncCh <- &sdkModel.AsyncValues{
		DeviceName:    driver.Config.ControllerName,
		CommandValues: []*sdkModel.CommandValue{value},
	}
}

func (driver *Driver) processResource(data jsonrpc.Notification) (modified []byte, err error) {
	switch data.Method {
	case sensorHeartbeat:
		// Register new (i.e., currently unregistered) sensors with EdgeX
		var deviceID string
		deviceID, err = data.GetParamStr(deviceIdKey)
		if err != nil {
			return
		}

		if _, notFound := sdk.RunningService().GetDeviceByName(deviceID); notFound != nil {
			driver.registerRSP(deviceID)
		}

	case inventoryEvent:
		var tagData, URI string
		tagData, err = data.GetParamStr(tagDataKey)
		if err == nil {
			URI, err = driver.DecoderRing.TagDataToURI(tagData)
		}
		if err == nil {
			err = data.SetParam(uriDataKey, URI)
		}
		if err == nil {
			modified, err = json.Marshal(data) // update the outgoing payload
		}
	}

	return
}
