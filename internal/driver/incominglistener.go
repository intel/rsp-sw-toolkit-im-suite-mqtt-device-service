package driver

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/models"
	"net/url"
	"strings"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	sdk "github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
	edgexModels "github.com/edgexfoundry/go-mod-core-contracts/models"
)

const sensorHeartbeat = "heartbeat"
const sensorDeviceProfile = "RSP.Device.MQTT.Profile"

func replaceMessagePlaceholders(message string) string {
	id := uuid.New().String()
	// replace {{uuid}} placeholder with generated id
	return strings.Replace(message, "{{uuid}}", id, 1)
}

func onMqttConnect(client mqtt.Client) {
	conf := *driver.Config

	driver.Logger.Info("mqtt incoming listener client connected")

	topic := conf.OnConnectPublishTopic
	if topic != "" {
		msg := replaceMessagePlaceholders(conf.OnConnectPublishMessage)
		driver.Logger.Debug("publish onconnect", "topic", topic, "message", msg)
		client.Publish(topic, qos, retained, msg)
	}
}

// startIncomingListening starts listening on all the configured IncomingTopics;
// when a new message comes in, the onIncomingDataReceived method converts it to
// an EdgeX message.
func startIncomingListening(done <-chan interface{}) error {
	conf := *driver.Config

	client, err := createClient(
		conf.IncomingClientId,
		&url.URL{
			Scheme: strings.ToLower(conf.IncomingScheme),
			Host:   fmt.Sprintf("%s:%s", conf.IncomingHost, conf.IncomingPort),
			User:   url.UserPassword(conf.IncomingUser, conf.IncomingPassword),
		},
		conf.IncomingKeepAlive, onMqttConnect)
	if err != nil {
		return err
	}

	defer client.Disconnect(5000)

	for _, topic := range conf.IncomingTopics {
		token := client.Subscribe(topic, byte(conf.IncomingQos), onIncomingDataReceived)
		if token.Wait() && token.Error() != nil {
			driver.Logger.Info("[Incoming listener] Stop incoming data listening.", "cause", token.Error())
			return token.Error()
		}
	}

	driver.Logger.Info("[Incoming listener] Start incoming data listener. ")
	<-done
	driver.Logger.Info("[Incoming listener] Stopping incoming data listener. ")
	return nil
}

func onIncomingDataReceived(_ mqtt.Client, message mqtt.Message) {
	var incomingData models.JsonRequest
	if err := json.Unmarshal(message.Payload(), &incomingData); err != nil {
		driver.Logger.Error(fmt.Sprintf("Unmarshal failed: %+v", err))
		return
	}

	if incomingData.Version != jsonRpcVersion {
		driver.Logger.Error(fmt.Sprintf("Invalid version: %s", incomingData.Version))
		return
	}

	// JsonRpc Responses do not contain a method field. We also do not want to send these to core-data
	resourceName := incomingData.Method
	if resourceName == "" {
		driver.Logger.Warn("[Incoming listener] "+
			"Incoming reading ignored. "+
			"No method field in message.",
			"msg", string(message.Payload()))
		return
	}

	// register new sensor device in Edgex to be able to send GET command requests with params to RSP Controller
	if resourceName == sensorHeartbeat {
		var heartbeat map[string]interface{}
		if err := json.Unmarshal(incomingData.Params, &heartbeat); err != nil {
			driver.Logger.Error(fmt.Sprintf("Unmarshalling of sensor heartbeat params failed: %+v", err))
		}
		deviceId := heartbeat["device_id"].(string)

		// registering the sensor only if it is already not registered
		if _, notFound := sdk.RunningService().GetDeviceByName(deviceId); notFound != nil {
			registerSensor(deviceId)
		}
	}

	origin := time.Now().UnixNano() / int64(time.Millisecond)
	value := sdkModel.NewStringValue(resourceName, origin, string(message.Payload()))

	driver.Logger.Info("[Incoming listener] Incoming reading received",
		"topic", message.Topic(),
		"method", incomingData.Method,
		"msgLen", len(message.Payload()))

	driver.AsyncCh <- &sdkModel.AsyncValues{
		DeviceName:    driver.Config.ControllerName,
		CommandValues: []*sdkModel.CommandValue{value},
	}
}

func registerSensor(deviceId string) {
	conf := *driver.Config

	// Registering sensor devices in Edgex
	_, err := sdk.RunningService().AddDevice(edgexModels.Device{
		Name:           deviceId,
		AdminState:     edgexModels.Unlocked,
		OperatingState: edgexModels.Enabled,
		Protocols: map[string]edgexModels.ProtocolProperties{
			"mqtt": {
				"Scheme":   conf.IncomingScheme,
				"Host":     conf.IncomingHost,
				"Port":     conf.IncomingPort,
				"User":     conf.IncomingUser,
				"Password": conf.IncomingPassword,
				"ClientId": conf.RSPMqttClientId,
				"Topics":   conf.OnConnectPublishTopic,
			},
		},
		Profile: edgexModels.DeviceProfile{
			Name: sensorDeviceProfile,
		},
	})
	if err != nil {
		driver.Logger.Error(fmt.Sprintf("Registering of sensor device %v failed: %v", deviceId, err))
	}
}
