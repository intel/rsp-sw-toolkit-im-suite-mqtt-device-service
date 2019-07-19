// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2019 IOTech Ltd
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
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"net/url"
	"strings"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	commandModel "github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/models"
)

var once sync.Once
var driver *Driver

const (
	jsonRpcVersion = "2.0"
	qos            = byte(1)
	retained       = false
)

type Driver struct {
	Logger           logger.LoggingClient
	AsyncCh          chan<- *sdkModel.AsyncValues
	CommandResponses sync.Map
	Config           *configuration

	done chan interface{}
}

// NewProtocolDriver returns the package-level driver instance.
func NewProtocolDriver() sdkModel.ProtocolDriver {
	once.Do(func() {
		driver = new(Driver)
	})
	return driver
}

// GetServiceName gets the name of the running service, as it's known to EdgeX,
// or returns an empty string if there is no running service.
func GetServiceName() string {
	srv := device.RunningService()

	if srv == nil {
		return ""
	}
	return srv.Name()
}

// Initialize an MQTT driver.
//
// Once initialized, the driver listens on the configured MQTT topics. When a
// message comes in on a data topic, the driver formats the message appropriately
// and forwards it to EdgeX. When a message comes in on a command response topic,
// the driver checks for a corresponding command it sent previously. Assuming it
// finds one, it formats the response appropriately for EdgeX and forwards it on.
func (d *Driver) Initialize(lc logger.LoggingClient, asyncCh chan<- *sdkModel.AsyncValues) error {
	d.Logger = lc
	d.AsyncCh = asyncCh

	config, err := CreateDriverConfig(device.DriverConfigs())
	if err != nil {
		panic(fmt.Errorf("read MQTT driver configuration failed: %v", err))
	}
	d.Config = config

	done := make(chan interface{})
	d.done = done
	go func() {
		err := startCommandResponseListening(done)
		if err != nil {
			panic(errors.Wrap(err, "start command response Listener failed, please check MQTT broker settings are correct"))
		}
	}()

	go func() {
		err := startIncomingListening(done)
		if err != nil {
			panic(errors.Wrap(err, "start incoming data Listener failed, please check MQTT broker settings are correct"))
		}
	}()

	return nil
}

// HandleReadCommands handles CommandRequests to read data via MQTT.
//
// It satisfies them by creating a new MQTT client with the protocol, sending the
// requests as JSON RPC messages on all configured topics, then waiting for a
// response on any of the response topics; once a response comes in, it returns
// that result.
func (d *Driver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModel.CommandRequest) ([]*sdkModel.CommandValue, error) {
	var responses = make([]*sdkModel.CommandValue, len(reqs))
	var err error

	// create device client and open connection
	connectionInfo, err := CreateConnectionInfo(protocols)
	if err != nil {
		return responses, err
	}

	uri := &url.URL{
		Scheme: strings.ToLower(connectionInfo.Scheme),
		Host:   fmt.Sprintf("%s:%s", connectionInfo.Host, connectionInfo.Port),
		User:   url.UserPassword(connectionInfo.User, connectionInfo.Password),
	}

	client, err := createClient(connectionInfo.ClientId, uri, 30, nil)
	if err != nil {
		return responses, err
	}
	defer client.Disconnect(5000)

	for i, req := range reqs {
		res, err := d.handleReadCommandRequest(client, req, connectionInfo.Topics)
		if err != nil {
			driver.Logger.Info(fmt.Sprintf("Handle read commands failed: %v", err))
			return responses, err
		}

		responses[i] = res
	}

	return responses, err
}

// handleReadCommandRequest takes care of the JSON RPC command/response portion
// of the HandleReadCommands.
//
// The command request is published on all of the incoming connection info topics.
func (d *Driver) handleReadCommandRequest(deviceClient MQTT.Client, req sdkModel.CommandRequest, topics []string) (*sdkModel.CommandValue, error) {
	var result = &sdkModel.CommandValue{}
	var err error

	request := commandModel.JsonRequest{
		Version: jsonRpcVersion,
		Method:  req.DeviceResourceName,
		Id:      uuid.New().String(),
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		err = fmt.Errorf("marshalling of command request failed: error=%v", err)
		return result, err
	}

	for _, topic := range topics {
		deviceClient.Publish(topic, qos, retained, jsonData)
	}

	driver.Logger.Info(fmt.Sprintf("Publish command: %v", string(jsonData)))

	// fetch response from MQTT broker after publish command successful
	cmdResponse, ok := d.fetchCommandResponse(request.Id)
	if !ok {
		err = fmt.Errorf("can not fetch command response: method=%v", request.Method)
		return result, err
	}
	driver.Logger.Info(fmt.Sprintf("Command response: %v", cmdResponse))

	var responseMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(cmdResponse), &responseMap); err != nil {
		err = fmt.Errorf("unmarshalling of command response failed: error=%v", err)
		return nil, err
	}

	// Parse response to extract result or error field
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
			err = fmt.Errorf("invalid command response: %v", cmdResponse)
			return nil, err
		}
	}

	result, err = newResult(req, reading)
	if err != nil {
		return nil, err
	} else {
		driver.Logger.Info(fmt.Sprintf("Get command finished: %v", result))
	}

	return result, err
}

// HandleWriteCommands ignores all requests; write commands are not currently supported.
func (d *Driver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModel.CommandRequest, params []*sdkModel.CommandValue) error {
	return nil
}

// Stop instructs the protocol-specific DS code to shutdown gracefully, or
// if the force parameter is 'true', immediately. The driver is responsible
// for closing any in-use channels, including the channel used to send async
// readings (if supported).
func (d *Driver) Stop(force bool) error {
	close(d.done)
	close(d.AsyncCh)
	return nil
}

// Create a MQTT client
func createClient(clientID string, uri *url.URL, keepAlive int, onConn MQTT.OnConnectHandler) (MQTT.Client, error) {
	driver.Logger.Info(fmt.Sprintf("Create MQTT client and connection: uri=%v clientID=%v ", uri.String(), clientID))
	opts := MQTT.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s", uri.Scheme, uri.Host))
	opts.SetClientID(clientID)
	opts.SetUsername(uri.User.Username())
	password, _ := uri.User.Password()
	opts.SetPassword(password)
	opts.SetKeepAlive(time.Second * time.Duration(keepAlive))

	if onConn != nil {
		opts.SetOnConnectHandler(onConn)
	}

	opts.SetConnectionLostHandler(func(client MQTT.Client, e error) {
		driver.Logger.Warn(fmt.Sprintf("Connection lost : %v", e))
		token := client.Connect()
		if token.Wait() && token.Error() != nil {
			// todo: the main incomingListener client should probably panic() if it can't re-connect after X tries
			driver.Logger.Warn(fmt.Sprintf("Reconnection failed : %v", token.Error()))
		} else {
			driver.Logger.Warn(fmt.Sprintf("Reconnection sucessful"))
		}
	})

	// todo: this should probably *not* be hardcoded
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})

	// todo: this method could probably keep a cache of clients, hashed by their
	//   incoming uri + clientID; if it's not a bottleneck, then it's probably not worth it.
	client := MQTT.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return client, token.Error()
	}

	return client, nil
}

// newResult constructs a new CommandValue from the original incoming request and
// the returned reading that resulted from that request.
func newResult(req sdkModel.CommandRequest, reading interface{}) (*sdkModel.CommandValue, error) {
	var result = &sdkModel.CommandValue{}
	var err error
	var resTime = time.Now().UnixNano() / int64(time.Millisecond)
	castError := "fail to parse %v reading, %v"

	if !checkValueInRange(req.Type, reading) {
		err = fmt.Errorf("parse reading fail. Reading %v is out of the value type(%v)'s range", reading, req.Type)
		driver.Logger.Error(err.Error())
		return result, err
	}

	switch req.Type {
	case sdkModel.Bool:
		val, err := cast.ToBoolE(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewBoolValue(req.DeviceResourceName, resTime, val)
	case sdkModel.String:
		val, err := cast.ToStringE(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result = sdkModel.NewStringValue(req.DeviceResourceName, resTime, val)
	case sdkModel.Uint8:
		val, err := cast.ToUint8E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewUint8Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Uint16:
		val, err := cast.ToUint16E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewUint16Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Uint32:
		val, err := cast.ToUint32E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewUint32Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Uint64:
		val, err := cast.ToUint64E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewUint64Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Int8:
		val, err := cast.ToInt8E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewInt8Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Int16:
		val, err := cast.ToInt16E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewInt16Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Int32:
		val, err := cast.ToInt32E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewInt32Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Int64:
		val, err := cast.ToInt64E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewInt64Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Float32:
		val, err := cast.ToFloat32E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewFloat32Value(req.DeviceResourceName, resTime, val)
	case sdkModel.Float64:
		val, err := cast.ToFloat64E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, req.DeviceResourceName, err)
		}
		result, err = sdkModel.NewFloat64Value(req.DeviceResourceName, resTime, val)
	default:
		err = fmt.Errorf("return result fail, none supported value type: %v", req.Type)
	}

	return result, err
}

func newCommandValue(valueType sdkModel.ValueType, param *sdkModel.CommandValue) (interface{}, error) {
	var commandValue interface{}
	var err error
	switch valueType {
	case sdkModel.Bool:
		commandValue, err = param.BoolValue()
	case sdkModel.String:
		commandValue, err = param.StringValue()
	case sdkModel.Uint8:
		commandValue, err = param.Uint8Value()
	case sdkModel.Uint16:
		commandValue, err = param.Uint16Value()
	case sdkModel.Uint32:
		commandValue, err = param.Uint32Value()
	case sdkModel.Uint64:
		commandValue, err = param.Uint64Value()
	case sdkModel.Int8:
		commandValue, err = param.Int8Value()
	case sdkModel.Int16:
		commandValue, err = param.Int16Value()
	case sdkModel.Int32:
		commandValue, err = param.Int32Value()
	case sdkModel.Int64:
		commandValue, err = param.Int64Value()
	case sdkModel.Float32:
		commandValue, err = param.Float32Value()
	case sdkModel.Float64:
		commandValue, err = param.Float64Value()
	default:
		err = fmt.Errorf("fail to convert param, none supported value type: %v", valueType)
	}

	return commandValue, err
}

// fetchCommandResponse use to wait and fetch response from CommandResponses map
func (d *Driver) fetchCommandResponse(cmdUuid string) (string, bool) {
	var cmdResponse interface{}
	var ok bool
	for i := 0; i < 5; i++ {
		cmdResponse, ok = d.CommandResponses.Load(cmdUuid)
		if ok {
			d.CommandResponses.Delete(cmdUuid)
			break
		} else {
			time.Sleep(time.Second * time.Duration(1))
		}
	}

	return fmt.Sprintf("%v", cmdResponse), ok
}
