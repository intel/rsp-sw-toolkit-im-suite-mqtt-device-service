// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2019 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package driver

const (
	Protocol = "mqtt"

	Scheme   = "Scheme"
	Host     = "Host"
	Port     = "Port"
	User     = "User"
	Password = "Password"
	ClientId = "ClientId"
	Topics   = "Topics"

	// Driver config
	DeviceName = "DeviceName"
	OnConnectPublishTopic   = "OnConnectPublishTopic"
	OnConnectPublishMessage = "OnConnectPublishMessage"

	// Incoming connection info
	IncomingScheme                = "IncomingScheme"
	IncomingHost                  = "IncomingHost"
	IncomingPort                  = "IncomingPort"
	IncomingUser                  = "IncomingUser"
	IncomingPassword              = "IncomingPassword"
	IncomingQos                   = "IncomingQos"
	IncomingKeepAlive             = "IncomingKeepAlive"
	IncomingClientId              = "IncomingClientId"
	IncomingTopics                = "IncomingTopics"
	IncomingTopicResourceMappings = "IncomingTopicResourceMappings"

	// Response connection info
	ResponseScheme    = "ResponseScheme"
	ResponseHost      = "ResponseHost"
	ResponsePort      = "ResponsePort"
	ResponseUser      = "ResponseUser"
	ResponsePassword  = "ResponsePassword"
	ResponseQos       = "ResponseQos"
	ResponseKeepAlive = "ResponseKeepAlive"
	ResponseClientId  = "ResponseClientId"
	ResponseTopics    = "ResponseTopics"
)
