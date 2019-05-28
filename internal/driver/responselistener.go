// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/eclipse/paho.mqtt.golang"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/models"
)

func startCommandResponseListening() error {
	var scheme = driver.Config.Response.Protocol
	var brokerUrl = driver.Config.Response.Host
	var brokerPort = driver.Config.Response.Port
	var username = driver.Config.Response.Username
	var password = driver.Config.Response.Password
	var mqttClientId = driver.Config.Response.MqttClientId
	var qos = byte(driver.Config.Response.Qos)
	var keepAlive = driver.Config.Response.KeepAlive
	var topics = driver.Config.Response.Topics

	uri := &url.URL{
		Scheme: strings.ToLower(scheme),
		Host:   fmt.Sprintf("%s:%d", brokerUrl, brokerPort),
		User:   url.UserPassword(username, password),
	}

	client, err := createClient(mqttClientId, uri, keepAlive)
	defer client.Disconnect(5000)
	if err != nil {
		return err
	}

	for _, topic := range topics {
		token := client.Subscribe(topic, qos, onCommandResponseReceived)
		if token.Wait() && token.Error() != nil {
			driver.Logger.Info(fmt.Sprintf("[Response listener] Stop command response listening. Cause:%v", token.Error()))
			return token.Error()
		}
	}

	driver.Logger.Info("[Response listener] Start command response listening. ")
	select {}
}

func onCommandResponseReceived(client mqtt.Client, message mqtt.Message) {
	var response models.JsonResponse

	if err := json.Unmarshal(message.Payload(), &response); err != nil {
		driver.Logger.Error(fmt.Sprintf("[Response listener] Unmarshal failed: %+v", err))
		return
	}

	if response.Id != "" {
		driver.CommandResponses[response.Id] = string(message.Payload())
		driver.Logger.Info(fmt.Sprintf("[Response listener] Command response received: topic=%v msg=%v", message.Topic(), string(message.Payload())))
	} else {
		driver.Logger.Warn(fmt.Sprintf("[Response listener] Command response ignored. No ID found in the message: topic=%v msg=%v", message.Topic(), string(message.Payload())))
	}
}
