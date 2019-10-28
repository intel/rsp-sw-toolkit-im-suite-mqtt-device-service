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
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestCreateDriverConfig(t *testing.T) {
	configs := map[string]string{
		ControllerName:             "rsp-controller",
		MaxWaitTimeForReq:          "10",
		MaxReconnectWaitSeconds:    "600",
		TlsInsecureSkipVerify:      "true",
		CommandTopic:               "rfid/controller/command",
		ResponseTopic:              "rfid/controller/response",
		IncomingTopics:             "rfid/controller/alerts,rfid/controller/heartbeat,rfid/controller/notification,rfid/rsp/data/+,rfid/rsp/rsp_status/+",
		SchemasDir:                 "schemas",
		RspControllerNotifications: "scheduler_run_state,sensor_config_notification,sensor_connection_state_notification",
		MqttScheme:                 "tcp",
		MqttHost:                   "mosquitto-server",
		MqttPort:                   "1883",
		MqttUser:                   "",
		MqttPassword:               "",
		MqttKeepAlive:              "120",
		IncomingQos:                "1",
		ResponseQos:                "1",
		CommandQos:                 "1",
		MqttClientId:               "MqttDeviceService",
		TagFormats:                 "sgtin,bittag",
		TagBitBoundary:             "8,44,44",
		TagURIAuthorityName:        "example.com",
		TagURIAuthorityDate:        "2019-01-31",
		SGTINStrictDecoding:        "true",
	}

	cfg, err := CreateDriverConfig(configs)
	if err != nil {
		t.Fatalf("Fail to load config, %v", err)
	}

	if cfg.ControllerName != configs[ControllerName] ||
		cfg.MaxWaitTimeForReq != convertInt(configs[MaxWaitTimeForReq]) ||
		cfg.MaxReconnectWaitSeconds != convertInt(configs[MaxReconnectWaitSeconds]) ||
		cfg.TlsInsecureSkipVerify != convertBool(configs[TlsInsecureSkipVerify]) ||
		convertSlice(cfg.IncomingTopics) != configs[IncomingTopics] ||
		cfg.CommandTopic != configs[CommandTopic] ||
		cfg.ResponseTopic != configs[ResponseTopic] ||
		convertSlice(cfg.RspControllerNotifications) != configs[RspControllerNotifications] ||
		cfg.SchemasDir != configs[SchemasDir] ||
		cfg.MqttScheme != configs[MqttScheme] ||
		cfg.MqttHost != configs[MqttHost] ||
		cfg.MqttPort != configs[MqttPort] ||
		cfg.MqttUser != configs[MqttUser] ||
		cfg.MqttPassword != configs[MqttPassword] ||
		cfg.MqttKeepAlive != convertInt(configs[MqttKeepAlive]) ||
		cfg.MqttClientId != configs[MqttClientId] ||
		cfg.CommandQos != convertByte(configs[CommandQos]) ||
		cfg.ResponseQos != convertByte(configs[ResponseQos]) ||
		cfg.IncomingQos != convertByte(configs[IncomingQos]) {

		t.Fatalf("Driver config didn't load correctly")
	}
}

func convertBool(str string) bool {
	val, err := strconv.ParseBool(str)
	if err != nil {
		panic(fmt.Sprintf("cannot convert %s to bool", str))
	}
	return val
}

func convertInt(str string) int {
	val, err := strconv.Atoi(str)
	if err != nil {
		panic(fmt.Sprintf("cannot convert %s to int", str))
	}
	return val
}

func convertByte(str string) byte {
	val, err := strconv.ParseUint(str, 10, 8)
	if err != nil {
		panic(fmt.Sprintf("cannot convert %s to int", str))
	}
	return byte(val)
}

func convertSlice(strSlice []string) string {
	return strings.Join(strSlice, ",")
}

func TestCreateDriverConfig_fail(t *testing.T) {
	configs := map[string]string{}
	_, err := CreateDriverConfig(configs)
	if err == nil {
		t.Fatal("Unexpected test result; err should not be nil")
	}
}
