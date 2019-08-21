package driver

import (
	"encoding/json"
	"fmt"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/jsonrpc"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	sdk "github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
)

const sensorHeartbeat = "heartbeat"

func (d *Driver) onIncomingDataReceived(_ mqtt.Client, message mqtt.Message) {
	go func(message mqtt.Message) {
		var incomingData jsonrpc.JsonRequest
		if err := json.Unmarshal(message.Payload(), &incomingData); err != nil {
			d.Logger.Error(fmt.Sprintf("Unmarshal failed: %+v", err))
			return
		}

		if incomingData.Version != jsonRpcVersion {
			d.Logger.Error(fmt.Sprintf("Invalid version: %s", incomingData.Version))
			return
		}

		// JsonRpc Responses do not contain a method field. We also do not want to send these to core-data
		resourceName := incomingData.Method
		if resourceName == "" {
			d.Logger.Warn("[Incoming listener] "+
				"Incoming reading ignored. "+
				"No method field in message.",
				"msg", string(message.Payload()))
			return
		}

		// register new sensor device in Edgex to be able to send GET command requests with params to RSP Controller
		if resourceName == sensorHeartbeat {
			var heartbeat map[string]interface{}
			if err := json.Unmarshal(incomingData.Params, &heartbeat); err != nil {
				d.Logger.Error(fmt.Sprintf("Unmarshalling of sensor heartbeat params failed: %+v", err))
			}
			deviceId := heartbeat["device_id"].(string)

			// registering the sensor only if it is already not registered
			if _, notFound := sdk.RunningService().GetDeviceByName(deviceId); notFound != nil {
				d.registerRSP(deviceId)
			}
		}

		origin := time.Now().UnixNano() / int64(time.Millisecond)
		value := sdkModel.NewStringValue(resourceName, origin, string(message.Payload()))

		d.Logger.Info("[Incoming listener] Incoming reading received",
			"topic", message.Topic(),
			"method", incomingData.Method,
			"msgLen", len(message.Payload()))

		d.AsyncCh <- &sdkModel.AsyncValues{
			DeviceName:    d.Config.ControllerName,
			CommandValues: []*sdkModel.CommandValue{value},
		}
	}(message)
}
