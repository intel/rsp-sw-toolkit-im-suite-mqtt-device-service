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
	sdk "github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
)

func startIncomingListening() error {
	var scheme = driver.Config.Incoming.Protocol
	var brokerUrl = driver.Config.Incoming.Host
	var brokerPort = driver.Config.Incoming.Port
	var username = driver.Config.Incoming.Username
	var password = driver.Config.Incoming.Password
	var mqttClientId = driver.Config.Incoming.MqttClientId
	var qos = byte(driver.Config.Incoming.Qos)
	var keepAlive = driver.Config.Incoming.KeepAlive
	var topics = driver.Config.Incoming.Topics

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
		token := client.Subscribe(topic, qos, onIncomingDataReceived)
		if token.Wait() && token.Error() != nil {
			driver.Logger.Info(
				fmt.Sprintf("[Incoming listener] Stop incoming data listening. Cause:%v",
					token.Error(),
				),
			)
			return token.Error()
		}
	}

	driver.Logger.Info("[Incoming listener] Start incoming data listening.")
	select {}
}

type JSONNotification struct {
	Version string `json:"jsonrpc"`
	Method  string
	Params  EitherID
}

type eitherID string
type EitherID struct {
	GatewayID *eitherID `json:"gateway_id"`
	DeviceID  *eitherID `json:"device_id"`
}

func (id *eitherID) isNilOrEmpty() bool {
	return id == nil || *id == ""
}

func (jn *JSONNotification) getID() string {
	if jn == nil {
		return ""
	}
	if !jn.Params.GatewayID.isNilOrEmpty() {
		return string(*(jn.Params.GatewayID))
	}
	if !jn.Params.DeviceID.isNilOrEmpty() {
		return string(*(jn.Params.DeviceID))
	}
	return ""
}

func onIncomingDataReceived(client mqtt.Client, message mqtt.Message) {
	var jn JSONNotification
	if err := json.Unmarshal(message.Payload(), &jn); err != nil {
		driver.Logger.Error(fmt.Sprintf("Unmarshal failed: %+v", err))
		return
	}

	if jn.Version != "2.0" {
		driver.Logger.Error(fmt.Sprintf("Invalid version: %s", jn.Version))
		return
	}

	deviceName := jn.getID()
	if deviceName == "" {
		driver.Logger.Error("Message is missing a device/gateway ID")
		return
	}
	cmd := "gwevent"
	reading := string(message.Payload())

	service := sdk.RunningService()

	deviceObject, ok := service.DeviceObject(deviceName, cmd, "get")
	if !ok {
		driver.Logger.Warn(fmt.Sprintf("[Incoming listener] "+
			"Incoming reading ignored. "+
			"No DeviceObject found: topic=%v msg=%v",
			message.Topic(), string(message.Payload())))
		return
	}

	ro, ok := service.ResourceOperation(deviceName, cmd, "get")
	if !ok {
		driver.Logger.Warn(fmt.Sprintf("[Incoming listener] "+
			"Incoming reading ignored. "+
			"No ResourceOperation found: topic=%v msg=%v",
			message.Topic(), string(message.Payload())))
		return
	}

	result, err := newResult(deviceObject, ro, reading)

	if err != nil {
		driver.Logger.Warn(fmt.Sprintf("[Incoming listener] "+
			"Incoming reading ignored. "+
			"topic=%v msg=%v error=%v",
			message.Topic(), string(message.Payload()), err))
		return
	}

	asyncValues := &sdkModel.AsyncValues{
		DeviceName:    deviceName,
		CommandValues: []*sdkModel.CommandValue{result},
	}

	driver.Logger.Info(fmt.Sprintf("[Incoming listener] "+
		"Incoming reading received: "+
		"topic=%v msg=%v",
		message.Topic(), string(message.Payload())))

	driver.AsyncCh <- asyncValues
}
