package driver

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/models"
	"net/url"
	"regexp"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	sdk "github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
)

var (
	topicMappings = map[*regexp.Regexp]string{
		compileOrPanic("rfid/gw/heartbeat"): "gw_heartbeat",
		compileOrPanic("rfid/gw/events"): "gw_event",
		compileOrPanic("rfid/gw/alerts"): "gw_alert",
		compileOrPanic("rfid/gw/response"): "gw_notification",
		compileOrPanic("rfid/rsp/data/.+"): "rsp_data",
	}
)

func compileOrPanic(pattern string) *regexp.Regexp {
	res, err := regexp.Compile(pattern)
	if err != nil {
		panic(errors.Wrapf(err,"unable to compile regex %v", pattern))
	}

	return res
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
		conf.IncomingKeepAlive)
	if err != nil {
		return err
	}

	defer func() {
		if client.IsConnected() {
			client.Disconnect(5000)
		}
	}()

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

func onIncomingDataReceived(client mqtt.Client, message mqtt.Message) {
	var jn models.JSONRPC
	if err := json.Unmarshal(message.Payload(), &jn); err != nil {
		driver.Logger.Error(fmt.Sprintf("Unmarshal failed: %+v", err))
		return
	}

	if jn.Version != jsonRPC20 {
		driver.Logger.Error(fmt.Sprintf("Invalid version: %s", jn.Version))
		return
	}

	topic := message.Topic()
	var valueDescriptor string

	for pattern, descriptor := range topicMappings {
		if pattern.MatchString(topic) {
			valueDescriptor = descriptor
			driver.Logger.Info(fmt.Sprintf("valueDescriptor: %s", valueDescriptor))
			break
		}
	}

	if valueDescriptor == "" {
		driver.Logger.Warn(fmt.Sprintf("unable to determine valueDescriptor for topic: %s", topic))
		return
	}

	reading := string(message.Payload())
	service := sdk.RunningService()
	deviceName := driver.Config.DeviceName

	resource, ok := service.DeviceResource(deviceName, valueDescriptor, "get")
	if !ok {
		driver.Logger.Warn(fmt.Sprintf("[Incoming listener] "+
			"Incoming reading ignored. "+
			"No DeviceObject found: topic=%v device=%v method=%v",
			message.Topic(), deviceName, jn.Method))
		return
	}

	req := sdkModel.CommandRequest{
		DeviceResourceName: valueDescriptor,
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
		message.Topic(), jn.Method, len(message.Payload())))

	driver.AsyncCh <- asyncValues
}
