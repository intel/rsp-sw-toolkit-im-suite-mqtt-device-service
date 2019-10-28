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

const (
	ControllerName          = "ControllerName"
	MaxWaitTimeForReq       = "MaxWaitTimeForReq"
	MaxReconnectWaitSeconds = "MaxReconnectWaitSeconds"
	TlsInsecureSkipVerify   = "TlsInsecureSkipVerify"

	// IncomingTopics provide reads to be sent to EdgeX.
	IncomingTopics = "IncomingTopics"
	CommandTopic   = "CommandTopic"
	ResponseTopic  = "ResponseTopic"
	SchemasDir     = "SchemasDir"

	// RspControllerNotifications a slice of the notification types we want to receive from the rsp controller
	RspControllerNotifications = "RspControllerNotifications"

	MqttScheme    = "MqttScheme"
	MqttHost      = "MqttHost"
	MqttPort      = "MqttPort"
	MqttUser      = "MqttUser"
	MqttPassword  = "MqttPassword"
	MqttKeepAlive = "MqttKeepAlive"
	MqttClientId  = "MqttClientId"

	CommandQos  = "CommandQos"
	ResponseQos = "ResponseQos"
	IncomingQos = "IncomingQos"

	TagFormats          = "TagFormats"
	TagBitBoundary      = "TagBitBoundary"
	TagURIAuthorityName = "TagURIAuthorityName"
	TagURIAuthorityDate = "TagURIAuthorityDate"
	SGTINStrictDecoding = "SGTINStrictDecoding"
)
