REGISTRY?=
NAMESPACE?=rsp
SERVICE_NAME?=mqtt-device-service
MODULE_NAME?=github.com/intel/rsp-sw-toolkit-im-suite-mqtt-device-service
VERSION?=$(shell cat ./VERSION)
IMAGE_NAME?=$(if $(REGISTRY),$(REGISTRY)/,)$(if $(NAMESPACE),$(NAMESPACE)/,)$(SERVICE_NAME)

# See https://github.com/golang/go/wiki/GcToolchainTricks#including-build-information-in-the-executable
GOFLAGS:=-ldflags "-X $(MODULE_NAME)/cmd/main.Version=$(VERSION)"
GO:=CGO_ENABLED=0 GOARCH=amd64 GOOS=linux GO111MODULE=on go
TAGS?=$(VERSION) dev latest
LABELS?="git_sha=$(shell git rev-parse HEAD)"

.PHONY: build image test clean clean-img

build: $(SERVICE_NAME)
DEPENDS=internal/driver/*.go internal/jsonrpc/*.go cmd/*.go \
	cmd/res/*.toml cmd/res/*.yml cmd/res/docker/*.toml \
	cmd/res/schemas/incoming/*.json cmd/res/schemas/responses/*.json
$(SERVICE_NAME): go.mod VERSION $(DEPENDS)
	$(GO) build $(GOFLAGS) -o $@ ./cmd

image: Dockerfile
	docker build \
		$(addprefix --label ,$(LABELS)) \
		$(addprefix -t $(IMAGE_NAME):,$(TAGS)) \
		.

test:
	go test ./... -cover

clean:
	-rm -f $(SERVICE_NAME)

clean-img:
	-docker rmi $(addprefix $(IMAGE_NAME):,$(TAGS))
