// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 IOTech Ltd
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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	sdk "github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/clients"
)

/*File is modified by Intel by adding some structs and functions to register Intel open source gateway
to Edgex and get data from the gateway into Edgex*/

type Addressable struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
}

type Device struct {
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	AdminState     string            `json:"adminState"`
	OperatingState string            `json:"operatingState"`
	Service        map[string]string `json:"service"`
	Profile        map[string]string `json:"profile"`
	Addressable    map[string]string `json:"addressable"`
}

// Modified by Intel to fix minor formatting issues.
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
	Method  string `json:"method"`
	// Topic will be set by us and sent upstream, indicating the topic on which
	// the original JSON message came.
	Topic string `json:"topic"`
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

// Modified by Intel to add better error handling and handle incoming data from Intel open source gateway
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

		driver.Logger.Info("Registering a new device...")

		// Register new Addressable
		if err := postAddressable(deviceName); err != nil {
			driver.Logger.Warn(fmt.Sprintf("Unable to register new addressable %s, error %s", deviceName, err.Error()))
			return
		}
		// Register new Device
		if err := postDevice(deviceName); err != nil {
			driver.Logger.Warn(fmt.Sprintf("Unable to register new device %s, error %s", deviceName, err.Error()))
		}
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

func postAddressable(deviceName string) error {

	endPointURL := fmt.Sprintf("http://%s:%d%s", clients.CoreMetaDataServiceKey, driver.Config.Incoming.MetaDataPort, clients.ApiAddressableRoute)

	driver.Logger.Debug(fmt.Sprintf("Adding new device to %s", endPointURL))

	payload := Addressable{Name: deviceName,
		Protocol: "TCP",
		Address:  deviceName,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", endPointURL, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		driver.Logger.Debug(fmt.Sprintf("Response Code error %d", resp.StatusCode))
		body, _ := ioutil.ReadAll(resp.Body)
		driver.Logger.Debug(fmt.Sprintf("response Body: %s", string(body)))
		return errors.New("Unable to register addressable")
	}

	return nil

}

func postDevice(deviceName string) error {

	endPointURL := fmt.Sprintf("http://%s:%d%s", clients.CoreMetaDataServiceKey, driver.Config.Incoming.MetaDataPort, clients.ApiDeviceRoute)

	driver.Logger.Debug(fmt.Sprintf("Adding new device to %s", endPointURL))

	payload := Device{
		Name:           deviceName,
		Description:    "Gateway Device MQTT Broker Connection",
		AdminState:     "unlocked",
		OperatingState: "enabled",
		Service: map[string]string{
			"name": "mqtt-device-service",
		},
		Profile: map[string]string{
			"name": "Gateway.Device.MQTT.Profile",
		},
		Addressable: map[string]string{
			"name": deviceName,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", endPointURL, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		driver.Logger.Debug(fmt.Sprintf("Response Code error %d", resp.StatusCode))
		body, _ := ioutil.ReadAll(resp.Body)
		driver.Logger.Debug(fmt.Sprintf("response Body: %s", string(body)))
		return errors.New("Unable to register device")
	}

	return nil

}
