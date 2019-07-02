// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2019 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

/*
 * INTEL CONFIDENTIAL
 * Copyright (2017) Intel Corporation.
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
	"reflect"
	"strconv"
	"strings"

	"github.com/edgexfoundry/go-mod-core-contracts/models"
)

type ConnectionInfo struct {
	Scheme   string
	Host     string
	Port     string
	User     string
	Password string
	ClientId string
	Topics   []string
}

type configuration struct {
	IncomingScheme    string
	IncomingHost      string
	IncomingPort      int
	IncomingUser      string
	IncomingPassword  string
	IncomingQos       int
	IncomingKeepAlive int
	IncomingClientId  string
	IncomingTopics    []string

	ResponseScheme    string
	ResponseHost      string
	ResponsePort      int
	ResponseUser      string
	ResponsePassword  string
	ResponseQos       int
	ResponseKeepAlive int
	ResponseClientId  string
	ResponseTopics    []string
}

// CreateDriverConfig use to load driver config for incoming listener and response listener
func CreateDriverConfig(configMap map[string]string) (*configuration, error) {
	config := new(configuration)
	err := load(configMap, config)
	if err != nil {
		return config, err
	}
	return config, nil
}

// CreateConnectionInfo use to load MQTT connectionInfo for read and write command
func CreateConnectionInfo(protocols map[string]models.ProtocolProperties) (*ConnectionInfo, error) {
	info := new(ConnectionInfo)
	protocol, ok := protocols[Protocol]
	if !ok {
		return info, fmt.Errorf("config is missing %s protocol", Protocol)
	}

	err := load(protocol, info)
	if err != nil {
		return info, err
	}
	return info, nil
}

// load by reflect to check map key and then fetch the value
func load(config map[string]string, des interface{}) error {
	val := reflect.ValueOf(des).Elem()
	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		valueField := val.Field(i)

		val, ok := config[typeField.Name]
		if !ok {
			return fmt.Errorf("config is missing property '%s'", typeField.Name)
		}
		if !valueField.CanSet() {
			return fmt.Errorf("cannot set field '%s'", typeField.Name)
		}

		switch valueField.Kind() {
		case reflect.Int:
			intVal, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			valueField.SetInt(int64(intVal))
		case reflect.String:
			valueField.SetString(val)
		case reflect.Slice:
			splitVals := strings.Split(val, ",")
			var slice reflect.Value
			switch typeField.Type.Elem().Kind() {
			case reflect.String:
				slice = reflect.ValueOf(splitVals)
			case reflect.Int:
				slice = reflect.MakeSlice(valueField.Elem().Type(), len(splitVals), len(splitVals))
				for idx, toConvert := range splitVals {
					intVal, err := strconv.Atoi(toConvert)
					if err != nil {
						return err
					}
					slice.Index(idx).SetInt(int64(intVal))
				}
			}
			slice = reflect.AppendSlice(valueField, slice)
			valueField.Set(slice)
		default:
			return fmt.Errorf("config uses unsupported property kind "+
				"%v for field %v", valueField.Kind(), typeField.Name)
		}
	}
	return nil
}
