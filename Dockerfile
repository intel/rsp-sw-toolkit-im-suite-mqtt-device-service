#
# Copyright (C) 2018 IOTech Ltd
#
# SPDX-License-Identifier: Apache-2.0

# INTEL CONFIDENTIAL
# Copyright (2019) Intel Corporation.
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

# ---------------------------------------------------------
# can't create files in a scratch container, nor can you run chown, so this adds
# the file structure here so it can be copied. This also adds the prebuilt image
# so the service image itself is the same for this and the Dockerfile_dev
FROM alpine as builder
ARG SERVICE=mqtt-device-service
WORKDIR /app
COPY  ${SERVICE} service
RUN mkdir logs

# ---------------------------------------------------------
FROM scratch as service
ARG APP_PORT=49982

# ARG variable substitution doesn't work with --chown below 19.03.0 :(
# https://github.com/moby/moby/issues/35018
COPY --from=builder --chown=2000:2000 /app/logs /logs
COPY --from=builder --chown=2000:2000 /app/service /
COPY LICENSE .
COPY cmd/res /res

USER 2000
ENV APP_PORT=$APP_PORT
EXPOSE $APP_PORT
ENTRYPOINT ["/service"]
CMD ["--registry=consul://edgex-core-consul:8500", "--profile=docker","--confdir=/res"]

