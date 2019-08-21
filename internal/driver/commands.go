package driver

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/jsonrpc"
	"strings"
	"time"

	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
)

// HandleReadCommands handles CommandRequests to read data via MQTT.
//
// It satisfies them by creating a new MQTT client with the protocol, sending the
// requests as JSON RPC messages on all configured topics, then waiting for a
// response on any of the response topics; once a response comes in, it returns
// that result.
func (d *Driver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModel.CommandRequest) ([]*sdkModel.CommandValue, error) {
	var responses = make([]*sdkModel.CommandValue, len(reqs))
	var err error

	for i, req := range reqs {
		res, err := d.handleReadCommandRequest(deviceName, req)
		if err != nil {
			d.Logger.Warn("Handle read commands failed", "cause", err)
			return responses, err
		}

		responses[i] = res
	}

	return responses, err
}

// handleReadCommandRequest takes care of the JSON RPC command/response portion
// of the HandleReadCommands.
//
// The command request is published on all of the incoming connection info topics.
func (d *Driver) handleReadCommandRequest(deviceName string, req sdkModel.CommandRequest) (*sdkModel.CommandValue, error) {
	var err error
	request := jsonrpc.JsonRequest{
		Version: jsonRpcVersion,
		Method:  req.DeviceResourceName,
		Id:      uuid.New().String(),
	}

	// Sensor devices start with "RSP", this will not be needed in near future as Edgex is going to support GET requests with query parameters
	// If the device is sensor add the device_id as params to the command request
	if strings.HasPrefix(deviceName, "RSP") {
		deviceIdParam := jsonrpc.DeviceIdParam{DeviceId: deviceName}
		request.Params, err = json.Marshal(deviceIdParam)
		if err != nil {
			err = fmt.Errorf("marshalling of command parameters failed: error=%v", err)
			return nil, err
		}
	}

	// marshal request to jsonrpc format
	jsonRpcRequest, err := json.Marshal(request)
	if err != nil {
		err = fmt.Errorf("marshalling of command request failed: error=%v", err)
		return nil, err
	}

	// Publish the command request
	d.Logger.Info("Publish command", "command", string(jsonRpcRequest))
	d.Client.Publish(d.Config.CommandTopic, d.Config.CommandQos, retained, jsonRpcRequest)

	cmdResponse, ok := d.fetchCommandResponse(request.Id)
	if !ok {
		err = fmt.Errorf("no command response or getting response delayed for method=%v", request.Method)
		return nil, err
	}

	var responseMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(cmdResponse), &responseMap); err != nil {
		err = fmt.Errorf("unmarshalling of command response failed: error=%v", err)
		return nil, err
	}

	// Parse response to extract result or error field from the jsonrpc response
	var reading string
	_, ok = responseMap["result"]
	if ok {
		reading = string(responseMap["result"])
	} else {
		_, ok = responseMap["error"]
		if ok {
			reading = string(responseMap["error"])
		} else {
			err = fmt.Errorf("invalid command response: %v", cmdResponse)
			return nil, err
		}
	}

	origin := time.Now().UnixNano() / int64(time.Millisecond)
	value := sdkModel.NewStringValue(req.DeviceResourceName, origin, reading)

	d.Logger.Info("Get command finished", "response", cmdResponse)

	return value, err
}

// HandleWriteCommands ignores all requests; write commands (PUT requests) are not currently supported.
func (d *Driver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModel.CommandRequest, params []*sdkModel.CommandValue) error {
	return nil
}

// onCommandResponseReceived handles messages on the response topic and parses them as jsonrpc 2.0 Response messages
func (d *Driver) onCommandResponseReceived(_ mqtt.Client, message mqtt.Message) {
	go func(message mqtt.Message) {
		var response jsonrpc.JsonResponse

		if err := json.Unmarshal(message.Payload(), &response); err != nil {
			d.Logger.Error("[Response listener] Unmarshalling of command response failed", "cause", err)
			return
		}

		if response.Id != "" {
			d.CommandResponses.Store(response.Id, string(message.Payload()))
			d.Logger.Info("[Response listener] Command response received", "topic", message.Topic(), "msg", string(message.Payload()))
		} else {
			d.Logger.Debug("[Response listener] Command response ignored. No ID found in the message",
				"topic", message.Topic(), "msg", string(message.Payload()))
		}
	}(message)
}

// fetchCommandResponse use to wait and fetch response from CommandResponses map
func (d *Driver) fetchCommandResponse(cmdUuid string) (string, bool) {
	var cmdResponse interface{}
	var ok bool
	for i := 0; i < d.Config.MaxWaitTimeForReq; i++ {
		cmdResponse, ok = d.CommandResponses.Load(cmdUuid)
		if ok {
			d.CommandResponses.Delete(cmdUuid)
			break
		} else {
			time.Sleep(time.Second * time.Duration(1))
		}
	}

	return fmt.Sprintf("%v", cmdResponse), ok
}
