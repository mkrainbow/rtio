# 1. RTIO（Real Time Input Output Service for IoT）

> English | [简体中文](./README-CN.md)  
> The author's native language is Chinese. This document is translated using AI.

- [1. RTIO（Real Time Input Output Service for IoT）](#1-rtioreal-time-input-output-service-for-iot)
  - [1.1. Goals](#11-goals)
  - [1.2. Theory](#12-theory)
  - [1.3. Features](#13-features)
  - [1.4. Limitations](#14-limitations)
  - [1.5. Comparison with MQTT](#15-comparison-with-mqtt)
  - [1.6. Quick Start](#16-quick-start)
    - [1.6.1. Build and Run](#161-build-and-run)
    - [1.6.2. Device-Side Code](#162-device-side-code)
  - [1.7. Device SDK](#17-device-sdk)
    - [1.7.1. Golang](#171-golang)
    - [1.7.2. C](#172-c)
  - [1.8. More](#18-more)

---

## 1.1. Goals

With just a few lines of key code, the "device capabilities" can be mapped to the cloud or edge as a URI, allowing direct access and control of the device via HTTP.

## 1.2. Theory

```text
                                                                 
   Device:PRINTER-001           Native │ Could or Edge                 
  ┌───────────────────────┐            │             ┌────────────┐ 
  │                       │            │             │            │ 
  │ /printer/action       │            │             │            │ 
  │ /printer/status       │  tcp/tls   │             │            │ 
  │                       │ ───────────┼───────────► │            │ 
  │                       │            │             │            │ 
  └───────────────────────┘            │             │            │ 
                                       │             │            │ 
   HTTPClient                          │             │            │ 
  ┌───────────────────────┐            │             │            │ 
  │ post $RTIO/PRINTER-001│            │             │            │ 
  │ req:                  │            │             │    rtio    │ 
  │ {                     │            │             │            │ 
  │   uri: /printer/action│            │             │            │ 
  │   data: cmd=start     │            │             │            │ 
  │ }                     │  http/https│             │            │ 
  │                       │ ───────────┼───────────► │            │ 
  │                       │ ◄─ ─ ─ ─ ─ ┼ ─ ─ ─ ─ ─ ─ │            │ 
  │ resp:                 │            │             │            │ 
  │ {                     │            │             │            │ 
  │   code: ok            │            │             │            │ 
  │   data: starting      │            │             │            │ 
  │ }                     │            │             │            │ 
  └───────────────────────┘            │             └────────────┘ 
   Phone/WEB/PC...                    

```

The above is a schematic diagram.

- The device `PRINTER-001` establishes a long connection with the RTIO server.
- The HTTPClient sends a request to the device's URI `/printer/action`.
- The RTIO forwards the request to the device and returns the device processing result back to the HTTPClient via an HTTP response.

From the caller's perspective, the "device capabilities" are mapped to the cloud through the RTIO service, without needing to worry about implementation or communication details.

## 1.3. Features

The device caller is not limited to mobile devices, PCs, or backend services, and will be collectively referred to as the "caller".

- The caller directly uses HTTP to invoke the device, receiving the device's processing result within a specified time limit; otherwise, a timeout error is reported.
- The caller can observe the device in real time via HTTP (without polling), where the device typically provides a continuous processing update.
- The device can initiate requests to the cloud URI using `CoPOST` (similar to an `HTTP-POST` request).
- The device URI is transmitted using a `hash digest` (only 4 bytes) to reduce bandwidth waste from frequent URI transmissions.
- HTTP access supports JWT authentication.
- Supports TLS and HTTPS.
- A single RTIO node supports millions of connections; [Previous Stress Report](https://github.com/guowenhe/rtio/blob/master/docs/stress_report.md).
- The RTIO implementation is a single executable file (written in Golang), making it easy to deploy on cloud or edge environments.

## 1.4. Limitations

- The current version primarily focuses on `control command` communication. In a single call, both the request payload and the response payload have a maximum length of 504 bytes. For more details, refer to the [Device Access Protocol](./docs/device_access_protocol.md).

## 1.5. Comparison with MQTT

MQTT is a publish-subscribe model, which is more suitable for many-to-many communication.

RTIO is a point-to-point communication model, making it better suited for remote control scenarios, such as mobile devices controlling equipment. Below is a detailed comparison.

| Feature                     | MQTT                    | RTIO                             |
|:---------------------------|:-------------------------|:-----------------------------------|
| Communication Model         | Publish-Subscribe Model   | Point-to-Point, REST-Like Model¹, supports Observing Mode² |
| Client SDK Integration      | Required                  | Not required, uses HTTP protocol    |
| Remote Control Implementation Difficulty | High         | Low³                       |
| Lightweight and Efficient Protocol | Yes                 | Yes⁴                       |
| Bi-directional Communication | Yes                      | Yes                                 |
| Millions of Connections     | Supported                 | Supported                           |
| Reliable Message Delivery    | Supported                 | Supported⁵                  |
| Unreliable Network          | Supported                 | Supported⁶                 |
| Secure Communication (TLS)  | Supported                 | Supported                           |
| JWT Authentication         | Not supported             | Supported⁷                           |  

**Notes:**

1. REST-Like refers to a model similar to RESTful (widely used on the internet), but not relying on HTTP as the underlying protocol. It provides `CoPOST` (Constrained-Post, similar to HTTP-POST) and `ObGET` (Observe-GET, for observer mode) methods, identifying resources or capabilities through `URI`.
2. The observing mode is similar to MQTT message subscription but is more flexible, allowing observations and cancellation at any time. The caller still uses the HTTP protocol without additional operations (e.g., "subscribe").
3. The point-to-point model of RTIO is simpler; unlike MQTT, it does not require "subscribe" and "publish" processes. In point-to-point communication scenarios, such as remotely controlling devices, RTIO can reduce interaction times by more than half compared to MQTT, and there is no topic coupling (typically, MQTT requires defining pairs of topics for requests and responses: `*_req` and `*_resp`). For more details, refer to the [FAQ](./docs/rtio_faq.md).
4. Supports constrained devices, typically those that can run on devices with RAM between tens of KB and a hundred KB. RTIO does not need to transmit the URI with each communication (unlike topics in MQTT), resulting in higher communication efficiency.
5. RTIO does not have QOS; reliability depends on TCP. If a message cannot be delivered within the time limit, it will report an error directly to the caller.
6. The RTIO client will reconnect after a network disconnection.
7. HTTP callers can authenticate with JWT. Only requests with valid JWTs can access RTIO services to initiate calls to devices.  

**References:**

- [MQTT official](https://mqtt.org/)
  
## 1.6. Quick Start

### 1.6.1. Build and Run

You need to have Golang (version 1.21 or above) and make tools installed.

```sh
git clone https://github.com/mkrainbow/rtio.git
cd rtio
make
```

After a successful build, executable files will be generated in the `out` directory, with the main files as follows.

```sh
$ tree out/
out/
├── examples
│   ├── certificates
│   ├── simple_device
│   └── simple_device_tls
├── rtio
```

To run the RTIO server, you can check the help information using the `-h` parameter.

```sh
$ ./out/rtio -disable.deviceverify -disable.hubconfiger -log.level=info
INF cmd/rtio/rtio.go:83 > rtio starting ...
```

In another terminal, simulate the device. The following log indicates a successful connection to the RTIO service.

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

Here, "aGVsbG8=" is the base64 encoding of "hello", and "d29ybGQ=" is the base64 encoding of "world". You can encode and decode in the terminal using the following commands.

```sh
$ echo -n "hello" | base64       # Encode
aGVsbG8=
$ echo -n "d29ybGQ=" | base64 -d # Decode
world
```

### 1.6.2. Device-Side Code

Below is the Golang implementation for the device side.

```go
import (
    "github.com/mkrainbow/rtio-device-sdk-go/rtio"
)

func main() {

    // Connect to rtio service.
    session, err := rtio.Connect(context.Background(), *deviceID, *deviceSecret, *serverAddr)

    // ...
    
    // Register handler for URI.
    session.RegisterCoPostHandler("/rainbow", func(req []byte) ([]byte, error) {
        log.Printf("received [%s] and reply [world]", string(req))
        return []byte("world"), nil
    })

    // Session serve in the background.
    session.Serve(context.Background())

    // Do other things.
    time.Sleep(time.Hour * 8760)
}
```

## 1.7. Device SDK

### 1.7.1. Golang

RTIO device-side SDK, Golang version:  

- rtio-device-sdk-go - <https://github.com/mkrainbow/rtio-device-sdk-go>

Typically suitable for devices running full Linux, such as ARM32-bit or higher single-board computers.

### 1.7.2. C

RTIO device-side SDK, C language version:  

- rtio-device-sdk-c - <https://github.com/mkrainbow/rtio-device-sdk-c>

Suitable for resource-constrained devices, such as those running on real-time operating systems like FreeRTOS.

## 1.8. More

- [More Demos](./docs/rtio_demos.md)
- [Device Access Protocol](./docs/device_access_protocol.md)
- [HTTP API](./docs/http_access_protocol.md)
- [FQA](./docs/rtio_faq.md)
