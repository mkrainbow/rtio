# 1. Device Access Protocol

> English | [简体中文](./cn/device_access_protocol.md)  
> The author's native language is Chinese. This document is translated using AI.

- [1. Device Access Protocol](#1-device-access-protocol)
  - [1.1. Message Types](#11-message-types)
  - [1.2. Message Format](#12-message-format)
  - [1.3. Device Verification](#13-device-verification)
    - [1.3.1. Request](#131-request)
    - [1.3.2. Response](#132-response)
  - [1.4. Device Heartbeat](#14-device-heartbeat)
    - [1.4.1. Request](#141-request)
    - [1.4.2. Response](#142-response)
  - [1.5. Sending Messages](#15-sending-messages)
    - [1.5.1. Request](#151-request)
    - [1.5.2. Response](#152-response)
  - [1.6. REST-Like Communication Layer](#16-rest-like-communication-layer)
  - [REST-Like Methods](#rest-like-methods)
    - [1.6.1 ConstrainedPost](#161-constrainedpost)
    - [1.6.2 ObservedGet](#162-observedget)
  - [1.7. Status Code Description](#17-status-code-description)
  - [1.8. Response Code Description](#18-response-code-description)

## 1.1. Message Types

| Type Name           | Type Value | Description        | Direction               |
|:---------------------|:----------|:-------------------|:------------------------|
| DeviceVerifyReq      | 1         | Device verification request | Device -> Server         |
| DeviceVerifyResp     | 2         | Device verification response | Server -> Device         |
| DevicePingReq        | 3         | Device heartbeat request | Device -> Server         |
| DevicePingResp       | 4         | Device heartbeat response | Server -> Device         |
| DeviceSendReq        | 5         | Send message request | Device -> Server         |
| DeviceSendResp       | 6         | Send message response | Server -> Device         |
| ServerSendReq        | 7         | Send message request | Server -> Device         |
| ServerSendResp       | 8         | Send message response | Device -> Server         |

## 1.2. Message Format

```text
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  | Type  |V|Code |          MessageID            |          Body-     
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   Length         |                    Body...                    
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Type**: 4-bit message type
- **V (Version)**: 1-bit protocol version, currently fixed at 0
- **Code**: 3-bit response code, requests (*Req) are fixed at 0, responses (*Resp) reference [Status Code Description](#17-status-code-description)
- **MessageID**: 16-bit message identifier, "0" is invalid, used to match requests and responses (the same pair of Req and Resp has the same value, which increments by 1 with each request)
- **BodyLength**: 16-bit length of the message body (Payload) in bytes
- **Body**: Message body (Payload)

Messages consist of a header and a body, with the first 5 bytes forming the message header (Message Header) followed by the message body (Message Body). The body length is specified by the BodyLength field in the header.

## 1.3. Device Verification

### 1.3.1. Request

- The header Type is DeviceVerifyReq.
- The header BodyLength is the length of VerifyData (in bytes).

```text
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |CL | Reserves  |     VerifyData...       
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Body** defines the following parts:
  - **Specifics**: 8-bit device attributes
    - **CL (Capacity Level)**: 2-bit maximum payload capacity of the body, **currently only supports 0**
      - 0 - 512 bytes
      - 1 - 1024 bytes
      - 2 - 2048 bytes
      - 3 - 4096 bytes
    - **Reserves**: Reserved field
- **VerifyData**: A concatenation of device ID and secret (deviceID:deviceSecret), connected by ":", with a total length not exceeding 512 bytes.

### 1.3.2. Response

- The header Type is DeviceVerifyResp.
- The header MessageID must match the request.
- The header Code is the response code.
- The body is empty.

**Note**: If verification is not completed within 15 seconds after the connection is established, the server will disconnect without returning response data.

## 1.4. Device Heartbeat

### 1.4.1. Request

- The header Type is DevicePingReq.
- The header BodyLength of 0 indicates a default heartbeat interval of 300 seconds; otherwise, it should be 2 (2-byte body).

```text
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |    TimeOut (option)           |   
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Body** defines the following part:
  - **TimeOut**: 16-bit heartbeat sending interval (optional), ranging from 30 to 43200 seconds (30 seconds to 12 hours). The server considers the client disconnected if it does not receive a heartbeat for more than 1.5 times the TimeOut; incorrect parameter settings return “Invalid Parameter.”

### 1.4.2. Response

- The header Type is DevicePingResp.
- The header MessageID must match the request.
- The header Code is the response code.

**Note**: After the connection is established and verification is successful, if heartbeats are not sent at intervals and no data communication occurs, the server will disconnect without returning response data.

## 1.5. Sending Messages

The message types for ServerSendReq and DeviceSendReq are used to send messages to each other and respond accordingly. However, they are generally not used directly at the application layer, serving as a lower-level transport protocol.

### 1.5.1. Request

- The header Type is DeviceSendReq or ServerSendReq.
- The header BodyLength is the byte count of the body, with a maximum value defined by the capacity specified in DeviceVerifyReq.

```text
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |    Data...  
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Body** defines the following part:
  - **Data**: The data being sent.

### 1.5.2. Response

- The header Type is the corresponding DeviceSendResp or ServerSendResp.
- The header MessageID must match the request.
- The header Code is the response code.
- The header BodyLength is the byte count of the body, with a maximum value defined by the capacity specified in DeviceVerifyReq.

```text
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |    Data...  
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Body** defines the following part:
  - **Data**: The data returned in the response.

## 1.6. REST-Like Communication Layer

This layer provides a REST-style API to the application layer. It is based on message types DeviceSendReq and ServerSendReq and their responses, implementing “Http to Machine” and “Machine to Server” models.

## REST-Like Methods

| Type Name           | Type Value | Description        | Remarks               |
|:---------------------|:----------|:-------------------|:----------------------|
| ConstrainedPost      | 2         | Modify resource (constrained mode) | Supports U2M and M2S |
| ObservedGet          | 3         | Retrieve resource (observer mode) | Only supports U2M     |

In constrained mode, the maximum body length is limited to the capacity specified in DeviceVerifyReq.

### 1.6.1 ConstrainedPost

Request Message:

```text
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |Method |Reserve|                                    URIDigest
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
                  |   Data...                    
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Method**: 4-bit request method.
- **Reserve**: 4-bit reserved.
- **URIDigest**: 32-bit resource address, computed using CRC32 on the string form of the URI.
- **Data**: Request content.

Response

 Message:

```text
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |Method |Status |   Data...        
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Method**: 4-bit request method.
- **Status**: 4-bit status code, refer to StatusCode description section.
- **Data**: The data carried in the response.

### 1.6.2 ObservedGet

The observe mode is typically used for users to monitor devices, such as continuously retrieving sensor temperatures. Either party can terminate the interaction by setting the `StatusCode` field in the message to `Terminate`.

**obGetEstebReq**:

```text
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |Method |Reserve|          ObserverID           |                        
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   URIDigest                                      |   Data...                    
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Method**: 4-bit request method.
- **Reserve**: 4-bit reserved.
- **ObserverID**: 16-bit observer ID, generated by the server; "0" is considered an invalid ID.
- **URIDigest**: 32-bit resource address, computed using CRC32 on the string form of the URI.
- **Data**: Request content.

**obGetEstebResp**:

```text
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |Method |Status |          ObserverID           |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Method**: 4-bit request method.
- **Status**: 4-bit status code, refer to StatusCode description section.
- **ObserverID**: 16-bit observer ID.

**obGetNotifyReq**:

```text
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |Method |Status |          ObserverID           | Data...
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Method**: 4-bit request method.
- **Status**: 4-bit status code, refer to StatusCode description section.
- **ObserverID**: 16-bit observer ID.
- **Data**: The data being sent.

**obGetNotifyResp**:

```text
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |Method |Status |          ObserverID           |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Method**: 4-bit request method.
- **Status**: 4-bit status code, refer to StatusCode description section.
- **ObserverID**: 16-bit observer ID.

The communication process between the device and the cloud is illustrated as follows:

```text
  +----------+                        +------------+
  |  device  |                        |   server   | 
  +----------+                        +------------+
       |                                    |
       |  obGetEstebReq over ServerSendReq  | 
       |<-----------------------------------| 
       | obGetEstebResp over ServerSendResp | 
       |----------------------------------->| 
       |                                    |   
       | obGetNotifyReq over DeviceSendReq  | 
       |----------------------------------->| 
       |obGetNotifyResp over DeviceSendResp | 
       |<-----------------------------------|  
       | obGetNotifyReq over DeviceSendReq  | 
       |----------------------------------->| 
       |obGetNotifyResp over DeviceSendResp | 
       |<-----------------------------------|  
  +----------+                        +------------+
  |  device  |                        |   server   | 
  +----------+                        +------------+ 
```

## 1.7. Status Code Description

| StatusCode | Description            | Remarks   |
|:-----------|:-----------------------|:---------|
| 0          | Unknown                | Unknown error |
| 1          | InternalServerError    | Internal error |
| 2          | OK                     | Success   |
| 3          | Continue               | Used to indicate continued notification in observe mode |
| 4          | Terminate              | Used to indicate termination in observe mode |
| 5          | NotFound               | Resource not found |
| 6          | BadRequest             | Invalid request |
| 7          | MethodNotAllowed       | Method error |
| 8          | TooManyRequests        | Too many requests |
| 9          | TooManyObservers       | Too many observers |

## 1.8. Response Code Description

| Code | Description      | Remarks   |
|:-----|:-----------------|:---------|
| 0    | Failure, unknown reason |   |
| 1    | Success          |   |
| 2    | Message type error |   |
| 3    | Verification failed |   |
| 4    | Invalid parameter |   |
| 5    | BodyLength error |   |
