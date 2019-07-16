package driver

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/internal/models"
	"net/url"
	"regexp"
	"strings"

	"github.com/eclipse/paho.mqtt.golang"
	sdk "github.com/edgexfoundry/device-sdk-go"
	sdkModel "github.com/edgexfoundry/device-sdk-go/pkg/models"
)

var (
	topicMappings map[*regexp.Regexp]string
)

// pre-compute regexes for topic->deviceResource value descriptor mappings
func compileTopicMappings(conf configuration) (map[*regexp.Regexp]string, error) {
	// make sure that there is exactly one mapping for every topic
	if len(conf.IncomingTopics) != len(conf.IncomingTopicResourceMappings) {
		return nil, fmt.Errorf("incoming topics (len: %d) %#v has a different length than topic mappings (len: %d) %#v",
			len(conf.IncomingTopics), conf.IncomingTopics,
			len(conf.IncomingTopicResourceMappings), conf.IncomingTopicResourceMappings)
	}

	mappings := make(map[*regexp.Regexp]string, len(conf.IncomingTopicResourceMappings))

	for index, topic := range conf.IncomingTopics {
		pattern := topic
		// note, replacing '+' needs to happen first to avoid multiple substitution

		// '+' is a single-level wildcard for mqtt topics. we only want to match from the last / to the / after the +
		//    replacements are unlimited
		pattern = strings.ReplaceAll(pattern, "+", "[^/]+")
		// escape the '$' character as it is used in `$SYS` topics
		pattern = strings.ReplaceAll(pattern, "$", "\\$")
		// '#' is a multi-level wildcard for mqtt topics. once we see this, we match anything after it.
		//   it should only exist at the end of the topic, and only once
		pattern = strings.Replace(pattern, "#", ".+", 1)
		// make it be exact matches
		pattern = fmt.Sprintf("^%s$", pattern)

		driver.Logger.Debug(fmt.Sprintf("topic: %s, pattern: %s", topic, pattern))

		res, err := regexp.Compile(pattern)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to compile regex %s for topic %s", pattern, topic)
		}

		mappings[res] = conf.IncomingTopicResourceMappings[index]
	}

	return mappings, nil
}

// mapTopicToValueDescriptor takes a topic (non-wildcard) and attempts to match it against the
// configured topic mappings in order to determine the valueDescriptor / device resource to use
// returns an error if nothing matches
func mapTopicToValueDescriptor(topic string) (string, error) {
	for pattern, descriptor := range topicMappings {
		if pattern.MatchString(topic) {
			driver.Logger.Info(fmt.Sprintf("topic: %s, valueDescriptor: %s", topic, descriptor))
			return descriptor, nil
		}
	}

	return "", fmt.Errorf("unable to determine valueDescriptor for topic: %s", topic)
}

// startIncomingListening starts listening on all the configured IncomingTopics;
// when a new message comes in, the onIncomingDataReceived method converts it to
// an EdgeX message.
func startIncomingListening(done <-chan interface{}) error {
	conf := *driver.Config

	mappings, err := compileTopicMappings(conf)
	if err != nil {
		return errors.Wrap(err, "issue creating topic mappings to device resource value descriptors")
	}
	topicMappings = mappings

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

func onIncomingDataReceived(_ mqtt.Client, message mqtt.Message) {
	var jn models.JSONRPC
	if err := json.Unmarshal(message.Payload(), &jn); err != nil {
		driver.Logger.Error(fmt.Sprintf("Unmarshal failed: %+v", err))
		return
	}

	if jn.Version != jsonRPC20 {
		driver.Logger.Error(fmt.Sprintf("Invalid version: %s", jn.Version))
		return
	}

	valueDescriptor, err := mapTopicToValueDescriptor(message.Topic())
	if err != nil {
		driver.Logger.Warn(err.Error())
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
