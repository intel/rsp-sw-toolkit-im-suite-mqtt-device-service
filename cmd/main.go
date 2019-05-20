// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/edgexfoundry/device-sdk-go/pkg/startup"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/driver"
)

const (
	version     string = device_mqtt.Version
	serviceName string = "mqtt-device-service"
)

func main() {
	sd := driver.NewProtocolDriver()
	startup.Bootstrap(serviceName, version, sd)
}
