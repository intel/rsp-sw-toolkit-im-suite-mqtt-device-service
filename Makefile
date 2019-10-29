SERVICE_NAME?=mqtt-device-service
MODULE_NAME?=github.impcloud.net/RSP-Inventory-Suite/$(SERVICE_NAME)
VERSION?=$(shell cat ./VERSION)

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

image: $(SERVICE_NAME) Dockerfile
	docker build \
		$(addprefix --label ,$(LABELS)) \
		$(addprefix -t $(SERVICE_NAME):,$(TAGS)) \
		.

test:
	go test ./... -cover

clean:
	-rm -f $(SERVICE_NAME)

clean-img:
	-docker rmi $(addprefix $(SERVICE_NAME):,$(TAGS))
