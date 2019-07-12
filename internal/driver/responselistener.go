// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018-2019 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

/*
 * INTEL CONFIDENTIAL
 * Copyright (2017) Intel Corporation.
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
	"net/url"
	"strings"

	"github.com/eclipse/paho.mqtt.golang"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/models"
)

// startCommandResponseListening begins listening for messages on the command
// response channel.
func startCommandResponseListening(done <-chan interface{}) error {
	conf := driver.Config
	client, err := createClient(conf.ResponseClientId, &url.URL{
		Scheme: strings.ToLower(conf.ResponseScheme),
		Host:   fmt.Sprintf("%s:%d", conf.ResponseHost, conf.ResponsePort),
		User:   url.UserPassword(conf.ResponseUser, conf.ResponsePassword),
	}, conf.ResponseKeepAlive)
	if err != nil {
		return err
	}

	defer func() {
		if client.IsConnected() {
			client.Disconnect(5000)
		}
	}()

	for _, topic := range conf.ResponseTopics {
		token := client.Subscribe(topic, byte(conf.ResponseQos), onCommandResponseReceived)
		if token.Wait() && token.Error() != nil {
			driver.Logger.Info(
				fmt.Sprintf("[Incoming listener] Stop command response listening. Cause:%v",
					token.Error(),
				),
			)
			return token.Error()
		}
	}

	driver.Logger.Info("[Response listener] Start command response listener. ")
	<-done
	driver.Logger.Info("[Response listener] Stopping command response listener. ")
	return nil
}

// Modified by Intel to handle responses from Intel open source gateway
func onCommandResponseReceived(client mqtt.Client, message mqtt.Message) {
	var response models.JsonResponse

	if err := json.Unmarshal(message.Payload(), &response); err != nil {
		driver.Logger.Error(fmt.Sprintf("[Response listener] Unmarshal failed: %+v", err))
		return
	}

	if response.Id != "" {
		driver.CommandResponses.Store(response.Id, string(message.Payload()))
		driver.Logger.Info(fmt.Sprintf("[Response listener] Command response received: topic=%v msg=%v", message.Topic(), string(message.Payload())))
	} else {
		driver.Logger.Warn(fmt.Sprintf("[Response listener] Command response ignored. No ID found in the message: topic=%v msg=%v", message.Topic(), string(message.Payload())))
	}
}
