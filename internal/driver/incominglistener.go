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
	var topic = driver.Config.Incoming.Topic

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

	token := client.Subscribe(topic, qos, onIncomingDataReceived)
	if token.Wait() && token.Error() != nil {
		driver.Logger.Info(
			fmt.Sprintf("[Incoming listener] Stop incoming data listening. Cause:%v",
				token.Error(),
			),
		)
		return token.Error()
	}

	driver.Logger.Info("[Incoming listener] Start incoming data listening.")
	select {}
}

type TagEvent struct {
	EpcCode         string `json:"epc_code"`
	Tid             string `json:"tid"`
	EpcEncodeFormat string `json:"epc_encode_format"`
	FacilityID      string `json:"facility_id"`
	Location        string `json:"location"`
	EventType       string `json:"event_type,omitempty"`
	Timestamp       int64  `json:"timestamp"`
}

type JSONRPC struct {
	Version string `json:"jsonrpc"`
	Method  string
	Params  json.RawMessage
}

type GatewayEventIn struct {
	GatewayID     string `json:"gateway_id"`
	SentOn        int64  `json:"sent_on"`
	TotalSegments int    `json:"total_event_segments"`
	SegmentNumber int    `json:"event_segment_number"`
	// Data          []TagEvent
	Data json.RawMessage
}

func (jrpc *JSONRPC) process() (gwEvent GatewayEventIn, err error) {
	if jrpc.Version != "2.0" {
		err = errors.Errorf("invalid version: %s", jrpc.Version)
		return
	}
	switch jrpc.Method {
	case "inventory_event":
		err = json.Unmarshal(jrpc.Params, &gwEvent)
	default:
		err = errors.Errorf("unknown method: %s", jrpc.Method)
	}
	return
}

func onIncomingDataReceived(client mqtt.Client, message mqtt.Message) {
	var jrpc JSONRPC
	if err := json.Unmarshal(message.Payload(), &jrpc); err != nil {
		driver.Logger.Error(fmt.Sprintf("Unmarshal failed: %+v", err))
		return
	}

	gwEvent, err := jrpc.process()
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("Processing failed: %+v", err))
		return
	}

	deviceName := gwEvent.GatewayID
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
