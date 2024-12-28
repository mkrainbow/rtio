# 设备服务

> 简体中文 | [English](../http_deviceservice.md)

## 设备服务接口

RTIO代理服务将物联网设备请求转发给设备服务。使用HTTP通信，POST方法发送。

URL示例：

```text
http://$HOST/deviceservice
```

请求参数，编码为JSON字符串。

|参数 |类型   |长度|必选 | 描述|
|:---|:------|:-------|:---|:-----|
| method|string | 1-10  |是|这里method为`call`|
| id |uint32 |-   |是|请求标识，每个请求唯一，响应中该字段会与之匹配|
| deviceid|string | 30-40  |是|设备ID|
| data |base64 | 0-5462|否|为base64字符串|

响应参数，编码为JSON字符串。

|参数 |类型   |长度|必选 | 描述|
|:---|:------|:-------|:---|:-----|
| code|string | 0-64 |是|错误码|
| id |uint32 | - |是|应答标识，与请求中该字段匹配|
| data |base64 | 0-5462|否|为base64字符串|

## 错误码

以下为RTIO错误码，HTTP作为RTIO的传输层，通常HTTP响应码为200时才能正确返回JSON数据。

| code                    | 描述      |
|:------------------------|:---------|
| INTERNAL_SERVER_ERROR  | 内部错误  |
| OK                     | 成功      |
| BAD_REQUEST            | 请求数据无效 |
| METHOD_NOT_ALLOWED     | 请求设备方法错误 |

```sh
$ echo -n "hello bb" | base64
aGVsbG8gYmI=

$ curl http://localhost:17517/deviceservice/aa/bb -d '{"method":"call","id":1999,"deviceid":"cfa09baa-4913-4ad7-a936-2e26f9671b05","data":"aGVsbG8gYmI="}'
{"id":1999,"code":"OK","data":"ZGV2aWNlc2VydmljZTogcmVzcG9uZSB3aXRoIGJi"}

$ echo -n "ZGV2aWNlc2VydmljZTogcmVzcG9uZSB3aXRoIGJi" | base64 -d
deviceservice: respone with bb
```
