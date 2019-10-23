// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

// Intel modified the project name from device-mqtt-go to mqtt-device-service

package mqtt_device_service

// Global version for mqtt-device-service; overwritten during build process via
// -ldflags="-X 'importpath.Version=value'".
// See https://github.com/golang/go/wiki/GcToolchainTricks#including-build-information-in-the-executable
var Version = "1.0.0"
