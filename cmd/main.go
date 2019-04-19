// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/edgexfoundry/device-sdk-go/pkg/startup"
	"github.impcloud.net/RSP-Inventory-Suite/gateway-device-service"
	"github.impcloud.net/RSP-Inventory-Suite/gateway-device-service/internal/driver"
)

const (
	version     string = device_mqtt.Version
	serviceName string = "gateway-device-service"
)

func main() {
	sd := driver.NewProtocolDriver()
	startup.Bootstrap(serviceName, version, sd)
}
