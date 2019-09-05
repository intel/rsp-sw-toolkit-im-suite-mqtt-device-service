// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2019 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

/*
 * INTEL CONFIDENTIAL
 * Copyright (2019) Intel Corporation.
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

		t.Fatalf("Unexpected test result; driver config doesn't load correctly")
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
