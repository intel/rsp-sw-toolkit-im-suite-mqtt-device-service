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
)

// onCommandResponseReceived handles messages on the response topic and parses them as jsonrpc 2.0 Response messages
func (driver *Driver) onCommandResponseReceived(_ mqtt.Client, message mqtt.Message) {
	var response jsonrpc.Response

	if err := json.Unmarshal(message.Payload(), &response); err != nil {
		driver.Logger.Error("[Response listener] Unmarshalling of command response failed", "cause", err)
		return
	}

	if response.Id != "" {
		driver.CommandResponses.Store(response.Id, string(message.Payload()))
		driver.Logger.Info("[Response listener] Command response received", "topic", message.Topic(), "msg", string(message.Payload()))
	} else {
		driver.Logger.Debug("[Response listener] Command response ignored. No ID found in the message",
			"topic", message.Topic(), "msg", string(message.Payload()))
	}
}

// fetchCommandResponse use to wait and fetch response from CommandResponses map
func (driver *Driver) fetchCommandResponse(requestId string) (string, bool) {
	var response interface{}
	var ok bool
	for i := 0; i < driver.Config.MaxWaitTimeForReq; i++ {
		response, ok = driver.CommandResponses.Load(requestId)
		if ok {
			driver.CommandResponses.Delete(requestId)
			break
		} else {
			time.Sleep(time.Second * time.Duration(1))
		}
	}

	return fmt.Sprintf("%v", response), ok
}
