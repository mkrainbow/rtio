# Device Authentication Service

> English | [简体中文](./cn/http_deviceverifier.md)  
> The author's native language is Chinese. This document is translated using AI.


## Device Authentication Interface

When RTIO enables device verification, requests can be sent to the authentication server to verify the legitimacy of a device. The communication uses HTTP with the POST method.

### URL

```text
http://$HOST/deviceverifier
```

### Request Parameters

The parameters are encoded as a JSON string.

| Parameter      | Type   | Length  | Required | Description                                   |
|:---------------|:-------|:--------|:---------|:----------------------------------------------|
| method         | string | 1-10    | Yes      | The `method` is `verify` |
| id             | uint32 | -       | Yes      | Request identifier, must be unique for each request; this field will match in the response |
| deviceid       | string | 30-40   | Yes      | Device ID                                     |
| devicesecret   | string | 3-128   | Yes      | Device secret                                 |

### Response Parameters

The response is also encoded as a JSON string.

| Parameter | Type   | Length  | Required | Description                                   |
|:----------|:-------|:--------|:---------|:----------------------------------------------|
| code      | string | 0-128   | Yes      | Error code                                    |
| id        | uint32 | -       | Yes      | Response identifier, matches the request      |

## Error Codes

The following are the RTIO error codes. The HTTP response code should typically be 200 for the JSON data to be returned correctly.

| Code                   | Description                            |
|:-----------------------|:---------------------------------------|
| INTERNAL_SERVER_ERROR  | Internal error                         |
| OK                     | Success                                |
| NOT_FOUND              | Device not found or invalid device ID  |
| BAD_REQUEST            | Invalid request data                   |
| METHOD_NOT_ALLOWED     | Incorrect method for device request    |
| VERIFICATION_FAILED    | Verification failed                    |

## Example

```sh
$ curl http://localhost:17217/deviceverifier -d '{"method":"verify","id": 1999,"deviceid":"cfa09baa-4913-4ad7-a936-2e26f9671b05", "devicesecret": "mb6bgso4EChvyzA05thF9+wH"}'
{"id":1999,"code":"OK"}

$ curl http://localhost:17217/deviceverifier -d '{"method":"verify","id": 1999,"deviceid":"cfa09baa-4913-4ad7-a936-2e26f9671b05", "devicesecret": ""}'
{"id":1999,"code":"VERIFICATION_FAILED"}
```
