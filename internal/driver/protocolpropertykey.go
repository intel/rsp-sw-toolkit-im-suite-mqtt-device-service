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

const (
	ControllerName          = "ControllerName"
	MaxWaitTimeForReq       = "MaxWaitTimeForReq"
	MaxReconnectWaitSeconds = "MaxReconnectWaitSeconds"
	TlsInsecureSkipVerify   = "TlsInsecureSkipVerify"

	// IncomingTopics provide reads to be sent to EdgeX.
	IncomingTopics = "IncomingTopics"
	CommandTopic   = "CommandTopic"
	ResponseTopic  = "ResponseTopic"

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
	TagProductField     = "TagProductField"
	TagURIAuthorityName = "TagURIAuthorityName"
	TagURIAuthorityDate = "TagURIAuthorityDate"
	SGTINStrictDecoding = "SGTINStrictDecoding"
)
