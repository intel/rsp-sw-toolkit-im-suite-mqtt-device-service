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
		IncomingScheme: "tcp", IncomingHost: "0.0.0.0", IncomingPort: "1883",
		IncomingUser: "admin", IncomingPassword: "public", IncomingQos: "0",
		IncomingKeepAlive: "3600", IncomingClientId: "IncomingDataSubscriber", IncomingTopics: "DataTopic",

		ResponseScheme: "tcp", ResponseHost: "0.0.0.0", ResponsePort: "1883",
		ResponseUser: "admin", ResponsePassword: "public", ResponseQos: "0",
		ResponseKeepAlive: "3600", ResponseClientId: "CommandResponseSubscriber", ResponseTopics: "ResponseTopic",
	}
	diverConfig, err := CreateDriverConfig(configs)
	if err != nil {
		t.Fatalf("Fail to load config, %v", err)
	}
	if diverConfig.IncomingScheme != configs[IncomingScheme] || diverConfig.IncomingHost != configs[IncomingHost] ||
		diverConfig.IncomingPort != 1883 || diverConfig.IncomingUser != configs[IncomingUser] ||
		diverConfig.IncomingPassword != configs[IncomingPassword] || diverConfig.IncomingQos != 0 ||
		diverConfig.IncomingKeepAlive != 3600 || diverConfig.IncomingClientId != configs[IncomingClientId] ||
		diverConfig.IncomingTopics[0] != configs[IncomingTopics] ||
		diverConfig.ResponseScheme != configs[ResponseScheme] || diverConfig.ResponseHost != configs[ResponseHost] ||
		diverConfig.ResponsePort != 1883 || diverConfig.ResponseUser != configs[ResponseUser] ||
		diverConfig.ResponsePassword != configs[ResponsePassword] || diverConfig.ResponseQos != 0 ||
		diverConfig.ResponseKeepAlive != 3600 || diverConfig.ResponseClientId != configs[ResponseClientId] ||
		diverConfig.ResponseTopics[0] != configs[ResponseTopics] {

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
