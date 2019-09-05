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
	data := []byte(`{"jsonrpc":"2.0","method":"my_method","params":{"a":"b"}}`)

	var n Notification
	w.ShouldSucceed(json.Unmarshal(data, &n))
	w.ShouldContain(n.Params, []string{"a"})
	w.ShouldBeEqual(w.ShouldHaveResult(n.GetParamStr("a")).(string), "b")
	w.ShouldBeEqual(w.ShouldHaveResult(json.Marshal(n)).([]byte), data)

	n = Notification{}
	data = []byte(`{"jsonrpc":"2.0","method":"my_method"}`)
	w.ShouldSucceed(json.Unmarshal(data, &n))
	w.ShouldHaveError(n.GetParamStr("b"))
	w.ShouldBeEqual(w.ShouldHaveResult(json.Marshal(n)).([]byte), data)
}
