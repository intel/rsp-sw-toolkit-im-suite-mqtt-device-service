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

package jsonrpc

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	Version                      = "2.0"
	RSPControllerSubscribeMethod = "subscribe"
)

type Message interface{}

// Response represents a JsonRPC 2.0 Response
type Response struct {
	Version string          `json:"jsonrpc"`
	Id      string          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

type Parameters map[string]json.RawMessage

// Notification represents a JsonRPC 2.0 Notification
type Notification struct {
	Version string     `json:"jsonrpc"`
	Method  string     `json:"method"`
	Params  Parameters `json:"params,omitempty"`
}

// Request represents a JsonRPC 2.0 Request
type Request struct {
	Version string          `json:"jsonrpc"`
	Id      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type RSPCommandRequest struct {
	Request // embed
	Params DeviceIdParams `json:"params"`
}

type RSPControllerSubscribeRequest struct {
	Request // embed
	Params []string `json:"params"`
}

// DeviceIdParams holds the device id parameter used in command requests to RSP Controller
type DeviceIdParams struct {
	DeviceId string `json:"device_id"`
}

func NewRequest(method string) Request {
	return Request{
		Version: Version,
		Id:      uuid.New().String(),
		Method:  method,
	}
}

func NewRSPCommandRequest(method string, deviceId string) RSPCommandRequest {
	return RSPCommandRequest{
		Request: NewRequest(method),
		Params: DeviceIdParams{
			DeviceId: deviceId,
		},
	}
}

func NewRSPControllerSubscribeRequest(topics []string) RSPControllerSubscribeRequest {
	return RSPControllerSubscribeRequest{
		Request: NewRequest(RSPControllerSubscribeMethod),
		Params:  topics,
	}
}

// GetParam looks for the parameter 'key' and unmarshals it into `out`. If the
// key is not in the parameters or fails to unmarshal, it returns an error.
func (n Notification) GetParam(key string, out interface{}) error {
	if n.Params == nil || len(n.Params) == 0 {
		return errors.New("notification has no parameters")
	}
	paramVal, ok := n.Params[key]
	if !ok {
		return errors.Errorf("no such parameter %q", key)
	}
	return errors.Wrapf(json.Unmarshal(paramVal, out),
		"failed to unmarshal %q", key)
}

// SetParam inserts or replaces the parameter 'key' with a JSON marshal value.
//
// If the notification had no parameters, a new parameter map is created.
// If marshaling the value fails, this returns the marshaling error.
func (n *Notification) SetParam(key string, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "failed to marshal parameter value")
	}
	if n.Params == nil {
		n.Params = map[string]json.RawMessage{key: b}
	} else {
		n.Params[key] = b
	}
	return nil
}
