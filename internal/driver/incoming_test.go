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
			paramDataKey: []byte(`[{"epc":"30143639F84191AD22901607"},{"epc":"30143639F84191AD23901607"}]`),
		},
	}

	modified := w.ShouldHaveResult(driverInstance.processResource(n)).([]byte)
	w.ShouldNotBeNil(modified)

	var result jsonrpc.Notification
	var data []jsonrpc.Parameters
	w.ShouldSucceed(json.Unmarshal(modified, &result))
	w.ShouldContain(result.Params, []string{paramDataKey})
	w.ShouldSucceed(json.Unmarshal(result.Params[paramDataKey], &data))
	w.ShouldHaveLength(data, 2)
	w.ShouldContain(data[0], []string{tagDataKey, uriDataKey})
	w.ShouldContain(data[1], []string{tagDataKey, uriDataKey})
	w.ShouldBeEqual(data[0][uriDataKey],
		json.RawMessage(`"urn:epc:id:sgtin:0888446.067142.193853396487"`))
	w.ShouldBeEqual(data[0][tagDataKey],
		json.RawMessage(`"30143639F84191AD22901607"`))
	w.ShouldBeEqual(data[1][uriDataKey],
		json.RawMessage(`"urn:epc:id:sgtin:0888446.067142.193870173703"`))
	w.ShouldBeEqual(data[1][tagDataKey],
		json.RawMessage(`"30143639F84191AD23901607"`))
}

func TestProcessTagData_fullMessage(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()

	jsonData := []byte(`{
    "jsonrpc" : "2.0",
    "method" : "inventory_data",
    "params" : {
        "sent_on" : 1570840098444.0,
        "period" : 500.0,
        "device_id" : "RSP-15077a",
        "location" : {
            "latitude" : 0.0,
            "longitude" : 0.0,
            "altitude" : 0.0
        },
        "facility_id" : "DEFAULT_FACILITY",
        "motion_detected" : false,
        "data" : [ 
            {
                "epc" : "30143639F84191AD22901607",
                "tid" : null,
                "antenna_id" : 0.0,
                "last_read_on" : 1570840098434.0,
                "rssi" : -608.0,
                "phase" : -32.0,
                "frequency" : 903250.0
            },
            {
                "epc" : "30143639F84191AD23901607",
                "tid" : null,
                "antenna_id" : 0.0,
                "last_read_on" : 1570840098434.0,
                "rssi" : -608.0,
                "phase" : -32.0,
                "frequency" : 903250.0
            }
        ]
    }
}`)

	var n jsonrpc.Notification
	w.ShouldSucceed(json.Unmarshal(jsonData, &n))

	modified := w.ShouldHaveResult(driverInstance.processResource(n)).([]byte)
	w.ShouldNotBeNil(modified)

	var result jsonrpc.Notification
	var data []jsonrpc.Parameters
	w.ShouldSucceed(json.Unmarshal(modified, &result))
	w.ShouldContain(result.Params, []string{paramDataKey})
	w.ShouldSucceed(json.Unmarshal(result.Params[paramDataKey], &data))
	w.ShouldHaveLength(data, 2)
	w.ShouldContain(data[0], []string{tagDataKey, uriDataKey})
	w.ShouldContain(data[1], []string{tagDataKey, uriDataKey})
	w.ShouldBeEqual(data[0][uriDataKey],
		json.RawMessage(`"urn:epc:id:sgtin:0888446.067142.193853396487"`))
	w.ShouldBeEqual(data[0][tagDataKey],
		json.RawMessage(`"30143639F84191AD22901607"`))
	w.ShouldBeEqual(data[1][uriDataKey],
		json.RawMessage(`"urn:epc:id:sgtin:0888446.067142.193870173703"`))
	w.ShouldBeEqual(data[1][tagDataKey],
		json.RawMessage(`"30143639F84191AD23901607"`))
}
