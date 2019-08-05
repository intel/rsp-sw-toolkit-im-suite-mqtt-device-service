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
	jsonRpcVersion    = "2.0"
	qos               = byte(1)
	retained          = false
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
		panic(errors.Wrap(err, "read MQTT driver configuration failed"))
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
		res, err := d.handleReadCommandRequest(deviceName, client, req, connectionInfo.Topics)
		if err != nil {
			driver.Logger.Warn("Handle read commands failed", "cause", err)
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
func (d *Driver) handleReadCommandRequest(deviceName string, deviceClient MQTT.Client, req sdkModel.CommandRequest, topics []string) (*sdkModel.CommandValue, error) {
	var err error
	request := commandModel.JsonRequest{
		Version: jsonRpcVersion,
		Method:  req.DeviceResourceName,
		Id:      uuid.New().String(),
	}

	// Sensor devices start with "RSP", this will not be needed in near future as Edgex is going to support GET requests with query parameters
	// If the device is sensor add the device_id as params to the command request
	if strings.HasPrefix(deviceName, "RSP") {
		deviceIdParam := commandModel.DeviceIdParam{deviceName}
		request.Params, err = json.Marshal(deviceIdParam)
		if err != nil {
			err = fmt.Errorf("marshalling of command parameters failed: error=%v", err)
			return nil, err
		}
	}

	// marshal request to jsonrpc format
	jsonRpcRequest, err := json.Marshal(request)
	if err != nil {
		err = fmt.Errorf("marshalling of command request failed: error=%v", err)
		return nil, err
	}

	//Publish the command request
	for _, topic := range topics {
		deviceClient.Publish(topic, qos, retained, jsonRpcRequest)
	}
	driver.Logger.Info("Publish command", "command", string(jsonRpcRequest))

	cmdResponse, ok := d.fetchCommandResponse(request.Id)
	if !ok {
		err = fmt.Errorf("no command response or getting response delayed for method=%v", request.Method)
		return nil, err
	}

	var responseMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(cmdResponse), &responseMap); err != nil {
		err = fmt.Errorf("unmarshalling of command response failed: error=%v", err)
		return nil, err
	}

	// Parse response to extract result or error field from the jsonrpc response
	var reading string
	_, ok = responseMap["result"]
	if ok {
		reading = string(responseMap["result"])
	} else {
		_, ok = responseMap["error"]
		if ok {
			reading = string(responseMap["error"])
		} else {
			err = fmt.Errorf("invalid command response: %v", cmdResponse)
			return nil, err
		}
	}

	origin := time.Now().UnixNano() / int64(time.Millisecond)
	value := sdkModel.NewStringValue(req.DeviceResourceName, origin, reading)

	driver.Logger.Info("Get command finished", "response", cmdResponse)

	return value, err
}

// HandleWriteCommands ignores all requests; write commands (PUT requests) are not currently supported.
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
	driver.Logger.Info("Create MQTT client and connection", "uri", uri.String(), "clientId", clientID)
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
		driver.Logger.Warn("Connection lost", "cause", e)
		token := client.Connect()
		if token.Wait() && token.Error() != nil {
			// todo: the main incomingListener client should probably panic() if it can't re-connect after X tries
			driver.Logger.Warn("Reconnection failed", "cause", token.Error())
			// PANIC!!!
			panic(errors.Wrap(token.Error(), "unable to re-connect to mqtt broker"))
		} else {
			driver.Logger.Warn("Reconnection successful")
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

// fetchCommandResponse use to wait and fetch response from CommandResponses map
func (d *Driver) fetchCommandResponse(cmdUuid string) (string, bool) {
	var cmdResponse interface{}
	var ok bool
	for i := 0; i < d.Config.MaxWaitTimeForReq; i++ {
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
