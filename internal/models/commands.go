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

package models

import (
	"encoding/json"
	"github.com/pkg/errors"
)

// Json response from the gateway
type JsonResponse struct {
	Version string      `json:"jsonrpc"`
	Id      string      `json:"id"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
}

// todo: can JsonResponse be an JSONRPC message too?
// JSONRPC represents a JSON RPC message.
//
// It's used for data messages from the Gateway and command requests to the Gateway.
type JSONRPC struct {
	Version string `json:"jsonrpc"`
	Id      string `json:"id"`
	Method  string `json:"method"`
	// Topic will be set by us and sent upstream, indicating the topic on which
	// the original JSON message came.
	Topic string `json:"topic,omitempty"` // TODO: this should probably be moved into Params to fit the spec
	// Params is rest of the message from which we'll extract the Gateway's ID.
	Params json.RawMessage `json:"params,omitempty"`
}

// EitherID is used to unmarshal the Gateway's ID, regardless of how it came
type EitherID struct {
	GatewayID *OptString `json:"gateway_id"`
	DeviceID  *OptString `json:"device_id"`
}

// OptString represents an optional string (and should be used as a pointer)
type OptString string

// IsPresent returns true if the OptString is neither nil nor empty
func (id *OptString) IsPresent() bool {
	return id != nil && *id != ""
}

func (jn *JSONRPC) GetID() (string, error) {
	if jn == nil || len(jn.Params) == 0 {
		return "", errors.New("JSON notification is nil or is missing parameters")
	}

	var ids EitherID
	if err := json.Unmarshal(jn.Params, &ids); err != nil {
		return "", errors.Wrap(err, "unable to unmarshal the gateway ID")
	}

	if ids.GatewayID.IsPresent() {
		return string(*(ids.GatewayID)), nil
	} else if ids.DeviceID.IsPresent() {
		return string(*(ids.DeviceID)), nil
	}
	return "", errors.New("neither gateway_id nor device_id found in message")
}
