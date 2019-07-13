// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2019 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"reflect"
	"strings"
	"testing"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
)

func TestCreateConnectionInfo(t *testing.T) {
	expected := &ConnectionInfo{
		Scheme:   "tcp",
		Host:     "0.0.0.0",
		Port:     "1883",
		User:     "admin",
		Password: "password",
		ClientId: "CommandPublisher",
		Topics:   []string{"CommandTopic1"},
	}
	protocols := map[string]models.ProtocolProperties{
		Protocol: {
			Scheme:   expected.Scheme,
			Host:     expected.Host,
			Port:     expected.Port,
			User:     expected.User,
			Password: expected.Password,
			ClientId: expected.ClientId,
			Topics:   strings.Join(expected.Topics, ","),
		},
	}

	connectionInfo, err := CreateConnectionInfo(protocols)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if !reflect.DeepEqual(connectionInfo, expected) {
		t.Fatalf("connectionInfo: %+v\nexpected: %+v", connectionInfo, expected)
	}
}

func TestLoadConfig_multipleTopics(t *testing.T) {
	expected := &ConnectionInfo{
		Scheme:   "tcp",
		Host:     "0.0.0.0",
		Port:     "1883",
		User:     "admin",
		Password: "password",
		ClientId: "CommandPublisher",
		Topics:   []string{"CommandTopic1", "CommandTopic2"},
	}
	protocols := map[string]models.ProtocolProperties{
		Protocol: {
			Scheme:   expected.Scheme,
			Host:     expected.Host,
			Port:     expected.Port,
			User:     expected.User,
			Password: expected.Password,
			ClientId: expected.ClientId,
			Topics:   strings.Join(expected.Topics, ","),
		},
	}

	connectionInfo, err := CreateConnectionInfo(protocols)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if !reflect.DeepEqual(connectionInfo, expected) {
		t.Fatalf("connectionInfo: %+v\nexpected: %+v", connectionInfo, expected)
	}
}

func TestCreateConnectionInfo_fail(t *testing.T) {
	protocols := map[string]models.ProtocolProperties{
		Protocol: {},
	}

	_, err := CreateConnectionInfo(protocols)
	if err == nil {
		t.Fatal("Unexpected test result; err should not be nil")
	}
}

func TestCreateDriverConfig(t *testing.T) {
	configs := map[string]string{
		DeviceName:     "test-device",
		IncomingScheme: "tcp", IncomingHost: "0.0.0.0", IncomingPort: "1883",
		IncomingUser: "admin", IncomingPassword: "public", IncomingQos: "0",
		IncomingKeepAlive: "3600", IncomingClientId: "IncomingDataSubscriber", IncomingTopics: "DataTopic",

		ResponseScheme: "tcp", ResponseHost: "0.0.0.0", ResponsePort: "1883",
		ResponseUser: "admin", ResponsePassword: "public", ResponseQos: "0",
		ResponseKeepAlive: "3600", ResponseClientId: "CommandResponseSubscriber", ResponseTopics: "ResponseTopic",
	}
	driverConfig, err := CreateDriverConfig(configs)
	if err != nil {
		t.Fatalf("Fail to load config, %v", err)
	}
	if driverConfig.DeviceName != configs[DeviceName] ||
		driverConfig.IncomingScheme != configs[IncomingScheme] || driverConfig.IncomingHost != configs[IncomingHost] ||
		driverConfig.IncomingPort != 1883 || driverConfig.IncomingUser != configs[IncomingUser] ||
		driverConfig.IncomingPassword != configs[IncomingPassword] || driverConfig.IncomingQos != 0 ||
		driverConfig.IncomingKeepAlive != 3600 || driverConfig.IncomingClientId != configs[IncomingClientId] ||
		driverConfig.IncomingTopics[0] != configs[IncomingTopics] ||
		driverConfig.ResponseScheme != configs[ResponseScheme] || driverConfig.ResponseHost != configs[ResponseHost] ||
		driverConfig.ResponsePort != 1883 || driverConfig.ResponseUser != configs[ResponseUser] ||
		driverConfig.ResponsePassword != configs[ResponsePassword] || driverConfig.ResponseQos != 0 ||
		driverConfig.ResponseKeepAlive != 3600 || driverConfig.ResponseClientId != configs[ResponseClientId] ||
		driverConfig.ResponseTopics[0] != configs[ResponseTopics] {

		t.Fatalf("Unexpected test result; driver config doesn't load correctly")
	}
}

func TestCreateDriverConfig_fail(t *testing.T) {
	configs := map[string]string{}
	_, err := CreateDriverConfig(configs)
	if err == nil {
		t.Fatal("Unexpected test result; err should not be nil")
	}
}
