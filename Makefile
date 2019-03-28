GOOS:=linux
GOARCH:=amd64
GOPATH:=$$HOME/go
SRC_PATH=github.impcloud.net/Responsive-Retail-Inventory/gateway-device-service

REPO:=saites/gateway-device-service
GIT_SHA:=$(shell git rev-parse HEAD)
VERSION=$(shell cat ./VERSION)

GO=GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=off go
GOFLAGS=-ldflags "-X github.com/edgexfoundry/device-mqtt-go.Version=$(VERSION)"

.PHONY: build image run down clean clean-image dev

build: gateway-device-service
gateway-device-service:
	docker run \
		--rm \
		-it \
		--name=gobuilder \
		-v gobuildcache:/cache \
		-v $(GOPATH)/src/$(SRC_PATH):/go/src/$(SRC_PATH) \
		-w /go/src/$(SRC_PATH) \
		-e GOCACHE=/cache \
		golang:1.12 \
		sh -c '$(GO) build -v -o $@ ./cmd'

image: gateway-device-service.docker
gateway-device-service.docker: Dockerfile gateway-device-service
	docker build \
		-t $(REPO):v$(GIT_SHA) \
		.
	touch $@

run: gateway-device-service.docker
	IMAGE=$(REPO):v$(GIT_SHA) docker-compose up -d

EDGEX_NETWORK:=$(shell docker network ls -qf name=edgex-network)
override docker_args += --name gateway-device-service \
	-d -p "49982:49982" --net $(EDGEX_NETWORK) -v $(GOPATH)/src/$(SRC_PATH)/cmd/res:/res \
	-e no_proxy="*" -e NO_PROXY="*" \
	--add-host "mosquitto-server:192.168.99.100"
override cmd_args += --profile=dev --confdir=/res
dev: gateway-device-service.docker
	docker rm gateway-device-service || true
	docker run $(docker_args) $(REPO):v$(GIT_SHA) $(cmd_args)

down:
	IMAGE=$(REPO):v$(GIT_SHA) docker-compose down

clean-image:
	rm -rf gateway-device-service.docker

clean: clean-image
	rm -rf gateway-device-service

