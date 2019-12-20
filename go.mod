module github.com/intel/rsp-sw-toolkit-im-suite-mqtt-device-service

go 1.12

require (
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/edgexfoundry/device-sdk-go v1.0.0
	github.com/edgexfoundry/go-mod-core-contracts v0.1.0
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.0 // indirect
	github.com/intel/rsp-sw-toolkit-im-suite-expect v1.1.4
	github.com/intel/rsp-sw-toolkit-im-suite-gojsonschema v1.0.0
	github.com/intel/rsp-sw-toolkit-im-suite-tagcode v1.2.1
	github.com/kr/pretty v0.1.0 // indirect
	github.com/pkg/errors v0.8.1
	golang.org/x/net v0.0.0-20190213061140-3a22650c66bd // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)

replace github.com/intel/rsp-sw-toolkit-im-suite-expect => github.impcloud.net/RSP-Inventory-Suite/expect v1.1.4

replace github.com/intel/rsp-sw-toolkit-im-suite-gojsonschema => github.impcloud.net/RSP-Inventory-Suite/gojsonschema v1.2.0

replace github.com/intel/rsp-sw-toolkit-im-suite-tagcode => github.impcloud.net/RSP-Inventory-Suite/tagcode v1.2.1
