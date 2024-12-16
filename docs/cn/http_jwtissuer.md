# JWT签发服务

> 简体中文 | [English](../http_jwtissuer.md)

该服务非RTIO必要服务，主要演示如何签发JWT，用于RTIO验证HTTP请求。亦可通过其他服务进行JWT签发，参考[通过其他服务签发](#通过其他服务签发)。

- [JWT签发服务](#jwt签发服务)
  - [JWT证书签发接口](#jwt证书签发接口)
  - [证书生成](#证书生成)
  - [签发服务](#签发服务)
  - [通过`curl`请求签发服务](#通过curl请求签发服务)
  - [通过其他服务签发](#通过其他服务签发)

## JWT证书签发接口

URL示例：

```text
http://$HOST/jwtissuer
```

请求参数，编码为JSON字符串。

|参数 |类型   |长度|必选 | 描述|
|:---|:------|:-------|:---|:-----|
| method|string | 1-10  |是|这里method为`jwtissuer`|
| id |uint32 |-   |是|请求标识，每个请求唯一，响应中该字段会与之匹配|
| deviceid|string | 30-40  |是|设备ID|
| expires |uint32 | 1-604800|否|有效时间，单位为秒，最大7天, 默认7天|

响应参数，编码为JSON字符串。

|参数 |类型   |长度|必选 | 描述|
|:---|:------|:-------|:---|:-----|
| code|string | 0-64 |是|错误码|
| id |uint32 | - |是|应答标识，与请求中该字段匹配|
| jwt |string | 0-160|否|jwt字符串|

## 证书生成

目前仅支持ed25519签名算法,生成密ed25519钥对。

```sh
$ openssl version
OpenSSL 3.0.13 30 Jan 2024 (Library: OpenSSL 3.0.13 30 Jan 2024)

$ openssl genpkey -algorithm Ed25519 -out private.pem
$ openssl pkey -in private.pem -pubout -out public.pem
```

示例证书已生成到`./out/examples/certificates/`目录下。

## 签发服务

启动签发服务。

```sh
./out/examples/jwtissuer -private.ed25519 ./out/examples/certificates/ed25519.private
```

## 通过`curl`请求签发服务

以下示例签发JWT，设备ID为"cfa09baa-4913-4ad7-a936-3e26f9671b09" 、有效时间为1天（86400秒）。

```sh
$ curl http://localhost:17019/jwtissuer -d '{"method":"jwtissuer", "id":12667,"deviceid":"cfa09baa-4913-4ad7-a936-3e26f9671b09", "expires": 86400}'
{"id":12667,"code":"OK","jwt":"eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjY0NjQ1MzAsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0zZTI2Zjk2NzFiMDkifQ.sM80Zu3nMsmahzmLpmSC4GlsI8G0xKbnIk8kZIJvLH9IbadEc3sOM3tSb7m_L_ZY2eWy4Ipl8EiYS7t9y_NmCA"}
```

## 通过其他服务签发

RTIO JWT校验时，主题`sub`与deviceID需匹配，同时为合法的JWT。主要声明如下（Golang代码）。算法仅支持`ed25519`。

```go
claims := jwt.NewWithClaims(&jwt.SigningMethodEd25519{},
    jwt.MapClaims{
        "iss": "rtio",  // not checked
        "sub": req.DeviceID,
        "exp": time.Now().Unix() + int64(req.Expires),
    })
```
