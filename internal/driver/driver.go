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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"net/url"
	"strings"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/edgexfoundry/edgex-go/pkg/clients/logging"
	"github.com/edgexfoundry/edgex-go/pkg/models"
	commandModel "github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/models"
)

var once sync.Once
var driver *Driver

const (
	jsonRpc  = "2.0"
	qos      = byte(1)
	retained = false
)

type Driver struct {
	Logger           logger.LoggingClient
	AsyncCh          chan<- *sdkModel.AsyncValues
	CommandResponses map[string]string
	Config           *configuration
}

func NewProtocolDriver() sdkModel.ProtocolDriver {
	once.Do(func() {
		driver = new(Driver)
		driver.CommandResponses = make(map[string]string)
	})
	return driver
}

func (d *Driver) Initialize(lc logger.LoggingClient, asyncCh chan<- *sdkModel.AsyncValues) error {
	d.Logger = lc
	d.AsyncCh = asyncCh

	config, err := LoadConfigFromFile()
	if err != nil {
		panic(fmt.Errorf("read MQTT driver configuration failed: %v", err))
	}
	d.Config = config

	go func() {
		err := startCommandResponseListening()
		if err != nil {
			panic(fmt.Errorf("start command response Listener failed: %+v", err))
		}
	}()

	go func() {
		err := startIncomingListening()
		if err != nil {
			panic(fmt.Errorf("start incoming data Listener failed: %+v", err))
		}
	}()

	return nil
}

func (d *Driver) DisconnectDevice(address *models.Addressable) error {
	panic("implement me")
}

// Modified by Intel to add better error handling
func (d *Driver) HandleReadCommands(addr *models.Addressable, reqs []sdkModel.CommandRequest) ([]*sdkModel.CommandValue, error) {
	var responses = make([]*sdkModel.CommandValue, len(reqs))
	var err error

	// create device client and open connection
	var brokerUrl = d.Config.Command.Host
	var brokerPort = d.Config.Command.Port
	var username = d.Config.Command.Username
	var password = d.Config.Command.Password
	var mqttClientId = d.Config.Command.MqttClientId
	var topics = d.Config.Command.Topics

	uri := &url.URL{
		Scheme: strings.ToLower(d.Config.Command.Protocol),
		Host:   fmt.Sprintf("%s:%d", brokerUrl, brokerPort),
		User:   url.UserPassword(username, password),
	}

	client, err := createClient(mqttClientId, uri, 30)
	if err != nil {
		return responses, err
	}
	defer client.Disconnect(5000)

	for i, req := range reqs {
		res, err := d.handleReadCommandRequest(client, req, topics)
		if err != nil {
			driver.Logger.Info(fmt.Sprintf("Handle read commands failed: %v", err))
			return responses, err
		}

		responses[i] = res
	}

	return responses, err
}

// Modified by Intel to handle command requests and responses related to Intel open source gateway
func (d *Driver) handleReadCommandRequest(deviceClient MQTT.Client, req sdkModel.CommandRequest, topics []string) (*sdkModel.CommandValue, error) {
	var result = &sdkModel.CommandValue{}
	var err error

	var request commandModel.JsonRequest
	request.JsonRpc = jsonRpc
	request.Method = req.DeviceObject.Name
	// create a unique id to track every response
	request.Id = bson.NewObjectId().Hex()

	jsonData, err := json.Marshal(request)
	if err != nil {
		return result, err
	}

	for _, topic := range topics {
		deviceClient.Publish(topic, qos, retained, jsonData)
	}

	driver.Logger.Info(fmt.Sprintf("Publish command: %v", string(jsonData)))

	// fetch response from MQTT broker after publish command successful
	cmdResponse, ok := fetchCommandResponse(d.CommandResponses, request.Id)
	if !ok {
		err = fmt.Errorf("can not fetch command response: method=%v", request.Method)
		return result, err
	}

	var responseMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(cmdResponse), &responseMap); err != nil {
		return nil, err
	}

	// extract specific values from the response
	var reading string
	_, ok = responseMap["result"]
	if ok {
		reading = string(responseMap["result"])
	} else {
		_, ok = responseMap["error"]
		// error response is handled as ok (200 http code) as EdgeX command service returns only 500 error code with no message
		if ok {
			reading = string(responseMap["error"])
		} else {
			err = fmt.Errorf("incorrect command response from rsp-gateway: %v", cmdResponse)
			return nil, err
		}

	}
	if reading != "" {
		result, err = newResult(req.DeviceObject, req.RO, reading)
		if err != nil {
			return nil, err
		}
	}

	driver.Logger.Info(fmt.Sprintf("Get command finished: %v", result))
	return result, err
}

// Modified by Intel as Intel is not handling command put requests, so this function is just used for implementing ProtocolDriver Interface
func (d *Driver) HandleWriteCommands(addr *models.Addressable, reqs []sdkModel.CommandRequest, params []*sdkModel.CommandValue) error {
	var err error

	/*// create device client and open connection
	var brokerUrl = addr.Address
	var brokerPort = addr.Port
	var username = addr.User
	var password = addr.Password
	var mqttClientId = addr.Publisher

	uri := &url.URL{
		Scheme: strings.ToLower(addr.Protocol),
		Host:   fmt.Sprintf("%s:%d", brokerUrl, brokerPort),
		User:   url.UserPassword(username, password),
	}

	client, err := createClient(mqttClientId, uri, 30)
	if err != nil {
		return err
	}
	defer client.Disconnect(5000)

	for i, req := range reqs {
		err = d.handleWriteCommandRequest(client, req, addr.Topic, params[i])
		if err != nil {
			driver.Logger.Info(fmt.Sprintf("Handle write commands failed: %v", err))
			return err
		}
	}*/

	return err
}

// Modified by Intel as Intel is not handling command put requests, so this function is just used for implementing ProtocolDriver Interface
func (d *Driver) handleWriteCommandRequest(deviceClient MQTT.Client, req sdkModel.CommandRequest, topic string, param *sdkModel.CommandValue) error {
	/*var err error
	var qos = byte(0)
	var retained = false

	var method = "set"
	var cmdUuid = bson.NewObjectId().Hex()
	var cmd = req.DeviceObject.Name

	data := make(map[string]interface{})
	data["uuid"] = cmdUuid
	data["method"] = method
	data["cmd"] = cmd

	commandValue, err := newCommandValue(req.DeviceObject, param)
	if err != nil {
		return err
	}
	data[cmd] = commandValue

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	deviceClient.Publish(topic, qos, retained, jsonData)

	driver.Logger.Info(fmt.Sprintf("Publish command: %v", string(jsonData)))

	// wait and fetch response from CommandResponses map
	var cmdResponse string
	var ok bool
	for i := 0; i < 5; i++ {
		cmdResponse, ok = d.CommandResponses[cmdUuid]
		if ok {
			break
		} else {
			time.Sleep(time.Second * time.Duration(1))
		}
	}

	if !ok {
		err = fmt.Errorf("can not fetch command response: method=%v cmd=%v", method, cmd)
		return err
	}

	driver.Logger.Info(fmt.Sprintf("Put command finished: %v", cmdResponse))*/

	return nil
}

func (*Driver) Stop(force bool) error {
	panic("implement me")
}

// Create a MQTT client
func createClient(clientID string, uri *url.URL, keepAlive int) (MQTT.Client, error) {
	driver.Logger.Info(fmt.Sprintf("Create MQTT client and connection: uri=%v clientID=%v ", uri.String(), clientID))
	opts := MQTT.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s", uri.Scheme, uri.Host))
	opts.SetClientID(clientID)
	opts.SetUsername(uri.User.Username())
	password, _ := uri.User.Password()
	opts.SetPassword(password)
	opts.SetKeepAlive(time.Second * time.Duration(keepAlive))
	opts.SetConnectionLostHandler(func(client MQTT.Client, e error) {
		driver.Logger.Warn(fmt.Sprintf("Connection lost : %v", e))
		token := client.Connect()
		if token.Wait() && token.Error() != nil {
			driver.Logger.Warn(fmt.Sprintf("Reconnection failed : %v", token.Error()))
		} else {
			driver.Logger.Warn(fmt.Sprintf("Reconnection sucessful"))
		}
	})
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})

	client := MQTT.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return client, token.Error()
	}

	return client, nil
}

func newResult(deviceObject models.DeviceObject, ro models.ResourceOperation, reading interface{}) (*sdkModel.CommandValue, error) {
	var result = &sdkModel.CommandValue{}
	var err error
	var resTime = time.Now().UnixNano() / int64(time.Millisecond)

	switch deviceObject.Properties.Value.Type {
	case "Bool":
		result, err = sdkModel.NewBoolValue(&ro, resTime, reading.(bool))
	case "String":
		result = sdkModel.NewStringValue(&ro, resTime, reading.(string))
	case "Uint8":
		result, err = sdkModel.NewUint8Value(&ro, resTime, reading.(uint8))
	case "Uint16":
		result, err = sdkModel.NewUint16Value(&ro, resTime, reading.(uint16))
	case "Uint32":
		result, err = sdkModel.NewUint32Value(&ro, resTime, reading.(uint32))
	case "Uint64":
		result, err = sdkModel.NewUint64Value(&ro, resTime, reading.(uint64))
	case "Int8":
		result, err = sdkModel.NewInt8Value(&ro, resTime, reading.(int8))
	case "Int16":
		result, err = sdkModel.NewInt16Value(&ro, resTime, reading.(int16))
	case "Int32":
		result, err = sdkModel.NewInt32Value(&ro, resTime, reading.(int32))
	case "Int64":
		result, err = sdkModel.NewInt64Value(&ro, resTime, reading.(int64))
	case "Float32":
		result, err = sdkModel.NewFloat32Value(&ro, resTime, reading.(float32))
	case "Float64":
		result, err = sdkModel.NewFloat64Value(&ro, resTime, reading.(float64))
	default:
		err = fmt.Errorf("return result fail, none supported value type: %v", deviceObject.Properties.Value.Type)
	}

	return result, err
}

// Commented out as Intel is not handling command put requests
/*func newCommandValue(deviceObject models.DeviceObject, param *sdkModel.CommandValue) (interface{}, error) {
	var commandValue interface{}
	var err error
	switch deviceObject.Properties.Value.Type {
	case "Bool":
		commandValue, err = param.BoolValue()
	case "String":
		commandValue, err = param.StringValue()
	case "Uint8":
		commandValue, err = param.Uint8Value()
	case "Uint16":
		commandValue, err = param.Uint16Value()
	case "Uint32":
		commandValue, err = param.Uint32Value()
	case "Uint64":
		commandValue, err = param.Uint64Value()
	case "Int8":
		commandValue, err = param.Int8Value()
	case "Int16":
		commandValue, err = param.Int16Value()
	case "Int32":
		commandValue, err = param.Int32Value()
	case "Int64":
		commandValue, err = param.Int64Value()
	case "Float32":
		commandValue, err = param.Float32Value()
	case "Float64":
		commandValue, err = param.Float64Value()
	default:
		err = fmt.Errorf("return result fail, none supported value type: %v", deviceObject.Properties.Value.Type)
	}

	return commandValue, err
}*/

// fetchCommandResponse use to wait and fetch response from CommandResponses map
func fetchCommandResponse(commandResponses map[string]string, id string) (string, bool) {
	var cmdResponse string
	var ok bool
	for i := 0; i < 5; i++ {
		cmdResponse, ok = commandResponses[id]
		if ok {
			break
		} else {
			time.Sleep(time.Second * time.Duration(1))
		}
	}

	return cmdResponse, ok
}
