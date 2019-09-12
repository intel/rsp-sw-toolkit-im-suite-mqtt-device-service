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

import (
	"encoding/json"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.impcloud.net/RSP-Inventory-Suite/expect"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/jsonrpc"
	"testing"
)

func init() {
	driverInstance = new(Driver)
	driverInstance.Logger = logger.NewClient("test", false, "", "DEBUG")

	driverInstance.Config = &configuration{
		TagFormats: []string{"sgtin"},
	}
	err := driverInstance.setupDecoderRing()
	if err != nil {
		panic(err)
	}
}

func TestProcessTagData(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()

	n := jsonrpc.Notification{
		Version: jsonrpc.Version,
		Method:  inventoryEvent,
		Params: map[string]json.RawMessage{
			tagDataKey: []byte(`"30143639F84191AD22901607"`),
		},
	}

	modified := w.ShouldHaveResult(driverInstance.processResource(n)).([]byte)
	w.ShouldNotBeNil(modified)

	var n2 jsonrpc.Notification
	w.ShouldSucceed(json.Unmarshal(modified, &n2))
	w.ShouldContain(n2.Params, []string{tagDataKey, uriDataKey})
	w.ShouldBeEqual(n2.Params[uriDataKey],
		json.RawMessage(`"urn:epc:id:sgtin:0888446.067142.193853396487"`))
	w.ShouldBeEqual(n2.Params[tagDataKey],
		json.RawMessage(`"30143639F84191AD22901607"`))
}
