package driver

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/models"
	"net/url"
	"strings"

	"github.com/eclipse/paho.mqtt.golang"
	sdk "github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
)

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
		driver.Logger.Debug(fmt.Sprintf("publish onconnect topic: %s, message: %s", topic, msg))
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
			Host:   fmt.Sprintf("%s:%d", conf.IncomingHost, conf.IncomingPort),
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
			driver.Logger.Info(
				fmt.Sprintf("[Incoming listener] Stop incoming data listening. Cause:%v",
					token.Error(),
				),
			)
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
		driver.Logger.Warn(fmt.Sprintf("[Incoming listener] "+
			"Incoming reading ignored. "+
			"No method field in message. msg=%s",
			string(message.Payload())))
		return
	}

	deviceName := driver.Config.DeviceName
	reading := string(message.Payload())
	service := sdk.RunningService()

	resource, ok := service.DeviceResource(deviceName, resourceName, "get")
	if !ok {
		driver.Logger.Warn(fmt.Sprintf("[Incoming listener] "+
			"Incoming reading ignored. "+
			"No DeviceObject found: topic=%v device=%v method=%v",
			message.Topic(), deviceName, incomingData.Method))
		return
	}

	req := sdkModel.CommandRequest{
		DeviceResourceName: resourceName,
		Type:               sdkModel.ParseValueType(resource.Properties.Value.Type),
	}
	result, err := newResult(req, reading)

	if err != nil {
		driver.Logger.Warn(fmt.Sprintf("[Incoming listener] "+
			"Incoming reading ignored. "+
			"topic=%v msg=%v error=%v",
			message.Topic(), string(message.Payload()), err))
		return
	}

	asyncValues := &sdkModel.AsyncValues{
		DeviceName:    deviceName,
		CommandValues: []*sdkModel.CommandValue{result},
	}

	driver.Logger.Info(fmt.Sprintf("[Incoming listener] "+
		"Incoming reading received: "+
		"topic=%v method=%v msgLen=%v",
		message.Topic(), incomingData.Method, len(message.Payload())))

	driver.AsyncCh <- asyncValues
}
