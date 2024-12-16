# 设备认证服务

> 简体中文 | [English](../http_deviceverifier.md)


## 设备认证接口

RTIO使能设备验证功能后，通过该接口向认证服务器发起请求，验证设备是否合法。使用HTTP通信，POST方法发送。

URL示例：

```text
http://$HOST/deviceverifier
```

请求参数，编码为JSON字符串。

|参数 |类型   |长度|必选 | 描述|
|:---|:------|:-------|:---|:-----|
| method|string | 1-10  |是|这里method为`verify`|
| id |uint32 |-   |是|请求标识，每个请求唯一，响应中该字段会与之匹配|
| deviceid|string | 30-40  |是|设备ID|
| devicesecret|string |3-128  |是|设备密钥|

响应参数，编码为JSON字符串。

|参数 |类型   |长度|必选 | 描述|
|:---|:------|:-------|:---|:-----|
| code|string | 0-128 |是|错误码|
| id |uint32 | - |是|应答标识，与请求中该字段匹配|

## 错误码

以下为RTIO错误码，HTTP作为RTIO的传输层，通常HTTP响应码为200时才能正确返回JSON数据。

| code                   | 描述      |
|:-----------------------|:---------|
| INTERNAL_SERVER_ERROR  | 内部错误  |
| OK                     | 成功      |
| NOT_FOUND              | 找不到设备或设备ID无效 |
| BAD_REQUEST            | 请求数据无效 |
| METHOD_NOT_ALLOWED     | 请求设备方法错误 |
| VERIFICATION_FAILED    | 验证失败 |

## 示例

```sh
$ curl http://localhost:17217/deviceverifier -d '{"method":"verify","id": 1999,"deviceid":"cfa09baa-4913-4ad7-a936-2e26f9671b05", "devicesecret": "mb6bgso4EChvyzA05thF9+wH"}'
{"id":1999,"code":"OK"}

$ curl http://localhost:17217/deviceverifier -d '{"method":"verify","id": 1999,"deviceid":"cfa09baa-4913-4ad7-a936-2e26f9671b05", "devicesecret": ""}'
{"id":1999,"code":"VERIFICATION_FAILED"}

```
