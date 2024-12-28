# Device Service

> English | [简体中文](./cn/http_deviceservice.md)  
> The author's native language is Chinese. This document is translated using AI.

## Device Service Interface

The RTIO proxy service forwards IoT device requests to the device service. The interface uses HTTP communication with the POST method.

### URL

```text
http://$HOST/deviceservice
```

### Request Parameters

The parameters are encoded as a JSON string.

| Parameter | Type   | Length  | Required | Description                                   |
|:----------|:-------|:--------|:---------|:----------------------------------------------|
| method    | string | 1-10    | Yes      | The `method` is `call` |
| id        | uint32 | -       | Yes      | Request identifier, must be unique for each request; this field will match in the response |
| deviceid  | string | 30-40   | Yes      | Device ID                                     |
| data      | base64 | 0-5462  | No       | A base64-encoded string                       |

### Response Parameters

The response is also encoded as a JSON string.

| Parameter | Type   | Length  | Required | Description                                   |
|:----------|:-------|:--------|:---------|:----------------------------------------------|
| code      | string | 0-64    | Yes      | Error code                                    |
| id        | uint32 | -       | Yes      | Response identifier, matches the request      |
| data      | base64 | 0-5462  | No       | A base64-encoded string                       |

## Error Codes

The following are the RTIO error codes. The HTTP response code should typically be 200 for the JSON data to be returned correctly.

| Code                     | Description                            |
|:-------------------------|:---------------------------------------|
| INTERNAL_SERVER_ERROR    | Internal error                         |
| OK                       | Success                                |
| BAD_REQUEST              | Invalid request data                   |
| METHOD_NOT_ALLOWED       | Incorrect method for device request    |

### Example

```sh
$ echo -n "hello bb" | base64
aGVsbG8gYmI=

$ curl http://localhost:17517/deviceservice/aa/bb -d '{"method":"call","id":1999,"deviceid":"cfa09baa-4913-4ad7-a936-2e26f9671b05","data":"aGVsbG8gYmI="}'
{"id":1999,"code":"OK","data":"ZGV2aWNlc2VydmljZTogcmVzcG9uZSB3aXRoIGJi"}

$ echo -n "ZGV2aWNlc2VydmljZTogcmVzcG9uZSB3aXRoIGJi" | base64 -d
deviceservice: respone with bb
```
