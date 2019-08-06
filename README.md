
# RSP Controller Device Service
Based on the Edgex Go MQTT Device Service service, modified to support multiple
topics and extract jsonrpc method field from RSP Controller messages.

## Requisite
* core-data
* core-metadata
* core-command

## Sending Commands to RSP Controller
To send commands from Edgex to Intel open source RSP Controller we can use some client such as POSTMAN [https://www.getpostman.com/].
 
Open POSTMAN or any similar tool and execute the following apis:

- Replace `localhost` in the below api with your respective server IP address if not running on localhost. This api is
used to find all the executable commands for a particular device (rsp-controller is the default name of the Intel open
source RSP Controller)
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

  
