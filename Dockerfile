#
# Copyright (C) 2018 IOTech Ltd
#
# SPDX-License-Identifier: Apache-2.0

# INTEL CONFIDENTIAL
# Copyright (2017) Intel Corporation.
#
# The source code contained or described herein and all documents related to the source code ("Material")
# are owned by Intel Corporation or its suppliers or licensors. Title to the Material remains with
# Intel Corporation or its suppliers and licensors. The Material may contain trade secrets and proprietary
# and confidential information of Intel Corporation and its suppliers and licensors, and is protected by
# worldwide copyright and trade secret laws and treaty provisions. No part of the Material may be used,
# copied, reproduced, modified, published, uploaded, posted, transmitted, distributed, or disclosed in
# any way without Intel/'s prior express written permission.
# No license under any patent, copyright, trade secret or other intellectual property right is granted
# to or conferred upon you by disclosure or delivery of the Materials, either expressly, by implication,
# inducement, estoppel or otherwise. Any license under such intellectual property rights must be express
# and approved by Intel in writing.
# Unless otherwise agreed by Intel in writing, you may not remove or alter this notice or any other
# notice embedded in Materials by Intel or Intel's suppliers or licensors in any way.

# Intel modified the project name from device-mqtt-go to mqtt-device-service

ARG ALPINE=golang:1.11-alpine
FROM ${ALPINE} AS builder
ARG ALPINE_PKG_BASE="build-base git openssh-client"
ARG ALPINE_PKG_EXTRA=""

# Replicate the APK repository override.
# If it is no longer necessary to avoid the CDN mirros we should consider dropping this as it is brittle.
RUN sed -e 's/dl-cdn[.]alpinelinux.org/nl.alpinelinux.org/g' -i~ /etc/apk/repositories
# Install our build time packages.
RUN apk add --no-cache ${ALPINE_PKG_BASE} ${ALPINE_PKG_EXTRA}

WORKDIR $GOPATH/src/github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service

ENV GO111MODULE=on
# Download go modules first so they can be cached for faster subsequent builds
COPY go.mod go.mod
RUN go mod download

COPY . .

# To run tests in the build container:
#   docker build --build-arg 'MAKE=build test' .
# This is handy of you do your Docker business on a Mac
ARG MAKE=build
RUN make $MAKE


FROM scratch

ENV APP_PORT=49982
EXPOSE $APP_PORT

COPY --from=builder /go/src/github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service/cmd /

ENTRYPOINT ["/mqtt-device-service","--profile=docker","--confdir=/res"]