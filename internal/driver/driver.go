// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2019 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package driver

import (
	"crypto/tls"
	"fmt"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/jsonrpc"
	"net/url"
	"os"
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

const (
	jsonRpcVersion          = "2.0"
	notRetained             = false
	rspDeviceProfile        = "RSP.Device.MQTT.Profile"
	disconnectQuiesceMillis = 5000
	connectFailureSleep     = 5 * time.Second
	subscribeFailureSleep   = 5 * time.Second
	// maximum amount of incoming data mqtt messages to handle at one time
	incomingDataMessageBuffer = 100
	// maximujm amount of incoming mqtt responses to handle at one time
	incomingResponseMessageBuffer = 10
)

var (
	once           sync.Once
	driverInstance *Driver
)

type Driver struct {
	Logger      logger.LoggingClient
	AsyncCh     chan<- *sdkModel.AsyncValues
	Config      *configuration
	Client      mqtt.Client
	DecoderRing *DecoderRing

	watchdogTimer  *time.Timer
	watchdogStatus *time.Ticker

	responseMap sync.Map // [string]chan *jsonrpc.Response

	// mqttDataChan is a channel to send incoming mqtt messages from any of the incoming topics
	mqttDataChan chan mqtt.Message
	// mqttResponseChan is a channel to only send incoming mqtt messages from the command response topic
	mqttResponseChan chan mqtt.Message

	started chan bool
	done    chan interface{}
}

// NewProtocolDriver returns the package-level driver instance.
func NewProtocolDriver() sdkModel.ProtocolDriver {
	once.Do(func() {
		driverInstance = new(Driver)
		driverInstance.mqttDataChan = make(chan mqtt.Message, incomingDataMessageBuffer)
		driverInstance.mqttResponseChan = make(chan mqtt.Message, incomingResponseMessageBuffer)
	})
	return driverInstance
}

// Initialize an MQTT driver.
//
// Once initialized, the driver listens on the configured MQTT topics. When a
// message comes in on a data topic, the driver formats the message appropriately
// and forwards it to EdgeX. When a message comes in on a command response topic,
// the driver checks for a corresponding command it sent previously. Assuming it
// finds one, it formats the response appropriately for EdgeX and forwards it on.
func (driver *Driver) Initialize(lc logger.LoggingClient, asyncCh chan<- *sdkModel.AsyncValues) error {
	driver.Logger = lc
	driver.AsyncCh = asyncCh

	// driver.responseChan = make(chan *jsonrpc.Response)
	driver.started = make(chan bool)
	driver.done = make(chan interface{})

	config, err := CreateDriverConfig(device.DriverConfigs())
	if err != nil {
		panic(errors.Wrap(err, "read MQTT driver configuration failed"))
	}
	driver.Config = config

	if err := driver.setupDecoderRing(); err != nil {
		return err
	}

	driver.setupWatchdog()

	go driver.Start()

	// wait for the initial connection before telling EdgeX we have been initialized
	<-driver.started

	return nil
}

func (driver *Driver) Start() {
	driver.createClient()
	go driver.connect()

	driver.runUntilCancelled()

	driver.Logger.Warn("Disconnecting client from MQTT broker")
	driver.Client.Disconnect(disconnectQuiesceMillis)
	driver.Logger.Warn("Exiting...")
	// Call this to make sure the process is actually stopped
	os.Exit(0)
}

// runUntilCancelled will block forever until done is signaled or a timer is fired causing a panic()
func (driver *Driver) runUntilCancelled() {
	for {
		select {
		case msg := <-driver.mqttResponseChan:
			driver.onCommandResponseReceived(msg)

		case msg := <-driver.mqttDataChan:
			driver.onIncomingDataReceived(msg)

		case <-driver.done:
			driver.Logger.Info("done signaled. stopping service.")
			return

		case <-driver.watchdogTimer.C:
			panic(errors.New("Timed out waiting for mqtt client to connect/re-connect. Exiting..."))
		}
	}
}

// Stop instructs the protocol-specific DS code to shutdown gracefully, or
// if the force parameter is 'true', immediately. The driver is responsible
// for closing any in-use channels, including the channel used to send async
// readings (if supported).
func (driver *Driver) Stop(force bool) error {
	close(driver.done)
	close(driver.AsyncCh)
	return nil
}

func (driver *Driver) setupWatchdog() {
	// setup watchdog timer but immediately stop it (we require a non-nil timer)
	driver.watchdogTimer = time.NewTimer(time.Duration(driver.Config.MaxReconnectWaitSeconds) * time.Second)
	driver.watchdogTimer.Stop()
}

func (driver *Driver) startWatchdog() {
	wait := time.Duration(driver.Config.MaxReconnectWaitSeconds) * time.Second
	driver.watchdogTimer.Reset(wait)

	driver.watchdogStatus = time.NewTicker(wait / 10)
	go driver.periodicWatchdogStatus(driver.watchdogStatus)
}

func (driver *Driver) stopWatchdog() {
	driver.watchdogTimer.Stop()
	if driver.watchdogStatus != nil {
		driver.watchdogStatus.Stop()
	}
}

func (driver *Driver) onMqttConnectionLost(client mqtt.Client, e error) {
	driver.Logger.Warn("MQTT connection lost", "cause", e)

	// IsConnected returns true if we are trying to reconnect still
	if client.IsConnected() {
		driver.Logger.Warn("Attempting to auto reconnect to MQTT broker...")
		driver.startWatchdog()
	} else {
		panic(errors.Wrap(e, "Connection to MQTT broker has been lost, and does not appear to be auto re-connecting"))
	}
}

// periodicWatchdogStatus will print a status message every so often to let the user know we are still waiting
func (driver *Driver) periodicWatchdogStatus(watchdogStatus *time.Ticker) {
	for range watchdogStatus.C {
		driver.Logger.Warn("still waiting for a connection to MQTT broker...")
	}
}

func (driver *Driver) onMqttConnect(client mqtt.Client) {
	driver.stopWatchdog()

	driver.Logger.Info("MQTT client connected/re-connected successfully")

	driver.subscribeAll()

	driver.configureControllerNotifications()

	driver.started <- true
}

// subscribe attempts to subscribe to a specific mqtt topic with a given qos and handler
// it will try forever until it succeeds or is cancelled. should be called in a goroutine
func (driver *Driver) subscribe(topic string, qos byte, handler mqtt.MessageHandler) {
	for {
		// keep trying to subscribe forever unless done is signaled
		select {
		case <-driver.done:
			driver.Logger.Info("done signaled. stopping subscription attempt", "topic", topic)
			// get out of the infinite loop
			return

		default:
			token := driver.Client.Subscribe(topic, qos, handler)
			if token.Wait() && token.Error() != nil {
				driver.Logger.Warn("subscription error", "cause", token.Error(), "topic", topic, "qos", qos)
			} else {
				driver.Logger.Info("subscription successful", "topic", topic, "qos", qos)
				// get out of the infinite loop
				return
			}
		}

		time.Sleep(subscribeFailureSleep)
	}
}

// subscribeAll will setup the subscriptions and handlers for all incoming topics and response topic
// this should be called in the onConnect handler as we need to setup the subscriptions
// every time we connect or reconnect to the mqtt broker
func (driver *Driver) subscribeAll() {
	// subscriptions are done in goroutines to allow them to retry over and over again
	// without interrupting the flow of the program

	// response subscription
	go driver.subscribe(driver.Config.ResponseTopic, driver.Config.ResponseQos, func(_ mqtt.Client, message mqtt.Message) {
		driver.mqttResponseChan <- message
	})

	// incoming subscriptions
	for _, topic := range driver.Config.IncomingTopics {
		go driver.subscribe(topic, driver.Config.IncomingQos, func(_ mqtt.Client, message mqtt.Message) {
			driver.mqttDataChan <- message
		})
	}
}

// createClient creates an MQTT client based on the driver config but does not connect it yet
func (driver *Driver) createClient() {
	opts := mqtt.NewClientOptions()

	uri := &url.URL{
		Scheme: strings.ToLower(driver.Config.MqttScheme),
		Host:   fmt.Sprintf("%s:%s", driver.Config.MqttHost, driver.Config.MqttPort),
	}

	// use `append()` because `opts.AddBroker()` does superfluous url parsing
	opts.Servers = append(opts.Servers, uri)

	opts.SetClientID(driver.Config.MqttClientId)
	opts.SetUsername(driver.Config.MqttUser)
	opts.SetPassword(driver.Config.MqttPassword)
	opts.SetKeepAlive(time.Second * time.Duration(driver.Config.MqttKeepAlive))
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: driver.Config.TlsInsecureSkipVerify})
	// just let the mqtt library handle reconnecting for us
	opts.SetAutoReconnect(true)

	opts.SetConnectionLostHandler(driver.onMqttConnectionLost)
	opts.SetOnConnectHandler(driver.onMqttConnect)

	driver.Logger.Info("Create MQTT client", "uri", uri.String(), "clientId", driver.Config.MqttClientId)

	driver.Client = mqtt.NewClient(opts)
}

// connect tries multiple times to establish the initial connection to the mqtt broker
// This is NOT called when re-connecting!
// MUST call createClient first!
func (driver *Driver) connect() {
	driver.startWatchdog()

	for {
		driver.Logger.Info("attempting to establish connection to mqtt broker...")
		token := driver.Client.Connect()
		if token.Wait() && token.Error() == nil {
			driver.Logger.Info("mqtt connection successful")
			return
		}

		driver.Logger.Error("unable to connect to mqtt broker", "cause", token.Error())
		driver.Logger.Info(fmt.Sprintf("attempting to connect to mqtt broker again in %v...", connectFailureSleep))
		time.Sleep(connectFailureSleep)
	}
}

// registerRSP registers a newly seen RSP sensor within EdgeX for the purposes of calling commands with parameters
func (driver *Driver) registerRSP(deviceId string) {
	// Registering sensor devices in Edgex
	_, err := sdk.RunningService().AddDevice(edgexModels.Device{
		Name:           deviceId,
		AdminState:     edgexModels.Unlocked,
		OperatingState: edgexModels.Enabled,
		Protocols: map[string]edgexModels.ProtocolProperties{
			"mqtt": {
				"Scheme": driver.Config.MqttScheme,
			},
		},
		Profile: edgexModels.DeviceProfile{
			Name: rspDeviceProfile,
		},
	})
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("Registering of sensor device %v failed: %v", deviceId, err))
	}
}
func (driver *Driver) setupDecoderRing() error {
	driver.DecoderRing = &DecoderRing{}
	for idx, f := range driver.Config.TagFormats {
		switch strings.ToLower(f) {
		case "bittag":
			if err := driver.DecoderRing.AddBitTagDecoder(
				driver.Config.TagURIAuthorityName,
				driver.Config.TagURIAuthorityDate,
				driver.Config.TagBitBoundary); err != nil {
				return err
			}
		case "sgtin":
			driver.DecoderRing.AddSGTINDecoder(driver.Config.SGTINStrictDecoding)
		default:
			return errors.Errorf("unknown tag format: %s", f)
		}
		driver.Logger.Info("added decoder %+v", driver.DecoderRing.Decoders[idx])
	}
	return nil
}

// configureControllerNotifications tells the RSP Controller which notifications it should send over MQTT
func (driver *Driver) configureControllerNotifications() {
	// tell the RSP Controller what notifications we would like to receive
	if driver.Config.RspControllerNotifications != nil && len(driver.Config.RspControllerNotifications) > 0 {
		if err := driver.publishCommand(jsonrpc.NewRSPControllerSubscribeRequest(driver.Config.RspControllerNotifications)); err != nil {
			driver.Logger.Warn("unable to subscribe to rsp controller notifications", "cause", err)
		}
	}
}
