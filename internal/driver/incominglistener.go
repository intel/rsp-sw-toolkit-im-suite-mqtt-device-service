// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
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
	// Topic will be set by us and sent upstream, indicating the topic on which
	// the original JSON message came.
	Topic   string `json:"topic"`
	// Params is rest of the message from which we'll extract the Gateway's ID.
	Params json.RawMessage `json:"params"`
}

// EitherID is used to unmarshal the Gateway's ID, regardless of how it came
type EitherID struct {
	GatewayID *optString `json:"gateway_id"`
	DeviceID  *optString `json:"device_id"`
}

// optString is used for optional strings (and should be used as a pointer)
type optString string
func (id *optString) isNilOrEmpty() bool {
	return id == nil || *id == ""
}

func (jn *JSONNotification) getID() (string, error) {
	if jn == nil || len(jn.Params) == 0 {
		return "", errors.New("JSON notification is nil or is missing parameters")
	}

	var ids EitherID
	if err := json.Unmarshal(jn.Params, &ids); err != nil {
		return "", errors.Wrap(err, "unable to unmarshal the gateway ID")
	}

	if !ids.GatewayID.isNilOrEmpty() {
		return string(*(ids.GatewayID)), nil
	}
	if !ids.DeviceID.isNilOrEmpty() {
		return string(*(ids.DeviceID)), nil
	}
	return "", errors.New("neither gateway_id nor device_id found in message")
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

	deviceName, err := jn.getID()
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("Failed to get device ID: %+v", err))
		return
	}

	jn.Topic = message.Topic()
	remarshaled, err := json.Marshal(jn)
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("Failed to remashal message: %+v", err))
		return
	}

	event := "gwevent"
	reading := string(remarshaled)
	service := sdk.RunningService()

	deviceObject, ok := service.DeviceObject(deviceName, event, "get")
	if !ok {
		driver.Logger.Warn(fmt.Sprintf("[Incoming listener] "+
			"Incoming reading ignored. "+
			"No DeviceObject found: topic=%v msg=%v",
			message.Topic(), string(message.Payload())))
		return
	}

	ro, ok := service.ResourceOperation(deviceName, event, "get")
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
