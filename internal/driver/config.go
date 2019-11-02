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
	"github.com/google/uuid"
	"math/rand"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRandomLength = 10
)

var (
	templateRegex = regexp.MustCompile("{{ *(random|uuid|epoch|millis|nanos)[_ ]*\\(?([0-9]+)?\\)? *}}")
	runes         = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

// configuration holds the values for the device configuration, including what
// MQTT broker to connect to for incoming data and command responses.
type configuration struct {
	// ControllerName is the device name used for sending data received on the IncomingTopics into Edgex
	ControllerName string
	// MaxWaitTimeForReq is the maximum wait time in seconds for a command request to time out
	MaxWaitTimeForReq int
	// MaxReconnectWaitSeconds is the maximum amount of time to wait for connection/re-connection to mqtt broker before we panic()
	MaxReconnectWaitSeconds int
	// TlsInsecureSkipVerify when set to "true", this will disable certificate checking of TLS connections to the MQTT broker
	TlsInsecureSkipVerify bool

	// IncomingTopics is a list of all topics containing data to be ingested
	IncomingTopics []string

	// CommandTopic is the topic to send commands on
	CommandTopic string
	// ResponseTopic is the topic to listen for responses on
	ResponseTopic string
	// RspControllerNotifications a slice of the notification types we want to receive from the rsp controller
	RspControllerNotifications []string
	// SchemasDir is the root directory of schema files. They're looked up as
	// <schemasDir>/<incoming | responses>/<method>_schema.json where "method"
	// is the jsonrpc method on the incoming data or the command request.
	SchemasDir string

	// Mqtt connection info
	MqttScheme   string
	MqttHost     string
	MqttPort     string
	MqttUser     string
	MqttPassword string
	// MqttKeepAlive is the keep alive in seconds
	MqttKeepAlive int
	MqttClientId  string

	// CommandQos is the MQTT Quality of Service 0, 1, or 2 for sending commands
	CommandQos byte
	// ResponseQos is the MQTT Quality of Service 0, 1, or 2 for subscribing to responses
	ResponseQos byte
	// IncomingQos is the MQTT Quality of Service 0, 1, or 2 for subscribing to incoming data
	IncomingQos byte

	// Tag decoding
	TagFormats          []string
	TagBitBoundary      []int
	TagURIAuthorityName string
	TagURIAuthorityDate string
	SGTINStrictDecoding bool
}

// CreateDriverConfig use to load driver config for incoming listener and response listener
func CreateDriverConfig(configMap map[string]string) (*configuration, error) {
	config := new(configuration)
	err := load(configMap, config)
	if err == nil {
		config.MqttClientId, err = replaceTemplateVars(config.MqttClientId)
	}
	return config, err
}

// load by reflect to check map key and then fetch the value
func load(configMap map[string]string, config *configuration) error {
	configValue := reflect.ValueOf(config).Elem()
	for i := 0; i < configValue.NumField(); i++ {
		typeField := configValue.Type().Field(i)
		valueField := configValue.Field(i)

		val, ok := configMap[typeField.Name]
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
		case reflect.Uint8:
			// uint8 is the same as byte
			byteVal, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			valueField.SetUint(uint64(byteVal))
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(val)
			if err != nil {
				return err
			}
			valueField.SetBool(boolVal)
		case reflect.String:
			valueField.SetString(val)
		case reflect.Slice:
			splitVals := strings.Split(val, ",")
			var slice reflect.Value
			switch typeField.Type.Elem().Kind() {
			case reflect.String:
				slice = reflect.ValueOf(splitVals)
			case reflect.Int:
				slice = reflect.MakeSlice(typeField.Type, len(splitVals), len(splitVals))
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

func generateRandomString(length int) string {
	randomStr := make([]rune, length)
	for i := range randomStr {
		randomStr[i] = runes[rand.Intn(len(runes))]
	}
	return string(randomStr)
}

func replaceTemplateVars(val string) (string, error) {
	var err error
	var replacement string
	var length int

	for {
		groups := templateRegex.FindStringSubmatch(val)
		if len(groups) < 2 {
			break
		}

		// determine optional length to truncate
		if len(groups) == 3 && groups[2] != "" {
			length, err = strconv.Atoi(groups[2])
			if err != nil {
				return "", err
			}
		}

		switch groups[1] {
		case "uuid":
			replacement = uuid.New().String()
		case "epoch":
			replacement = strconv.FormatInt(time.Now().Unix(), 10)
		case "millis":
			replacement = strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
		case "nanos":
			replacement = strconv.FormatInt(time.Now().UnixNano(), 10)
		case "random":
			// random does not have an inherent size like a uuid or similar,
			// so give it one here to allow it to be called without a parameter
			if length == 0 {
				length = defaultRandomLength
			}
			replacement = generateRandomString(length)
		default:
			return "", fmt.Errorf("invalid template variable specified: %s", groups[1])
		}

		if length > 0 && length < len(replacement) {
			replacement = replacement[:length]
		}

		val = strings.Replace(val, groups[0], replacement, 1)
	}

	if strings.Contains(val, "{{") || strings.Contains(val, "}}") {
		return "", fmt.Errorf("invalid template string: %s", val)
	}

	return val, nil
}
