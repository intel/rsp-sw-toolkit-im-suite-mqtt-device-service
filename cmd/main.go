/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

// Intel modified the project name from device-mqtt-go to mqtt-device-service

package main

import (
	"github.com/edgexfoundry/device-sdk-go/pkg/startup"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/driver"
)

const (
	serviceName string = "mqtt-device-service"
)

func main() {
	mqttDriver := driver.NewProtocolDriver()
	startup.Bootstrap(serviceName, Version, mqttDriver)
}
