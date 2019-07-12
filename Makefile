.PHONY: build test clean prepare update docker

GO = CGO_ENABLED=0 GO111MODULE=on go

MICROSERVICES=cmd/mqtt-device-service

.PHONY: $(MICROSERVICES)

DOCKERS=docker_mqtt-device-service_go

.PHONY: $(DOCKERS)

VERSION=$(shell cat ./VERSION)
GIT_SHA=$(shell git rev-parse HEAD)

GOFLAGS=-ldflags "-X github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service-go.Version=$(VERSION)"

build: $(MICROSERVICES)
	$(GO) build ./...

cmd/mqtt-device-service:
	$(GO) build $(GOFLAGS) -o $@ ./cmd

test:
	$(GO) test ./... -cover

clean:
	rm -f $(MICROSERVICES)

run:
	cd bin && ./edgex-launch.sh

docker: $(DOCKERS)

docker_mqtt-device-service_go:
	docker build \
		--label "git_sha=$(GIT_SHA)" \
		-t mqtt-device-service-go:$(GIT_SHA) \
		-t mqtt-device-service-go:$(VERSION)-dev \
		.
