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
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/edgexfoundry/device-sdk-go"
	sdk "github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	edgexModels "github.com/edgexfoundry/go-mod-core-contracts/models"
)

var once sync.Once
var instance *Driver

const (
	jsonRpcVersion   = "2.0"
	retained         = false
	rspDeviceProfile = "RSP.Device.MQTT.Profile"
)

type Driver struct {
	Logger           logger.LoggingClient
	AsyncCh          chan<- *sdkModel.AsyncValues
	CommandResponses sync.Map
	Config           *configuration
	Client           mqtt.Client

	done chan interface{}
}

// NewProtocolDriver returns the package-level driver instance.
func NewProtocolDriver() sdkModel.ProtocolDriver {
	once.Do(func() {
		instance = new(Driver)
	})
	return instance
}

// Initialize an MQTT d.
//
// Once initialized, the driver listens on the configured MQTT topics. When a
// message comes in on a data topic, the driver formats the message appropriately
// and forwards it to EdgeX. When a message comes in on a command response topic,
// the driver checks for a corresponding command it sent previously. Assuming it
// finds one, it formats the response appropriately for EdgeX and forwards it on.
func (d *Driver) Initialize(lc logger.LoggingClient, asyncCh chan<- *sdkModel.AsyncValues) error {
	d.Logger = lc
	d.AsyncCh = asyncCh
	d.done = make(chan interface{})

	config, err := CreateDriverConfig(device.DriverConfigs())
	if err != nil {
		panic(errors.Wrap(err, "read MQTT driver configuration failed"))
	}
	d.Config = config

	go d.Run()

	return nil
}

func (d *Driver) Run() {
	d.createClient()
	d.connect()
	d.Logger.Info("Mqtt client connected. Listening for data.")

	defer d.Client.Disconnect(5000)

	// Block forever until done is signaled
	<-d.done

	d.Logger.Info("Stopping mqtt client connection")
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

func (d *Driver) onMqttConnectionLost(client mqtt.Client, e error) {
	d.Logger.Warn("Connection lost", "cause", e)
	if client.IsConnected() {
		// todo: we can keep track of a timer/watchdog that will call panic() if it takes too long to re-connect
		// todo: 	the timer will be reset inside of the onConnect callback
		d.Logger.Warn("Attempting to auto reconnect to MQTT broker...")
	} else {
		panic(errors.Wrap(e, "Connection to MQTT broker has been lost, and does not appear to be auto reconnecting"))
	}
}

func (d *Driver) onMqttConnect(client mqtt.Client) {
	d.Logger.Info("mqtt incoming listener client connected")

	topic := d.Config.OnConnectPublishTopic
	if topic != "" {
		msg := replaceMessagePlaceholders(d.Config.OnConnectPublishMessage)
		d.Logger.Debug("publish onconnect", "topic", topic, "message", msg)
		client.Publish(topic, d.Config.CommandQos, retained, msg)
	}

	d.subscribeAll()
}

func (d *Driver) subscribe(topic string, qos byte, callback mqtt.MessageHandler) {
	// todo: should this have a max amount of retries??
	// todo: should this panic() after the max retries?
	for {
		// keep trying to subscribe forever unless done is signaled
		select {
		case <-d.done:
			d.Logger.Info("done signaled. stopping subscription attempt", "topic", topic)
			return

		default:
			token := d.Client.Subscribe(topic, qos, callback)
			if token.Wait() && token.Error() != nil {
				d.Logger.Warn("subscription error", "cause", token.Error(), "topic", topic, "qos", qos)
			} else {
				d.Logger.Info("subscription successful", "topic", topic, "qos", qos)
				return
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func (d *Driver) subscribeAll() {
	// response subscription
	go d.subscribe(d.Config.ResponseTopic, d.Config.ResponseQos, d.onCommandResponseReceived)

	// incoming subscriptions
	for _, topic := range d.Config.IncomingTopics {
		go d.subscribe(topic, d.Config.IncomingQos, d.onIncomingDataReceived)
	}
}

// Create an MQTT client
func (d *Driver) createClient() {
	opts := mqtt.NewClientOptions()
	clientId := replaceMessagePlaceholders(d.Config.MqttClientId)

	d.Logger.Info("create client")

	uri := &url.URL{
		Scheme: strings.ToLower(d.Config.MqttScheme),
		Host:   fmt.Sprintf("%s:%s", d.Config.MqttHost, d.Config.MqttPort),
	}

	// use `append()` because `opts.AddBroker()` does superfluous url parsing
	opts.Servers = append(opts.Servers, uri)

	opts.SetClientID(clientId)
	opts.SetUsername(d.Config.MqttUser)
	opts.SetPassword(d.Config.MqttPassword)
	opts.SetKeepAlive(time.Second * time.Duration(d.Config.MqttKeepAlive))
	opts.SetAutoReconnect(true)

	opts.SetConnectionLostHandler(d.onMqttConnectionLost)
	opts.SetOnConnectHandler(d.onMqttConnect)

	// todo: this should probably *not* be hardcoded
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})

	d.Logger.Info("Create MQTT client and connection", "uri", uri.String(), "clientId", clientId)

	d.Client = mqtt.NewClient(opts)
}

func (d *Driver) connect() {
	retries := d.Config.InitialConnectionTries
	for {
		token := d.Client.Connect()
		if token.Wait() && token.Error() != nil {
			d.Logger.Error("unable to connect to mqtt broker", "cause", token.Error())
			retries -= 1
		} else {
			d.Logger.Info("mqtt connection successful")
			return
		}

		if retries == 0 {
			panic(errors.Wrap(token.Error(), fmt.Sprintf("unable to connect to mqtt broker after %d tries!", d.Config.InitialConnectionTries)))
		}
		d.Logger.Info(fmt.Sprintf("attempting to connect to mqtt broker again in 5 seconds... %d retries left", retries))
		time.Sleep(5 * time.Second)
	}
}

func (d *Driver) registerRSP(deviceId string) {
	// Registering sensor devices in Edgex
	_, err := sdk.RunningService().AddDevice(edgexModels.Device{
		Name:           deviceId,
		AdminState:     edgexModels.Unlocked,
		OperatingState: edgexModels.Enabled,
		Protocols: map[string]edgexModels.ProtocolProperties{
			"mqtt": {
				"Scheme": d.Config.MqttScheme,
			},
		},
		Profile: edgexModels.DeviceProfile{
			Name: rspDeviceProfile,
		},
	})
	if err != nil {
		d.Logger.Error(fmt.Sprintf("Registering of sensor device %v failed: %v", deviceId, err))
	}
}

func replaceMessagePlaceholders(message string) string {
	res := message
	uuid := uuid.New().String()
	tokens := strings.Split(uuid, "-")
	shortUuid := tokens[len(tokens)-1]
	// replace {{uuid}} placeholder with generated id
	res = strings.Replace(res, "{{uuid}}", uuid, -1)
	res = strings.Replace(res, "{{short_uuid}}", shortUuid, -1)
	return res
}
