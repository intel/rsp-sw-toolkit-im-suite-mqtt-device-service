
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
  * [Prerequisites](#prerequisites)
  * [Make Targets](#make-targets)
  * [Docker Usage](#docker-usage)
  * [Sending Commands to RSP Controller](#sending-commands-to-rsp-controller)
  
## Prerequisites

### EdgeX [Edinburgh Release](https://www.edgexfoundry.org/release-1-0-edinburgh/)
*   Must have EdgeX - [Core Services](https://docs.edgexfoundry.org/Ch-CoreServices.html) microservices installed and running.
### Intel® RSP Controller Application
*   Must have the Intel® RSP Controller Application [*Getting Started with Intel® RFID Sensor Platform (RSP) on Linux*](https://software.intel.com/en-us/getting-started-with-intel-rfid-sensor-platform-on-linux) installed and running.  This will allow for the RSP MQTT Device service to register the RSP Controller Application and the list of commands that are made available.

## Make Targets
The included [Makefile](Makefile) has some useful targets for building and 
testing the service. Here's a description of these targets:

- `$(SERVICE_NAME)` (default is `mqtt-device-service`): builds the service 
- `build`: alias for `$(SERVICE_NAME)` 
- `test`: runs the test suite with coverage 
- `clean`: deletes the service executable
- `image`: builds and tags a Docker image
- `clean-img` deletes the Docker image

## Docker Usage
You can use this service with Docker by adding it to your `docker-compose.yml`
and giving it network access to the EdgeX services and the MQTT broker. If the
EdgeX services are reachable on a network named `edgex-network` and the MQTT 
broker is reachable via `172.17.0.1`, here's one way of doing this:

```yaml
  mqtt-device-service:
    image: mqtt-device-service:latest
    networks:
        - edgex-network 
    extra_hosts:
      - "mosquitto-server:172.17.0.1"
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

  
