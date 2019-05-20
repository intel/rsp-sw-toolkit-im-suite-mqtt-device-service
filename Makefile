GOOS:=linux
GOARCH:=amd64
GOPATH:=$$HOME/go
SRC_PATH=github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service

REPO:=saites/mqtt-device-service
GIT_SHA:=$(shell git rev-parse HEAD)
VERSION=$(shell cat ./VERSION)

GO=GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=off go
GOFLAGS=-ldflags "-X github.com/edgexfoundry/device-mqtt-go.Version=$(VERSION)"

.PHONY: build image run down clean clean-image dev

build: mqtt-device-service
mqtt-device-service:
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

image: mqtt-device-service.docker
mqtt-device-service.docker: Dockerfile mqtt-device-service
	docker build \
		-t $(REPO):v$(GIT_SHA) \
		.
	touch $@

run: mqtt-device-service.docker
	IMAGE=$(REPO):v$(GIT_SHA) docker-compose up -d

EDGEX_NETWORK:=$(shell docker network ls -qf name=edgex-network)
override docker_args += --name mqtt-device-service \
	-d -p "49982:49982" --net $(EDGEX_NETWORK) -v $(GOPATH)/src/$(SRC_PATH)/cmd/res:/res \
	-e no_proxy="*" -e NO_PROXY="*" \
	--add-host "mosquitto-server:192.168.99.100"
override cmd_args += --profile=dev --confdir=/res
dev: mqtt-device-service.docker
	docker rm mqtt-device-service || true
	docker run $(docker_args) $(REPO):v$(GIT_SHA) $(cmd_args)

down:
	IMAGE=$(REPO):v$(GIT_SHA) docker-compose down

clean-image:
	rm -rf mqtt-device-service.docker

clean: clean-image
	rm -rf mqtt-device-service

