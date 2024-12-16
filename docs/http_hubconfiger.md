# DeviceHub Configuration Service

> English | [简体中文](./cn/http_hubconfiger.md)  
> The author's native language is Chinese. This document is translated using AI.

## Interface

Uses HTTP communication with the POST method.

### URL

```text
http://$HOST/hubconfiger
```

### Request Parameters

The parameters are encoded as a JSON string.

| Parameter | Type   | Length  | Required | Description                                   |
|:----------|:-------|:--------|:---------|:----------------------------------------------|
| method    | string | 1-10    | Yes      | The `method` is `getconfig` |
| id        | uint32 | -       | Yes      | Request identifier, must be unique for each request; this field will match in the response |

### Response Parameters

The response is also encoded as a JSON string.

| Parameter | Type   | Length  | Required | Description                                   |
|:----------|:-------|:--------|:---------|:----------------------------------------------|
| code      | string | 0-128   | Yes      | Error code                                    |
| id        | uint32 | -       | Yes      | Response identifier, matches the request      |
| config    | string | 0-2048  | Yes      | JSON-encoded string                           |
| digest    | uint32 | -       | Yes      | The digest of the `config` string, using the CRC32 hash function |

## Error Codes

The following are the RTIO error codes. The HTTP response code should typically be 200 for the JSON data to be returned correctly.

| Code                   | Description                            |
|:-----------------------|:---------------------------------------|
| INTERNAL_SERVER_ERROR  | Internal error                         |
| OK                     | Success                                |
| BAD_REQUEST            | Invalid request data                   |
| METHOD_NOT_ALLOWED     | Incorrect method for device request    |

## Example

```sh
$ curl http://localhost:17317/hubconfiger -d '{"method":"getconfig","id": 1999 }'
{"id":1999,"code":"OK","config":"{\"deviceservicemap\":{\"/aa/bb\":\"http://localhost:17517/deviceservice/aa/bb\",\"/aa/cc\":\"http://localhost:17517/deviceservice/aa/cc\",\"/aa/dd\":\"http://localhost:17517/deviceservice/aa/dd\"}}","digest":785914363}
```
