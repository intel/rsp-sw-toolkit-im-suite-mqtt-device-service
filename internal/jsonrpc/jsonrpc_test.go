/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package jsonrpc

import (
	"encoding/json"
	"github.impcloud.net/RSP-Inventory-Suite/expect"
	"testing"
)

func TestNotification_MarshalJSON(t *testing.T) {
	w := expect.WrapT(t)
	n := Notification{
		Version: Version,
		Method:  "my_method",
	}

	marshaled := w.ShouldHaveResult(json.Marshal(n)).([]byte)
	w.ShouldBeEqual(marshaled, []byte(`{"jsonrpc":"2.0","method":"my_method"}`))

	w.ShouldSucceed(n.SetParam("a", "b"))
	marshaled = w.ShouldHaveResult(json.Marshal(&n)).([]byte)
	w.ShouldBeEqual(marshaled, []byte(`{"jsonrpc":"2.0","method":"my_method","params":{"a":"b"}}`))

	w.ShouldSucceed(n.SetParam("a", "c"))
	w.ShouldContain(n.Params, []string{"a"})
	marshaled = w.ShouldHaveResult(json.Marshal(n)).([]byte)
	w.ShouldBeEqual(marshaled, []byte(`{"jsonrpc":"2.0","method":"my_method","params":{"a":"c"}}`))

	w.ShouldSucceed(n.SetParam("d", "e"))
	w.ShouldContain(n.Params, []string{"a", "d"})
	marshaled = w.ShouldHaveResult(json.Marshal(n)).([]byte)
	w.ShouldBeEqual(marshaled, []byte(`{"jsonrpc":"2.0","method":"my_method","params":{"a":"c","d":"e"}}`))
}

func TestNotification_UnmarshalJSON(t *testing.T) {
	w := expect.WrapT(t)
	data := []byte(`{"jsonrpc":"2.0","method":"my_method","params":{` +
		`"strParam":"imma string",` +
		`"objParam":{"nestStr":"soam I","nestFloat":3.14},` +
		`"numParam":1e10}}`)

	var n Notification
	w.ShouldSucceed(json.Unmarshal(data, &n))
	w.ShouldContain(n.Params, []string{"strParam", "objParam", "numParam"})
	w.ShouldContainValues(n.Params, []json.RawMessage{
		json.RawMessage(`"imma string"`),
		json.RawMessage(`{"nestStr":"soam I","nestFloat":3.14}`),
		json.RawMessage(`1e10`),
	})

	var s string
	w.ShouldSucceed(n.GetParam("strParam", &s))
	w.ShouldBeEqual("imma string", s)

	var o map[string]interface{}
	w.ShouldSucceed(n.GetParam("objParam", &o))
	w.ShouldContain(o, []string{"nestStr", "nestFloat"})
	w.ShouldContainValues(o, []interface{}{"soam I", 3.14})

	var i float64
	w.ShouldSucceed(n.GetParam("numParam", &i))
	w.ShouldBeEqual(1e10, i)

	n = Notification{}
	data = []byte(`{"jsonrpc":"2.0","method":"my_method"}`)
	w.ShouldSucceed(json.Unmarshal(data, &n))
	w.As("no parameters").ShouldFail(n.GetParam("b", s))
}
