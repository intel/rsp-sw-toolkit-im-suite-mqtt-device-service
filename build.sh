#!/bin/sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=on go build -v -o mqtt-device-service ./cmd