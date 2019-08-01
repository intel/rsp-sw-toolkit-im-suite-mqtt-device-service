.PHONY: build test clean prepare update docker

GO = CGO_ENABLED=0 GO111MODULE=on go

MICROSERVICES=cmd/mqtt-device-service

.PHONY: $(MICROSERVICES)

DOCKERS=docker_mqtt-device-service_go

.PHONY: $(DOCKERS)

VERSION=$(shell cat ./VERSION)
GIT_SHA=$(shell git rev-parse HEAD)

GOFLAGS=-ldflags "-X github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service-go.Version=$(VERSION)"

STACK_NAME ?= Inventory-Suite-Dev
SERVICE_NAME ?= mqtt-device-service
PROJECT_NAME ?= mqtt-device-service

default: build

scale=docker service scale $(STACK_NAME)_$(SERVICE_NAME)=$1 $2

wait_for_service=	@printf "Waiting for $(SERVICE_NAME) service to$1..."; \
					while [  $2 -z `docker ps -qf name=$(STACK_NAME)_$(SERVICE_NAME).1` ]; \
                 	do \
                 		printf "."; \
                 		sleep 0.3;\
                 	done; \
                 	printf "\n";

log=docker logs $1$2 `docker ps -qf name=$(STACK_NAME)_$(SERVICE_NAME).1` 2>&1

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

build-dev:
	$(MAKE) -C .. $(PROJECT_NAME)

iterate:
	$(call scale,0,-d)
	$(MAKE) build-dev
	$(call wait_for_service, stop, !)
	$(call scale,1,-d)
	$(call wait_for_service, start)
	$(MAKE) tail

restart:
	$(call scale,0,-d)
	$(call wait_for_service, stop, !)
	$(call scale,1,-d)
	$(call wait_for_service, start)

tail:
	$(call log,-f,$(args))

scale:
	$(call scale,$(n),$(args))

fmt:
	go fmt ./...
