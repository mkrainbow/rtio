# JWT Issuance Service

> English | [简体中文](./cn/http_jwtissuer.md)  
> The author's native language is Chinese. This document is translated using AI.

This service is not a necessary part of RTIO but demonstrates how to issue JWTs for RTIO to validate HTTP requests. JWTs can also be issued through other services; see [Issuing via Other Services](#issuing-via-other-services) for reference.

## JWT Certificate Issuance Interface

### URL

```text
http://$HOST/jwtissuer
```

### Request Parameters

The parameters are encoded as a JSON string.

| Parameter  | Type   | Length  | Required | Description                                   |
|:-----------|:-------|:--------|:---------|:----------------------------------------------|
| method     | string | 1-10    | Yes      | The `method` is `jwtissuer` |
| id         | uint32 | -       | Yes      | Request identifier, must be unique for each request; this field will match in the response |
| deviceid   | string | 30-40   | Yes      | Device ID                                    |
| expires    | uint32 | 1-604800| No       | Expiration time in seconds, maximum of 7 days; default is 7 days |

### Response Parameters

The response is also encoded as a JSON string.

| Parameter | Type   | Length  | Required | Description                                   |
|:----------|:-------|:--------|:---------|:----------------------------------------------|
| code      | string | 0-64    | Yes      | Error code                                    |
| id        | uint32 | -       | Yes      | Response identifier, matches the request      |
| jwt       | string | 0-160   | No       | JWT string                                    |

## Certificate Generation

Currently, only the Ed25519 signing algorithm is supported for generating Ed25519 key pairs.

```sh
$ openssl version
OpenSSL 3.0.13 30 Jan 2024 (Library: OpenSSL 3.0.13 30 Jan 2024)

$ openssl genpkey -algorithm Ed25519 -out private.pem
$ openssl pkey -in private.pem -pubout -out public.pem
```

Example certificates have been generated and can be found in the `./out/examples/certificates/` directory.

## Issuance Service

Start the issuance service.

```sh
./out/examples/jwtissuer -private.ed25519 ./out/examples/certificates/ed25519.private
```

## Requesting the Issuance Service via `curl`

The following example issues a JWT for the device ID "cfa09baa-4913-4ad7-a936-3e26f9671b09" with a validity of 1 day (86400 seconds).

```sh
$ curl http://localhost:17019/jwtissuer -d '{"method":"jwtissuer", "id":12667,"deviceid":"cfa09baa-4913-4ad7-a936-3e26f9671b09", "expires": 86400}'
{"id":12667,"code":"OK","jwt":"eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjY0NjQ1MzAsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0zZTI2Zjk2NzFiMDkifQ.sM80Zu3nMsmahzmLpmSC4GlsI8G0xKbnIk8kZIJvLH9IbadEc3sOM3tSb7m_L_ZY2eWy4Ipl8EiYS7t9y_NmCA"}
```

## Issuing via Other Services

When RTIO performs HTTP JWT verification, the subject (`sub`) must match the device ID and be a valid JWT. The main claims are as follows (in Golang code). The algorithm only supports `ed25519`.

```go
claims := jwt.NewWithClaims(&jwt.SigningMethodEd25519{},
    jwt.MapClaims{
        "iss": "rtio",  // not checked
        "sub": req.DeviceID,
        "exp": time.Now().Unix() + int64(req.Expires),
    })
```
