# 1. FAQ

> English | [简体中文](./cn/rtio_faq.md)  
> The author's native language is Chinese. This document is translated using AI.

- [1. FAQ](#1-faq)
  - [1.1. About RTIO](#11-about-rtio)
    - [1.1.1. Why Was RTIO Created?](#111-why-was-rtio-created)
    - [1.1.2. How Does RTIO Work?](#112-how-does-rtio-work)
    - [1.1.3. When to Choose Point-to-Point Communication?](#113-when-to-choose-point-to-point-communication)
  - [1.2. Complexity and Efficiency Evaluation](#12-complexity-and-efficiency-evaluation)
    - [1.2.1. IoT Application Maintenance Complexity x 1/100](#121-iot-application-maintenance-complexity-x-1100)
    - [1.2.2. IoT Application Development Speed x 10](#122-iot-application-development-speed-x-10)
  - [1.3. Comparison with MQTT Interaction](#13-comparison-with-mqtt-interaction)
    - [1.3.1. Controlling IoT Devices](#131-controlling-iot-devices)
    - [1.3.2. Requesting Data from IoT Devices to the Cloud](#132-requesting-data-from-iot-devices-to-the-cloud)
    - [1.3.3. Observer Pattern](#133-observer-pattern)
  - [1.4. Comparison Summary with MQTT](#14-comparison-summary-with-mqtt)

## 1.1. About RTIO  

### 1.1.1. Why Was RTIO Created?  

RTIO was designed to simplify IoT application development for **point-to-point remote communication**.  

IoT devices communicate with the RTIO server in a manner similar to MQTT but use TCP-based long connections. RTIO enables the cloud to push messages to devices while avoiding the complexity of using MQTT's publish-subscribe model for point-to-point communication. It integrates the following ideas:  

- **Synchronous results:** Each request returns the device's execution result or a timeout.  
- **Decoupled versions:** Devices, calling endpoints (e.g., mobile apps), and application servers are independent and can maintain their own versions.  
- **Device as a service:** Devices act as service providers, offering a development experience similar to web services, enhancing efficiency.  
- **Cross-platform integration:** RTIO bridges the technological gap between mobile, server, and embedded systems. Each platform uses its own tech stack while working toward the same IoT application.  

### 1.1.2. How Does RTIO Work?  

- **HTTP-Based Communication:** Calling endpoints (e.g., mobile apps) use HTTP to communicate directly with the RTIO service, invoking device actions without SDK integration. Tools like `curl` or `postman` can validate device functionality.  
- **Resource URI:** Devices expose resources or capabilities via URIs, such as `/printer/action` or `/printer/action/v2`, simplifying categorization and iteration of device capabilities.  
- **Point-to-Point Efficiency:** RTIO supports point-to-point remote communication, carrying both request and response data in a single interaction, reducing communication overhead by more than half compared to MQTT.  
- **Device SDK:** RTIO provides a lightweight, portable SDK for devices. Developers register `handler functions` to specific URIs, allowing easy implementation of device-side logic.  

### 1.1.3. When to Choose Point-to-Point Communication?  

**Point-to-Point Communication vs. Publish-Subscribe:**  
Point-to-point communication involves two devices interacting directly, whereas publish-subscribe allows multiple devices to interact.  

For **User-to-Machine (U2M)** scenarios—where users (e.g., via smartphones) control IoT devices—point-to-point communication is simpler and more efficient. While a publish-subscribe model can also achieve U2M communication, it introduces unnecessary complexity for these cases[^1].  

**One User Controlling Multiple Devices:**  
Even when a user controls multiple IoT devices, point-to-point communication is suitable if there’s no complex interaction between devices. In such cases, adding batch control logic suffices, and point-to-point communication remains simpler than a publish-subscribe model.  

**Cross-Network Communication:**  
RTIO’s proxy service enables point-to-point remote communication even when the user (e.g., a smartphone) and the IoT device are not on the same local network.  

## 1.2. Complexity and Efficiency Evaluation  

### 1.2.1. IoT Application Maintenance Complexity x 1/100  

This section compares IoT applications developed with RTIO and MQTT, focusing on remote control scenarios, specifically their maintenance and version iteration complexities.  

**Key factors defining IoT applications:**

- **N1:** Number of coexisting versions of the user-side application (APP).  
- **N2:** Number of OTA (Over-The-Air) versions for IoT devices (Device).  
- **N3:** Number of IoT device types, as an APP typically supports multiple devices.  
- **N4:** Device scale (10^N).  

**Effective formula:** Coupled factors are multiplied, while uncoupled factors are summed. In MQTT, a pair of `*_Req` and `*_Resp` topics is usually defined, with publish and subscribe on different ends, creating coupling[^1]. RTIO resources are described in the form of URIs, where adding versions (e.g., v1/v2) on the device side does not create coupling with the user side.  

- **MQTT Complexity:** N1 x N2 x N3 x N4 = O(N^4)  
- **RTIO Complexity:** (N1 + N2 + N3) x N4 = O(N^2)  

For a clearer demonstration, assume N = 10. The resulting complexity is 1/100 of the traditional MQTT-based approach[^2].  

[^1]: Reference: See the [Comparison with MQTT Interactions](#13-comparison-with-mqtt-interaction) section.  
[^2]: This is a non-rigorous model intended to show that RTIO reduces IoT application maintenance complexity. For reference only.  

### 1.2.2. IoT Application Development Speed x 10  

This section compares the development speed of IoT applications using RTIO and MQTT, focusing on remote control scenarios.  

1. In a C-language IoT device-side demo, RTIO’s code is one-tenth the size of the AWS MQTT demo (122 lines vs. 1500+ lines)[^3]. Key RTIO code spans just a few lines.  
2. On the user side (mobile/PC/WEB), HTTP can directly communicate with RTIO services without requiring SDK integration.  
3. RTIO reduces interaction volume between endpoints by over half compared to MQTT[^1].  
4. Tools like `curl` can verify device interfaces via RTIO services, decoupling endpoint debugging.  
5. RTIO is typically deployed within the same LAN as IoT application services, minimizing authentication requirements and simplifying interfaces for efficiency.  
6. RTIO focuses on IoT remote communication with simpler functionality and lower usage barriers. In contrast to the hundreds of AWS IoT Core documents[^4], RTIO requires fewer than 10.  

**Conclusion:** These factors collectively estimate that RTIO improves IoT application development speed by a factor of 10[^5].  

[^3]: Reference: [aws-iot-device-sdk-embedded-C](https://github.com/aws/aws-iot-device-sdk-embedded-C) project’s [mqtt_demo_basic_tls.c](https://github.com/aws/aws-iot-device-sdk-embedded-C/blob/main/demos/mqtt/mqtt_demo_basic_tls/mqtt_demo_basic_tls.c).  
[^4]: Reference: [What is AWS IoT](https://docs.aws.amazon.com/iot/latest/developerguide/what-is-aws-iot.html).  
[^5]: This is a non-rigorous model intended to highlight the improvement in IoT application development speed with RTIO. For reference only.

## 1.3. Comparison with MQTT Interaction  

### 1.3.1. Controlling IoT Devices  

Comparing RTIO and MQTT interaction processes for remote control scenarios. 

User controls a device via a mobile app, RTIO interaction.

```text
    +--------+         +------------+           +--------+
    |  app   |         |    rtio    |           | device |
    +--------+         +------------+           +--------+
        |                     |                      |
        | user power on       |                      |
        |-----+               |                      |
        |     |               |                      |
        |<----+               |                      |
        |                     |                      |
        | HTTP-POST           |                      |
        | $device/led_power   |                      |
        |-------------------->|                      |
        |                     |                      |
        |                     | CoPOST /led_power    |
        |                     |--------------------->|
        |                     |                      |
        |                     |                      |-----+
        |                     |     led power action |     |
        |                     |                      |<----+
        |                     |   led_power status   |
        |                     |<---------------------|
        |                     |                      |
        |    led_power status |                      |
        |<--------------------|                      |  
        |                     |                      |
    +--------+         +------------+           +--------+
    |  app   |         |    rtio    |           | device |
    +--------+         +------------+           +--------+
```  

User controls a device via a mobile app, MQTT interaction.

```text
     +--------+         +------------+           +--------+
    |  app   |         |   broker   |           | device |
    +--------+         +------------+           +--------+
        |                     |                      |
        |                     | SUBSCRIBE            |
        |                     | $diviceid_led_power_req
        |                     |<---------------------|
        |                     | SUBACK               |
        |                     |--------------------->|
        | SUBSCRIBE           |                      |
        | $diviceid_led_power_resp                   |
        |-------------------->|                      |
        | SUBACK              |                      |
        |<--------------------|                      |
        |                     |                      |
        |                     |                      |
        | user power on       |                      |
        |-----+               |                      |
        |     |               |                      |
        |<----+               |                      |
        |                     |                      |
        | PUBLISH             |                      |
        | $diviceid_led_power_req                    |
        |-------------------->|                      |
        | PUBACK              |                      |
        |<--------------------|                      |
      (PUBACK:means brocker successfully received not device)
        |                     |                      |
        |                     | PUBLISH              |
        |                     | $diviceid_led_power_req
        |                     |--------------------->|
        |                     | PUBACK               |
        |                     |<---------------------|
        |                     |                      |-----+
        |                     |     led power action |     |
        |                     |                      |<----+
        |                     |                      |
        |                     | PUBLISH              |  
        |                     | $diviceid_led_power_resp
        |                     |<---------------------|
        |                     | PUBACK               |
        |                     |--------------------->|
        |                     |                      |
        | PUBLISH             |                      |
        | $diviceid_led_power_resp                   |
        |<--------------------|                      |
        | PUBACK              |                      |
        |-------------------->|                      |
        |                     |                      |
    +--------+         +------------+           +--------+
    |  app   |         |   broker   |           | device |
    +--------+         +------------+           +--------+
```

Conclusion:  
- The first diagram, based on RTIO, involves 4 interactions between endpoints.  
- The second diagram, based on MQTT, involves 12 interactions between endpoints.  

Suppose the new version adds the `led_power_b` feature.  
- Using RTIO, the `led_power_b` URI and handler can be registered on the device. If the new app version calls the old device version, the device will immediately return a "resource not found" error (which can prompt the user to update the device).  
- If using MQTT, the device side needs to subscribe to the `led_power_b_req` topic, while the application side needs to subscribe to the `led_power_b_resp` topic. These two topics are coupled on both ends. When a new version of the application interacts with an older version of the device, the older device won't subscribe to the `led_power_b_req` topic or publish messages to the `led_power_b_resp` topic. As a result, there will be no response. Typically, this requires the device side to be upgraded together.

### 1.3.2. Requesting Data from IoT Devices to the Cloud  

Requesting the version list, RTIO interaction.

```text
    +--------+         +------------+          +----------+
    | device |         |    rtio    |          |app-server|
    +--------+         +------------+          +----------+
        |                     |                      |
        |     auto discover resource and registry    |
        |                     |-----+                |
        |                     |     |                |
        |                     |<----+                |
        |                     |                      |
        | CoPOST              |                      |
        | /resource/versions  |                      |
        |-------------------->|                      |
        |                     | HTTP-POST            |
        |                     | /resource/versions   |
        |                     |--------------------->|
        |                     |                      |
        |                     |                      |-----+
        |                     |      query versioins |     |
        |                     |                      |<----+
        |                     | versions             |
        |                     |<---------------------|
        |                     |                      |
        |versions             |                      |
        |<--------------------|                      |
        |                     |                      |
    +--------+         +------------+          +----------+
    | device |         |    rtio    |          |app-server|
    +--------+         +------------+          +----------+
```

Requesting the version list, MQTT interaction.

```text 
    +--------+         +------------+          +----------+
    | device |         |   broker   |          |app-server|
    +--------+         +------------+          +----------+
        |                     |                      |
        |                     | SUBSCRIBE            |
        |                     | $deviceid_get_versions_req
        |                     |<---------------------|
        |                     | SUBACK               |
        |                     |--------------------->|
        | SUBSCRIBE           |                      |
        | $deviceid_get_versions_resp                |
        |-------------------->|                      |
        | SUBACK              |                      |
        |<--------------------|                      |
        |                     |                      |
        |                     |                      |
        | PUBLISH             |                      |
        | $deviceid_get_versions_req                 |
        |-------------------->|                      |
        | PUBACK              |                      |
        |<--------------------|                      |
        |                     |                      |
        |                     | PUBLISH              |
        |                     | $deviceid_get_versions_req
        |                     |--------------------->|
        |                     | PUBACK               |
        |                     |<---------------------|
        |                     |                      |-----+
        |                     |      query versioins |     |
        |                     |                      |<----+
        |                     |                      |
        |                     | PUBLISH              |  
        |                     | $deviceid_get_versions_resp
        |                     |<---------------------|
        |                     | PUBACK               |
        |                     |--------------------->|
        |                     |                      |
        | PUBLISH             |                      |
        | $deviceid_get_versions_resp                |
        |<--------------------|                      |
        | PUBACK              |                      |
        |-------------------->|                      |
        |                     |                      |
    +--------+         +------------+          +----------+
    | device |         |   broker   |          |app-server|
    +--------+         +------------+          +----------+
```

Conclusion:  
- The first diagram, based on RTIO, involves 4 interactions between endpoints.  
- The second diagram, based on MQTT, involves 12 interactions between endpoints.  
  
### 1.3.3. Observer Pattern  

Refer to the following interaction diagram. Click to expand.  

The user 'observes' the device processing progress via a mobile app, RTIO interaction.

```text
    +--------+         +------------+           +--------+
    |  app   |         |    rtio    |           | device |
    +--------+         +------------+           +--------+
        |                     |                      |
        |                     |                      |
        | HTTP-GET            |                      |
        | $diviceid/progress  |                      |
        |-------------------->|                      |
        |                     |                      |
        |                     | ObGET /progress      |
        |                     |--------------------->|
        |                     |  20%                 |
        |                     |<---------------------|
        | 20%                 |                      |
        |<--------------------|                      |
        |                     |  30%                 |
        |                     |<---------------------|
        | 30%                 |                      |
        |<--------------------|                      |
        |                     |                      |
     (The observation can be terminated by device or app)
        |                     |                      |
        |                     |  TERMINAT            |
        |                     |<---------------------|
        | TERMINAT            |                      |
        |<--------------------|                      |
        |                     |                      |
    +--------+         +------------+           +--------+
    |  app   |         |    rtio    |           | device |
    +--------+         +------------+           +--------+
```

The observer mode is similar to MQTT message subscriptions but is more flexible. Observations can be initiated and canceled at any time. The caller continues to use the HTTP protocol without requiring additional actions such as subscriptions.  

MQTT does not have this mode.

## 1.4. Comparison Summary with MQTT  

MQTT is a lightweight and efficient communication protocol suitable for constrained devices. The design of the RTIO communication protocol has also been inspired by MQTT. Below is a comparison of the two:  

- MQTT uses a publish-subscribe model, which is more suitable for many-to-many communication.
- RTIO employs a point-to-point communication model, making it more appropriate for scenarios like controlling devices remotely through a mobile app. Detailed comparisons are as follows:  

| Aspect            | MQTT         | RTIO                                   |
|:------------------|:-------------|:---------------------------------------|
| Communication Model | Publish-subscribe model | Point-to-Point, REST-Like model[^6], supports Observer Mode[^7] |
| Client SDK Integration | Required      | Not required, uses HTTP protocol        |
| Remote Control Complexity | Complex      | Simple[^8]                              |
| Lightweight and Efficient | Yes          | Yes[^9]                                 |
| Bidirectional Communication | Yes         | Yes                                     |
| Supports Millions of Connections | Yes    | Yes                                     |
| Reliable Message Transmission | Yes       | Yes[^10]                                |
| Unreliable Networks | Supported   | Supported[^11]                           |
| Secure Communication (TLS) | Supported   | Supported                               |
| JWT Authentication | Not supported | Supported[^12]                           |  

[^6]: REST-like refers to a model similar to RESTful architecture, offering `CoPOST` (Constrained-Post, similar to HTTP-POST) and `ObGET` (Observe-GET, Observer mode) methods, with resources or capabilities identified by URIs.  
[^7]: Observer mode is similar to MQTT message subscriptions but is more flexible, allowing observations to be started and canceled at any time. The caller continues using the HTTP protocol without additional actions (such as `subscriptions`).  
[^8]: RTIO eliminates the need for MQTT's `subscribe` and `publish` processes. In point-to-point communication scenarios like remote device control, it reduces interaction steps by more than half and avoids topic coupling. Refer to [Comparison with MQTT Interactions](#13-comparison-with-mqtt-interaction).  
[^9]: RTIO is lightweight and can run on devices with tens to hundreds of kilobytes of RAM. Its URI transmission uses a hash-compressed representation, which is only 4 bytes long, improving communication efficiency.  
[^10]: RTIO does not implement QoS. Reliability relies on TCP, and if a message cannot be delivered within the timeout, an error is immediately reported to the caller.  
[^11]: The RTIO IoT device SDK automatically reconnects after a network disconnection.  
[^12]: HTTP callers can authenticate with JWT. Only requests with valid JWTs can access RTIO services to initiate calls to devices.  
