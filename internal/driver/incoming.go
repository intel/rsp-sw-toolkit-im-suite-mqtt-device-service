/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package driver

import (
	"encoding/json"
	"github.com/intel/rsp-sw-toolkit-im-suite-mqtt-device-service/internal/jsonrpc"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
)

const (
	sensorHeartbeat        = "heartbeat"
	inventoryEvent         = "inventory_data"
	controllerStatusUpdate = "rsp_controller_status_update"

	deviceIdKey  = "device_id"
	tagDataKey   = "epc"
	uriDataKey   = "uri"
	statusKey    = "status"
	paramDataKey = "data"

	controllerReady = "controller_ready"
)

func (driver *Driver) onIncomingDataReceived(message mqtt.Message) {
	outgoing := message.Payload()

	var incomingData jsonrpc.Notification
	if err := json.Unmarshal(message.Payload(), &incomingData); err != nil {
		driver.Logger.Error("Unmarshal failed.",
			"cause", err.Error(),
			"payload", string(outgoing),
			"message", message)
		return
	}

	if incomingData.Version != jsonRpcVersion {
		driver.Logger.Error("Invalid JSON RPC version",
			"incoming", incomingData.Version, "expected", jsonRpcVersion)
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

	if err := driver.validateIncoming(incomingData.Method, outgoing); err != nil {
		driver.Logger.Error("Schema validation failed",
			"resourceName", resourceName, "cause", err.Error())
		return
	}

	modified, err := driver.processResource(incomingData)
	if err != nil {
		driver.Logger.Error("Incoming resource processing failed",
			"resourceName", resourceName, "cause", err.Error())
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
		var deviceId string
		err = data.GetParam(deviceIdKey, &deviceId)
		if err != nil {
			return
		}
		driver.registerDeviceIfNeeded(deviceId, rspDeviceProfile)

	case inventoryEvent:
		var inventoryData []jsonrpc.Parameters
		err = data.GetParam(paramDataKey, &inventoryData)
		if err != nil {
			return
		}

		for i := 0; i < len(inventoryData); i++ {
			var tagData, URI string
			err = inventoryData[i].Get(tagDataKey, &tagData)
			if err != nil {
				return
			}
			URI, err = driver.DecoderRing.TagDataToURI(tagData)
			if err != nil {
				return
			}
			err = inventoryData[i].Set(uriDataKey, URI)
			if err != nil {
				return
			}
		}

		err = data.SetParam(paramDataKey, inventoryData)
		if err != nil {
			return
		}
		modified, err = json.Marshal(data) // update the outgoing payload

	case controllerStatusUpdate:
		var status string
		err = data.GetParam(statusKey, &status)
		if err != nil {
			return
		}

		if status == controllerReady {
			// tell the RSP controller which notifications we want to subscribe to
			go driver.configureControllerNotifications()
		}
	}

	return
}
