# 1. RTIO Demos

> English | [简体中文](./cn/rtio_demos.md)  
> The author's native language is Chinese. This document is translated using AI.

- [1. RTIO Demos](#1-rtio-demos)
  - [1.1. Enabling TLS for Device and RTIO](#11-enabling-tls-for-device-and-rtio)
  - [1.2. Observing the Device](#12-observing-the-device)
  - [1.3. Device Requests to the Server "Device Services"](#13-device-requests-to-the-server-device-services)
    - [1.3.1. Starting the RTIO Configuration Service](#131-starting-the-rtio-configuration-service)
    - [1.3.2. Starting the RTIO Server](#132-starting-the-rtio-server)
    - [1.3.3. Accessing Device Services](#133-accessing-device-services)
  - [1.4. Enabling Device Authentication for RTIO](#14-enabling-device-authentication-for-rtio)
    - [1.4.1. Enabling Device Authentication Service](#141-enabling-device-authentication-service)
    - [1.4.2. Starting the RTIO Server](#142-starting-the-rtio-server)
    - [1.4.3. Starting the Device](#143-starting-the-device)
  - [1.5. Enabling HTTPS and JWT Authentication for RTIO](#15-enabling-https-and-jwt-authentication-for-rtio)
    - [1.5.1. Obtaining Device Access JWT](#151-obtaining-device-access-jwt)
    - [1.5.2. Starting the RTIO Server](#152-starting-the-rtio-server)
    - [1.5.3. Starting the Device](#153-starting-the-device)
    - [1.5.4. HTTP Client Access](#154-http-client-access)

## 1.1. Enabling TLS for Device and RTIO

After a successful build, the `out` directory will contain the executable files, as shown below.

```sh
$ tree out/
out/
├── examples
│   ├── certificates
│   ├── simple_device
│   └── simple_device_tls
├── rtio
```

To run the RTIO Server, you can view the help with the `-h` parameter.

```sh
$ ./out/rtio -disable.deviceverify  \
             -disable.hubconfiger \
             -log.level=debug \
             -enable.hub.tls \
             -tls.hub.certfile=./out/examples/certificates/demo_server.crt \
             -tls.hub.keyfile=./out/examples/certificates/demo_server.key

INF cmd/rtio/rtio.go:83 > rtio starting ...
INF internal/devicehub/server/devicetcp/devicetls.go:123 > TLS access enabled
```

In another terminal, simulate a device. The following log indicates a successful connection to the RTIO server.

```sh
$ ./out/examples/simple_device
INF internal/devicehub/client/devicesession/devicesession.go:660 > serving device_ip=127.0.0.1:47032 deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09
INF internal/devicehub/client/devicesession/devicesession.go:601 > verify pass
```

Open another terminal and use `curl` to request the device's URI `/rainbow` through the RTIO service, sending the string "hello" to the device, which responds with "world".

```sh
$ curl http://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b09 -d '{"method":"copost", "uri":"/rainbow","id":12667,"data":"aGVsbG8="}'
{"id":12667,"fid":0,"code":"OK","data":"d29ybGQ="}
```

Here, "aGVsbG8=" is the base64 encoding of "hello", and "d29ybGQ=" is the base64 encoding of "world". You can use the following commands to encode and decode in the terminal.

```sh
$ echo -n "hello" | base64       # Encode
aGVsbG8=
$ echo -n "d29ybGQ=" | base64 -d # Decode
world
```

## 1.2. Observing the Device

After a successful build, the `out` directory will again contain the executable files, as shown below.

```sh
$ tree out/
out/
├── examples
│   ├── certificates
│   ├── simple_device
│   └── simple_device_obget
├── rtio
```

To run the RTIO Server, you can view the help with the `-h` parameter.

```sh
$ ./out/rtio -disable.deviceverify  \
             -disable.hubconfiger \
             -log.level=debug \
             -enable.hub.tls \
             -tls.hub.certfile=./out/examples/certificates/demo_server.crt \
             -tls.hub.keyfile=./out/examples/certificates/demo_server.key

INF cmd/rtio/rtio.go:83 > rtio starting ...
INF internal/devicehub/server/devicetcp/devicetls.go:123 > TLS access enabled
```

In another terminal, simulate a device. The following log indicates a successful connection to the RTIO server.

```sh
$ ./out/examples/simple_device_obget
INF internal/devicehub/client/devicesession/devicesession.go:660 > serving device_ip=127.0.0.1:47032 deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09
INF internal/devicehub/client/devicesession/devicesession.go:601 > verify pass
```

Open another terminal and use `curl` to send a request through the RTIO service to the device's URI `/rainbow`, sending the string "hello" to the device. The device responds with "world0" to "world9".

```sh
$ curl http://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b09 -d '{"method":"obget", "uri":"/rainbow","id":12667,"data":"aGVsbG8="}'
{"id":12667,"fid":0,"code":"CONTINUE","data":"d29ybGQhIDA="}
{"id":12667,"fid":1,"code":"CONTINUE","data":"d29ybGQhIDE="}
{"id":12667,"fid":2,"code":"CONTINUE","data":"d29ybGQhIDI="}
{"id":12667,"fid":3,"code":"CONTINUE","data":"d29ybGQhIDM="}
{"id":12667,"fid":4,"code":"CONTINUE","data":"d29ybGQhIDQ="}
{"id":12667,"fid":5,"code":"CONTINUE","data":"d29ybGQhIDU="}
{"id":12667,"fid":6,"code":"CONTINUE","data":"d29ybGQhIDY="}
{"id":12667,"fid":7,"code":"CONTINUE","data":"d29ybGQhIDc="}
{"id":12667,"fid":8,"code":"CONTINUE","data":"d29ybGQhIDg="}
{"id":12667,"fid":9,"code":"CONTINUE","data":"d29ybGQhIDk="}
{"id":12667,"fid":10,"code":"TERMINATE","data":""}
```

You can use the following command in the terminal to encode and decode the base64 `data`.

```sh
$ echo -n "hello" | base64           # Encode
aGVsbG8=
$ echo -n "d29ybGQhIDA=" | base64 -d # Decode
world! 0
$ echo -n "d29ybGQhIDE=" | base64 -d # Decode
world! 1

# ...

$ echo -n "d29ybGQhIDk=" | base64 -d # Decode
world! 9
```

## 1.3. Device Requests to the Server "Device Services"

The device requests the backend service, referred to here as the device service, proxied by RTIO through the CoPOST method. The dynamic updates to this service are achieved through the configuration service, hubconfiger.

For the implementation of the configuration service, please refer to the [Configuration Service Interface](./http_hubconfiger.md) and the [Configuration Service Example Code](../examples/hubconfiger/hubconfiger.go).

### 1.3.1. Starting the RTIO Configuration Service

```sh
$ ./out/examples/hubconfiger
INF hubconfiger.go:150 > hubconfiger started httpaddr=0.0.0.0:17317
INF hubconfiger.go:151 > hubconfiger URL, replace the 0.0.0.0 address with the external address.  url=http://0.0.0.0:17317/hubconfiger
```

### 1.3.2. Starting the RTIO Server

The logs will show the loaded device services.

```sh
$ ./out/rtio -disable.deviceverify -backend.hubconfiger=http://localhost:17317/hubconfiger -log.level=debug
DBG cmd/rtio/rtio.go:71 > Config:backend.hubconfiger=http://localhost:17317/hubconfiger
INF cmd/rtio/rtio.go:81 > rtio starting ...
# ...
DBG configer.go:156 > Config: backend.hubconfiger=http://localhost:17317/hubconfiger
DBG configer.go:156 > Config: deviceservice.a796d057=http://localhost:17517/deviceservice/aa/bb
DBG configer.go:156 > Config: deviceservice.c98ad180=http://localhost:17517/deviceservice/aa/cc
DBG configer.go:156 > Config: deviceservice.18afd2e4=http://localhost:17518/deviceservice/aa/dd
```

### 1.3.3. Accessing Device Services

The device requests the URI "/aa/bb" service every 5 seconds, as shown in the [example code](../examples/simple_device_copost_to_server/simple_device_copost_to_server.go).

```sh
$ ./out/examples/simple_device_copost_to_server
2024-09-16 13:02:33.266 INFdevicesession.go:772 > serving device_ip=127.0.0.1:41548 deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09

# after 5 seconds
2024/09/16 13:02:38 resp=deviceservice: respone with bb
```

The device service logs are as follows.

```sh
INF deviceservice.go:113 > req data data="test for device post" deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09 id=182274021
INF deviceservice.go:117 > resp data data="deviceservice: respone with bb" deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09 id=182274021
```

## 1.4. Enabling Device Authentication for RTIO

### 1.4.1. Enabling Device Authentication Service

Below is a demo of the authentication service. You can refer to the [device authentication service interface](./http_deviceverifier.md) for implementation details.

```sh
$ ./out/examples/deviceverifier -h
Usage of ./out/examples/deviceverifier:
  -http.addr string
        address for http connection (default "0.0.0.0:17217")
$ ./out/examples/deviceverifier
INF deviceverifier.go:127 > deviceverifier started httpaddr=0.0.0.0:17217
INF deviceverifier.go:129 > deviceverifier URL, replace the 0.0.0.0 address with the external address.  url=http://0.0.0.0:17217/deviceverifier
```

### 1.4.2. Starting the RTIO Server

Remove the `-disable.deviceverify` option and add the `-backend.deviceverifier=localhost:17217` option.

```sh
$ ./out/rtio -backend.deviceverifier=http://localhost:17217/deviceverifier -disable.hubconfiger -log.level=debug
DBG cmd/rtio/rtio.go:73 > Config:backend.deviceverifier=http://localhost:17217/deviceverifier
# ...
INF cmd/rtio/rtio.go:83 > rtio starting ...
```

### 1.4.3. Starting the Device

The default `DeviceID` and `DeviceSecret` are already set in the authentication service. [Refer to the code](../examples/deviceverifier/deviceverifier.go).

```sh
$ ./out/examples/simple_device -id="cfa09baa-4913-4ad7-a936-3e26f9671b09" -secret="mb6bgso4EChvyzA05thF9+wH"
INF devicesession.go:772 > serving device_ip=127.0.0.1:33456 deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09
# ...
INF devicesession.go:713 > verify pass
```

When using an incorrect `DeviceSecret`, the device cannot connect to the RTIO server.

```sh
$ ./out/examples/simple_device -id="cfa09baa-4913-4ad7-a936-3e26f9671b09" -secret="mb6bgso4EChvyzA05thF9+xx"
INF devicesession.go:772 > serving device_ip=127.0.0.1:39646 deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09
ERR devicesession.go:715 > verify fail resp.code=3
ERR devicesession.go:805 > device done when error error=ErrVerifyFailed
```

## 1.5. Enabling HTTPS and JWT Authentication for RTIO

### 1.5.1. Obtaining Device Access JWT

You can refer to the [JWT issuance example service](./http_jwtissuer.md). For example, you can obtain the JWT as follows.

```sh
$ curl http://localhost:17019/jwtissuer -d '{"method":"jwtissuer", "id":12667,"deviceid":"cfa09baa-4913-4ad7-a936-3e26f9671b09", "expires": 86400}'
{"id":12667,"code":"OK","jwt":"eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjY0NjQ1MzAsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0zZTI2Zjk2NzFiMDkifQ.sM80Zu3nMsmahzmLpmSC4GlsI8G0xKbnIk8kZIJvLH9IbadEc3sOM3tSb7m_L_ZY2eWy4Ipl8EiYS7t9y_NmCA"}
```

### 1.5.2. Starting the RTIO Server

```sh
$ ./out/rtio \
-disable.deviceverify  \
-disable.hubconfiger \
-log.level=debug \
-enable.hub.tls \
-tls.hub.certfile=./out/examples/certificates/demo_server.crt \
-tls.hub.keyfile=./out/examples/certificates/demo_server.key \
-enable.https \
-https.certfile=./out/examples/certificates/demo_server.crt \
-https.keyfile=./out/examples/certificates/demo_server.key \
-enable.jwt \
-jwt.ed25519=./out/examples/certificates/ed25519.public

INF cmd/rtio/rtio.go:83 > rtio starting ...
INF internal/devicehub/server/devicetcp/devicetls.go:123 > TLS access enabled
INF internal/httpaccess/server/httpgw/httpgw.go:464 > gateway started with TLS gwaddr=0.0.0.0:17917
```

### 1.5.3. Starting the Device

```sh
./out/examples/simple_device_tls -with-ca ./out/examples/certificates/ca.crt
```

### 1.5.4. HTTP Client Access

First, assign the JWT to a variable `TOKEN`, then specify it in the request header using `-H`.

```sh
$ export TOKEN="eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjY0NjQ1MzAsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0zZTI2Zjk2NzFiMDkifQ.sM80Zu3nMsmahzmLpmSC4GlsI8G0xKbnIk8kZIJvLH9IbadEc3sOM3tSb7m_L_ZY2eWy4Ipl8EiYS7t9y_NmCA"
$ curl https://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b09 -H "Authorization:Bearer $TOKEN" -d '{"method":"copost", "uri":"/rainbow","id":12667,"data":"aGVsbG8="}' --cacert ./out/examples/certificates/ca.crt
{"id":12667,"fid":0,"code":"OK","data":"d29ybGQ="}
```

The detailed logs are as follows:

```sh
$ curl https://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b09 -H "Authorization:Bearer $TOKEN" -d '{"method":"copost", "uri":"/rainbow","id":12667,"data":"aGVsbG8="}' --cacert ./out/examples/certificates/ca.crt  -v
> POST /cfa09baa-4913-4ad7-a936-3e26f9671b09 HTTP/2
> Host: localhost:17917
> user-agent: curl/7.81.0
> accept: */*
> authorization:Bearer eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjY0NjQ1MzAsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0zZTI2Zjk2NzFiMDkifQ.sM80Zu3nMsmahzmLpmSC4GlsI8G0xKbnIk8kZIJvLH9IbadEc3sOM3tSb7m_L_ZY2eWy4Ipl8EiYS7t9y_NmCA
> content-length: 66
> content-type: application/x-www-form-urlencoded
>

< HTTP/2 200
< access-control-allow-headers: Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization
< access-control-allow-methods: POST, GET, OPTIONS, PUT, DELETE
< access-control-allow-origin: *
< content-type: text/plain; charset=utf-8
< content-length: 50
< date: Sun, 15 Sep 2024 06:10:07 GMT
<
{"id":12667,"fid":0,"code":"OK","data":"d29ybGQ="}
```
