#!/bin/bash
#
# Copyright (c) 2018
# Mainflux
#
# SPDX-License-Identifier: Apache-2.0
#

###
# Launches all EdgeX Go binaries (must be previously built).
#
# Expects that Consul and MongoDB are already installed and running.
#
###

DIR=$PWD
CMD=../cmd

function cleanup {
	pkill mqtt-device-service

cd $CMD
exec -a mqtt-device-service ./mqtt-device-service &
cd $DIR


trap cleanup EXIT

while : ; do sleep 1 ; done