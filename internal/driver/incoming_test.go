/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package driver

import (
	"encoding/json"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.impcloud.net/RSP-Inventory-Suite/expect"
	"github.impcloud.net/RSP-Inventory-Suite/gojsonschema"
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

func TestJSONValidation(t *testing.T) {
	w := expect.WrapT(t)
	d := &Driver{
		incomingSchemas: map[string]*gojsonschema.Schema{},
		responseSchemas: map[string]*gojsonschema.Schema{},
		Config:          &configuration{SchemasDir: "testdata"},
	}

	w.As("empty method").ShouldHaveError(d.loadSchema(incomingDir, ""))
	w.As("empty type").ShouldHaveError(d.loadSchema("", "m1"))

	w.As("valid m1").ShouldSucceed(d.validateIncoming("m1", []byte(`{"id": 5}`)))
	w.As("valid m2").ShouldSucceed(d.validateIncoming("m2", []byte(`{"s": "123"}`)))

	w.As("needs id; s invalid").ShouldFail(d.validateIncoming("m1", []byte(`{"s": "123"}`)))
	w.As("needs s; id invalid").ShouldFail(d.validateIncoming("m2", []byte(`{"id": 5}`)))
	w.As("nil data").ShouldFail(d.validateIncoming("m1", nil))
	w.As("empty data").ShouldFail(d.validateIncoming("m1", []byte{}))
	w.As("one null byte").ShouldFail(d.validateIncoming("m1", []byte{0x00}))
	w.As("invalid json").ShouldFail(d.validateIncoming("m1", []byte(`id: 1`)))
	w.As("missing id").ShouldFail(d.validateIncoming("m1", []byte(`{}`)))
	w.As("id too small").ShouldFail(d.validateIncoming("m1", []byte(`{"id": -1}`)))
	w.As("id too large").ShouldFail(d.validateIncoming("m1", []byte(`{"id": 11}`)))
	w.As("no such method").ShouldFail(d.validateIncoming("no_such_method", []byte{0x00}))
	w.As("bad schema").ShouldFail(d.validateIncoming("invalid", []byte{0x00}))

	w.As("valid m1").ShouldSucceed(d.validateResponse("m1", []byte(`{"id": 5}`)))
	w.As("valid m2").ShouldSucceed(d.validateResponse("m2", []byte(`{"s": "123"}`)))

	w.As("needs id; s invalid").ShouldFail(d.validateResponse("m1", []byte(`{"s": "123"}`)))
	w.As("needs s; id invalid").ShouldFail(d.validateResponse("m2", []byte(`{"id": 5}`)))
	w.As("nil data").ShouldFail(d.validateResponse("m1", nil))
	w.As("empty data").ShouldFail(d.validateResponse("m1", []byte{}))
	w.As("one null byte").ShouldFail(d.validateResponse("m1", []byte{0x00}))
	w.As("invalid json").ShouldFail(d.validateResponse("m1", []byte(`id: 1`)))
	w.As("missing id").ShouldFail(d.validateResponse("m1", []byte(`{}`)))
	w.As("id too small").ShouldFail(d.validateResponse("m1", []byte(`{"id": -1}`)))
	w.As("id too large").ShouldFail(d.validateResponse("m1", []byte(`{"id": 11}`)))
	w.As("no such method").ShouldFail(d.validateResponse("no_such_method", []byte{0x00}))
	w.As("bad schema").ShouldFail(d.validateResponse("invalid", []byte{0x00}))
}
