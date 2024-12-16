# http access protocol

> English | [简体中文](./cn/http_access_protocol.md)  
> The author's native language is Chinese. This document is translated using AI.

## Interface Definition

The interface uses HTTP communication. If the RTIO service has JWT validation enabled, the "Authorization" field must be added to the HTTP request header.

The URL structure is as follows, where HOST is the RTIO service host address and DEVICE_ID is the device identifier:

```text
http://$HOST/$DEVICE_ID
```

### Request Parameters

The parameters are encoded as a JSON string, with a total length limit of 864$^{1}$ bytes.

| Parameter | Type   | Length | Required | Description |
|:----------|:-------|:-------|:---------|:------------|
| method    | string | 1-10   | Yes      | The method to request from the device, only supports `copost` and `obget` |
| id        | int    | -      | Yes      | Request identifier, must be unique for each request; this field will match in the response |
| uri       | string | 3-128  | Yes      | The internal URI of the device, which binds the handler to this URI |
| data      | base64 | 0-672$^{2}$ | No       | A base64-encoded string |

### Response Parameters

The response is also encoded as a JSON string.

| Parameter | Type   | Length | Required | Description |
|:----------|:-------|:-------|:---------|:------------|
| code      | string | 0-64   | Yes      | Error code |
| id        | int    | -      | Yes      | Response identifier, matches the request |
| fid       | int    | -      | No       | Required for `obget` method; frame identifier for each frame |
| data      | base64 | 0-672$^{2}$ | No       | A base64-encoded string |

**Notes:**

1. The RTIO service checks the total length of the JSON string when it receives a request. This includes the lengths of the data, JSON quotes, brackets, field names, etc., and is set to a maximum of 864 bytes.
2. When a device connects to the RTIO service (refer to the device access protocol), the maximum body length for each transmission is agreed upon, currently supporting only 512 bytes. The REST-like communication layer reserves 8 bytes, making the maximum length per communication 504 bytes, which when base64-encoded, results in a length of 672 bytes (504 * 4/3).

## Examples

### Method `copost` Request and Response

**Device Online:**

```sh
$ curl https://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b10 -d '{"method":"copost", "uri":"/greeter","id":12667,"data":"c3RhcnQ="}' 

{"id":12667, "code":"OK", "data":"d29ybGQ="}
```

**Device Offline:**

```sh
$ curl http://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b09 -d '{"method":"copost", "uri":"/greeter","id":12667,"data":"aGVsbG8="}'
{"id":12667,"fid":0,"code":"DEVICEID_OFFLINE","data":""}
```

**Device Timeout:**

```sh
$ curl https://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b10 -d '{"method":"copost", "uri":"/greeter","id":12667,"data":"c3RhcnQ="}' 

{"id":12667, "code":"DEVICEID_TIMEOUT"}
```

### Method `obget` Request and Response

```sh
curl https://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b10 -d '{"method":"obget", "uri":"/greeter","id":12667,"data":"c3RhcnQ="}' 

{"id":12334,"fid":0,"code":"CONTINUE","data":"cHJpbnRpbmcgMzMl"}
{"id":12334,"fid":1,"code":"CONTINUE","data":"cHJpbnRpbmcgMzcl"}
...
{"id":12334,"fid":21,"code":"TERMINATE","data":""}
```

## More Examples

Refer to：[RTIO Demos](./rtio_demos.md)

## Error Codes

The following are the RTIO error codes. The HTTP response code should typically be 200 for the JSON data to be returned correctly.

| code | Definition                 | Description                      |
|:-----|:---------------------------|:---------------------------------|
| 0    | INTERNAL_SERVER_ERROR      | Internal error                   |
| 1    | OK                         | Success                          |
| 2    | DEVICEID_OFFLINE          | Device is offline                |
| 3    | DEVICEID_TIMEOUT          | Device timeout; observers do not timeout, but due to network issues, the device may not respond |
| 4    | CONTINUE                   | In observer mode, there are subsequent data frames |
| 5    | TERMINATE                  | In observer mode, no subsequent data frames |
| 6    | NOT_FOUND                  | URI not found                    |
| 7    | BAD_REQUEST                | Invalid request data             |
| 8    | METHOD_NOT_ALLOWED         | Incorrect method for device request; currently only supports `copost` and `obget` |
| 9    | TOO_MANY_REQUESTS          | Too many requests                |
| 10   | TOO_MANY_OBSERVERS         | Too many observers               |
| 11   | REQUEST_TIMEOUT            | Request timed out                |
