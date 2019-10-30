
# RSP MQTT Device Service
Based on the Edgex Go MQTT Device Service, the RSP MQTT Device Service is a specific connector for the Intel® RSP Controller Application to EdgeX. 

RSP MQTT Device Service:
*   Registers the Intel® RSP Controller Application device with the EdgeX platform
*   Sends commands from EdgeX's [Command](https://docs.edgexfoundry.org/Ch-Command.html) service to the Intel® RSP Controller Application
*   Sends responses from the Intel® RSP Controller Application to the EdgeX [Command](https://docs.edgexfoundry.org/Ch-Command.html) service
*   Sends RFID reads from an Intel® RSP Sensor to EdgeX Core Data service

To accomplish this, modifications were made to:
*   Add multiple topics support
*   Consume RSP Controller Application messages 
*   Send commands to RSP controller Application and receive responses

## Contents
  * [Make Targets](#make-targets)
  * [Building and Launching the MQTT Device Service with EdgeX](#building-and-launching-the-mqtt-device-service-with-edgeX)
    + [Prerequisites](#prerequisites)
    + [Getting the source code](#getting-the-source-code)
    + [Building and creating the docker image](#building-and-creating-the-docker-image)
    + [Adding to EdgeX](#adding-to-edgeX)
    + [Starting the services](#starting-the-services)
  * [Sending Commands to RSP Controller Application](#sending-commands-to-rsp-controller-application)

## Make Targets
The included [Makefile](Makefile) has some useful targets for building and 
testing the service. Here's a description of these targets:

- `$(SERVICE_NAME)` (default is `mqtt-device-service`): builds the service 
- `build`: alias for `$(SERVICE_NAME)` 
- `test`: runs the test suite with coverage 
- `clean`: deletes the service executable
- `image`: builds and tags a Docker image
- `clean-img` deletes the Docker image

## Building and Launching the MQTT Device Service with EdgeX

### Prerequisites
#### Make
```bash
sudo apt -y install make
```

#### Golang (1.12+)
*   [Install Instructions](https://golang.org/doc/install)

#### EdgeX - Edinburgh Release
*   Download the latest EdgeX Edinburgh docker-compose file [here](https://raw.githubusercontent.com/edgexfoundry/developer-scripts/master/releases/edinburgh/compose-files/docker-compose-edinburgh-no-secty-1.0.1.yml) and save this as docker-compose.yml in your local directory. This file contains everything you need to deploy EdgeX with docker.

#### Intel® RSP Controller Application
*   Must have the Intel® RSP Controller Application [*Getting Started with Intel® RFID Sensor Platform (RSP) on Linux*](https://software.intel.com/en-us/getting-started-with-intel-rfid-sensor-platform-on-linux) installed and running.  This will allow for the RSP MQTT Device service to register the RSP Controller Application and the list of commands that are made available.

*   :heavy_check_mark: If you installed the DOCKER version of the Intel® RSP Controller Application, go straight to the [Getting the source code](#getting-the-source-code) section.

*   :warning: If you installed the NATIVE version of the Intel® RSP Controller Application you will need the following:

##### Curl 
```bash
sudo apt -y install curl
```

##### Docker
```bash
apt -y install docker.io
```

##### Docker Compose
```bash
curl -L "https://github.com/docker/compose/releases/download/1.24.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose && sudo chmod a+x /usr/local/bin/docker-compose
```

### Getting the source code
```bash
git clone https://github.impcloud.net/RSP-Inventory-Suite/mqtt-device-service.git
```

### Building and creating the docker image
```bash
cd mqtt-device-service
```

```bash
make build image 
```

### Adding to EdgeX
1. To use this service with Docker go to the directory with the EdgeX `docker-compose.yml` file you downloaded in the [EdgeX prerequisites section](#EdgeX). 
2. Add the following code snippet to the DEVICE SERVICES section of the EdgeX `docker-compose.yml`.  This snippet also gives it network access to the EdgeX services and the MQTT broker. If the EdgeX services are reachable on a network named `edgex-network` (this is the default name in the EdgeX Edinburgh docker-compose.yml) and the MQTT broker is reachable via `172.17.0.1`. 

Section to add to the `docker-compose.yml` (remember spacing and alignment is important!):

```yaml
  mqtt-device-service:
    image: mqtt-device-service:latest
    networks:
        - edgex-network 
    extra_hosts:
      - "mosquitto-server:172.17.0.1"
    depends_on:
      - logging
```

### Starting the services
```bash
docker-compose up -d
```


## Sending Commands to RSP Controller Application
To send commands from Edgex to RSP Controller Application we can use some client such as [Postman](https://www.getpostman.com/).
 
Open POSTMAN or any similar tool and execute the following apis:

- Replace `localhost` in the below api with your respective server IP address if not running on localhost. This api is
used to find all the executable commands for a particular device (rsp-controller is the default name of the RSP Controller)
```
GET to http://localhost:48082/api/v1/device/name/rsp-controller
```
- If the GET request is successful a json response is received from which all the executable commands can be found

![GET device](docs/Command_list.png)

- The commands can be be sent by modifying the above api. For e.g. the below api is used to send a command known as
`behavior_get_all` 
```
GET to http://localhost:48082/api/v1/device/name/rsp-controller/command/behavior_get_all
```

- If the above request is successful a json response is received from which the RSP Controller response can be found in the
`value` field.

![GET command](docs/Response.png)

- Also GET command requests which requires only `device_id` as parameter are supported. For e.g. command below can be used 
to get bist_results of a sensor named `RSP-150000`. Be sure to replace `RSP-150000` with the one you need.
Since Edgex does not support GET requests with query parameters this is an alternate solution.
```
GET to http://localhost:48082/api/v1/device/name/RSP-150000/command/sensor_get_bist_results
```

  
