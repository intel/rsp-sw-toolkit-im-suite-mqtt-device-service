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
)

// Json response from the gateway
type JsonResponse struct {
	Version string      `json:"jsonrpc"`
	Id      string      `json:"id"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
}

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